package ticket

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/pflow/shared/httpx"
)

var allowedStatuses = map[string]struct{}{
	StatusOpen:       {},
	StatusInProgress: {},
	StatusResolved:   {},
	StatusCancelled:  {},
}

// SubmissionCoordinator coordinates asynchronous submissions.
type SubmissionCoordinator interface {
	Submit(ctx context.Context, req SubmissionRequest) (*TicketSubmission, error)
	Lookup(ctx context.Context, id string) (*TicketSubmission, error)
	Metrics(ctx context.Context) (SubmissionMetrics, error)
}

// Handler exposes HTTP handlers for the ticket component.
type Handler struct {
	repo        Repository
	coordinator SubmissionCoordinator
}

// HandlerOption customises the handler behaviour.
type HandlerOption func(*Handler)

// WithSubmissionCoordinator attaches an asynchronous submission coordinator to the handler.
func WithSubmissionCoordinator(coordinator SubmissionCoordinator) HandlerOption {
	return func(h *Handler) {
		h.coordinator = coordinator
	}
}

// NewHandler builds a ticket HTTP handler backed by the given repository.
func NewHandler(repo Repository, opts ...HandlerOption) *Handler {
	handler := &Handler{repo: repo}
	for _, opt := range opts {
		if opt != nil {
			opt(handler)
		}
	}
	return handler
}

// Mount registers the ticket routes on the provided router under the supplied base path.
func (h *Handler) Mount(router chi.Router, basePath string) {
	path := strings.TrimSpace(basePath)
	if path == "" {
		path = "/tickets"
	}

	router.Route(path, func(r chi.Router) {
		r.Get("/", h.listTickets)
		r.Post("/", h.createTicket)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.getTicket)
			r.Patch("/", h.updateTicket)
			r.Delete("/", h.deleteTicket)
			r.Post("/resolve", h.resolveTicket)
		})

		if h.coordinator != nil {
			r.Route("/submissions", func(r chi.Router) {
				r.Post("/", h.submitTicket)
				r.Get("/{id}", h.getSubmission)
			})
			r.Get("/queue-metrics", h.queueMetrics)
		}
	})
}

type createTicketRequest struct {
	Title      string         `json:"title"`
	Status     string         `json:"status"`
	FormID     string         `json:"formId"`
	AssigneeID string         `json:"assigneeId"`
	Priority   string         `json:"priority"`
	Metadata   map[string]any `json:"metadata"`
}

type createSubmissionRequest struct {
	createTicketRequest
	ClientReference string `json:"clientReference"`
}

type updateTicketRequest struct {
	Title      *string        `json:"title"`
	Status     *string        `json:"status"`
	AssigneeID *string        `json:"assigneeId"`
	Priority   *string        `json:"priority"`
	Metadata   map[string]any `json:"metadata"`
}

func (h *Handler) listTickets(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	assignee := strings.TrimSpace(r.URL.Query().Get("assigneeId"))

	tickets, err := h.repo.List(r.Context(), status, assignee)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]map[string]any, 0, len(tickets))
	for _, entity := range tickets {
		items = append(items, entity.ToDTO())
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) createTicket(w http.ResponseWriter, r *http.Request) {
	var payload createTicketRequest
	if err := decodeJSON(r, &payload); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	entity, _, err := normalizeTicketPayload(payload)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.repo.Create(r.Context(), entity); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{"data": entity.ToDTO()})
}

func (h *Handler) getTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	entity, err := h.repo.Find(r.Context(), id)
	if err != nil {
		if IsNotFound(err) {
			httpx.Error(w, http.StatusNotFound, "ticket not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": entity.ToDTO()})
}

