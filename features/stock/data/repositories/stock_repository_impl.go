package repositories

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	budgetrepo "github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	pricing "github.com/RodolfoBonis/spooliq/features/budget/domain/services"
	"github.com/RodolfoBonis/spooliq/features/stock/data/models"
	"github.com/RodolfoBonis/spooliq/features/stock/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/stock/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type stockRepositoryImpl struct {
	db *gorm.DB
}

// NewStockRepository creates a new filament stock repository.
func NewStockRepository(db *gorm.DB) repositories.StockRepository {
	return &stockRepositoryImpl{db: db}
}

// FilamentExistsInOrg reports whether a live filament belongs to the organization.
func (r *stockRepositoryImpl) FilamentExistsInOrg(ctx context.Context, filamentID uuid.UUID, organizationID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Table("filaments").
		Where("id = ? AND organization_id = ? AND deleted_at IS NULL", filamentID, organizationID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check filament existence: %w", err)
	}
	return count > 0, nil
}

// lockedFilament is the projection read under FOR UPDATE when applying a movement.
type lockedFilament struct {
	ID                     uuid.UUID `gorm:"column:id"`
	StockGrams             int64     `gorm:"column:stock_grams"`
	TrackStock             bool      `gorm:"column:track_stock"`
	LowStockThresholdGrams *int      `gorm:"column:low_stock_threshold_grams"`
}

