package user

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts identity handlers on the router.
func RegisterRoutes(router gin.IRouter, repo *Repository) {
	router.GET("/users", listUsersHandler(repo))
	router.POST("/users", createUserHandler(repo))
	router.GET("/users/:id", getUserHandler(repo))
	router.PUT("/users/:id", updateUserHandler(repo))
	router.DELETE("/users/:id", deleteUserHandler(repo))
}

type createUserRequest struct {
	Name  string `json:"name" binding:"required,min=1"`
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,min=2"`
}

type updateUserRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Role  *string `json:"role"`
}

func listUsersHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.Query("role")
		search := c.Query("search")

		users, err := repo.List(c.Request.Context(), role, search)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		items := make([]map[string]any, 0, len(users))
		for _, entity := range users {
			items = append(items, entity.ToDTO())
		}

		c.JSON(http.StatusOK, gin.H{"data": items})
	}
}

func createUserHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload createUserRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		entity := &User{
			Name:  strings.TrimSpace(payload.Name),
			Email: strings.ToLower(strings.TrimSpace(payload.Email)),
			Role:  strings.TrimSpace(payload.Role),
		}

		if err := repo.Create(c.Request.Context(), entity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": entity.ToDTO()})
	}
}

func getUserHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		entity, err := repo.Find(c.Request.Context(), id)
		if err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": entity.ToDTO()})
	}
}

func updateUserHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var payload updateUserRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updates := make(map[string]any)
		if payload.Name != nil {
			updates["name"] = strings.TrimSpace(*payload.Name)
		}
		if payload.Email != nil {
			updates["email"] = strings.ToLower(strings.TrimSpace(*payload.Email))
		}
		if payload.Role != nil {
			updates["role"] = strings.TrimSpace(*payload.Role)
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no updates provided"})
			return
		}

		entity, err := repo.Update(c.Request.Context(), id, updates)
		if err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": entity.ToDTO()})
	}
}

func deleteUserHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := repo.Delete(c.Request.Context(), id); err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	}
}
