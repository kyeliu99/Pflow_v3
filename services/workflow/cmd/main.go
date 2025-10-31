package main

import (
	"fmt"
	"log"
	"net/http"

	workflowcmp "github.com/pflow/components/workflow"

	"github.com/pflow/shared/config"
	"github.com/pflow/shared/database"
	"github.com/pflow/shared/httpx"
)

func main() {
	cfg := config.Load()
	dsn := cfg.DatabaseDSN("workflow")
	db := database.ConnectWithDSN("workflow", dsn)

	if err := db.AutoMigrate(&workflowcmp.Definition{}); err != nil {
		log.Fatalf("workflow service: failed to run migrations: %v", err)
	}

	repository := workflowcmp.NewGormRepository(db)
	handler := workflowcmp.NewHandler(repository)

	server := httpx.New()
	handler.Mount(server.Router, "")

	port := cfg.ResolveServiceHTTPPort("workflow", "8084")
	addr := fmt.Sprintf(":%s", port)
	log.Printf("workflow service listening on %s", addr)

	if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("workflow service stopped: %v", err)
	}
}
