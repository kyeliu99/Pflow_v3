package ticket

import (
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

// Handler exposes HTTP handlers for the ticket component.
type Handler struct {
    repo Repository
}

// NewHandler builds a ticket HTTP handler backed by the given repository.
func NewHandler(repo Repository) *Handler {
    return &Handler{repo: repo}
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

    title := strings.TrimSpace(payload.Title)
    if len(title) < 3 {
        httpx.Error(w, http.StatusBadRequest, "title must be at least 3 characters")
        return
    }

    formID := strings.TrimSpace(payload.FormID)
    if _, err := uuid.Parse(formID); err != nil {
        httpx.Error(w, http.StatusBadRequest, "formId must be a valid UUID")
        return
    }

    status := strings.ToLower(strings.TrimSpace(payload.Status))
    if status == "" {
        status = StatusOpen
    }
    if !isValidStatus(status) {
        httpx.Error(w, http.StatusBadRequest, "invalid status")
        return
    }

    entity := &Ticket{
        Title:      title,
        Status:     status,
        FormID:     formID,
        AssigneeID: strings.TrimSpace(payload.AssigneeID),
    }
    if payload.Priority != "" {
        entity.Priority = strings.ToLower(strings.TrimSpace(payload.Priority))
    }
    if payload.Metadata != nil {
        entity.Metadata = datatypes.JSONMap(payload.Metadata)
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
