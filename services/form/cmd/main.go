package main

import (
	"fmt"
	"log"
	"net/http"

	formcmp "github.com/pflow/components/form"

	"github.com/pflow/shared/config"
	"github.com/pflow/shared/database"
	"github.com/pflow/shared/httpx"
)

func main() {
	cfg := config.Load()
	dsn := cfg.DatabaseDSN("form")
	db := database.ConnectWithDSN("form", dsn)

	if err := db.AutoMigrate(&formcmp.Form{}); err != nil {
		log.Fatalf("form service: failed to run migrations: %v", err)
	}

	repository := formcmp.NewGormRepository(db)
	handler := formcmp.NewHandler(repository)

	server := httpx.New()
	handler.Mount(server.Router, "")

	port := cfg.ResolveServiceHTTPPort("form", "8081")
	addr := fmt.Sprintf(":%s", port)
	log.Printf("form service listening on %s", addr)

	if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("form service stopped: %v", err)
	}
}
