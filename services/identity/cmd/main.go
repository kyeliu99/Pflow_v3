package main

import (
	"fmt"
	"log"
	"net/http"

	identitycmp "github.com/pflow/components/identity"

	"github.com/pflow/shared/config"
	"github.com/pflow/shared/database"
	"github.com/pflow/shared/httpx"
)

func main() {
	cfg := config.Load()
	dsn := cfg.DatabaseDSN("identity")
	db := database.ConnectWithDSN("identity", dsn)

	if err := db.AutoMigrate(&identitycmp.User{}); err != nil {
		log.Fatalf("identity service: failed to run migrations: %v", err)
	}

	repository := identitycmp.NewGormRepository(db)
	handler := identitycmp.NewHandler(repository)

	server := httpx.New()
	handler.Mount(server.Router, "")

	port := cfg.ResolveServiceHTTPPort("identity", "8082")
	addr := fmt.Sprintf(":%s", port)
	log.Printf("identity service listening on %s", addr)

	if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("identity service stopped: %v", err)
	}
}
