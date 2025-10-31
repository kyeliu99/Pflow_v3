package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	ticketcmp "github.com/pflow/components/ticket"

	"github.com/pflow/shared/config"
	"github.com/pflow/shared/database"
	"github.com/pflow/shared/httpx"
	"github.com/pflow/shared/mq"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dsn := cfg.DatabaseDSN("ticket")
	db := database.ConnectWithDSN("ticket", dsn)

	if err := db.AutoMigrate(&ticketcmp.Ticket{}, &ticketcmp.TicketSubmission{}); err != nil {
		log.Fatalf("ticket service: failed to run migrations: %v", err)
	}

	repository := ticketcmp.NewGormRepository(db)
	submissionStore := ticketcmp.NewSubmissionRepository(db)

	brokers := cfg.KafkaBrokerList("ticket")
	topic := cfg.ResolveServiceQueueTopic("ticket", cfg.KafkaTopic)
	if len(brokers) == 0 || strings.TrimSpace(topic) == "" {
		log.Fatalf("ticket service: kafka brokers/topic must be configured (brokers=%v topic=%s)", brokers, topic)
	}

	producer, err := mq.NewProducer(mq.ProducerConfig{
		Brokers:  brokers,
		Topic:    topic,
		ClientID: fmt.Sprintf("%s-ticket-api", cfg.ServiceName),
		Timeout:  2 * time.Second,
	})
	if err != nil {
		log.Fatalf("ticket service: failed to initialise producer: %v", err)
	}
	defer producer.Close(context.Background())

	coordinator := ticketcmp.NewQueueCoordinator(submissionStore, producer)

	handler := ticketcmp.NewHandler(repository, ticketcmp.WithSubmissionCoordinator(coordinator))

	server := httpx.New()
	handler.Mount(server.Router, "")

	port := cfg.ResolveServiceHTTPPort("ticket", "8083")
	addr := fmt.Sprintf(":%s", port)
	log.Printf("ticket service listening on %s", addr)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("ticket service: graceful shutdown error: %v", err)
		}
	}()

	if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("ticket service stopped: %v", err)
	}

	log.Println("ticket service stopped")
}
