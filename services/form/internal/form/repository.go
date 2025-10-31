package form

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// Repository provides persistence helpers for forms.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a repository from a database connection.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// List returns all forms, optionally filtered by a case-insensitive name search.
func (r *Repository) List(ctx context.Context, search string) ([]Form, error) {
	query := r.db.WithContext(ctx).Model(&Form{}).Order("created_at DESC")
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("LOWER(name) LIKE LOWER(?)", like)
	}

	var forms []Form
	if err := query.Find(&forms).Error; err != nil {
		return nil, err
	}
	return forms, nil
}

// Create persists a new form.
func (r *Repository) Create(ctx context.Context, payload *Form) error {
	return r.db.WithContext(ctx).Create(payload).Error
}

// Find returns a form by ID.
func (r *Repository) Find(ctx context.Context, id string) (*Form, error) {
	var entity Form
	if err := r.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

// Update applies partial updates to a form.
func (r *Repository) Update(ctx context.Context, id string, updates map[string]any) (*Form, error) {
	var entity Form
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

// Delete removes a form by ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Form{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// IsNotFound reports whether an error indicates a missing record.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
