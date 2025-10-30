package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pflow/workflow/internal/workflow"

	"github.com/pflow/shared/config"
	"github.com/pflow/shared/database"
	"github.com/pflow/shared/httpx"
)

func main() {
	cfg := config.Load()
	db := database.Connect()

	if err := db.AutoMigrate(&workflow.Definition{}); err != nil {
		log.Fatalf("workflow service: failed to run migrations: %v", err)
	}

	repository := workflow.NewRepository(db)

	server := httpx.New()
	workflow.RegisterRoutes(server.Router, repository)

	port := cfg.ResolveHTTPPort("8084")
	addr := fmt.Sprintf(":%s", port)
	log.Printf("workflow service listening on %s", addr)

	if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("workflow service stopped: %v", err)
	}
}
