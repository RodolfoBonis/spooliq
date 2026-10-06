package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	pricing "github.com/RodolfoBonis/spooliq/features/budget/domain/services"
	"github.com/google/uuid"
)

// BudgetRepository defines the interface for budget data operations
type BudgetRepository interface {
	// Transaction wraps the given function in a single database transaction.
	// The repository passed to fn is bound to the transaction; any returned
	// error rolls back all changes made within fn.
	WithTransaction(ctx context.Context, fn func(repo BudgetRepository) error) error

	// Basic CRUD operations
	Create(ctx context.Context, budget *entities.BudgetEntity) error
	FindByID(ctx context.Context, id uuid.UUID, organizationID string) (*entities.BudgetEntity, error)
	// Update persists editable budget fields. It is guarded to draft budgets:
	// it returns entities.ErrBudgetNotEditable if the budget is not a draft (or
	// does not belong to the organization). Status is owned by UpdateStatus and
	// the PDF URL may be cleared here but is otherwise owned by UpdatePDFURL.
	Update(ctx context.Context, budget *entities.BudgetEntity) error
	// UpdateStatus writes only the status column (and updated_at), scoped by
	// organization and guarded by the expected current status. It is the sole owner
	// of the status column so a concurrent full Update cannot revert an approval.
	// It returns entities.ErrBudgetStatusConflict when no row matches (the budget
	// changed status concurrently or does not belong to the organization), so the
	// caller can surface a 409.
	UpdateStatus(ctx context.Context, budgetID uuid.UUID, organizationID string, expectedCurrent, newStatus entities.BudgetStatus) error
	// UpdatePDFURL writes only the pdf_url column (and updated_at), scoped by
	// organization and allowed in any status (PDFs are generated for approved
	// budgets, not only drafts).
	UpdatePDFURL(ctx context.Context, budgetID uuid.UUID, organizationID string, pdfURL *string) error
	Delete(ctx context.Context, id uuid.UUID, organizationID string) error

	// SearchBudgets returns a filtered, sorted, org-scoped page of budgets WITHOUT
	// any relationship preloads (the list response is assembled by the use case via
	// the batch loaders below, keeping the per-page query count constant). orderBy is
	// a safe "column asc|desc" expression built from the sort whitelist by the caller;
	// an empty orderBy falls back to created_at DESC. Recognized filter keys: "name"
	// (case-insensitive LIKE), "customer_id" (uuid.UUID), "status" (string),
	// "start_date"/"end_date" (time.Time, inclusive on created_at).
	SearchBudgets(ctx context.Context, organizationID string, filters map[string]interface{}, orderBy string, limit, offset int) ([]*entities.BudgetEntity, int, error)

	// Batch loaders for the list response. Each performs a single org-scoped query
	// over a set of IDs and returns a map keyed by the entity ID, so building a page
	// of N budgets costs a constant number of queries instead of O(N).
	//
	// GetCustomersInfo loads the {id, name, ...} of every given customer.
	GetCustomersInfo(ctx context.Context, customerIDs []uuid.UUID, organizationID string) (map[uuid.UUID]*entities.CustomerInfo, error)
	// GetItemsByBudgetIDs loads the items of every given budget, grouped by budget ID
	// and ordered by item "order" within each budget.
	GetItemsByBudgetIDs(ctx context.Context, budgetIDs []uuid.UUID, organizationID string) (map[uuid.UUID][]*entities.BudgetItemEntity, error)
	// GetFilamentUsageInfoByItemIDs loads the filament usage of every given item,
	// grouped by item ID and ordered by the color-change order within each item.
	GetFilamentUsageInfoByItemIDs(ctx context.Context, itemIDs []uuid.UUID, organizationID string) (map[uuid.UUID][]entities.FilamentUsageInfo, error)
	// GetCostPresetNames loads the display name of every given cost preset. Names are
	// resolved WITHOUT a deleted_at filter for historical consistency with budgets
	// that reference a since-soft-deleted preset (matching GetPresetInfo).
	GetCostPresetNames(ctx context.Context, presetIDs []uuid.UUID, organizationID string) (map[uuid.UUID]string, error)
	// GetProfileNames loads the display name of every given live print profile.
	GetProfileNames(ctx context.Context, profileIDs []uuid.UUID, organizationID string) (map[uuid.UUID]string, error)

	// Item operations
	AddItem(ctx context.Context, item *entities.BudgetItemEntity) error
	RemoveItem(ctx context.Context, itemID uuid.UUID) error
	UpdateItem(ctx context.Context, item *entities.BudgetItemEntity) error
	GetItems(ctx context.Context, budgetID uuid.UUID) ([]*entities.BudgetItemEntity, error)
	DeleteAllItems(ctx context.Context, budgetID uuid.UUID, organizationID string) error

	// Item Filament operations (NEW - multi-filament support)
	AddItemFilament(ctx context.Context, filament *entities.BudgetItemFilamentEntity) error
	RemoveItemFilament(ctx context.Context, filamentID uuid.UUID) error
	GetItemFilaments(ctx context.Context, itemID uuid.UUID) ([]*entities.BudgetItemFilamentEntity, error)
	DeleteAllItemFilaments(ctx context.Context, itemID uuid.UUID) error
	GetFilamentUsageInfo(ctx context.Context, itemID uuid.UUID, organizationID string) ([]entities.FilamentUsageInfo, error)

	// Status history operations
	AddStatusHistory(ctx context.Context, history *entities.BudgetStatusHistoryEntity) error
	// GetStatusHistory returns a page of a budget's status history (newest first)
	// together with the total row count, so the handler can build a paginated envelope.
	GetStatusHistory(ctx context.Context, budgetID uuid.UUID, limit, offset int) ([]entities.BudgetStatusHistoryEntity, int64, error)

	// Calculation operations
	// CalculateCosts recomputes and PERSISTS all costs for a stored budget. The
	// initial budget fetch is scoped by organization (defense-in-depth against a
	// cross-tenant budget ID).
	CalculateCosts(ctx context.Context, budgetID uuid.UUID, organizationID string) error
	// ComputeBudgetPricing loads the org-scoped rates for the given input and runs
	// the pure pricing engine WITHOUT persisting anything. It is shared by
	// CalculateCosts (which then persists) and the stateless preview endpoint.
	ComputeBudgetPricing(ctx context.Context, in entities.PricingComputationInput) (pricing.PricingResult, error)

	// Ownership validation (multi-tenant guards)
	ValidateFilamentsInOrg(ctx context.Context, filamentIDs []uuid.UUID, organizationID string) error
	// ValidatePresetInOrg ensures the preset belongs to the organization AND is
	// of the expected type (machine/energy/cost). It only accepts live presets
	// (deleted_at IS NULL) because it validates freshly provided references.
	ValidatePresetInOrg(ctx context.Context, presetID uuid.UUID, presetType string, organizationID string) error
	// ValidateModel3DsInOrg ensures every referenced 3D model belongs to the
	// organization (live rows only). Empty input is a no-op.
	ValidateModel3DsInOrg(ctx context.Context, model3dIDs []uuid.UUID, organizationID string) error

	// Relationship helpers
	GetCustomerInfo(ctx context.Context, customerID uuid.UUID, organizationID string) (*entities.CustomerInfo, error)
	GetFilamentInfo(ctx context.Context, filamentID uuid.UUID, organizationID string) (*entities.FilamentInfo, error)
	GetPresetInfo(ctx context.Context, presetID uuid.UUID, presetType string, organizationID string) (*entities.PresetInfo, error)
	GetCompanyByOrganizationID(ctx context.Context, organizationID string) (*entities.CompanyInfo, error)
	FindItemsByBudgetID(ctx context.Context, budgetID uuid.UUID) ([]*entities.BudgetItemEntity, error)
}
