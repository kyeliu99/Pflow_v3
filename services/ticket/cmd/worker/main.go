package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"strings"
	"syscall"

	ticketcmp "github.com/pflow/components/ticket"

	"github.com/pflow/shared/config"
	"github.com/pflow/shared/database"
	"github.com/pflow/shared/mq"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dsn := cfg.DatabaseDSN("ticket")
	db := database.ConnectWithDSN("ticket-worker", dsn)

	if err := db.AutoMigrate(&ticketcmp.Ticket{}, &ticketcmp.TicketSubmission{}); err != nil {
		log.Fatalf("ticket worker: failed to run migrations: %v", err)
	}

	brokers := cfg.KafkaBrokerList("ticket")
	topic := cfg.ResolveServiceQueueTopic("ticket", cfg.KafkaTopic)
	group := cfg.ResolveServiceQueueGroup("ticket", fmt.Sprintf("%s-ticket-workers", cfg.ServiceName))
	if len(brokers) == 0 || strings.TrimSpace(topic) == "" {
		log.Fatalf("ticket worker: kafka brokers/topic must be configured (brokers=%v topic=%s)", brokers, topic)
	}

	store := ticketcmp.NewSubmissionRepository(db)
	repo := ticketcmp.NewGormRepository(db)
	worker := ticketcmp.NewQueueWorker(store, repo)

	consumer, err := mq.NewConsumer(mq.ConsumerConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  group,
		ClientID: fmt.Sprintf("%s-ticket-worker", cfg.ServiceName),
	}, worker.HandleMessage)
	if err != nil {
		log.Fatalf("ticket worker: failed to create consumer: %v", err)
	}
	defer consumer.Close()

	log.Printf("ticket worker consuming topic=%s group=%s", topic, group)

	if err := consumer.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("ticket worker stopped: %v", err)
	}

	log.Println("ticket worker stopped")
}
