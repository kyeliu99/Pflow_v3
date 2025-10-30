package ticket

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

var allowedStatuses = map[string]struct{}{
	StatusOpen:       {},
	StatusInProgress: {},
	StatusResolved:   {},
	StatusCancelled:  {},
}

// RegisterRoutes mounts the ticket handlers.
func RegisterRoutes(router gin.IRouter, repo *Repository) {
	router.GET("", listTicketsHandler(repo))
	router.POST("", createTicketHandler(repo))
	router.GET(":id", getTicketHandler(repo))
	router.PATCH(":id", updateTicketHandler(repo))
	router.DELETE(":id", deleteTicketHandler(repo))
	router.POST(":id/resolve", resolveTicketHandler(repo))
}

type createTicketRequest struct {
	Title      string         `json:"title" binding:"required,min=3"`
	Status     string         `json:"status"`
	FormID     string         `json:"formId" binding:"required,uuid4"`
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

func listTicketsHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		assignee := c.Query("assigneeId")

		tickets, err := repo.List(c.Request.Context(), status, assignee)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		items := make([]map[string]any, 0, len(tickets))
		for _, entity := range tickets {
			items = append(items, entity.ToDTO())
		}

		c.JSON(http.StatusOK, gin.H{"data": items})
	}
}

func createTicketHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload createTicketRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		status := strings.ToLower(strings.TrimSpace(payload.Status))
		if status == "" {
			status = StatusOpen
		}
		if !isValidStatus(status) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}

		entity := &Ticket{
			Title:      strings.TrimSpace(payload.Title),
			Status:     status,
			FormID:     strings.TrimSpace(payload.FormID),
			AssigneeID: strings.TrimSpace(payload.AssigneeID),
		}
		if payload.Priority != "" {
			entity.Priority = strings.ToLower(strings.TrimSpace(payload.Priority))
		}
		if payload.Metadata != nil {
			entity.Metadata = datatypes.JSONMap(payload.Metadata)
		}

		if err := repo.Create(c.Request.Context(), entity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": entity.ToDTO()})
	}
}

func getTicketHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		entity, err := repo.Find(c.Request.Context(), id)
		if err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": entity.ToDTO()})
	}
}

func updateTicketHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var payload updateTicketRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updates := make(map[string]any)
		if payload.Title != nil {
			updates["title"] = strings.TrimSpace(*payload.Title)
		}
		if payload.Status != nil {
			status := strings.ToLower(strings.TrimSpace(*payload.Status))
			if !isValidStatus(status) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "no updates provided"})
			return
		}

		entity, err := repo.Update(c.Request.Context(), id, updates)
		if err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": entity.ToDTO()})
	}
}

func deleteTicketHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := repo.Delete(c.Request.Context(), id); err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func resolveTicketHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		entity, err := repo.Resolve(c.Request.Context(), id)
		if err != nil {
			if IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": entity.ToDTO()})
	}
}

func isValidStatus(status string) bool {
	_, ok := allowedStatuses[strings.ToLower(status)]
	return ok
}
