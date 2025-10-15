package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
)

// BudgetRepository defines the interface for budget data operations
type BudgetRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, budget *entities.BudgetEntity) error
	FindByID(ctx context.Context, id uuid.UUID, organizationID string) (*entities.BudgetEntity, error)
	Update(ctx context.Context, budget *entities.BudgetEntity) error
	Delete(ctx context.Context, id uuid.UUID) error

	// List operations
	FindAll(ctx context.Context, organizationID string, limit, offset int) ([]*entities.BudgetEntity, int, error)
	FindByCustomer(ctx context.Context, customerID uuid.UUID, organizationID string) ([]*entities.BudgetEntity, error)

	// Search operations
	SearchBudgets(ctx context.Context, organizationID string, filters map[string]interface{}, limit, offset int) ([]*entities.BudgetEntity, int, error)

	// Item operations
	AddItem(ctx context.Context, item *entities.BudgetItemEntity) error
	RemoveItem(ctx context.Context, itemID uuid.UUID) error
	UpdateItem(ctx context.Context, item *entities.BudgetItemEntity) error
	GetItems(ctx context.Context, budgetID uuid.UUID) ([]*entities.BudgetItemEntity, error)
	DeleteAllItems(ctx context.Context, budgetID uuid.UUID) error

	// Item Filament operations (NEW - multi-filament support)
	AddItemFilament(ctx context.Context, filament *entities.BudgetItemFilamentEntity) error
	RemoveItemFilament(ctx context.Context, filamentID uuid.UUID) error
	GetItemFilaments(ctx context.Context, itemID uuid.UUID) ([]*entities.BudgetItemFilamentEntity, error)
	DeleteAllItemFilaments(ctx context.Context, itemID uuid.UUID) error
	GetFilamentUsageInfo(ctx context.Context, itemID uuid.UUID) ([]entities.FilamentUsageInfo, error)

	// Status history operations
	AddStatusHistory(ctx context.Context, history *entities.BudgetStatusHistoryEntity) error
	GetStatusHistory(ctx context.Context, budgetID uuid.UUID) ([]entities.BudgetStatusHistoryEntity, error)

	// Calculation operations
	CalculateCosts(ctx context.Context, budgetID uuid.UUID) error

	// Relationship helpers
	GetCustomerInfo(ctx context.Context, customerID uuid.UUID) (*entities.CustomerInfo, error)
	GetFilamentInfo(ctx context.Context, filamentID uuid.UUID) (*entities.FilamentInfo, error)
	GetPresetInfo(ctx context.Context, presetID uuid.UUID, presetType string) (*entities.PresetInfo, error)
	GetCompanyByOrganizationID(ctx context.Context, organizationID string) (*entities.CompanyInfo, error)
	FindItemsByBudgetID(ctx context.Context, budgetID uuid.UUID) ([]*entities.BudgetItemEntity, error)
}
