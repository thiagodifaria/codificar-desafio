package ticket

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func integrationStore(t *testing.T) (*Store, *pgxpool.Pool) {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL não configurada")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("não foi possível criar o pool de testes: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, `TRUNCATE tickets, assignees RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("não foi possível limpar o banco de testes: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO assignees (name)
		VALUES ('Ana Souza'), ('Bruno Lima'), ('Carla Mendes')
	`); err != nil {
		t.Fatalf("não foi possível criar os responsáveis: %v", err)
	}

	return NewStore(pool), pool
}

func validTicketInput(title string) UpsertInput {
	return UpsertInput{
		Title:          title,
		Description:    "Descrição usada pelo teste de integração.",
		RequesterName:  "Equipe de testes",
		Priority:       PriorityMedium,
		Status:         StatusOpen,
		AssignmentMode: AssignmentAutomatic,
	}
}

// TestStoreAutomaticAssignmentSelectsLeastLoadedAssignee valida a regra principal de distribuição.
func TestStoreAutomaticAssignmentSelectsLeastLoadedAssignee(t *testing.T) {
	store, _ := integrationStore(t)
	ctx := context.Background()

	for index := 0; index < 2; index++ {
		input := validTicketInput("Carga da Ana")
		input.AssignmentMode = AssignmentManual
		input.AssigneeID = 1
		if _, err := store.Create(ctx, input); err != nil {
			t.Fatalf("não foi possível criar carga da Ana: %v", err)
		}
	}

	input := validTicketInput("Carga do Bruno")
	input.AssignmentMode = AssignmentManual
	input.AssigneeID = 2
	if _, err := store.Create(ctx, input); err != nil {
		t.Fatalf("não foi possível criar carga do Bruno: %v", err)
	}

	created, err := store.Create(ctx, validTicketInput("Chamado automático"))
	if err != nil {
		t.Fatalf("não foi possível criar chamado automático: %v", err)
	}
	if created.AssigneeID != 3 {
		t.Fatalf("responsável automático = %d, esperado 3", created.AssigneeID)
	}
}

// TestStoreCompletedTicketsDoNotAffectLoad garante que chamados concluídos saiam da distribuição.
func TestStoreCompletedTicketsDoNotAffectLoad(t *testing.T) {
	store, _ := integrationStore(t)
	ctx := context.Background()

	input := validTicketInput("Chamado concluído")
	input.AssignmentMode = AssignmentManual
	input.AssigneeID = 1
	input.Status = StatusResolved
	if _, err := store.Create(ctx, input); err != nil {
		t.Fatalf("não foi possível criar chamado concluído: %v", err)
	}

	for _, assigneeID := range []int64{2, 3} {
		input = validTicketInput("Chamado ativo")
		input.AssignmentMode = AssignmentManual
		input.AssigneeID = assigneeID
		if _, err := store.Create(ctx, input); err != nil {
			t.Fatalf("não foi possível criar chamado ativo: %v", err)
		}
	}

	created, err := store.Create(ctx, validTicketInput("Nova distribuição"))
	if err != nil {
		t.Fatalf("não foi possível criar chamado automático: %v", err)
	}
	if created.AssigneeID != 1 {
		t.Fatalf("responsável automático = %d, esperado 1", created.AssigneeID)
	}

	assignees, err := store.ListAssignees(ctx)
	if err != nil {
		t.Fatalf("não foi possível consultar os concluídos: %v", err)
	}
	if assignees[0].CompletedTickets != 1 {
		t.Fatalf("concluídos de %s = %d, esperado 1", assignees[0].Name, assignees[0].CompletedTickets)
	}
}

