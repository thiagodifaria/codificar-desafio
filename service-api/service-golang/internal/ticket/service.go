package ticket

import "context"

// Service define os casos de uso oferecidos pelo módulo de chamados.
type Service interface {
	Create(ctx context.Context, input UpsertInput) (Ticket, error)
	Update(ctx context.Context, id int64, input UpsertInput) (Ticket, error)
	ChangeStatus(ctx context.Context, id int64, input StatusInput) (Ticket, error)
	Get(ctx context.Context, id int64) (Ticket, error)
	List(ctx context.Context, filter ListFilter) ([]Ticket, error)
	ListAssignees(ctx context.Context) ([]Assignee, error)
	CreateAssignee(ctx context.Context, input AssigneeInput) (Assignee, error)
	UpdateAssignee(ctx context.Context, id int64, input AssigneeInput) (Assignee, error)
	DeleteAssignee(ctx context.Context, id int64) error
	Dashboard(ctx context.Context) (Dashboard, error)
}

type service struct {
	repository Repository
}

// NewService cria o serviço responsável pelas validações e casos de uso.
func NewService(repository Repository) Service {
	return &service{repository: repository}
}

// Create normaliza e valida os dados antes de registrar um chamado.
func (s *service) Create(ctx context.Context, input UpsertInput) (Ticket, error) {
	input.Normalize()
	if err := input.Validate(); err != nil {
		return Ticket{}, err
	}
	return s.repository.Create(ctx, input)
}

// Update normaliza e valida os dados antes de alterar um chamado.
func (s *service) Update(ctx context.Context, id int64, input UpsertInput) (Ticket, error) {
	input.Normalize()
	if err := input.Validate(); err != nil {
		return Ticket{}, err
	}
	return s.repository.Update(ctx, id, input)
}

// ChangeStatus valida e executa uma transição direta de status.
func (s *service) ChangeStatus(ctx context.Context, id int64, input StatusInput) (Ticket, error) {
	input.Normalize()
	if err := input.Validate(); err != nil {
		return Ticket{}, err
	}
	return s.repository.ChangeStatus(ctx, id, input.Status)
}

// Get retorna um chamado pelo identificador.
func (s *service) Get(ctx context.Context, id int64) (Ticket, error) {
	return s.repository.Get(ctx, id)
}

// List retorna os chamados de acordo com os filtros informados.
func (s *service) List(ctx context.Context, filter ListFilter) ([]Ticket, error) {
	if filter.Limit < 0 {
		filter.Limit = 0
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	return s.repository.List(ctx, filter)
}

// ListAssignees retorna os responsáveis ativos.
func (s *service) ListAssignees(ctx context.Context) ([]Assignee, error) {
	return s.repository.ListAssignees(ctx)
}

// CreateAssignee valida e registra um novo membro da equipe.
func (s *service) CreateAssignee(ctx context.Context, input AssigneeInput) (Assignee, error) {
	input.Normalize()
	if err := input.Validate(); err != nil {
		return Assignee{}, err
	}
	return s.repository.CreateAssignee(ctx, input)
}

// UpdateAssignee valida e altera os dados operacionais de um membro.
func (s *service) UpdateAssignee(ctx context.Context, id int64, input AssigneeInput) (Assignee, error) {
	input.Normalize()
	if err := input.Validate(); err != nil {
		return Assignee{}, err
	}
	return s.repository.UpdateAssignee(ctx, id, input)
}

// DeleteAssignee remove um membro sem vínculos com chamados.
func (s *service) DeleteAssignee(ctx context.Context, id int64) error {
	return s.repository.DeleteAssignee(ctx, id)
}

// Dashboard retorna os indicadores operacionais da aplicação.
func (s *service) Dashboard(ctx context.Context) (Dashboard, error) {
	return s.repository.Dashboard(ctx)
}
