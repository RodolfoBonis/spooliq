package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
)

// PaymentGatewayLinkRepository defines the interface for payment gateway link data operations
type PaymentGatewayLinkRepository interface {
	Create(ctx context.Context, link *entities.PaymentGatewayLinkEntity) error
	FindByOrganizationID(ctx context.Context, organizationID string) (*entities.PaymentGatewayLinkEntity, error)
	Update(ctx context.Context, link *entities.PaymentGatewayLinkEntity) error
	Delete(ctx context.Context, organizationID string) error
}
