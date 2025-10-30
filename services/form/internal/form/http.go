package form

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// RegisterRoutes wires the HTTP handlers to the provided router group.
func RegisterRoutes(router gin.IRouter, repo *Repository) {
	router.GET("", listFormsHandler(repo))
	router.POST("", createFormHandler(repo))
	router.GET(":id", getFormHandler(repo))
	router.PUT(":id", updateFormHandler(repo))
	router.DELETE(":id", deleteFormHandler(repo))
}

type createFormRequest struct {
	Name        string         `json:"name" binding:"required,min=1"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema"`
}

type updateFormRequest struct {
	Name        *string        `json:"name"`
	Description *string        `json:"description"`
	Schema      map[string]any `json:"schema"`
}

func listFormsHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		search := c.Query("search")
		forms, err := repo.List(c.Request.Context(), search)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		items := make([]map[string]any, 0, len(forms))
		for _, entity := range forms {
			items = append(items, entity.ToDTO())
		}

		c.JSON(http.StatusOK, gin.H{"data": items})
	}
}

func createFormHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload createFormRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		entity := &Form{
			Name:        payload.Name,
			Description: payload.Description,
		}
		if payload.Schema != nil {
			entity.Schema = datatypes.JSONMap(payload.Schema)
		}

		if err := repo.Create(c.Request.Context(), entity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": entity.ToDTO()})
	}
}

func getFormHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		entity, err := repo.Find(c.Request.Context(), id)
		if err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "form not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": entity.ToDTO()})
	}
}

func updateFormHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var payload updateFormRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updates := make(map[string]any)
		if payload.Name != nil {
			updates["name"] = *payload.Name
		}
		if payload.Description != nil {
			updates["description"] = *payload.Description
		}
		if payload.Schema != nil {
			updates["schema"] = datatypes.JSONMap(payload.Schema)
		}
		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no updates provided"})
			return
		}

		entity, err := repo.Update(c.Request.Context(), id, updates)
		if err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "form not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": entity.ToDTO()})
	}
}

func deleteFormHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := repo.Delete(c.Request.Context(), id); err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "form not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	}
}