// CreateManualMovement records a manual movement and updates the filament balance in
// one transaction. The filament row is locked FOR UPDATE so concurrent movements
// serialize and the resulting stock_grams is always correct.
func (r *stockRepositoryImpl) CreateManualMovement(ctx context.Context, m *entities.StockMovementEntity) (*entities.StockMovementEntity, *entities.FilamentStockSummary, error) {
	var stored *entities.StockMovementEntity
	var summary *entities.FilamentStockSummary

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var fil lockedFilament
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Table("filaments").
			Select("id, stock_grams, track_stock, low_stock_threshold_grams").
			Where("id = ? AND organization_id = ? AND deleted_at IS NULL", m.FilamentID, m.OrganizationID).
			Take(&fil).Error; err != nil {
			return err // gorm.ErrRecordNotFound bubbles up to a 404 in the use case
		}

		model := &models.StockMovementModel{
			OrganizationID: m.OrganizationID,
			FilamentID:     m.FilamentID,
			Type:           string(m.Type),
			Grams:          m.Grams,
			UnitPricePerKg: m.UnitPricePerKg,
			BudgetID:       m.BudgetID,
			Note:           m.Note,
			CreatedBy:      m.CreatedBy,
		}
		if err := tx.Create(model).Error; err != nil {
			return fmt.Errorf("failed to insert stock movement: %w", err)
		}

		newStock := fil.StockGrams + m.Grams
		if err := tx.
			Table("filaments").
			Where("id = ? AND organization_id = ?", m.FilamentID, m.OrganizationID).
			Updates(map[string]interface{}{
				"stock_grams": newStock,
				"track_stock": true, // any manual movement enables tracking
				"updated_at":  time.Now(),
			}).Error; err != nil {
			return fmt.Errorf("failed to update filament stock: %w", err)
		}

		stored = model.ToEntity()
		summary = &entities.FilamentStockSummary{
			ID:                     m.FilamentID,
			StockGrams:             newStock,
			TrackStock:             true,
			LowStockThresholdGrams: fil.LowStockThresholdGrams,
			IsLowStock:             fil.LowStockThresholdGrams != nil && newStock <= int64(*fil.LowStockThresholdGrams),
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return stored, summary, nil
}

// movementRow is the join projection read by ListMovements (ledger row + the linked
// budget's quote number, when any).
type movementRow struct {
	ID                uuid.UUID  `gorm:"column:id"`
	FilamentID        uuid.UUID  `gorm:"column:filament_id"`
	Type              string     `gorm:"column:type"`
	Grams             int64      `gorm:"column:grams"`
	UnitPricePerKg    *int64     `gorm:"column:unit_price_per_kg"`
	BudgetID          *uuid.UUID `gorm:"column:budget_id"`
	BudgetQuoteNumber *int       `gorm:"column:budget_quote_number"`
	Note              *string    `gorm:"column:note"`
	CreatedBy         string     `gorm:"column:created_by"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
}

// ListMovements returns an org-scoped page of a filament's movements plus the total.
func (r *stockRepositoryImpl) ListMovements(ctx context.Context, filamentID uuid.UUID, organizationID, typeFilter, order string, limit, offset int) ([]entities.StockMovementResponse, int64, error) {
	base := r.db.WithContext(ctx).
		Table("filament_stock_movements AS m").
		Where("m.filament_id = ? AND m.organization_id = ?", filamentID, organizationID)
	if typeFilter != "" {
		base = base.Where("m.type = ?", typeFilter)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count stock movements: %w", err)
	}

	if order == "" {
		order = "m.created_at desc, m.id desc"
	}

	var rows []movementRow
	if err := base.
		Select("m.id, m.filament_id, m.type, m.grams, m.unit_price_per_kg, m.budget_id, b.quote_number AS budget_quote_number, m.note, m.created_by, m.created_at").
		Joins("LEFT JOIN budgets b ON b.id = m.budget_id").
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list stock movements: %w", err)
	}

	out := make([]entities.StockMovementResponse, len(rows))
	for i, row := range rows {
		out[i] = entities.StockMovementResponse{
			ID:                row.ID,
			FilamentID:        row.FilamentID,
			Type:              entities.MovementType(row.Type),
			Grams:             row.Grams,
			UnitPricePerKg:    row.UnitPricePerKg,
			BudgetID:          row.BudgetID,
			BudgetQuoteNumber: row.BudgetQuoteNumber,
			Note:              row.Note,
			CreatedBy:         row.CreatedBy,
			CreatedAt:         row.CreatedAt,
		}
	}
	return out, total, nil
}

// DeductForCompletedBudget writes idempotent consumption movements for a completed
// budget inside the provided transaction. See the interface doc for the contract.
func (r *stockRepositoryImpl) DeductForCompletedBudget(ctx context.Context, tx *gorm.DB, budgetID uuid.UUID, organizationID, userID string) error {
	tx = tx.WithContext(ctx)

	// 1. PHYSICAL grams required per filament across the budget's items: each filament's
	// quantity PLUS its equal share of the color-change purge waste of every
	// multi-filament item. The waste is counted whenever an item has N > 1 filaments,
	// REGARDLESS of the budget's include_waste_cost flag, because the mass is physically
	// purged on the print bed and must leave stock even when it was not billed. The shared
	// budget loader + aggregator are the single source of truth so stock and pricing never
	// disagree on what a budget consumes.
	items, err := budgetrepo.LoadBudgetRequirementItems(ctx, tx, budgetID, organizationID)
	if err != nil {
		return err
	}
	required := pricing.FilamentRequirements(items)
	if len(required) == 0 {
		return nil
	}
	wasteByFilament := pricing.FilamentWasteGrams(items)

	ids := make([]uuid.UUID, 0, len(required))
	for id := range required {
		ids = append(ids, id)
	}
	// Iterate in a consistent (sorted) order to avoid deadlocks with concurrent manual
	// movements; the bulk lock below uses the same ordering.
	sort.Slice(ids, func(i, j int) bool { return bytes.Compare(ids[i][:], ids[j][:]) < 0 })

	// 2. Lock every referenced filament row in a consistent (sorted) order to avoid
	// deadlocks with concurrent manual movements, and learn which ones are tracked.
	var locked []struct {
		ID         uuid.UUID `gorm:"column:id"`
		TrackStock bool      `gorm:"column:track_stock"`
	}
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Table("filaments").
		Select("id, track_stock").
		Where("organization_id = ? AND id IN ?", organizationID, ids).
		Order("id").
		Find(&locked).Error; err != nil {
		return fmt.Errorf("failed to lock filaments for deduction: %w", err)
	}
	tracked := make(map[uuid.UUID]bool, len(locked))
	for _, l := range locked {
		tracked[l.ID] = l.TrackStock
	}

	// 3. For each tracked filament: insert the consumption movement (idempotent via
	// the partial unique index) and decrement stock only when the insert happened.
	for _, id := range ids {
		if !tracked[id] {
			continue // untracked filaments are skipped
		}
		grams := int64(math.Round(required[id]))
		if grams <= 0 {
			continue
		}
		signed := -grams

		// pt-BR consumption note: when purge waste is part of the deduction, record how
		// many grams of the total are purge so the ledger explains the extra consumption.
		var note *string
		if w := int64(math.Round(wasteByFilament[id])); w > 0 {
			n := fmt.Sprintf("Consumo do orçamento (inclui %d g de purga)", w)
			note = &n
		}

		res := tx.Exec(`
			INSERT INTO filament_stock_movements
				(id, organization_id, filament_id, type, grams, budget_id, note, created_by, created_at)
			VALUES (gen_random_uuid(), ?, ?, 'consumption', ?, ?, ?, ?, now())
			ON CONFLICT (budget_id, filament_id) WHERE type = 'consumption' DO NOTHING
		`, organizationID, id, signed, budgetID, note, userID)
		if res.Error != nil {
			return fmt.Errorf("failed to insert consumption movement: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			continue // already deducted for this (budget, filament) — do not double count
		}

		if err := tx.Exec(`
			UPDATE filaments SET stock_grams = stock_grams + ?, updated_at = now()
			WHERE id = ? AND organization_id = ?
		`, signed, id, organizationID).Error; err != nil {
			return fmt.Errorf("failed to decrement filament stock: %w", err)
		}
	}

	return nil
}
