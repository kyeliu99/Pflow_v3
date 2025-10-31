package ticket

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SubmissionMetrics exposes aggregated queue insights.
type SubmissionMetrics struct {
	Pending              int `json:"pending"`
	Processing           int `json:"processing"`
	Completed            int `json:"completed"`
	Failed               int `json:"failed"`
	OldestPendingSeconds int `json:"oldestPendingSeconds"`
}

// SubmissionStore handles persistence of ticket submissions.
type SubmissionStore interface {
	Create(ctx context.Context, submission *TicketSubmission) error
	Save(ctx context.Context, submission *TicketSubmission) error
	FindByID(ctx context.Context, id string) (*TicketSubmission, error)
	FindByClientReference(ctx context.Context, ref string) (*TicketSubmission, error)
	Metrics(ctx context.Context) (SubmissionMetrics, error)
}

// GormSubmissionRepository persists submissions via GORM.
type GormSubmissionRepository struct {
	db *gorm.DB
}

// NewSubmissionRepository constructs a repository backed by the provided DB connection.
func NewSubmissionRepository(db *gorm.DB) *GormSubmissionRepository {
	return &GormSubmissionRepository{db: db}
}

// Create inserts a new submission.
func (r *GormSubmissionRepository) Create(ctx context.Context, submission *TicketSubmission) error {
	return r.db.WithContext(ctx).Create(submission).Error
}

// Save persists changes to a submission.
func (r *GormSubmissionRepository) Save(ctx context.Context, submission *TicketSubmission) error {
	return r.db.WithContext(ctx).Save(submission).Error
}

// FindByID locates a submission by primary key.
func (r *GormSubmissionRepository) FindByID(ctx context.Context, id string) (*TicketSubmission, error) {
	var entity TicketSubmission
	if err := r.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

// FindByClientReference locates a submission by idempotency key.
func (r *GormSubmissionRepository) FindByClientReference(ctx context.Context, ref string) (*TicketSubmission, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var entity TicketSubmission
	if err := r.db.WithContext(ctx).First(&entity, "client_reference = ?", ref).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

// Metrics aggregates queue counts and wait times.
func (r *GormSubmissionRepository) Metrics(ctx context.Context) (SubmissionMetrics, error) {
	metrics := SubmissionMetrics{}

	type result struct {
		Status string
		Total  int
	}

	var rows []result
	if err := r.db.WithContext(ctx).
		Model(&TicketSubmission{}).
		Select("status, COUNT(*) as total").
		Group("status").
		Find(&rows).Error; err != nil {
		return metrics, err
	}

	for _, row := range rows {
		switch row.Status {
		case SubmissionPending:
			metrics.Pending = row.Total
		case SubmissionProcessing:
			metrics.Processing = row.Total
		case SubmissionCompleted:
			metrics.Completed = row.Total
		case SubmissionFailed:
			metrics.Failed = row.Total
		}
	}

	var oldest TicketSubmission
	err := r.db.WithContext(ctx).
		Model(&TicketSubmission{}).
		Where("status IN ?", []string{SubmissionPending, SubmissionProcessing}).
		Order("created_at ASC").
		Limit(1).
		Find(&oldest).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return metrics, err
	}

	if !oldest.CreatedAt.IsZero() {
		wait := time.Since(oldest.CreatedAt)
		if wait < 0 {
			wait = 0
		}
		metrics.OldestPendingSeconds = int(wait.Seconds())
	}

	return metrics, nil
}
