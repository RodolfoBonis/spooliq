package usecases

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StockDeductor decrements filament stock when a budget is completed. It is a narrow,
// one-directional port (implemented by the stock feature and wired via FX) so the
// budget package never imports the stock feature. The method runs inside the SAME
// transaction as the status change — it receives the transaction-bound *gorm.DB — so
// the status transition and the stock movements commit (or roll back) atomically.
type StockDeductor interface {
	DeductForCompletedBudget(ctx context.Context, tx *gorm.DB, budgetID uuid.UUID, organizationID, userID string) error
}
