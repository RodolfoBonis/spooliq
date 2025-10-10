package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/google/uuid"
)

// CustomerRepository defines the interface for customer data operations
type CustomerRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, customer *entities.CustomerEntity) error
	FindByID(ctx context.Context, id uuid.UUID, organizationID string) (*entities.CustomerEntity, error)
	Update(ctx context.Context, customer *entities.CustomerEntity) error
	Delete(ctx context.Context, id uuid.UUID) error

	// List operations
	FindAll(ctx context.Context, organizationID string, limit, offset int) ([]*entities.CustomerEntity, int, error)

	// Search operations
	SearchCustomers(ctx context.Context, organizationID string, filters map[string]interface{}, limit, offset int) ([]*entities.CustomerEntity, int, error)

	// Validation operations
	ExistsByEmail(ctx context.Context, email string, organizationID string, excludeID *uuid.UUID) (bool, error)

	// Relationship validation
	CountBudgetsByCustomer(ctx context.Context, customerID uuid.UUID) (int64, error)
}
