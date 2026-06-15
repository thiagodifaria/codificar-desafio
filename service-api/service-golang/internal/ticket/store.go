package ticket

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNoAssignee indica que não existem responsáveis ativos para receber o chamado.
	ErrNoAssignee = errors.New("no active assignee available")
	// ErrInvalidAssignee indica que o responsável escolhido não existe ou está inativo.
	ErrInvalidAssignee = errors.New("assignee is not available")
	// ErrAssigneeNameConflict indica que já existe um membro com o nome informado.
	ErrAssigneeNameConflict = errors.New("assignee name already exists")
	// ErrAssigneeHasTickets impede remover quem faz parte do histórico de chamados.
	ErrAssigneeHasTickets = errors.New("assignee has tickets")
)

// Store concentra as operações de persistência e as regras transacionais dos chamados.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore cria uma nova instância do repositório de chamados.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Create registra um chamado e resolve seu responsável dentro da mesma transação.
func (s *Store) Create(ctx context.Context, input UpsertInput) (Ticket, error) {
	// A atribuição e a criação precisam ser atômicas para preservar o equilíbrio sob concorrência.
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Ticket{}, fmt.Errorf("begin create ticket transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Resolve a escolha manual ou automática antes de persistir o chamado.
	assigneeID, err := resolveAssignee(ctx, tx, input.AssignmentMode, input.AssigneeID, 0)
	if err != nil {
		return Ticket{}, err
	}

	// Registra a data de conclusão somente para status que encerram o trabalho.
	resolvedAt := resolvedTime(input.Status)
	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO tickets (
			title, description, requester_name, priority, status,
			assignee_id, assignment_mode, resolved_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, input.Title, input.Description, input.RequesterName, input.Priority,
		input.Status, assigneeID, input.AssignmentMode, resolvedAt,
	).Scan(&id)
	if err != nil {
		return Ticket{}, fmt.Errorf("insert ticket: %w", err)
	}

	// Atualiza a rotação para tornar o desempate das próximas atribuições previsível.
	if input.AssignmentMode == AssignmentAutomatic {
		if _, err := tx.Exec(ctx,
			`UPDATE assignees SET last_assigned_at = NOW() WHERE id = $1`,
			assigneeID,
		); err != nil {
			return Ticket{}, fmt.Errorf("update assignee rotation: %w", err)
		}
	}

	// Confirma todas as alterações somente após a criação e a rotação serem concluídas.
	if err := tx.Commit(ctx); err != nil {
		return Ticket{}, fmt.Errorf("commit create ticket: %w", err)
	}
	return s.Get(ctx, id)
}

// Update altera um chamado existente e permite recalcular sua atribuição.
func (s *Store) Update(ctx context.Context, id int64, input UpsertInput) (Ticket, error) {
	// Mantém leitura, redistribuição e atualização dentro de uma única transação.
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Ticket{}, fmt.Errorf("begin update ticket transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Bloqueia o chamado para impedir alterações concorrentes sobre a mesma versão.
	var (
		lockedTicketID        int64
		currentAssigneeID     int64
		currentAssignmentMode string
		currentStatus         string
		currentResolvedAt     *time.Time
	)
	if err := tx.QueryRow(ctx,
		`SELECT id, assignee_id, assignment_mode, status, resolved_at
		 FROM tickets
		 WHERE id = $1
		 FOR UPDATE`,
		id,
	).Scan(
		&lockedTicketID,
		&currentAssigneeID,
		&currentAssignmentMode,
		&currentStatus,
		&currentResolvedAt,
	); errors.Is(err, pgx.ErrNoRows) {
		return Ticket{}, ErrNotFound
	} else if err != nil {
		return Ticket{}, fmt.Errorf("lock ticket: %w", err)
	}

	// Mantém o responsável atual até que uma troca manual ou redistribuição seja solicitada.
	assigneeID := currentAssigneeID
	redistributed := false
	if input.AssignmentMode == AssignmentManual {
		assigneeID, err = resolveAssignee(ctx, tx, AssignmentManual, input.AssigneeID, id)
		if err != nil {
			return Ticket{}, err
		}
	} else if currentAssignmentMode != AssignmentAutomatic || input.Redistribute {
		// Desconsidera o próprio chamado no cálculo para não inflar a carga atual do responsável.
		assigneeID, err = resolveAssignee(ctx, tx, AssignmentAutomatic, 0, id)
		if err != nil {
			return Ticket{}, err
		}
		redistributed = true
	}

	commandTag, err := tx.Exec(ctx, `
		UPDATE tickets
		SET title = $2,
			description = $3,
			requester_name = $4,
			priority = $5,
			status = $6,
			assignee_id = $7,
			assignment_mode = $8,
			resolved_at = $9,
			updated_at = NOW()
		WHERE id = $1
	`, id, input.Title, input.Description, input.RequesterName, input.Priority,
		input.Status, assigneeID, input.AssignmentMode,
		transitionResolvedTime(currentStatus, input.Status, currentResolvedAt),
	)
	if err != nil {
		return Ticket{}, fmt.Errorf("update ticket: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return Ticket{}, ErrNotFound
	}

	// Atualiza a posição na rotação somente quando houve uma nova distribuição.
	if redistributed {
		if _, err := tx.Exec(ctx,
			`UPDATE assignees SET last_assigned_at = NOW() WHERE id = $1`,
			assigneeID,
		); err != nil {
			return Ticket{}, fmt.Errorf("update assignee rotation: %w", err)
		}
	}

	// Finaliza a transação somente após todas as etapas terem sido executadas.
	if err := tx.Commit(ctx); err != nil {
		return Ticket{}, fmt.Errorf("commit update ticket: %w", err)
	}
	return s.Get(ctx, id)
}

// ChangeStatus altera apenas o andamento do chamado e preserva as demais informações.
func (s *Store) ChangeStatus(ctx context.Context, id int64, status string) (Ticket, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Ticket{}, fmt.Errorf("begin status transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Bloqueia o chamado para calcular a transição sobre o estado mais recente.
	var (
		currentStatus     string
		currentResolvedAt *time.Time
	)
	if err := tx.QueryRow(ctx, `
		SELECT status, resolved_at
		FROM tickets
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&currentStatus, &currentResolvedAt); errors.Is(err, pgx.ErrNoRows) {
		return Ticket{}, ErrNotFound
	} else if err != nil {
		return Ticket{}, fmt.Errorf("lock ticket status: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE tickets
		SET status = $2,
			resolved_at = $3,
			updated_at = NOW()
		WHERE id = $1
	`, id, status, transitionResolvedTime(currentStatus, status, currentResolvedAt)); err != nil {
		return Ticket{}, fmt.Errorf("update ticket status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Ticket{}, fmt.Errorf("commit ticket status: %w", err)
	}
	return s.Get(ctx, id)
}

// resolveAssignee seleciona e valida o responsável conforme o modo de atribuição.
func resolveAssignee(
	ctx context.Context,
	tx pgx.Tx,
	mode string,
	requestedID int64,
	excludedTicketID int64,
) (int64, error) {
	// Na atribuição manual, apenas confirma que a pessoa escolhida está disponível.
	if mode == AssignmentManual {
		var active bool
		err := tx.QueryRow(ctx,
			`SELECT active FROM assignees WHERE id = $1`,
			requestedID,
		).Scan(&active)
		if errors.Is(err, pgx.ErrNoRows) || !active {
			return 0, ErrInvalidAssignee
		}
		if err != nil {
			return 0, fmt.Errorf("validate assignee: %w", err)
		}
		return requestedID, nil
	}

	// Bloqueia todos os responsáveis ativos para serializar o trecho crítico da distribuição.
	// Assim, duas requisições simultâneas não escolhem a mesma carga desatualizada.
	rows, err := tx.Query(ctx,
		`SELECT id FROM assignees WHERE active = TRUE ORDER BY id FOR UPDATE`,
	)
	if err != nil {
		return 0, fmt.Errorf("lock assignees: %w", err)
	}
	hasAssignee := false
	for rows.Next() {
		hasAssignee = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("read locked assignees: %w", err)
	}
	rows.Close()
	if !hasAssignee {
		return 0, ErrNoAssignee
	}

	// Seleciona a menor carga ativa e desempata por quem está há mais tempo sem receber chamado.
	var assigneeID int64
	err = tx.QueryRow(ctx, `
		SELECT a.id
		FROM assignees a
		LEFT JOIN tickets t
			ON t.assignee_id = a.id
			AND t.status IN ('open', 'in_progress')
			AND ($1::BIGINT = 0 OR t.id <> $1)
		WHERE a.active = TRUE
		GROUP BY a.id, a.last_assigned_at
		ORDER BY COUNT(t.id), a.last_assigned_at NULLS FIRST, a.id
		LIMIT 1
	`, excludedTicketID).Scan(&assigneeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNoAssignee
	}
	if err != nil {
		return 0, fmt.Errorf("select least loaded assignee: %w", err)
	}
	return assigneeID, nil
}

// Get busca um chamado pelo identificador e inclui o nome do responsável.
func (s *Store) Get(ctx context.Context, id int64) (Ticket, error) {
	ticket, err := scanTicket(s.pool.QueryRow(ctx, `
		SELECT
			t.id, t.title, t.description, t.requester_name, t.priority,
			t.status, t.assignee_id, a.name, t.assignment_mode,
			t.opened_at, t.resolved_at, t.created_at, t.updated_at
		FROM tickets t
		JOIN assignees a ON a.id = t.assignee_id
		WHERE t.id = $1
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Ticket{}, ErrNotFound
	}
	if err != nil {
		return Ticket{}, fmt.Errorf("get ticket: %w", err)
	}
	return ticket, nil
}

// List retorna os chamados conforme os filtros informados.
func (s *Store) List(ctx context.Context, filter ListFilter) ([]Ticket, error) {
	orderBy := `
		CASE t.priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END,
		t.opened_at DESC`
	switch filter.Sort {
	case "opened_asc":
		orderBy = "t.opened_at ASC"
	case "opened_desc":
		orderBy = "t.opened_at DESC"
	case "updated_desc":
		orderBy = "t.updated_at DESC"
	case "priority_desc", "":
		// Mantém a ordenação operacional padrão: prioridade e depois antiguidade.
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}

	// A consulta permite combinar busca textual, status, prioridade e responsável.
	query := `
		SELECT
			t.id, t.title, t.description, t.requester_name, t.priority,
			t.status, t.assignee_id, a.name, t.assignment_mode,
			t.opened_at, t.resolved_at, t.created_at, t.updated_at
		FROM tickets t
		JOIN assignees a ON a.id = t.assignee_id
		WHERE (
			$1::TEXT = ''
			OR t.title ILIKE '%' || $1 || '%'
			OR t.description ILIKE '%' || $1 || '%'
			OR t.requester_name ILIKE '%' || $1 || '%'
		)
		AND ($2::TEXT = '' OR t.status = $2)
		AND ($3::TEXT = '' OR t.priority = $3)
		AND ($4::BIGINT = 0 OR t.assignee_id = $4)
		ORDER BY ` + orderBy + `
		LIMIT $5`
	rows, err := s.pool.Query(ctx, query,
		filter.Search,
		filter.Status,
		filter.Priority,
		filter.AssigneeID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	defer rows.Close()

	// Converte cada linha retornada para o modelo exposto pela API.
	tickets := make([]Ticket, 0)
	for rows.Next() {
		item, err := scanTicket(rows)
		if err != nil {
			return nil, fmt.Errorf("scan ticket list: %w", err)
		}
		tickets = append(tickets, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tickets: %w", err)
	}
	return tickets, nil
}

// ListAssignees retorna os responsáveis ativos e suas respectivas cargas.
func (s *Store) ListAssignees(ctx context.Context) ([]Assignee, error) {
	// Apenas chamados abertos ou em andamento compõem a carga ativa.
	rows, err := s.pool.Query(ctx, `
		SELECT
			a.id,
			a.name,
			a.active,
			COUNT(t.id) FILTER (WHERE t.status IN ('open', 'in_progress')),
			COUNT(t.id) FILTER (WHERE t.status IN ('resolved', 'closed')),
			a.last_assigned_at
		FROM assignees a
		LEFT JOIN tickets t ON t.assignee_id = a.id
		WHERE a.active = TRUE
		GROUP BY a.id
		ORDER BY a.name
	`)
	if err != nil {
		return nil, fmt.Errorf("list assignees: %w", err)
	}
	defer rows.Close()

	// Monta a lista usada nos formulários e no painel de distribuição.
	assignees := make([]Assignee, 0)
	for rows.Next() {
		var assignee Assignee
		if err := rows.Scan(
			&assignee.ID,
			&assignee.Name,
			&assignee.Active,
			&assignee.OpenTickets,
			&assignee.CompletedTickets,
			&assignee.LastAssignedAt,
		); err != nil {
			return nil, fmt.Errorf("scan assignee: %w", err)
		}
		assignees = append(assignees, assignee)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assignees: %w", err)
	}
	return assignees, nil
}

// CreateAssignee registra um membro e devolve sua carga inicial.
func (s *Store) CreateAssignee(ctx context.Context, input AssigneeInput) (Assignee, error) {
	var assignee Assignee
	err := s.pool.QueryRow(ctx, `
		INSERT INTO assignees (name)
		VALUES ($1)
		RETURNING id, name, active, 0, 0, last_assigned_at
	`, input.Name).Scan(
		&assignee.ID,
		&assignee.Name,
		&assignee.Active,
		&assignee.OpenTickets,
		&assignee.CompletedTickets,
		&assignee.LastAssignedAt,
	)
	if isUniqueViolation(err) {
		return Assignee{}, ErrAssigneeNameConflict
	}
	if err != nil {
		return Assignee{}, fmt.Errorf("create assignee: %w", err)
	}
	return assignee, nil
}

// UpdateAssignee altera nome e disponibilidade sem remover o histórico do membro.
func (s *Store) UpdateAssignee(ctx context.Context, id int64, input AssigneeInput) (Assignee, error) {
	command, err := s.pool.Exec(ctx, `
		UPDATE assignees
		SET name = $2
		WHERE id = $1
	`, id, input.Name)
	if isUniqueViolation(err) {
		return Assignee{}, ErrAssigneeNameConflict
	}
	if err != nil {
		return Assignee{}, fmt.Errorf("update assignee: %w", err)
	}
	if command.RowsAffected() == 0 {
		return Assignee{}, ErrNotFound
	}

	assignees, err := s.ListAssignees(ctx)
	if err != nil {
		return Assignee{}, err
	}
	for _, assignee := range assignees {
		if assignee.ID == id {
			return assignee, nil
		}
	}
	return Assignee{}, ErrNotFound
}

// DeleteAssignee remove um membro somente quando nenhum chamado depende dele.
func (s *Store) DeleteAssignee(ctx context.Context, id int64) error {
	command, err := s.pool.Exec(ctx, `DELETE FROM assignees WHERE id = $1`, id)
	if isForeignKeyViolation(err) {
		return ErrAssigneeHasTickets
	}
	if err != nil {
		return fmt.Errorf("delete assignee: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Dashboard consolida os totais por status e a carga atual dos responsáveis.
func (s *Store) Dashboard(ctx context.Context) (Dashboard, error) {
	// Calcula todos os indicadores de status em uma única consulta.
	var dashboard Dashboard
	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'open'),
			COUNT(*) FILTER (WHERE status = 'in_progress'),
			COUNT(*) FILTER (WHERE status = 'resolved'),
			COUNT(*) FILTER (WHERE status = 'closed')
		FROM tickets
	`).Scan(
		&dashboard.Total,
		&dashboard.Open,
		&dashboard.InProgress,
		&dashboard.Resolved,
		&dashboard.Closed,
	)
	if err != nil {
		return Dashboard{}, fmt.Errorf("load dashboard totals: %w", err)
	}

	// Reaproveita a mesma leitura de responsáveis utilizada pelo restante da aplicação.
	dashboard.Assignees, err = s.ListAssignees(ctx)
	if err != nil {
		return Dashboard{}, err
	}

	// Exibe quem receberia o próximo chamado automático sem bloquear a operação.
	dashboard.NextAssignee, err = s.nextAssignee(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	return dashboard, nil
}

// nextAssignee calcula de forma informativa quem receberia a próxima atribuição.
func (s *Store) nextAssignee(ctx context.Context) (*Assignee, error) {
	var assignee Assignee
	err := s.pool.QueryRow(ctx, `
		SELECT
			a.id,
			a.name,
			a.active,
			COUNT(t.id) FILTER (WHERE t.status IN ('open', 'in_progress')),
			COUNT(t.id) FILTER (WHERE t.status IN ('resolved', 'closed')),
			a.last_assigned_at
		FROM assignees a
		LEFT JOIN tickets t ON t.assignee_id = a.id
		WHERE a.active = TRUE
		GROUP BY a.id
		ORDER BY
			COUNT(t.id) FILTER (WHERE t.status IN ('open', 'in_progress')),
			a.last_assigned_at NULLS FIRST,
			a.id
		LIMIT 1
	`).Scan(
		&assignee.ID,
		&assignee.Name,
		&assignee.Active,
		&assignee.OpenTickets,
		&assignee.CompletedTickets,
		&assignee.LastAssignedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select next assignee: %w", err)
	}
	return &assignee, nil
}

// rowScanner abstrai o Scan compartilhado entre consultas de uma linha e múltiplas linhas.
type rowScanner interface {
	Scan(dest ...any) error
}

// isUniqueViolation identifica conflitos de unicidade retornados pelo PostgreSQL.
func isUniqueViolation(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) && databaseError.Code == "23505"
}

// isForeignKeyViolation identifica remoções bloqueadas por vínculos existentes.
func isForeignKeyViolation(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) && databaseError.Code == "23503"
}

// scanTicket converte uma linha do PostgreSQL para o modelo Ticket.
func scanTicket(row rowScanner) (Ticket, error) {
	var item Ticket
	err := row.Scan(
		&item.ID,
		&item.Title,
		&item.Description,
		&item.RequesterName,
		&item.Priority,
		&item.Status,
		&item.AssigneeID,
		&item.AssigneeName,
		&item.AssignmentMode,
		&item.OpenedAt,
		&item.ResolvedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

// resolvedTime define a data de conclusão conforme o status informado.
func resolvedTime(status string) *time.Time {
	if !isCompleted(status) {
		return nil
	}
	now := time.Now().UTC()
	return &now
}

// transitionResolvedTime preserva o primeiro encerramento e limpa a data ao reabrir.
func transitionResolvedTime(currentStatus, nextStatus string, currentResolvedAt *time.Time) *time.Time {
	if !isCompleted(nextStatus) {
		return nil
	}
	if isCompleted(currentStatus) && currentResolvedAt != nil {
		return currentResolvedAt
	}
	return resolvedTime(nextStatus)
}

// isCompleted informa se o status representa trabalho concluído.
func isCompleted(status string) bool {
	return status == StatusResolved || status == StatusClosed
}
