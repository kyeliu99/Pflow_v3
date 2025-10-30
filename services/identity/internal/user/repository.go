package user

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// Repository manages persistence of users.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new user repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// List returns users optionally filtered by role or search query.
func (r *Repository) List(ctx context.Context, role, search string) ([]User, error) {
	query := r.db.WithContext(ctx).Model(&User{}).Order("created_at DESC")
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("LOWER(name) LIKE LOWER(?) OR LOWER(email) LIKE LOWER(?)", like, like)
	}

	var users []User
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Create persists a new user.
func (r *Repository) Create(ctx context.Context, entity *User) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// Find returns a user by ID.
func (r *Repository) Find(ctx context.Context, id string) (*User, error) {
	var entity User
	if err := r.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

// Update applies changes to an existing user.
func (r *Repository) Update(ctx context.Context, id string, updates map[string]any) (*User, error) {
	var entity User
	tx := r.db.WithContext(ctx)
	if err := tx.First(&entity, "id = ?", id).Error; err != nil {
		return nil, err
	}

	if err := tx.Model(&entity).Updates(updates).Error; err != nil {
		return nil, err
	}

	if err := tx.First(&entity, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &entity, nil
}

// Delete removes a user.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&User{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// IsNotFound indicates a missing record error.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
