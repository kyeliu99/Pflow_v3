package ticket

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	// StatusOpen indicates a ticket awaiting processing.
	StatusOpen = "open"
	// StatusInProgress indicates an active ticket.
	StatusInProgress = "in_progress"
	// StatusResolved indicates a completed ticket.
	StatusResolved = "resolved"
	// StatusCancelled indicates a ticket that was cancelled.
	StatusCancelled = "cancelled"
)

const (
	// SubmissionPending represents a queued submission.
	SubmissionPending = "pending"
	// SubmissionProcessing represents a submission currently being processed.
	SubmissionProcessing = "processing"
	// SubmissionCompleted marks a submission that finished successfully.
	SubmissionCompleted = "completed"
	// SubmissionFailed marks a submission that failed permanently.
	SubmissionFailed = "failed"
)

// Ticket represents a workflow-driven work item.
type Ticket struct {
	ID         string            `json:"id" gorm:"type:uuid;primaryKey"`
	Title      string            `json:"title" gorm:"not null"`
	Status     string            `json:"status" gorm:"not null;index"`
	FormID     string            `json:"formId" gorm:"type:uuid;not null;index"`
	AssigneeID string            `json:"assigneeId" gorm:"type:uuid;index"`
	Priority   string            `json:"priority" gorm:"default:'medium'"`
	Metadata   datatypes.JSONMap `json:"metadata" gorm:"type:jsonb"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
	ResolvedAt *time.Time        `json:"resolvedAt"`
}

// TicketSubmission captures asynchronous ticket creation requests.
type TicketSubmission struct {
	ID              string            `json:"id" gorm:"type:uuid;primaryKey"`
	ClientReference string            `json:"clientReference" gorm:"type:varchar(128);uniqueIndex"`
	Status          string            `json:"status" gorm:"not null;index"`
	ErrorMessage    string            `json:"errorMessage"`
	TicketID        *string           `json:"ticketId" gorm:"type:uuid;index"`
	RequestPayload  datatypes.JSONMap `json:"requestPayload" gorm:"type:jsonb"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
	CompletedAt     *time.Time        `json:"completedAt"`
}

// BeforeCreate assigns a UUID when missing.
func (t *Ticket) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}

// BeforeCreate assigns defaults on submissions.
func (s *TicketSubmission) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.ClientReference == "" {
		s.ClientReference = s.ID
	}
	if s.Status == "" {
		s.Status = SubmissionPending
	}
	return nil
}

// ToDTO converts a ticket into a serialisable map.
func (t Ticket) ToDTO() map[string]any {
	payload := map[string]any{
		"id":         t.ID,
		"title":      t.Title,
		"status":     t.Status,
		"formId":     t.FormID,
		"assigneeId": t.AssigneeID,
		"priority":   t.Priority,
		"createdAt":  t.CreatedAt,
		"updatedAt":  t.UpdatedAt,
	}
	if t.Metadata != nil {
		payload["metadata"] = map[string]any(t.Metadata)
	} else {
		payload["metadata"] = map[string]any{}
	}
	if t.ResolvedAt != nil {
		payload["resolvedAt"] = t.ResolvedAt
	}
	return payload
}

// ToDTO exposes submission data for clients.
func (s TicketSubmission) ToDTO() map[string]any {
	dto := map[string]any{
		"id":              s.ID,
		"clientReference": s.ClientReference,
		"status":          s.Status,
		"createdAt":       s.CreatedAt,
		"updatedAt":       s.UpdatedAt,
	}
	if s.TicketID != nil {
		dto["ticketId"] = *s.TicketID
	}
	if s.CompletedAt != nil {
		dto["completedAt"] = s.CompletedAt
	}
	if s.ErrorMessage != "" {
		dto["errorMessage"] = s.ErrorMessage
	}
	return dto
}

// ToTicket reconstructs a Ticket entity from the stored payload.
func (s TicketSubmission) ToTicket() (*Ticket, error) {
	payload := map[string]any(s.RequestPayload)
	title, _ := payload["title"].(string)
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("ticket submission missing title")
	}

	formID, _ := payload["formId"].(string)
	if strings.TrimSpace(formID) == "" {
		return nil, errors.New("ticket submission missing formId")
	}

	statusValue, _ := payload["status"].(string)
	if statusValue == "" {
		statusValue = StatusOpen
	}

	assignee, _ := payload["assigneeId"].(string)
	priorityValue, _ := payload["priority"].(string)
	if priorityValue == "" {
		priorityValue = "medium"
	}

	ticket := &Ticket{
		Title:      strings.TrimSpace(title),
		Status:     strings.TrimSpace(statusValue),
		FormID:     strings.TrimSpace(formID),
		AssigneeID: strings.TrimSpace(assignee),
		Priority:   strings.TrimSpace(priorityValue),
	}

	if metadataRaw, ok := payload["metadata"].(map[string]any); ok {
		ticket.Metadata = datatypes.JSONMap(metadataRaw)
	}
	return ticket, nil
}
