package user

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

// RegisterRoutes mounts identity handlers on the router.
func RegisterRoutes(router chi.Router, repo *Repository) {
	router.Route("/users", func(r chi.Router) {
		r.Get("/", listUsersHandler(repo))
		r.Post("/", createUserHandler(repo))
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getUserHandler(repo))
			r.Put("/", updateUserHandler(repo))
			r.Delete("/", deleteUserHandler(repo))
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

func listUsersHandler(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role := r.URL.Query().Get("role")
		search := r.URL.Query().Get("search")

		users, err := repo.List(r.Context(), role, search)
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
}

func createUserHandler(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		if err := repo.Create(r.Context(), entity); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		httpx.JSON(w, http.StatusCreated, map[string]any{"data": entity.ToDTO()})
	}
}

func getUserHandler(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		entity, err := repo.Find(r.Context(), id)
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
}

func updateUserHandler(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		entity, err := repo.Update(r.Context(), id, updates)
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
}

func deleteUserHandler(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := repo.Delete(r.Context(), id); err != nil {
			if IsNotFound(err) {
				httpx.Error(w, http.StatusNotFound, "user not found")
				return
			}
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
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
