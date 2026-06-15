package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thiagodifaria/codificar-chamados/internal/ticket"
)

// serviceStub permite exercitar os handlers sem banco ou servidor externo.
type serviceStub struct {
	create       func(context.Context, ticket.UpsertInput) (ticket.Ticket, error)
	changeStatus func(context.Context, int64, ticket.StatusInput) (ticket.Ticket, error)
	get          func(context.Context, int64) (ticket.Ticket, error)
}

func (stub serviceStub) Create(ctx context.Context, input ticket.UpsertInput) (ticket.Ticket, error) {
	return stub.create(ctx, input)
}

func (stub serviceStub) Update(context.Context, int64, ticket.UpsertInput) (ticket.Ticket, error) {
	panic("unexpected call")
}

func (stub serviceStub) ChangeStatus(ctx context.Context, id int64, input ticket.StatusInput) (ticket.Ticket, error) {
	return stub.changeStatus(ctx, id, input)
}

func (stub serviceStub) Get(ctx context.Context, id int64) (ticket.Ticket, error) {
	return stub.get(ctx, id)
}

func (stub serviceStub) List(context.Context, ticket.ListFilter) ([]ticket.Ticket, error) {
	panic("unexpected call")
}

func (stub serviceStub) ListAssignees(context.Context) ([]ticket.Assignee, error) {
	panic("unexpected call")
}

func (stub serviceStub) CreateAssignee(context.Context, ticket.AssigneeInput) (ticket.Assignee, error) {
	panic("unexpected call")
}

func (stub serviceStub) UpdateAssignee(context.Context, int64, ticket.AssigneeInput) (ticket.Assignee, error) {
	panic("unexpected call")
}

func (stub serviceStub) DeleteAssignee(context.Context, int64) error {
	panic("unexpected call")
}

func (stub serviceStub) Dashboard(context.Context) (ticket.Dashboard, error) {
	panic("unexpected call")
}

type healthCheckerStub struct{}

func (healthCheckerStub) Ping(context.Context) error {
	return nil
}

func TestGetTicketReturnsNotFound(t *testing.T) {
	handler := newTestHandler(serviceStub{
		get: func(context.Context, int64) (ticket.Ticket, error) {
			return ticket.Ticket{}, ticket.ErrNotFound
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/tickets/999", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
	assertErrorCode(t, response.Body.Bytes(), "not_found")
}

func TestCreateTicketReturnsCreatedTicket(t *testing.T) {
	var received ticket.UpsertInput
	handler := newTestHandler(serviceStub{
		create: func(_ context.Context, input ticket.UpsertInput) (ticket.Ticket, error) {
			received = input
			return ticket.Ticket{
				ID:             42,
				Title:          input.Title,
				Description:    input.Description,
				RequesterName:  input.RequesterName,
				Priority:       input.Priority,
				Status:         input.Status,
				AssignmentMode: input.AssignmentMode,
				AssigneeID:     3,
				AssigneeName:   "Carla",
			}, nil
		},
	})
	body := []byte(`{
		"title":"Impressora sem conexão",
		"description":"A impressora do financeiro está indisponível.",
		"requesterName":"Ana Souza",
		"priority":"high",
		"status":"open",
		"assignmentMode":"automatic",
		"assigneeId":0,
		"redistribute":false
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/tickets", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	if received.Title != "Impressora sem conexão" || received.AssignmentMode != ticket.AssignmentAutomatic {
		t.Fatalf("handler forwarded unexpected input: %+v", received)
	}

	var created ticket.Ticket
	decodeJSON(t, response.Body.Bytes(), &created)
	if created.ID != 42 || created.AssigneeName != "Carla" {
		t.Fatalf("unexpected created ticket: %+v", created)
	}
}

func TestCreateTicketRejectsInvalidJSON(t *testing.T) {
	handler := newTestHandler(serviceStub{})
	request := httptest.NewRequest(http.MethodPost, "/api/tickets", bytes.NewBufferString(`{"title":`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
	assertErrorCode(t, response.Body.Bytes(), "invalid_json")
}

func TestCreateTicketReturnsValidationFields(t *testing.T) {
	handler := newTestHandler(serviceStub{
		create: func(context.Context, ticket.UpsertInput) (ticket.Ticket, error) {
			return ticket.Ticket{}, ticket.ValidationError{
				"assigneeId": "Selecione um responsável.",
			}
		},
	})
	body := bytes.NewBufferString(`{
		"title":"Acesso bloqueado",
		"description":"Usuário sem acesso.",
		"requesterName":"Financeiro",
		"priority":"medium",
		"status":"open",
		"assignmentMode":"manual",
		"assigneeId":0
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/tickets", body)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, response.Code)
	}
	var apiError errorResponse
	decodeJSON(t, response.Body.Bytes(), &apiError)
	if apiError.Code != "validation_error" || apiError.Fields["assigneeId"] == "" {
		t.Fatalf("unexpected validation response: %+v", apiError)
	}
}

func TestGetTicketRejectsInvalidID(t *testing.T) {
	handler := newTestHandler(serviceStub{})
	request := httptest.NewRequest(http.MethodGet, "/api/tickets/invalid", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
	assertErrorCode(t, response.Body.Bytes(), "invalid_id")
}

func TestChangeTicketStatusReturnsUpdatedTicket(t *testing.T) {
	var receivedID int64
	var receivedInput ticket.StatusInput
	handler := newTestHandler(serviceStub{
		changeStatus: func(_ context.Context, id int64, input ticket.StatusInput) (ticket.Ticket, error) {
			receivedID = id
			receivedInput = input
			return ticket.Ticket{ID: id, Status: input.Status}, nil
		},
	})
	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/tickets/7/status",
		bytes.NewBufferString(`{"status":"resolved"}`),
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	if receivedID != 7 || receivedInput.Status != ticket.StatusResolved {
		t.Fatalf("handler forwarded id %d and input %+v", receivedID, receivedInput)
	}

	var updated ticket.Ticket
	decodeJSON(t, response.Body.Bytes(), &updated)
	if updated.ID != 7 || updated.Status != ticket.StatusResolved {
		t.Fatalf("unexpected updated ticket: %+v", updated)
	}
}

func TestOptionsReturnsCORSHeaders(t *testing.T) {
	handler := newTestHandler(serviceStub{})
	request := httptest.NewRequest(http.MethodOptions, "/api/tickets", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("expected CORS methods header")
	}
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected request id header")
	}
}

func newTestHandler(service ticket.Service) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(service, healthCheckerStub{}, logger)
}

func assertErrorCode(t *testing.T, body []byte, expected string) {
	t.Helper()
	var response errorResponse
	decodeJSON(t, body, &response)
	if response.Code != expected {
		t.Fatalf("expected error code %q, got %q", expected, response.Code)
	}
}

func decodeJSON(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
