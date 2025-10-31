package identity

import (
    "encoding/json"
    "errors"
    "io"
    "net/http"
    "net/mail"
    "strings"

    "github.com/go-chi/chi/v5"

    "github.com/pflow/shared/httpx"
)

// Handler exposes HTTP handlers for user management.
type Handler struct {
    repo Repository
}

// NewHandler creates a new identity Handler.
func NewHandler(repo Repository) *Handler {
    return &Handler{repo: repo}
}

// Mount registers user routes on the provided router under the supplied base path.
func (h *Handler) Mount(router chi.Router, basePath string) {
    path := strings.TrimSpace(basePath)
    if path == "" {
        path = "/users"
    }

    router.Route(path, func(r chi.Router) {
        r.Get("/", h.listUsers)
        r.Post("/", h.createUser)
        r.Route("/{id}", func(r chi.Router) {
            r.Get("/", h.getUser)
            r.Put("/", h.updateUser)
            r.Delete("/", h.deleteUser)
        })
    })
}

type createUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Role  string `json:"role"`
}

type updateUserRequest struct {
    Name  *string `json:"name"`
    Email *string `json:"email"`
    Role  *string `json:"role"`
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
    role := strings.TrimSpace(r.URL.Query().Get("role"))
    search := strings.TrimSpace(r.URL.Query().Get("search"))

    users, err := h.repo.List(r.Context(), role, search)
    if err != nil {
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    items := make([]map[string]any, 0, len(users))
    for _, entity := range users {
        items = append(items, entity.ToDTO())
    }

    httpx.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
    var payload createUserRequest
    if err := decodeJSON(r, &payload); err != nil {
        httpx.Error(w, http.StatusBadRequest, err.Error())
        return
    }

    name := strings.TrimSpace(payload.Name)
    email := strings.ToLower(strings.TrimSpace(payload.Email))
    role := strings.TrimSpace(payload.Role)

    if name == "" {
        httpx.Error(w, http.StatusBadRequest, "name is required")
        return
    }
    if email == "" {
        httpx.Error(w, http.StatusBadRequest, "email is required")
        return
    }
    if _, err := mail.ParseAddress(email); err != nil {
        httpx.Error(w, http.StatusBadRequest, "email is invalid")
        return
    }
    if role == "" {
        httpx.Error(w, http.StatusBadRequest, "role is required")
        return
    }

    entity := &User{
        Name:  name,
        Email: email,
        Role:  role,
    }

    if err := h.repo.Create(r.Context(), entity); err != nil {
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    httpx.JSON(w, http.StatusCreated, map[string]any{"data": entity.ToDTO()})
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    entity, err := h.repo.Find(r.Context(), id)
    if err != nil {
        if IsNotFound(err) {
            httpx.Error(w, http.StatusNotFound, "user not found")
            return
        }
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    httpx.JSON(w, http.StatusOK, map[string]any{"data": entity.ToDTO()})
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    var payload updateUserRequest
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
    if payload.Email != nil {
        email := strings.ToLower(strings.TrimSpace(*payload.Email))
        if email == "" {
            httpx.Error(w, http.StatusBadRequest, "email cannot be empty")
            return
        }
        if _, err := mail.ParseAddress(email); err != nil {
            httpx.Error(w, http.StatusBadRequest, "email is invalid")
            return
        }
        updates["email"] = email
    }
    if payload.Role != nil {
        role := strings.TrimSpace(*payload.Role)
        if role == "" {
            httpx.Error(w, http.StatusBadRequest, "role cannot be empty")
            return
        }
        updates["role"] = role
    }

    if len(updates) == 0 {
        httpx.Error(w, http.StatusBadRequest, "no updates provided")
        return
    }

    entity, err := h.repo.Update(r.Context(), id, updates)
    if err != nil {
        if IsNotFound(err) {
            httpx.Error(w, http.StatusNotFound, "user not found")
            return
        }
        httpx.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    httpx.JSON(w, http.StatusOK, map[string]any{"data": entity.ToDTO()})
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    if err := h.repo.Delete(r.Context(), id); err != nil {
        if IsNotFound(err) {
            httpx.Error(w, http.StatusNotFound, "user not found")
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
