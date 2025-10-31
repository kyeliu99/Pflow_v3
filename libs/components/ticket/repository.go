package ticket

import (
    "context"
    "errors"
    "time"

    "gorm.io/gorm"
)

// Repository defines the persistence contract for tickets.
type Repository interface {
    List(ctx context.Context, status, assignee string) ([]Ticket, error)
    Create(ctx context.Context, entity *Ticket) error
    Find(ctx context.Context, id string) (*Ticket, error)
    Update(ctx context.Context, id string, updates map[string]any) (*Ticket, error)
    Delete(ctx context.Context, id string) error
    Resolve(ctx context.Context, id string) (*Ticket, error)
}

// GormRepository persists tickets using a relational database via GORM.
type GormRepository struct {
    db *gorm.DB
}

// NewGormRepository constructs a ticket repository backed by the provided DB.
func NewGormRepository(db *gorm.DB) *GormRepository {
    return &GormRepository{db: db}
}

// List returns tickets filtered by optional status or assignee.
func (r *GormRepository) List(ctx context.Context, status, assignee string) ([]Ticket, error) {
    query := r.db.WithContext(ctx).Model(&Ticket{}).Order("created_at DESC")
    if status != "" {
        query = query.Where("status = ?", status)
    }
    if assignee != "" {
        query = query.Where("assignee_id = ?", assignee)
    }

    var tickets []Ticket
    if err := query.Find(&tickets).Error; err != nil {
        return nil, err
    }
    return tickets, nil
}

// Create persists a new ticket.
func (r *GormRepository) Create(ctx context.Context, entity *Ticket) error {
    return r.db.WithContext(ctx).Create(entity).Error
}

// Find retrieves a ticket by ID.
func (r *GormRepository) Find(ctx context.Context, id string) (*Ticket, error) {
    var entity Ticket
    if err := r.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
        return nil, err
    }
    return &entity, nil
}

// Update applies updates to a ticket.
func (r *GormRepository) Update(ctx context.Context, id string, updates map[string]any) (*Ticket, error) {
    var entity Ticket
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

// Delete removes a ticket.
func (r *GormRepository) Delete(ctx context.Context, id string) error {
    result := r.db.WithContext(ctx).Delete(&Ticket{}, "id = ?", id)
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return gorm.ErrRecordNotFound
    }
    return nil
}

// Resolve marks a ticket as resolved and sets the timestamp.
func (r *GormRepository) Resolve(ctx context.Context, id string) (*Ticket, error) {
    now := time.Now()
    updates := map[string]any{
        "status":      StatusResolved,
        "resolved_at": &now,
    }
    return r.Update(ctx, id, updates)
}

// IsNotFound returns true if the error represents a missing record.
func IsNotFound(err error) bool {
    return errors.Is(err, gorm.ErrRecordNotFound)
}
