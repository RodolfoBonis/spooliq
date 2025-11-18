package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/google/uuid"
)

// CustomerRepository defines the interface for customer data operations
type CustomerRepository interface {
	Create(ctx context.Context, customer *entities.CustomerEntity) error
	FindByID(ctx context.Context, id uuid.UUID, organizationID string) (*entities.CustomerEntity, error)
	Update(ctx context.Context, customer *entities.CustomerEntity) error
	Delete(ctx context.Context, id uuid.UUID) error

	FindAll(ctx context.Context, organizationID string, limit, offset int) ([]*entities.CustomerEntity, int, error)

	SearchCustomers(ctx context.Context, organizationID string, filters map[string]interface{}, limit, offset int) ([]*entities.CustomerEntity, int, error)

	ExistsByEmail(ctx context.Context, email string, organizationID string, excludeID *uuid.UUID) (bool, error)

	CountBudgetsByCustomer(ctx context.Context, customerID uuid.UUID) (int64, error)

	GetCustomerBudgets(ctx context.Context, customerID uuid.UUID) ([]entities.BudgetSummary, error)

	SumBudgetTotalsByCustomerAndStatus(ctx context.Context, customerID uuid.UUID, statuses []string) (int64, error)
}
