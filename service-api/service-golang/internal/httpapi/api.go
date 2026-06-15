package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/thiagodifaria/codificar-chamados/internal/ticket"
)

// API reúne as dependências utilizadas pelos handlers HTTP.
type API struct {
	service       ticket.Service
	healthChecker healthChecker
	logger        *slog.Logger
}

type healthChecker interface {
	Ping(ctx context.Context) error
}

// New registra as rotas da aplicação e aplica os middlewares compartilhados.
func New(service ticket.Service, healthChecker healthChecker, logger *slog.Logger) http.Handler {
	api := &API{
		service:       service,
		healthChecker: healthChecker,
		logger:        logger,
	}
	mux := http.NewServeMux()

	// Rotas operacionais e recursos expostos pelo backend.
	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("GET /api/assignees", api.listAssignees)
	mux.HandleFunc("POST /api/assignees", api.createAssignee)
	mux.HandleFunc("PUT /api/assignees/{id}", api.updateAssignee)
	mux.HandleFunc("DELETE /api/assignees/{id}", api.deleteAssignee)
	mux.HandleFunc("GET /api/dashboard", api.dashboard)
	mux.HandleFunc("GET /api/tickets", api.listTickets)
	mux.HandleFunc("POST /api/tickets", api.createTicket)
	mux.HandleFunc("GET /api/tickets/{id}", api.getTicket)
	mux.HandleFunc("PUT /api/tickets/{id}", api.updateTicket)
	mux.HandleFunc("PATCH /api/tickets/{id}/status", api.changeTicketStatus)

	return api.middleware(mux)
}

// health informa se o processo da API está disponível.
func (api *API) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := api.healthChecker.Ping(ctx); err != nil {
		api.logger.Error("database health check failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status":   "unavailable",
			"database": "down",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"database": "ok",
	})
}

// listAssignees retorna os responsáveis disponíveis e suas cargas atuais.
func (api *API) listAssignees(w http.ResponseWriter, r *http.Request) {
	assignees, err := api.service.ListAssignees(r.Context())
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, assignees)
}

// createAssignee valida e registra um novo membro da equipe.
func (api *API) createAssignee(w http.ResponseWriter, r *http.Request) {
	input, ok := api.decodeAssigneeInput(w, r)
	if !ok {
		return
	}
	assignee, err := api.service.CreateAssignee(r.Context(), input)
	if api.handleServiceError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, assignee)
}

// updateAssignee altera o nome ou a disponibilidade de um membro.
func (api *API) updateAssignee(w http.ResponseWriter, r *http.Request) {
	id, ok := api.resourceID(w, r, "membro")
	if !ok {
		return
	}
	input, ok := api.decodeAssigneeInput(w, r)
	if !ok {
		return
	}
	assignee, err := api.service.UpdateAssignee(r.Context(), id, input)
	if errors.Is(err, ticket.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "Membro não encontrado.", nil)
		return
	}
	if api.handleServiceError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, assignee)
}

