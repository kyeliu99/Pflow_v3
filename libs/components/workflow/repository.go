package workflow

import (
    "context"
    "errors"

    "gorm.io/gorm"
)

// Repository defines persistence operations for workflow definitions.
type Repository interface {
    List(ctx context.Context, published *bool) ([]Definition, error)
    Create(ctx context.Context, entity *Definition) error
    Find(ctx context.Context, id string) (*Definition, error)
    Update(ctx context.Context, id string, updates map[string]any) (*Definition, error)
    Delete(ctx context.Context, id string) error
    Publish(ctx context.Context, id string) (*Definition, error)
}

// GormRepository implements Repository using GORM.
type GormRepository struct {
    db *gorm.DB
}

// NewGormRepository constructs a repository.
func NewGormRepository(db *gorm.DB) *GormRepository {
    return &GormRepository{db: db}
}

// List returns definitions optionally filtered by published flag.
func (r *GormRepository) List(ctx context.Context, published *bool) ([]Definition, error) {
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
func (r *GormRepository) Create(ctx context.Context, entity *Definition) error {
    return r.db.WithContext(ctx).Create(entity).Error
}

// Find returns a definition by ID.
func (r *GormRepository) Find(ctx context.Context, id string) (*Definition, error) {
    var entity Definition
    if err := r.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
        return nil, err
    }
    return &entity, nil
}

// Update applies updates to a definition.
func (r *GormRepository) Update(ctx context.Context, id string, updates map[string]any) (*Definition, error) {
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
func (r *GormRepository) Delete(ctx context.Context, id string) error {
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
func (r *GormRepository) Publish(ctx context.Context, id string) (*Definition, error) {
    updates := map[string]any{
        "published": true,
    }
    return r.Update(ctx, id, updates)
}

// IsNotFound indicates whether the error is gorm.ErrRecordNotFound.
func IsNotFound(err error) bool {
    return errors.Is(err, gorm.ErrRecordNotFound)
}
