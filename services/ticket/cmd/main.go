package main

import (
	"fmt"
	"log"
	"net/http"

	ticketcmp "github.com/pflow/components/ticket"

	"github.com/pflow/shared/config"
	"github.com/pflow/shared/database"
	"github.com/pflow/shared/httpx"
)

func main() {
	cfg := config.Load()
	dsn := cfg.DatabaseDSN("ticket")
	db := database.ConnectWithDSN("ticket", dsn)

	if err := db.AutoMigrate(&ticketcmp.Ticket{}); err != nil {
		log.Fatalf("ticket service: failed to run migrations: %v", err)
	}

	repository := ticketcmp.NewGormRepository(db)
	handler := ticketcmp.NewHandler(repository)

	server := httpx.New()
	handler.Mount(server.Router, "")

	port := cfg.ResolveServiceHTTPPort("ticket", "8083")
	addr := fmt.Sprintf(":%s", port)
	log.Printf("ticket service listening on %s", addr)

	if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("ticket service stopped: %v", err)
	}
}
