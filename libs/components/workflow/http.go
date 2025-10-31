package workflow

import (
    "encoding/json"
    "errors"
    "io"
    "net/http"
    "strconv"
    "strings"

    "github.com/go-chi/chi/v5"
    "gorm.io/datatypes"

    "github.com/pflow/shared/httpx"
)

// Handler exposes workflow HTTP endpoints.
type Handler struct {
    repo Repository
}

// NewHandler builds a workflow Handler backed by the given repository.
func NewHandler(repo Repository) *Handler {
    return &Handler{repo: repo}
}

// Mount registers the workflow routes under the provided base path.
func (h *Handler) Mount(router chi.Router, basePath string) {
    path := strings.TrimSpace(basePath)
    if path == "" {
        path = "/workflows"
    }

    router.Route(path, func(r chi.Router) {
        r.Get("/", h.listDefinitions)
        r.Post("/", h.createDefinition)
        r.Route("/{id}", func(r chi.Router) {
            r.Get("/", h.getDefinition)
            r.Put("/", h.updateDefinition)
            r.Delete("/", h.deleteDefinition)
            r.Post("/publish", h.publishDefinition)
        })
    })
}

type createDefinitionRequest struct {
    Name        string         `json:"name"`
    Version     int            `json:"version"`
    Description string         `json:"description"`
    Blueprint   map[string]any `json:"blueprint"`
}

type updateDefinitionRequest struct {
    Name        *string        `json:"name"`
    Version     *int           `json:"version"`
    Description *string        `json:"description"`
    Blueprint   map[string]any `json:"blueprint"`
    Published   *bool          `json:"published"`
}

func (h *Handler) listDefinitions(w http.ResponseWriter, r *http.Request) {
    var publishedFilter *bool
    if value := strings.TrimSpace(r.URL.Query().Get("published")); value != "" {
        parsed, err := strconv.ParseBool(value)
        if err != nil {
            httpx.Error(w, http.StatusBadRequest, "invalid published filter")
            return
        }
        publishedFilter = &parsed
    }

    definitions, err := h.repo.List(r.Context(), publishedFilter)
    if err != nil {
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    items := make([]map[string]any, 0, len(definitions))
    for _, entity := range definitions {
        items = append(items, entity.ToDTO())
    }

    httpx.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) createDefinition(w http.ResponseWriter, r *http.Request) {
    var payload createDefinitionRequest
    if err := decodeJSON(r, &payload); err != nil {
        httpx.Error(w, http.StatusBadRequest, err.Error())
        return
    }

    name := strings.TrimSpace(payload.Name)
    if len(name) < 2 {
        httpx.Error(w, http.StatusBadRequest, "name must be at least 2 characters")
        return
    }

    version := payload.Version
    if version <= 0 {
        version = 1
    }

    entity := &Definition{
        Name:        name,
        Version:     version,
        Description: strings.TrimSpace(payload.Description),
    }
    if payload.Blueprint != nil {
        entity.Blueprint = datatypes.JSONMap(payload.Blueprint)
    }

    if err := h.repo.Create(r.Context(), entity); err != nil {
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    httpx.JSON(w, http.StatusCreated, map[string]any{"data": entity.ToDTO()})
}

func (h *Handler) getDefinition(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    entity, err := h.repo.Find(r.Context(), id)
    if err != nil {
        if IsNotFound(err) {
            httpx.Error(w, http.StatusNotFound, "workflow not found")
            return
        }
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    httpx.JSON(w, http.StatusOK, map[string]any{"data": entity.ToDTO()})
}

func (h *Handler) updateDefinition(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    var payload updateDefinitionRequest
    if err := decodeJSON(r, &payload); err != nil {
        httpx.Error(w, http.StatusBadRequest, err.Error())
        return
    }

    updates := make(map[string]any)
    if payload.Name != nil {
        name := strings.TrimSpace(*payload.Name)
        if len(name) < 2 {
            httpx.Error(w, http.StatusBadRequest, "name must be at least 2 characters")
            return
        }
        updates["name"] = name
    }
    if payload.Version != nil {
        if *payload.Version <= 0 {
            httpx.Error(w, http.StatusBadRequest, "version must be positive")
            return
        }
        updates["version"] = *payload.Version
    }
    if payload.Description != nil {
        updates["description"] = strings.TrimSpace(*payload.Description)
    }
    if payload.Blueprint != nil {
        updates["blueprint"] = datatypes.JSONMap(payload.Blueprint)
    }
    if payload.Published != nil {
        updates["published"] = *payload.Published
    }

    if len(updates) == 0 {
        httpx.Error(w, http.StatusBadRequest, "no updates provided")
        return
    }

    entity, err := h.repo.Update(r.Context(), id, updates)
    if err != nil {
        if IsNotFound(err) {
            httpx.Error(w, http.StatusNotFound, "workflow not found")
            return
        }
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    httpx.JSON(w, http.StatusOK, map[string]any{"data": entity.ToDTO()})
}

func (h *Handler) deleteDefinition(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    if err := h.repo.Delete(r.Context(), id); err != nil {
        if IsNotFound(err) {
            httpx.Error(w, http.StatusNotFound, "workflow not found")
            return
        }
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) publishDefinition(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    entity, err := h.repo.Publish(r.Context(), id)
    if err != nil {
        if IsNotFound(err) {
            httpx.Error(w, http.StatusNotFound, "workflow not found")
            return
        }
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    httpx.JSON(w, http.StatusOK, map[string]any{"data": entity.ToDTO()})
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
