// Package repositories defines the filament stock data-access contracts.
package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/stock/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StockRepository defines filament stock movement persistence operations.
type StockRepository interface {
	// FilamentExistsInOrg reports whether a live filament with the given ID belongs
	// to the organization. Used to return 404 before listing movements.
	FilamentExistsInOrg(ctx context.Context, filamentID uuid.UUID, organizationID string) (bool, error)

	// CreateManualMovement records a manual movement (purchase/adjustment/waste) in a
	// single transaction: it locks the filament row (FOR UPDATE, org-scoped), inserts
	// the movement with the already-signed grams, applies the delta to stock_grams and
	// forces track_stock = true. It returns the stored movement and the refreshed
	// filament stock summary, or gorm.ErrRecordNotFound when the filament is not in the
	// organization.
	CreateManualMovement(ctx context.Context, m *entities.StockMovementEntity) (*entities.StockMovementEntity, *entities.FilamentStockSummary, error)

	// ListMovements returns an org-scoped page of a filament's movements together with
	// the total count. order is a safe ORDER BY expression; typeFilter, when non-empty,
	// restricts to a single movement type.
	ListMovements(ctx context.Context, filamentID uuid.UUID, organizationID, typeFilter, order string, limit, offset int) ([]entities.StockMovementResponse, int64, error)

	// DeductForCompletedBudget writes the consumption movements for a budget that has
	// just been completed, within the provided transaction (so it commits atomically
	// with the status change). For every tracked filament used by the budget it inserts
	// one consumption movement (negative grams, budget_id) guarded by the partial unique
	// index (ON CONFLICT DO NOTHING) and decrements stock_grams ONLY when the insert
	// actually happened, making the whole operation idempotent. Untracked filaments are
	// skipped; stock may go negative.
	DeductForCompletedBudget(ctx context.Context, tx *gorm.DB, budgetID uuid.UUID, organizationID, userID string) error

	// LowStockForBudget returns the budget's tracked filaments that are at or below
	// their low-stock threshold (used to notify after a completion).
	LowStockForBudget(ctx context.Context, budgetID uuid.UUID, organizationID string) ([]entities.LowStockFilament, error)
}