// deleteAssignee remove um membro sem vínculos com chamados.
func (api *API) deleteAssignee(w http.ResponseWriter, r *http.Request) {
	id, ok := api.resourceID(w, r, "membro")
	if !ok {
		return
	}
	err := api.service.DeleteAssignee(r.Context(), id)
	if errors.Is(err, ticket.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "Membro não encontrado.", nil)
		return
	}
	if api.handleServiceError(w, r, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// dashboard retorna os indicadores utilizados na visão geral da aplicação.
func (api *API) dashboard(w http.ResponseWriter, r *http.Request) {
	dashboard, err := api.service.Dashboard(r.Context())
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

// listTickets lista os chamados conforme os filtros enviados pela query string.
func (api *API) listTickets(w http.ResponseWriter, r *http.Request) {
	// Converte o responsável opcional antes de montar os filtros de domínio.
	assigneeID, err := optionalInt64(r.URL.Query().Get("assigneeId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_filter", "Responsável inválido.", nil)
		return
	}

	// Normaliza os valores textuais para evitar filtros compostos apenas por espaços.
	filter := ticket.ListFilter{
		Search:     strings.TrimSpace(r.URL.Query().Get("search")),
		Status:     strings.TrimSpace(r.URL.Query().Get("status")),
		Priority:   strings.TrimSpace(r.URL.Query().Get("priority")),
		AssigneeID: assigneeID,
		Sort:       strings.TrimSpace(r.URL.Query().Get("sort")),
	}
	if value := r.URL.Query().Get("limit"); value != "" {
		filter.Limit, err = strconv.Atoi(value)
		if err != nil || filter.Limit < 1 {
			writeError(w, http.StatusBadRequest, "invalid_filter", "Limite inválido.", nil)
			return
		}
	}

	items, err := api.service.List(r.Context(), filter)
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// getTicket retorna os detalhes de um chamado específico.
func (api *API) getTicket(w http.ResponseWriter, r *http.Request) {
	id, ok := api.ticketID(w, r)
	if !ok {
		return
	}

	item, err := api.service.Get(r.Context(), id)
	if errors.Is(err, ticket.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "Chamado não encontrado.", nil)
		return
	}
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// createTicket valida a requisição e registra um novo chamado.
func (api *API) createTicket(w http.ResponseWriter, r *http.Request) {
	input, ok := api.decodeInput(w, r)
	if !ok {
		return
	}

	item, err := api.service.Create(r.Context(), input)
	if api.handleServiceError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// updateTicket valida a requisição e altera um chamado existente.
func (api *API) updateTicket(w http.ResponseWriter, r *http.Request) {
	id, ok := api.ticketID(w, r)
	if !ok {
		return
	}
	input, ok := api.decodeInput(w, r)
	if !ok {
		return
	}

	item, err := api.service.Update(r.Context(), id, input)
	if api.handleServiceError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// changeTicketStatus executa uma transição rápida no andamento do chamado.
func (api *API) changeTicketStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := api.ticketID(w, r)
	if !ok {
		return
	}

	var input ticket.StatusInput
	if !decodeJSONBody(w, r, &input) {
		return
	}

	item, err := api.service.ChangeStatus(r.Context(), id, input)
	if api.handleServiceError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// decodeInput interpreta o corpo recebido pela API.
func (api *API) decodeInput(w http.ResponseWriter, r *http.Request) (ticket.UpsertInput, bool) {
	var input ticket.UpsertInput
	if !decodeJSONBody(w, r, &input) {
		return ticket.UpsertInput{}, false
	}

	return input, true
}

// ticketID extrai e valida o identificador presente na rota.
// decodeAssigneeInput interpreta o corpo das operações de gestão da equipe.
func (api *API) decodeAssigneeInput(w http.ResponseWriter, r *http.Request) (ticket.AssigneeInput, bool) {
	var input ticket.AssigneeInput
	if !decodeJSONBody(w, r, &input) {
		return ticket.AssigneeInput{}, false
	}
	return input, true
}

// decodeJSONBody aceita exatamente um objeto JSON e rejeita campos desconhecidos.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Não foi possível interpretar os dados enviados.", nil)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", "Envie apenas um objeto JSON.", nil)
		return false
	}
	return true
}

func (api *API) ticketID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	return api.resourceID(w, r, "chamado")
}

// resourceID extrai um identificador positivo e personaliza a mensagem do recurso.
func (api *API) resourceID(w http.ResponseWriter, r *http.Request, resource string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_id", fmt.Sprintf("Identificador de %s inválido.", resource), nil)
		return 0, false
	}
	return id, true
}

// handleServiceError converte erros conhecidos do domínio em respostas HTTP.
func (api *API) handleServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	var validation ticket.ValidationError
	switch {
	case errors.As(err, &validation):
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "Revise os campos informados.", validation)
	case errors.Is(err, ticket.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Chamado não encontrado.", nil)
	case errors.Is(err, ticket.ErrInvalidAssignee):
		writeError(w, http.StatusUnprocessableEntity, "invalid_assignee", "O responsável selecionado não está disponível.", nil)
	case errors.Is(err, ticket.ErrNoAssignee):
		writeError(w, http.StatusConflict, "no_assignee", "Não há responsáveis disponíveis para atribuição automática.", nil)
	case errors.Is(err, ticket.ErrAssigneeNameConflict):
		writeError(w, http.StatusConflict, "assignee_name_conflict", "Já existe um membro com esse nome.", nil)
	case errors.Is(err, ticket.ErrAssigneeHasTickets):
		writeError(w, http.StatusConflict, "assignee_has_tickets", "Este membro faz parte do histórico de chamados e não pode ser removido.", nil)
	default:
		api.internalError(w, r, err)
	}
	return true
}

// internalError registra o erro técnico e retorna uma mensagem segura ao cliente.
func (api *API) internalError(w http.ResponseWriter, r *http.Request, err error) {
	api.logger.Error("request failed",
		"method", r.Method,
		"path", r.URL.Path,
		"error", err,
	)
	writeError(w, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.", nil)
}

// middleware aplica cabeçalhos, recuperação de pânico e logs a todas as requisições.
func (api *API) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Registra o início para medir a duração total da requisição.
		startedAt := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = newRequestID()
		}
		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

		// Permite o frontend local e adiciona proteções básicas às respostas.
		recorder.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		recorder.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
		recorder.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		recorder.Header().Set("X-Content-Type-Options", "nosniff")
		recorder.Header().Set("X-Request-ID", requestID)

		if r.Method == http.MethodOptions {
			recorder.WriteHeader(http.StatusNoContent)
			return
		}

		// Recupera falhas inesperadas e registra cada requisição concluída.
		defer func() {
			if recovered := recover(); recovered != nil {
				api.logger.Error("panic recovered", "request_id", requestID, "error", recovered)
				writeError(recorder, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.", nil)
			}
			api.logger.Info("request",
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"duration", time.Since(startedAt),
			)
		}()
		next.ServeHTTP(recorder, r)
	})
}

// responseRecorder captura o status final para os logs estruturados.
type responseRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader registra o status antes de enviá-lo ao cliente.
func (recorder *responseRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

// newRequestID gera um identificador curto e seguro para correlação dos logs.
func newRequestID() string {
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", value)
}

// optionalInt64 converte um parâmetro numérico opcional.
func optionalInt64(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

// errorResponse define o formato padrão de erros retornados pela API.
type errorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// writeError escreve uma resposta de erro utilizando o contrato padrão.
func writeError(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	writeJSON(w, status, errorResponse{
		Code:    code,
		Message: message,
		Fields:  fields,
	})
}

// writeJSON serializa um valor em JSON com o status HTTP informado.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
