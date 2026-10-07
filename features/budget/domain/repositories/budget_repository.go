package repositories

import (
	"context"
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	pricing "github.com/RodolfoBonis/spooliq/features/budget/domain/services"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BudgetRepository defines the interface for budget data operations
type BudgetRepository interface {
	// Transaction wraps the given function in a single database transaction.
	// The repository passed to fn is bound to the transaction; any returned
	// error rolls back all changes made within fn.
	WithTransaction(ctx context.Context, fn func(repo BudgetRepository) error) error

	// UnderlyingTx returns the *gorm.DB this repository is bound to. On a repository
	// handed to a WithTransaction callback it is the transaction handle, which lets a
	// cross-aggregate operation (filament stock deduction on budget completion) run in
	// the SAME transaction as the status change. Outside a transaction it is the base
	// connection.
	UnderlyingTx() *gorm.DB

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

	// AllocateQuoteNumber atomically increments and returns the next sequential quote
	// number for the organization (UPDATE companies SET next_quote_number =
	// next_quote_number + 1 ... RETURNING next_quote_number - 1). It must be called
	// inside the create/duplicate transaction so the number is allocated exactly once.
	AllocateQuoteNumber(ctx context.Context, organizationID string) (int, error)

	// FindByPublicToken returns the budget with the given public share token (exact
	// match, any organization). It returns entities.ErrBudgetNotFound when no live
	// budget carries that token, so unknown/revoked tokens map to the same 404.
	FindByPublicToken(ctx context.Context, token string) (*entities.BudgetEntity, error)

	// SetShareToken writes the public share token (and its created_at), org-scoped.
	SetShareToken(ctx context.Context, budgetID uuid.UUID, organizationID string, token string, createdAt time.Time) error

	// RevokeShareToken clears the public share token (sets it and its created_at to
	// NULL), org-scoped.
	RevokeShareToken(ctx context.Context, budgetID uuid.UUID, organizationID string) error

	// SetValidUntil writes only the valid_until column (and updated_at), org-scoped.
	SetValidUntil(ctx context.Context, budgetID uuid.UUID, organizationID string, validUntil *time.Time) error

	// ExpireOverdue marks every sent budget whose valid_until has passed as expired in
	// one statement, returning the number of rows updated. Used by the expiry job.
	ExpireOverdue(ctx context.Context) (int64, error)

	// RespondToPublicBudget applies a customer response (approve/reject) using a
	// race-safe conditional UPDATE guarded by status='sent' and a non-expired
	// valid_until. It returns the number of rows affected (0 when the budget is no
	// longer answerable), so the caller can re-read and map the precise error.
	RespondToPublicBudget(ctx context.Context, budgetID uuid.UUID, newStatus entities.BudgetStatus, name, ip, userAgent string, reason *string, respondedAt time.Time) (int64, error)

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
	// GetCompanyQuoteDefaults returns the organization's default quote validity (days)
	// and default payment terms, used when creating budgets and computing valid_until.
	GetCompanyQuoteDefaults(ctx context.Context, organizationID string) (validityDays int, paymentTerms *string, err error)
	FindItemsByBudgetID(ctx context.Context, budgetID uuid.UUID) ([]*entities.BudgetItemEntity, error)

	// GetStockWarnings returns, for a STORED budget, the tracked filaments whose
	// current stock is below the grams this budget requires. It is a single org-scoped
	// query joining budget_item_filaments -> budget_items -> filaments. The caller is
	// responsible for skipping already-completed budgets.
	GetStockWarnings(ctx context.Context, budgetID uuid.UUID, organizationID string) ([]entities.StockWarning, error)

	// GetStockWarningsForRequest computes stock warnings for a NON-persisted budget
	// (preview) from the required grams per filament. It issues ONE org-scoped query
	// over the given filament IDs and returns warnings only for tracked filaments whose
	// stock is below the requirement.
	GetStockWarningsForRequest(ctx context.Context, organizationID string, requiredByFilament map[uuid.UUID]float64) ([]entities.StockWarning, error)
}
