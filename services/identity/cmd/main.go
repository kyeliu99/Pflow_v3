package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pflow/identity/internal/user"

	"github.com/pflow/shared/config"
	"github.com/pflow/shared/database"
	"github.com/pflow/shared/httpx"
)

func main() {
	cfg := config.Load()
	db := database.Connect()

	if err := db.AutoMigrate(&user.User{}); err != nil {
		log.Fatalf("identity service: failed to run migrations: %v", err)
	}

	repository := user.NewRepository(db)

	server := httpx.New()
	user.RegisterRoutes(server.Router, repository)

	port := cfg.ResolveHTTPPort("8082")
	addr := fmt.Sprintf(":%s", port)
	log.Printf("identity service listening on %s", addr)

	if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("identity service stopped: %v", err)
	}
}
