package repositories

import (
	"context"
	"fmt"

	pricing "github.com/RodolfoBonis/spooliq/features/budget/domain/services"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// requirementRow is the join projection behind LoadBudgetRequirementItems: one filament
// row of a budget item plus the effective cost preset's per-color-change waste (NULL
// when the resolved preset has none, which the aggregation treats as the default).
type requirementRow struct {
	BudgetItemID   uuid.UUID `gorm:"column:budget_item_id"`
	FilamentID     uuid.UUID `gorm:"column:filament_id"`
	Quantity       float64   `gorm:"column:quantity"`
	WastePerChange *float64  `gorm:"column:waste_grams_per_color_change"`
}

// LoadBudgetRequirementItems loads, for a STORED budget, the filament rows of every item
// grouped into pricing.RequirementItem values, using ONE org-scoped query. Each item
// carries the per-color-change waste of its EFFECTIVE cost preset — the item's own
// cost_preset_id, falling back to the budget's — so the caller can aggregate the physical
// stock requirement (quantity + apportioned purge waste) via pricing.FilamentRequirements.
//
// It is exported so the stock feature can reuse the exact same loader + aggregation on its
// own transaction, keeping the "what does a budget consume" definition in one place. The
// provided db may be a transaction handle; all lookups are scoped by organization so
// cross-tenant rows cannot leak in.
func LoadBudgetRequirementItems(ctx context.Context, db *gorm.DB, budgetID uuid.UUID, organizationID string) ([]pricing.RequirementItem, error) {
	var rows []requirementRow
	if err := db.WithContext(ctx).Raw(`
		SELECT bif.budget_item_id, bif.filament_id, bif.quantity, cp.waste_grams_per_color_change
		FROM budget_item_filaments bif
		JOIN budget_items bi ON bi.id = bif.budget_item_id
		JOIN budgets b ON b.id = bi.budget_id AND b.organization_id = bif.organization_id
		LEFT JOIN cost_presets cp ON cp.id = COALESCE(bi.cost_preset_id, b.cost_preset_id) AND cp.organization_id = bif.organization_id
		WHERE bi.budget_id = ? AND bif.organization_id = ?
	`, budgetID, organizationID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load budget filament requirements: %w", err)
	}

	// Group rows into items in first-seen order (deterministic for tests). Every row of a
	// given item carries the same resolved waste value, so the first wins.
	order := make([]uuid.UUID, 0, len(rows))
	byItem := make(map[uuid.UUID]*pricing.RequirementItem, len(rows))
	for _, row := range rows {
		item, ok := byItem[row.BudgetItemID]
		if !ok {
			item = &pricing.RequirementItem{}
			if row.WastePerChange != nil {
				item.WasteGramsPerChange = *row.WastePerChange
			}
			byItem[row.BudgetItemID] = item
			order = append(order, row.BudgetItemID)
		}
		item.Filaments = append(item.Filaments, pricing.RequirementFilament{
			FilamentID: row.FilamentID,
			Quantity:   row.Quantity,
		})
	}

	items := make([]pricing.RequirementItem, 0, len(order))
	for _, id := range order {
		items = append(items, *byItem[id])
	}
	return items, nil
}
