package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
)

// CostProfileRepository defines the interface for cost profile data access operations.
type CostProfileRepository interface {
	Create(ctx context.Context, profile *entities.CostProfile) error
	GetByID(ctx context.Context, id uint, userID *string) (*entities.CostProfile, error)
	GetAll(ctx context.Context, userID *string) ([]*entities.CostProfile, error)
	Update(ctx context.Context, profile *entities.CostProfile, userID string) error
	Delete(ctx context.Context, id uint, userID string) error
	GetByOwner(ctx context.Context, userID string) ([]*entities.CostProfile, error)
	GetGlobal(ctx context.Context) ([]*entities.CostProfile, error)
}
