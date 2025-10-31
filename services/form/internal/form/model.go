package form

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Form represents a persisted form definition that can be attached to a workflow.
type Form struct {
	ID          string            `json:"id" gorm:"type:uuid;primaryKey"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Schema      datatypes.JSONMap `json:"schema" gorm:"type:jsonb"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

// BeforeCreate ensures that a UUID is present for new records.
func (f *Form) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.NewString()
	}
	return nil
}

// ToDTO converts the model into a response-friendly structure.
func (f Form) ToDTO() map[string]any {
	schema := map[string]any{}
	if f.Schema != nil {
		schema = map[string]any(f.Schema)
	}

	return map[string]any{
		"id":          f.ID,
		"name":        f.Name,
		"description": f.Description,
		"schema":      schema,
		"createdAt":   f.CreatedAt,
		"updatedAt":   f.UpdatedAt,
	}
}
