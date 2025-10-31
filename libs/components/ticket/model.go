package ticket

import (
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

// BeforeCreate assigns a UUID when missing.
func (t *Ticket) BeforeCreate(tx *gorm.DB) error {
    if t.ID == "" {
        t.ID = uuid.NewString()
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
