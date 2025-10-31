package form

import (
    "encoding/json"
    "errors"
    "io"
    "net/http"
    "strings"

    "github.com/go-chi/chi/v5"
    "gorm.io/datatypes"

    "github.com/pflow/shared/httpx"
)

// Handler exposes reusable HTTP endpoints for form management.
type Handler struct {
    repo Repository
}

// NewHandler constructs a Handler backed by the provided repository.
func NewHandler(repo Repository) *Handler {
    return &Handler{repo: repo}
}

// Mount registers the form routes on the provided router under the supplied base path.
func (h *Handler) Mount(router chi.Router, basePath string) {
    path := strings.TrimSpace(basePath)
    if path == "" {
        path = "/forms"
    }

    router.Route(path, func(r chi.Router) {
        r.Get("/", h.listForms)
        r.Post("/", h.createForm)
        r.Route("/{id}", func(r chi.Router) {
            r.Get("/", h.getForm)
            r.Put("/", h.updateForm)
            r.Delete("/", h.deleteForm)
        })
    })
}

type createFormRequest struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    Schema      map[string]any `json:"schema"`
}

type updateFormRequest struct {
    Name        *string        `json:"name"`
    Description *string        `json:"description"`
    Schema      map[string]any `json:"schema"`
}

func (h *Handler) listForms(w http.ResponseWriter, r *http.Request) {
    search := strings.TrimSpace(r.URL.Query().Get("search"))
    forms, err := h.repo.List(r.Context(), search)
    if err != nil {
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    items := make([]map[string]any, 0, len(forms))
    for _, entity := range forms {
        items = append(items, entity.ToDTO())
    }

    httpx.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) createForm(w http.ResponseWriter, r *http.Request) {
    var payload createFormRequest
    if err := decodeJSON(r, &payload); err != nil {
        httpx.Error(w, http.StatusBadRequest, err.Error())
        return
    }

    name := strings.TrimSpace(payload.Name)
    if name == "" {
        httpx.Error(w, http.StatusBadRequest, "name is required")
        return
    }

    entity := &Form{
        Name:        name,
        Description: strings.TrimSpace(payload.Description),
    }
    if payload.Schema != nil {
        entity.Schema = datatypes.JSONMap(payload.Schema)
    }

    if err := h.repo.Create(r.Context(), entity); err != nil {
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    httpx.JSON(w, http.StatusCreated, map[string]any{"data": entity.ToDTO()})
}

func (h *Handler) getForm(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    entity, err := h.repo.Find(r.Context(), id)
    if err != nil {
        if IsNotFound(err) {
            httpx.Error(w, http.StatusNotFound, "form not found")
            return
        }
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    httpx.JSON(w, http.StatusOK, map[string]any{"data": entity.ToDTO()})
}

func (h *Handler) updateForm(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    var payload updateFormRequest
    if err := decodeJSON(r, &payload); err != nil {
        httpx.Error(w, http.StatusBadRequest, err.Error())
        return
    }

    updates := make(map[string]any)
    if payload.Name != nil {
        name := strings.TrimSpace(*payload.Name)
        if name == "" {
            httpx.Error(w, http.StatusBadRequest, "name cannot be empty")
            return
        }
        updates["name"] = name
    }
    if payload.Description != nil {
        updates["description"] = strings.TrimSpace(*payload.Description)
    }
    if payload.Schema != nil {
        updates["schema"] = datatypes.JSONMap(payload.Schema)
    }
    if len(updates) == 0 {
        httpx.Error(w, http.StatusBadRequest, "no updates provided")
        return
    }

    entity, err := h.repo.Update(r.Context(), id, updates)
    if err != nil {
        if IsNotFound(err) {
            httpx.Error(w, http.StatusNotFound, "form not found")
            return
        }
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    httpx.JSON(w, http.StatusOK, map[string]any{"data": entity.ToDTO()})
}

func (h *Handler) deleteForm(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    if err := h.repo.Delete(r.Context(), id); err != nil {
        if IsNotFound(err) {
            httpx.Error(w, http.StatusNotFound, "form not found")
            return
        }
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    w.WriteHeader(http.StatusNoContent)
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
