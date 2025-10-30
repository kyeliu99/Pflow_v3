package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pflow/shared/config"
	"github.com/pflow/shared/database"
	"github.com/pflow/shared/httpx"
	"github.com/pflow/shared/messaging"
)

type ticket struct {
	ID         string                 `json:"id"`
	FormID     string                 `json:"formId"`
	WorkflowID string                 `json:"workflowId"`
	Payload    map[string]interface{} `json:"payload"`
	Status     string                 `json:"status"`
}

func main() {
	cfg := config.Load()
	database.Connect()

	server := httpx.New()
	api := server.Engine.Group("/tickets")
	{
		api.GET("", listTickets)
		api.POST("", createTicket)
		api.GET(":id", getTicket)
	}

	go consumeWorkflowEvents()

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Printf("ticket service listening on %s", addr)
	if err := server.Start(addr); err != nil {
		log.Fatalf("ticket service stopped: %v", err)
	}
}

func listTickets(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []ticket{}})
}

func createTicket(c *gin.Context) {
	var payload ticket
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	go func(t ticket) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = messaging.Publish(ctx, "ticket.created", []byte(t.ID))
	}(payload)

	c.JSON(http.StatusCreated, gin.H{"data": payload})
}

func getTicket(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": ticket{ID: c.Param("id"), Status: "pending"}})
}

func consumeWorkflowEvents() {
	ctx := context.Background()
	for {
		msg, err := messaging.Consume(ctx)
		if err != nil {
			log.Printf("ticket consumer error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("ticket event received: %s", string(msg.Value))
		if err := messaging.Commit(ctx, msg); err != nil {
			log.Printf("ticket commit error: %v", err)
		}
	}
}
