package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User captures an account within the identity service.
type User struct {
	ID        string    `json:"id" gorm:"type:uuid;primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Role      string    `json:"role" gorm:"not null"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BeforeCreate ensures a UUID exists.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return nil
}

// ToDTO renders the response payload.
func (u User) ToDTO() map[string]any {
	return map[string]any{
		"id":        u.ID,
		"name":      u.Name,
		"email":     u.Email,
		"role":      u.Role,
		"createdAt": u.CreatedAt,
		"updatedAt": u.UpdatedAt,
	}
}
