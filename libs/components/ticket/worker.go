package ticket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/pflow/shared/mq"
)

// QueueWorker processes submission messages and materialises tickets.
type QueueWorker struct {
	store SubmissionStore
	repo  Repository
}

// NewQueueWorker constructs a queue worker.
func NewQueueWorker(store SubmissionStore, repo Repository) *QueueWorker {
	return &QueueWorker{store: store, repo: repo}
}

// HandleMessage consumes a submission message from Kafka.
func (w *QueueWorker) HandleMessage(ctx context.Context, msg mq.Message) error {
	if w == nil || w.store == nil || w.repo == nil {
		return fmt.Errorf("ticket worker not initialised")
	}

	var payload struct {
		SubmissionID string `json:"submissionId"`
	}
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		return fmt.Errorf("decode submission message: %w", err)
	}
	if strings.TrimSpace(payload.SubmissionID) == "" {
		return fmt.Errorf("submission id missing from message")
	}

	submission, err := w.store.FindByID(ctx, payload.SubmissionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("ticket worker: submission %s not found, skipping", payload.SubmissionID)
			return nil
		}
		return err
	}

	if submission.Status == SubmissionCompleted {
		return nil
	}

	submission.Status = SubmissionProcessing
	submission.ErrorMessage = ""
	if err := w.store.Save(ctx, submission); err != nil {
		return err
	}

	ticket, err := submission.ToTicket()
	if err != nil {
		submission.Status = SubmissionFailed
		submission.ErrorMessage = err.Error()
		if saveErr := w.store.Save(ctx, submission); saveErr != nil {
			log.Printf("ticket worker: failed to persist submission failure: %v", saveErr)
		}
		return err
	}

	if err := w.repo.Create(ctx, ticket); err != nil {
		submission.Status = SubmissionFailed
		submission.ErrorMessage = err.Error()
		if saveErr := w.store.Save(ctx, submission); saveErr != nil {
			log.Printf("ticket worker: failed to persist submission failure: %v", saveErr)
		}
		return err
	}

	submission.Status = SubmissionCompleted
	submission.TicketID = &ticket.ID
	now := time.Now()
	submission.CompletedAt = &now
	if err := w.store.Save(ctx, submission); err != nil {
		return err
	}

	log.Printf("ticket worker: processed submission %s -> ticket %s", submission.ID, ticket.ID)
	return nil
}

// RunConsumer starts the provided consumer using the worker handler.
func (w *QueueWorker) RunConsumer(ctx context.Context, consumer *mq.Consumer) error {
	if consumer == nil {
		return fmt.Errorf("consumer is nil")
	}
	return consumer.Run(ctx)
}
