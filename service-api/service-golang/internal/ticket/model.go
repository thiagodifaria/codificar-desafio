package ticket

import (
	"errors"
	"strings"
	"time"
)

const (
	// Prioridades aceitas para classificação dos chamados.
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"

	// Status que representam o ciclo de vida de um chamado.
	StatusOpen       = "open"
	StatusInProgress = "in_progress"
	StatusResolved   = "resolved"
	StatusClosed     = "closed"

	// Modos disponíveis para seleção do responsável pelo atendimento.
	AssignmentManual    = "manual"
	AssignmentAutomatic = "automatic"
)

// ErrNotFound indica que o chamado solicitado não existe.
var ErrNotFound = errors.New("ticket not found")

// AssigneeInput reúne os dados aceitos na criação e edição de membros da equipe.
type AssigneeInput struct {
	Name string `json:"name"`
}

// Ticket representa um chamado interno e suas informações de atendimento.
type Ticket struct {
	ID             int64      `json:"id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	RequesterName  string     `json:"requesterName"`
	Priority       string     `json:"priority"`
	Status         string     `json:"status"`
	AssigneeID     int64      `json:"assigneeId"`
	AssigneeName   string     `json:"assigneeName"`
	AssignmentMode string     `json:"assignmentMode"`
	OpenedAt       time.Time  `json:"openedAt"`
	ResolvedAt     *time.Time `json:"resolvedAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// Assignee representa uma pessoa disponível para atender chamados.
type Assignee struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	Active           bool       `json:"active"`
	OpenTickets      int        `json:"openTickets"`
	CompletedTickets int        `json:"completedTickets"`
	LastAssignedAt   *time.Time `json:"lastAssignedAt"`
}

// Normalize remove espaços desnecessários do nome do membro.
func (input *AssigneeInput) Normalize() {
	input.Name = strings.TrimSpace(input.Name)
}

// Validate garante um nome útil e compatível com o armazenamento.
func (input AssigneeInput) Validate() error {
	problems := ValidationError{}
	if input.Name == "" {
		problems["name"] = "Informe o nome."
	} else if len([]rune(input.Name)) > 120 {
		problems["name"] = "O nome deve ter no máximo 120 caracteres."
	}
	if len(problems) > 0 {
		return problems
	}
	return nil
}

// UpsertInput reúne os dados aceitos na criação e na edição de chamados.
type UpsertInput struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	RequesterName  string `json:"requesterName"`
	Priority       string `json:"priority"`
	Status         string `json:"status"`
	AssignmentMode string `json:"assignmentMode"`
	AssigneeID     int64  `json:"assigneeId"`
	Redistribute   bool   `json:"redistribute"`
}

// StatusInput representa uma alteração direta no andamento do chamado.
type StatusInput struct {
	Status string `json:"status"`
}

// ListFilter define os filtros opcionais utilizados na listagem de chamados.
type ListFilter struct {
	Search     string
	Status     string
	Priority   string
	AssigneeID int64
	Sort       string
	Limit      int
}

// Dashboard reúne os indicadores apresentados na visão geral da aplicação.
type Dashboard struct {
	Total        int64      `json:"total"`
	Open         int64      `json:"open"`
	InProgress   int64      `json:"inProgress"`
	Resolved     int64      `json:"resolved"`
	Closed       int64      `json:"closed"`
	Assignees    []Assignee `json:"assignees"`
	NextAssignee *Assignee  `json:"nextAssignee"`
}

// ValidationError relaciona cada campo inválido à sua mensagem de validação.
type ValidationError map[string]string

// Error implementa a interface error para erros de validação.
func (e ValidationError) Error() string {
	return "invalid ticket data"
}

// Normalize remove espaços desnecessários dos campos textuais recebidos pela API.
func (input *UpsertInput) Normalize() {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.RequesterName = strings.TrimSpace(input.RequesterName)
	input.Priority = strings.TrimSpace(input.Priority)
	input.Status = strings.TrimSpace(input.Status)
	input.AssignmentMode = strings.TrimSpace(input.AssignmentMode)
}

// Validate aplica as regras de preenchimento e domínio de um chamado.
func (input UpsertInput) Validate() error {
	problems := ValidationError{}

	// Valida os campos textuais obrigatórios e seus limites de armazenamento.
	if input.Title == "" {
		problems["title"] = "Informe o título."
	} else if len([]rune(input.Title)) > 160 {
		problems["title"] = "O título deve ter no máximo 160 caracteres."
	}
	if input.Description == "" {
		problems["description"] = "Informe a descrição."
	}
	if input.RequesterName == "" {
		problems["requesterName"] = "Informe o solicitante."
	} else if len([]rune(input.RequesterName)) > 120 {
		problems["requesterName"] = "O solicitante deve ter no máximo 120 caracteres."
	}

	// Garante que somente valores reconhecidos pelo domínio cheguem ao banco.
	if !oneOf(input.Priority, PriorityLow, PriorityMedium, PriorityHigh) {
		problems["priority"] = "Selecione uma prioridade válida."
	}
	if !oneOf(input.Status, StatusOpen, StatusInProgress, StatusResolved, StatusClosed) {
		problems["status"] = "Selecione um status válido."
	}
	if !oneOf(input.AssignmentMode, AssignmentManual, AssignmentAutomatic) {
		problems["assignmentMode"] = "Selecione uma forma de atribuição válida."
	}
	if input.AssignmentMode == AssignmentManual && input.AssigneeID <= 0 {
		problems["assigneeId"] = "Selecione um responsável."
	}

	// Retorna todos os problemas de uma vez para permitir uma correção completa no formulário.
	if len(problems) > 0 {
		return problems
	}
	return nil
}

// Normalize remove espaços desnecessários do status recebido.
func (input *StatusInput) Normalize() {
	input.Status = strings.TrimSpace(input.Status)
}

// Validate garante que a transição solicitada utilize um status reconhecido.
func (input StatusInput) Validate() error {
	if !oneOf(input.Status, StatusOpen, StatusInProgress, StatusResolved, StatusClosed) {
		return ValidationError{"status": "Selecione um status válido."}
	}
	return nil
}

// oneOf verifica se um valor pertence ao conjunto de opções permitidas.
func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
