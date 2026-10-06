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

	// FindAll returns a page of customers for the organization. search is an
	// optional case-insensitive term matched against name/email/phone/document;
	// order is a safe ORDER BY clause already validated against a whitelist.
	FindAll(ctx context.Context, organizationID, search, order string, limit, offset int) ([]*entities.CustomerEntity, int64, error)

	// SearchCustomers returns a page of customers matching the structured filters
	// (name, email, phone, document, city, state, is_active, id) plus the
	// free-text search; order behaves as in FindAll.
	SearchCustomers(ctx context.Context, organizationID string, filters map[string]interface{}, search, order string, limit, offset int) ([]*entities.CustomerEntity, int64, error)

	ExistsByEmail(ctx context.Context, email string, organizationID string, excludeID *uuid.UUID) (bool, error)

	CountBudgetsByCustomer(ctx context.Context, customerID uuid.UUID) (int64, error)

	GetCustomerBudgets(ctx context.Context, customerID uuid.UUID) ([]entities.BudgetSummary, error)

	SumBudgetTotalsByCustomerAndStatus(ctx context.Context, customerID uuid.UUID, statuses []string) (int64, error)

	// GetBudgetStatsByCustomers computes, in a single grouped query, the budget
	// count and the summed total (restricted to totalStatuses) for every
	// customer in customerIDs. It is the N+1 fix for the list endpoints, which
	// previously issued two queries per row. The returned map is keyed by
	// customer ID; customers with no budgets are simply absent.
	GetBudgetStatsByCustomers(ctx context.Context, customerIDs []uuid.UUID, totalStatuses []string) (map[uuid.UUID]entities.CustomerBudgetStats, error)
}
