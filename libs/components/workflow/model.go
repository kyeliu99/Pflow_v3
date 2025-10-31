package workflow

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/datatypes"
    "gorm.io/gorm"
)

// Definition captures a workflow blueprint stored in the workflow service.
type Definition struct {
    ID          string            `json:"id" gorm:"type:uuid;primaryKey"`
    Name        string            `json:"name" gorm:"not null"`
    Version     int               `json:"version" gorm:"not null;default:1"`
    Description string            `json:"description"`
    Blueprint   datatypes.JSONMap `json:"blueprint" gorm:"type:jsonb"`
    Published   bool              `json:"published" gorm:"index"`
    CreatedAt   time.Time         `json:"createdAt"`
    UpdatedAt   time.Time         `json:"updatedAt"`
}

// BeforeCreate ensures a UUID exists.
func (d *Definition) BeforeCreate(tx *gorm.DB) error {
    if d.ID == "" {
        d.ID = uuid.NewString()
    }
    return nil
}

// ToDTO converts a definition into a response payload.
func (d Definition) ToDTO() map[string]any {
    payload := map[string]any{
        "id":          d.ID,
        "name":        d.Name,
        "version":     d.Version,
        "description": d.Description,
        "published":   d.Published,
        "createdAt":   d.CreatedAt,
        "updatedAt":   d.UpdatedAt,
    }
    if d.Blueprint != nil {
        payload["blueprint"] = map[string]any(d.Blueprint)
    } else {
        payload["blueprint"] = map[string]any{}
    }
    return payload
}
