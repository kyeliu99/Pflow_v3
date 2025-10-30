package workflow

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// RegisterRoutes exposes workflow HTTP endpoints.
func RegisterRoutes(router gin.IRouter, repo *Repository) {
	router.GET("", listDefinitionsHandler(repo))
	router.POST("", createDefinitionHandler(repo))
	router.GET(":id", getDefinitionHandler(repo))
	router.PUT(":id", updateDefinitionHandler(repo))
	router.DELETE(":id", deleteDefinitionHandler(repo))
	router.POST(":id/publish", publishDefinitionHandler(repo))
}

type createDefinitionRequest struct {
	Name        string         `json:"name" binding:"required,min=2"`
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

func listDefinitionsHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var publishedFilter *bool
		if value := c.Query("published"); value != "" {
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid published filter"})
				return
			}
			publishedFilter = &parsed
		}

		definitions, err := repo.List(c.Request.Context(), publishedFilter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		items := make([]map[string]any, 0, len(definitions))
		for _, entity := range definitions {
			items = append(items, entity.ToDTO())
		}

		c.JSON(http.StatusOK, gin.H{"data": items})
	}
}

func createDefinitionHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload createDefinitionRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		version := payload.Version
		if version <= 0 {
			version = 1
		}

		entity := &Definition{
			Name:        strings.TrimSpace(payload.Name),
			Version:     version,
			Description: strings.TrimSpace(payload.Description),
		}
		if payload.Blueprint != nil {
			entity.Blueprint = datatypes.JSONMap(payload.Blueprint)
		}

		if err := repo.Create(c.Request.Context(), entity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": entity.ToDTO()})
	}
}

func getDefinitionHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		entity, err := repo.Find(c.Request.Context(), id)
		if err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": entity.ToDTO()})
	}
}

func updateDefinitionHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var payload updateDefinitionRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updates := make(map[string]any)
		if payload.Name != nil {
			updates["name"] = strings.TrimSpace(*payload.Name)
		}
		if payload.Version != nil {
			if *payload.Version <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "version must be positive"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "no updates provided"})
			return
		}

		entity, err := repo.Update(c.Request.Context(), id, updates)
		if err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": entity.ToDTO()})
	}
}

func deleteDefinitionHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := repo.Delete(c.Request.Context(), id); err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func publishDefinitionHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		entity, err := repo.Publish(c.Request.Context(), id)
		if err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": entity.ToDTO()})
	}
}