func (h *Handler) updateTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var payload updateTicketRequest
	if err := decodeJSON(r, &payload); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	updates := make(map[string]any)
	if payload.Title != nil {
		title := strings.TrimSpace(*payload.Title)
		if len(title) < 3 {
			httpx.Error(w, http.StatusBadRequest, "title must be at least 3 characters")
			return
		}
		updates["title"] = title
	}
	if payload.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*payload.Status))
		if !isValidStatus(status) {
			httpx.Error(w, http.StatusBadRequest, "invalid status")
			return
		}
		updates["status"] = status
	}
	if payload.AssigneeID != nil {
		updates["assignee_id"] = strings.TrimSpace(*payload.AssigneeID)
	}
	if payload.Priority != nil {
		updates["priority"] = strings.ToLower(strings.TrimSpace(*payload.Priority))
	}
	if payload.Metadata != nil {
		updates["metadata"] = datatypes.JSONMap(payload.Metadata)
	}

	if len(updates) == 0 {
		httpx.Error(w, http.StatusBadRequest, "no updates provided")
		return
	}

	entity, err := h.repo.Update(r.Context(), id, updates)
	if err != nil {
		if IsNotFound(err) {
			httpx.Error(w, http.StatusNotFound, "ticket not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": entity.ToDTO()})
}

func (h *Handler) deleteTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if IsNotFound(err) {
			httpx.Error(w, http.StatusNotFound, "ticket not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) resolveTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	entity, err := h.repo.Resolve(r.Context(), id)
	if err != nil {
		if IsNotFound(err) {
			httpx.Error(w, http.StatusNotFound, "ticket not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": entity.ToDTO()})
}

func (h *Handler) submitTicket(w http.ResponseWriter, r *http.Request) {
	if h.coordinator == nil {
		httpx.Error(w, http.StatusNotImplemented, "ticket submissions are not configured")
		return
	}

	var payload createSubmissionRequest
	if err := decodeJSON(r, &payload); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	_, normalized, err := normalizeTicketPayload(payload.createTicketRequest)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	submission, err := h.coordinator.Submit(r.Context(), SubmissionRequest{
		ClientReference: strings.TrimSpace(payload.ClientReference),
		Payload:         normalized,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	statusCode := http.StatusAccepted
	if submission.Status == SubmissionCompleted {
		statusCode = http.StatusOK
	}
	httpx.JSON(w, statusCode, map[string]any{"data": submission.ToDTO()})
}

func (h *Handler) getSubmission(w http.ResponseWriter, r *http.Request) {
	if h.coordinator == nil {
		httpx.Error(w, http.StatusNotImplemented, "ticket submissions are not configured")
		return
	}

	id := chi.URLParam(r, "id")
	submission, err := h.coordinator.Lookup(r.Context(), id)
	if err != nil {
		if IsNotFound(err) {
			httpx.Error(w, http.StatusNotFound, "submission not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": submission.ToDTO()})
}

func (h *Handler) queueMetrics(w http.ResponseWriter, r *http.Request) {
	if h.coordinator == nil {
		httpx.Error(w, http.StatusNotImplemented, "ticket submissions are not configured")
		return
	}

	metrics, err := h.coordinator.Metrics(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": metrics})
}

func normalizeTicketPayload(payload createTicketRequest) (*Ticket, map[string]any, error) {
	title := strings.TrimSpace(payload.Title)
	if len(title) < 3 {
		return nil, nil, errors.New("title must be at least 3 characters")
	}

	formID := strings.TrimSpace(payload.FormID)
	if _, err := uuid.Parse(formID); err != nil {
		return nil, nil, errors.New("formId must be a valid UUID")
	}

	status := strings.ToLower(strings.TrimSpace(payload.Status))
	if status == "" {
		status = StatusOpen
	}
	if !isValidStatus(status) {
		return nil, nil, errors.New("invalid status")
	}

	entity := &Ticket{
		Title:      title,
		Status:     status,
		FormID:     formID,
		AssigneeID: strings.TrimSpace(payload.AssigneeID),
		Priority:   "medium",
	}

	priority := strings.ToLower(strings.TrimSpace(payload.Priority))
	if priority != "" {
		entity.Priority = priority
	}

	normalized := map[string]any{
		"title":    entity.Title,
		"status":   entity.Status,
		"formId":   entity.FormID,
		"priority": entity.Priority,
	}

	if entity.AssigneeID != "" {
		normalized["assigneeId"] = entity.AssigneeID
	}

	if payload.Metadata != nil {
		entity.Metadata = datatypes.JSONMap(payload.Metadata)
		normalized["metadata"] = payload.Metadata
	}

	return entity, normalized, nil
}

func isValidStatus(status string) bool {
	_, ok := allowedStatuses[status]
	return ok
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is empty")
		}
		return err
	}
	return nil
}
