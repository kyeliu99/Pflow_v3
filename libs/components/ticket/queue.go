package ticket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/pflow/shared/mq"
)

// SubmissionRequest captures the normalized payload for asynchronous creation.
type SubmissionRequest struct {
	ClientReference string
	Payload         map[string]any
}

// QueueCoordinator orchestrates submission persistence and queue publication.
type QueueCoordinator struct {
	store    SubmissionStore
	producer *mq.Producer
}

// NewQueueCoordinator constructs a queue-backed submission coordinator.
func NewQueueCoordinator(store SubmissionStore, producer *mq.Producer) *QueueCoordinator {
	return &QueueCoordinator{store: store, producer: producer}
}

// Submit persists a submission and enqueues it for asynchronous processing.
func (c *QueueCoordinator) Submit(ctx context.Context, req SubmissionRequest) (*TicketSubmission, error) {
	if c == nil || c.store == nil {
		return nil, errors.New("ticket submissions are not configured")
	}

	sanitized := make(map[string]any, len(req.Payload))
	for key, value := range req.Payload {
		sanitized[key] = value
	}

	ref := strings.TrimSpace(req.ClientReference)
	if ref != "" {
		if existing, err := c.store.FindByClientReference(ctx, ref); err == nil {
			switch existing.Status {
			case SubmissionCompleted:
				return existing, nil
			case SubmissionPending, SubmissionProcessing:
				return existing, nil
			case SubmissionFailed:
				existing.Status = SubmissionPending
				existing.ErrorMessage = ""
				existing.TicketID = nil
				existing.CompletedAt = nil
				existing.RequestPayload = datatypes.JSONMap(sanitized)
				if err := c.store.Save(ctx, existing); err != nil {
					return nil, err
				}
				if err := c.publish(ctx, existing); err != nil {
					existing.Status = SubmissionFailed
					existing.ErrorMessage = err.Error()
					_ = c.store.Save(ctx, existing)
					return nil, err
				}
				return existing, nil
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	submission := &TicketSubmission{
		ClientReference: ref,
		Status:          SubmissionPending,
		RequestPayload:  datatypes.JSONMap(sanitized),
	}

	if err := c.store.Create(ctx, submission); err != nil {
		return nil, err
	}

	if err := c.publish(ctx, submission); err != nil {
		submission.Status = SubmissionFailed
		submission.ErrorMessage = err.Error()
		_ = c.store.Save(ctx, submission)
		return nil, err
	}

	return submission, nil
}

// Lookup fetches a submission by ID.
func (c *QueueCoordinator) Lookup(ctx context.Context, id string) (*TicketSubmission, error) {
	if c == nil || c.store == nil {
		return nil, errors.New("ticket submissions are not configured")
	}
	return c.store.FindByID(ctx, id)
}

// Metrics exposes queue statistics for observability.
func (c *QueueCoordinator) Metrics(ctx context.Context) (SubmissionMetrics, error) {
	if c == nil || c.store == nil {
		return SubmissionMetrics{}, errors.New("ticket submissions are not configured")
	}
	return c.store.Metrics(ctx)
}

func (c *QueueCoordinator) publish(ctx context.Context, submission *TicketSubmission) error {
	if c.producer == nil {
		return errors.New("queue producer not configured")
	}

	payload, err := json.Marshal(struct {
		SubmissionID string `json:"submissionId"`
	}{SubmissionID: submission.ID})
	if err != nil {
		return fmt.Errorf("marshal submission payload: %w", err)
	}

	return c.producer.Publish(ctx, submission.ID, payload, map[string]string{
		"submitted_at": submission.CreatedAt.Format(time.RFC3339Nano),
	})
}
