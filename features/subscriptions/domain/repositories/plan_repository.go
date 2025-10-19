package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
)

// PlanRepository defines the interface for plan data access
type PlanRepository interface {
	FindAll(ctx context.Context, activeOnly bool) ([]*entities.PlanEntity, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entities.PlanEntity, error)
	FindBySlug(ctx context.Context, slug string) (*entities.PlanEntity, error)
	Create(ctx context.Context, plan *entities.PlanEntity) error
	Update(ctx context.Context, plan *entities.PlanEntity) error
	Delete(ctx context.Context, id uuid.UUID) error
	
	// Feature operations
	AddFeature(ctx context.Context, feature *entities.PlanFeature) error
	UpdateFeature(ctx context.Context, feature *entities.PlanFeature) error
	DeleteFeature(ctx context.Context, featureID uuid.UUID) error
}
