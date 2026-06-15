package ticket

import "context"

// Repository define as operações de persistência utilizadas pelo serviço de chamados.
type Repository interface {
	Create(ctx context.Context, input UpsertInput) (Ticket, error)
	Update(ctx context.Context, id int64, input UpsertInput) (Ticket, error)
	ChangeStatus(ctx context.Context, id int64, status string) (Ticket, error)
	Get(ctx context.Context, id int64) (Ticket, error)
	List(ctx context.Context, filter ListFilter) ([]Ticket, error)
	ListAssignees(ctx context.Context) ([]Assignee, error)
	CreateAssignee(ctx context.Context, input AssigneeInput) (Assignee, error)
	UpdateAssignee(ctx context.Context, id int64, input AssigneeInput) (Assignee, error)
	DeleteAssignee(ctx context.Context, id int64) error
	Dashboard(ctx context.Context) (Dashboard, error)
}
