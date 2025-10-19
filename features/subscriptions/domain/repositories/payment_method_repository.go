package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
)

// PaymentMethodRepository defines the interface for payment method data operations
type PaymentMethodRepository interface {
	// Create creates a new payment method
	Create(ctx context.Context, paymentMethod *entities.PaymentMethodEntity) error

	// FindByID finds a payment method by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entities.PaymentMethodEntity, error)

	// FindByOrganizationID finds all payment methods for an organization
	FindByOrganizationID(ctx context.Context, organizationID string) ([]*entities.PaymentMethodEntity, error)

	// FindPrimaryByOrganizationID finds the primary payment method for an organization
	FindPrimaryByOrganizationID(ctx context.Context, organizationID string) (*entities.PaymentMethodEntity, error)

	// Update updates a payment method
	Update(ctx context.Context, paymentMethod *entities.PaymentMethodEntity) error

	// Delete soft deletes a payment method
	Delete(ctx context.Context, id uuid.UUID) error

	// SetAsPrimary sets a payment method as primary (and unsets others)
	SetAsPrimary(ctx context.Context, organizationID string, paymentMethodID uuid.UUID) error
}
