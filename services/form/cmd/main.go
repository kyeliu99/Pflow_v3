package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pflow/services/form/internal/form"

	"github.com/pflow/shared/config"
	"github.com/pflow/shared/database"
	"github.com/pflow/shared/httpx"
)

func main() {
	cfg := config.Load()
	db := database.Connect()

	if err := db.AutoMigrate(&form.Form{}); err != nil {
		log.Fatalf("form service: failed to run migrations: %v", err)
	}

	repository := form.NewRepository(db)

	server := httpx.New()
	api := server.Engine.Group("/forms")
	form.RegisterRoutes(api, repository)

	port := cfg.ResolveHTTPPort("8081")
	addr := fmt.Sprintf(":%s", port)
	log.Printf("form service listening on %s", addr)

	if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("form service stopped: %v", err)
	}
}
