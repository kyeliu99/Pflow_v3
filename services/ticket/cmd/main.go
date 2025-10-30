package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pflow/services/ticket/internal/ticket"

	"github.com/pflow/shared/config"
	"github.com/pflow/shared/database"
	"github.com/pflow/shared/httpx"
)

func main() {
	cfg := config.Load()
	db := database.Connect()

	if err := db.AutoMigrate(&ticket.Ticket{}); err != nil {
		log.Fatalf("ticket service: failed to run migrations: %v", err)
	}

	repository := ticket.NewRepository(db)

	server := httpx.New()
	api := server.Engine.Group("/tickets")
	ticket.RegisterRoutes(api, repository)

	port := cfg.ResolveHTTPPort("8083")
	addr := fmt.Sprintf(":%s", port)
	log.Printf("ticket service listening on %s", addr)

	if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("ticket service stopped: %v", err)
	}
}
