package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
)

// SubscriptionPlanRepository defines the interface for subscription plan data operations
type SubscriptionPlanRepository interface {
	// Create creates a new subscription plan
	Create(ctx context.Context, plan *entities.SubscriptionPlanEntity) error

	// FindByID finds a subscription plan by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entities.SubscriptionPlanEntity, error)

	// FindByName finds a subscription plan by name
	FindByName(ctx context.Context, name string) (*entities.SubscriptionPlanEntity, error)

	// FindAll finds all subscription plans
	FindAll(ctx context.Context) ([]*entities.SubscriptionPlanEntity, error)

	// FindAllActive finds all active subscription plans
	FindAllActive(ctx context.Context) ([]*entities.SubscriptionPlanEntity, error)

	// Update updates a subscription plan
	Update(ctx context.Context, plan *entities.SubscriptionPlanEntity) error

	// Delete soft deletes a subscription plan
	Delete(ctx context.Context, id uuid.UUID) error
}