// TestStoreUpdatePreservesAssignmentAndResolutionDate protege o histórico durante edições comuns.
func TestStoreUpdatePreservesAssignmentAndResolutionDate(t *testing.T) {
	store, _ := integrationStore(t)
	ctx := context.Background()

	created, err := store.Create(ctx, validTicketInput("Chamado original"))
	if err != nil {
		t.Fatalf("não foi possível criar o chamado: %v", err)
	}

	updatedInput := validTicketInput("Título atualizado")
	updatedInput.AssignmentMode = AssignmentAutomatic
	updatedInput.Status = StatusResolved
	resolved, err := store.Update(ctx, created.ID, updatedInput)
	if err != nil {
		t.Fatalf("não foi possível resolver o chamado: %v", err)
	}
	if resolved.AssigneeID != created.AssigneeID {
		t.Fatalf("responsável mudou de %d para %d sem redistribuição", created.AssigneeID, resolved.AssigneeID)
	}
	if resolved.ResolvedAt == nil {
		t.Fatal("data de resolução não foi registrada")
	}
	firstResolution := *resolved.ResolvedAt

	time.Sleep(10 * time.Millisecond)
	updatedInput.Description = "Descrição alterada depois da resolução."
	edited, err := store.Update(ctx, created.ID, updatedInput)
	if err != nil {
		t.Fatalf("não foi possível editar o chamado resolvido: %v", err)
	}
	if edited.ResolvedAt == nil || !edited.ResolvedAt.Equal(firstResolution) {
		t.Fatalf("data de resolução foi alterada: primeira=%v atual=%v", firstResolution, edited.ResolvedAt)
	}

	reopened, err := store.ChangeStatus(ctx, created.ID, StatusOpen)
	if err != nil {
		t.Fatalf("não foi possível reabrir o chamado: %v", err)
	}
	if reopened.ResolvedAt != nil {
		t.Fatalf("chamado reaberto manteve resolvedAt=%v", reopened.ResolvedAt)
	}
}

// TestStoreConcurrentAutomaticAssignment mantém a carga equilibrada sob concorrência real.
func TestStoreConcurrentAutomaticAssignment(t *testing.T) {
	store, _ := integrationStore(t)
	ctx := context.Background()

	const total = 12
	var waitGroup sync.WaitGroup
	errs := make(chan error, total)

	for index := 0; index < total; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			_, err := store.Create(ctx, validTicketInput("Concorrente "+time.Now().Add(time.Duration(index)).String()))
			errs <- err
		}(index)
	}

	waitGroup.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("criação concorrente falhou: %v", err)
		}
	}

	assignees, err := store.ListAssignees(ctx)
	if err != nil {
		t.Fatalf("não foi possível consultar as cargas: %v", err)
	}

	minimum, maximum, sum := total, 0, 0
	for _, assignee := range assignees {
		if assignee.OpenTickets < minimum {
			minimum = assignee.OpenTickets
		}
		if assignee.OpenTickets > maximum {
			maximum = assignee.OpenTickets
		}
		sum += assignee.OpenTickets
	}
	if sum != total {
		t.Fatalf("total de chamados = %d, esperado %d", sum, total)
	}
	if maximum-minimum > 1 {
		t.Fatalf("carga desequilibrada: mínimo=%d máximo=%d", minimum, maximum)
	}
}

// TestStoreManageAssignees preserva o histórico e protege membros vinculados.
func TestStoreManageAssignees(t *testing.T) {
	store, _ := integrationStore(t)
	ctx := context.Background()

	removable, err := store.CreateAssignee(ctx, AssigneeInput{Name: "Daniel Rocha"})
	if err != nil {
		t.Fatalf("não foi possível criar o membro: %v", err)
	}
	if err := store.DeleteAssignee(ctx, removable.ID); err != nil {
		t.Fatalf("não foi possível remover o membro sem chamados: %v", err)
	}

	linked, err := store.CreateAssignee(ctx, AssigneeInput{Name: "Eduardo Alves"})
	if err != nil {
		t.Fatalf("não foi possível criar o membro vinculado: %v", err)
	}

	input := validTicketInput("Chamado do Eduardo")
	input.AssignmentMode = AssignmentManual
	input.AssigneeID = linked.ID
	if _, err := store.Create(ctx, input); err != nil {
		t.Fatalf("não foi possível criar a carga do membro: %v", err)
	}

	err = store.DeleteAssignee(ctx, linked.ID)
	if !errors.Is(err, ErrAssigneeHasTickets) {
		t.Fatalf("erro ao remover membro com histórico = %v, esperado %v", err, ErrAssigneeHasTickets)
	}
}
