package workflow

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// Repository manages workflow definitions.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// List returns definitions optionally filtered by published flag.
func (r *Repository) List(ctx context.Context, published *bool) ([]Definition, error) {
	query := r.db.WithContext(ctx).Model(&Definition{}).Order("updated_at DESC")
	if published != nil {
		query = query.Where("published = ?", *published)
	}

	var definitions []Definition
	if err := query.Find(&definitions).Error; err != nil {
		return nil, err
	}
	return definitions, nil
}

// Create persists a definition.
func (r *Repository) Create(ctx context.Context, entity *Definition) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// Find returns a definition by ID.
func (r *Repository) Find(ctx context.Context, id string) (*Definition, error) {
	var entity Definition
	if err := r.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

// Update applies updates to a definition.
func (r *Repository) Update(ctx context.Context, id string, updates map[string]any) (*Definition, error) {
	var entity Definition
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

// Delete removes a definition.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Definition{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Publish marks a workflow as published.
func (r *Repository) Publish(ctx context.Context, id string) (*Definition, error) {
	updates := map[string]any{
		"published": true,
	}
	return r.Update(ctx, id, updates)
}

// IsNotFound indicates whether the error is gorm.ErrRecordNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
