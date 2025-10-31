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

// RegisterRoutes wires the HTTP handlers to the provided router group.
func RegisterRoutes(router chi.Router, repo *Repository) {
	router.Route("/forms", func(r chi.Router) {
		r.Get("/", listFormsHandler(repo))
		r.Post("/", createFormHandler(repo))
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getFormHandler(repo))
			r.Put("/", updateFormHandler(repo))
			r.Delete("/", deleteFormHandler(repo))
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

func listFormsHandler(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search := r.URL.Query().Get("search")
		forms, err := repo.List(r.Context(), search)
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
}

func createFormHandler(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		if err := repo.Create(r.Context(), entity); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		httpx.JSON(w, http.StatusCreated, map[string]any{"data": entity.ToDTO()})
	}
}

func getFormHandler(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		entity, err := repo.Find(r.Context(), id)
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
}

func updateFormHandler(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		entity, err := repo.Update(r.Context(), id, updates)
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
}

func deleteFormHandler(repo *Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := repo.Delete(r.Context(), id); err != nil {
			if IsNotFound(err) {
				httpx.Error(w, http.StatusNotFound, "form not found")
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
