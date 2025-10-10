package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
)

// SubscriptionRepository defines the interface for subscription payment history data access
type SubscriptionRepository interface {
	FindAll(ctx context.Context, organizationID uuid.UUID, limit, offset int) ([]*entities.SubscriptionEntity, error)
	FindByID(ctx context.Context, id uuid.UUID, organizationID uuid.UUID) (*entities.SubscriptionEntity, error)
	FindByAsaasPaymentID(ctx context.Context, asaasPaymentID string) (*entities.SubscriptionEntity, error)
	Create(ctx context.Context, subscription *entities.SubscriptionEntity) error
	Update(ctx context.Context, id uuid.UUID, organizationID uuid.UUID, subscription *entities.SubscriptionEntity) error
	Delete(ctx context.Context, id uuid.UUID, organizationID uuid.UUID) error
}
