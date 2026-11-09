package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/data/models"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
)

// AddItemSupply persists a single budget item supply row (with its catalog snapshot).
func (r *budgetRepositoryImpl) AddItemSupply(ctx context.Context, supply *entities.BudgetItemSupplyEntity) error {
	model := &models.BudgetItemSupplyModel{}
	model.FromEntity(supply)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to add item supply: %w", err)
	}
	return nil
}

// GetItemSupplies returns one item's supplies (org-scoped), ordered by their display
// order.
func (r *budgetRepositoryImpl) GetItemSupplies(ctx context.Context, itemID uuid.UUID, organizationID string) ([]*entities.BudgetItemSupplyEntity, error) {
	var supplies []*models.BudgetItemSupplyModel
	if err := r.db.WithContext(ctx).
		Where("budget_item_id = ? AND organization_id = ?", itemID, organizationID).
		Order("\"order\" ASC").
		Find(&supplies).Error; err != nil {
		return nil, fmt.Errorf("failed to get item supplies: %w", err)
	}

	out := make([]*entities.BudgetItemSupplyEntity, len(supplies))
	for i, s := range supplies {
		out[i] = s.ToEntity()
	}
	return out, nil
}

// GetItemSuppliesByItemIDs batch-loads the supplies of every given item in ONE
// org-scoped query, grouped by item ID and ordered by the display order.
func (r *budgetRepositoryImpl) GetItemSuppliesByItemIDs(ctx context.Context, itemIDs []uuid.UUID, organizationID string) (map[uuid.UUID][]*entities.BudgetItemSupplyEntity, error) {
	out := make(map[uuid.UUID][]*entities.BudgetItemSupplyEntity)
	ids := uniqueIDs(itemIDs)
	if len(ids) == 0 {
		return out, nil
	}

	var supplies []*models.BudgetItemSupplyModel
	if err := r.db.WithContext(ctx).
		Where("budget_item_id IN ? AND organization_id = ?", ids, organizationID).
		Order("budget_item_id ASC, \"order\" ASC").
		Find(&supplies).Error; err != nil {
		return nil, fmt.Errorf("failed to load item supplies: %w", err)
	}

	for _, s := range supplies {
		out[s.BudgetItemID] = append(out[s.BudgetItemID], s.ToEntity())
	}
	return out, nil
}

// ValidateSuppliesInOrg ensures every supply ID belongs to the given organization
// and is live (deleted_at IS NULL). Supplies are strictly tenant-scoped (the supply
// model has a NOT NULL organization_id), so any supply outside the caller's
// organization — or soft-deleted — is rejected. Empty input is a no-op.
func (r *budgetRepositoryImpl) ValidateSuppliesInOrg(ctx context.Context, supplyIDs []uuid.UUID, organizationID string) error {
	unique := uniqueIDs(supplyIDs)
	if len(unique) == 0 {
		return nil
	}

	var count int64
	if err := r.db.WithContext(ctx).
		Table("supplies").
		Where("id IN ? AND organization_id = ? AND deleted_at IS NULL", unique, organizationID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate supplies: %w", err)
	}

	if int(count) != len(unique) {
		return fmt.Errorf("%w: one or more referenced supplies do not belong to your organization", entities.ErrSupplyNotFound)
	}

	return nil
}

// GetSuppliesInfo loads the snapshot fields (name, unit, unit_cost) for every given
// LIVE supply in ONE org-scoped query. It is used to snapshot supplies at
// create/update time and to render a preview's supply lines. Soft-deleted supplies
// are excluded (callers validate liveness first).
func (r *budgetRepositoryImpl) GetSuppliesInfo(ctx context.Context, supplyIDs []uuid.UUID, organizationID string) (map[uuid.UUID]entities.SupplySnapshot, error) {
	out := make(map[uuid.UUID]entities.SupplySnapshot)
	ids := uniqueIDs(supplyIDs)
	if len(ids) == 0 {
		return out, nil
	}

	var rows []struct {
		ID           uuid.UUID `gorm:"column:id"`
		Name         string    `gorm:"column:name"`
		Unit         string    `gorm:"column:unit"`
		PackageSize  float64   `gorm:"column:package_size"`
		PackagePrice int64     `gorm:"column:package_price"`
		UnitCost     int64     `gorm:"column:unit_cost"`
	}
	if err := r.db.WithContext(ctx).
		Table("supplies").
		Select("id, name, unit, package_size, package_price, unit_cost").
		Where("id IN ? AND organization_id = ? AND deleted_at IS NULL", ids, organizationID).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load supplies info: %w", err)
	}
	for _, row := range rows {
		out[row.ID] = entities.SupplySnapshot{
			Name:         row.Name,
			Unit:         row.Unit,
			PackageSize:  row.PackageSize,
			PackagePrice: row.PackagePrice,
			UnitCost:     row.UnitCost,
		}
	}
	return out, nil
}

// loadItemSupplySpecs batch-loads the supplies (supply_id + quantity_per_piece) of
// every given item in a single org-scoped query, grouped by item ID and ordered by
// the display order.
func (r *budgetRepositoryImpl) loadItemSupplySpecs(ctx context.Context, items []*entities.BudgetItemEntity, organizationID string) (map[uuid.UUID][]entities.PricingSupplySpec, error) {
	out := make(map[uuid.UUID][]entities.PricingSupplySpec, len(items))
	if len(items) == 0 {
		return out, nil
	}

	itemIDs := make([]uuid.UUID, len(items))
	for i, it := range items {
		itemIDs[i] = it.ID
	}

	var rows []struct {
		BudgetItemID     uuid.UUID
		SupplyID         uuid.UUID
		QuantityPerPiece float64
	}
	if err := r.db.WithContext(ctx).
		Table("budget_item_supplies").
		Select("budget_item_id, supply_id, quantity_per_piece").
		Where("budget_item_id IN ? AND organization_id = ?", itemIDs, organizationID).
		Order("\"order\" ASC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load item supplies: %w", err)
	}

	for _, row := range rows {
		out[row.BudgetItemID] = append(out[row.BudgetItemID], entities.PricingSupplySpec{
			SupplyID:         row.SupplyID,
			QuantityPerPiece: row.QuantityPerPiece,
		})
	}
	return out, nil
}

// supplyPackagePricing is the package pricing (CENTS price + size) of a catalog
// supply used to price a budget line via SupplyCostCents.
type supplyPackagePricing struct {
	PackagePrice int64
	PackageSize  float64
}

// loadSupplyPricing batch-loads the package pricing (package_price CENTS +
// package_size) of every distinct supply referenced by the items in a single
// org-scoped query. The lookup is NOT filtered by deleted_at (historical
// consistency with saved budgets, matching filament/cost preset loading): a supply
// soft-deleted from the catalog still prices an existing budget on recalc/preview.
// A referenced supply that does not resolve inside the organization is an error
// (not a silent zero).
func (r *budgetRepositoryImpl) loadSupplyPricing(ctx context.Context, items []entities.PricingItemSpec, organizationID string) (map[uuid.UUID]supplyPackagePricing, error) {
	seen := make(map[uuid.UUID]struct{})
	ids := make([]uuid.UUID, 0)
	for _, item := range items {
		for _, s := range item.Supplies {
			if _, ok := seen[s.SupplyID]; ok {
				continue
			}
			seen[s.SupplyID] = struct{}{}
			ids = append(ids, s.SupplyID)
		}
	}

	out := make(map[uuid.UUID]supplyPackagePricing, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	var rows []struct {
		ID           uuid.UUID `gorm:"column:id"`
		PackagePrice int64     `gorm:"column:package_price"`
		PackageSize  float64   `gorm:"column:package_size"`
	}
	if err := r.db.WithContext(ctx).
		Table("supplies").
		Select("id, package_price, package_size").
		Where("id IN ? AND organization_id = ?", ids, organizationID).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load supply pricing: %w", err)
	}
	for _, row := range rows {
		out[row.ID] = supplyPackagePricing{PackagePrice: row.PackagePrice, PackageSize: row.PackageSize}
	}
	if len(out) != len(ids) {
		return nil, fmt.Errorf("%w: one or more referenced supplies do not belong to your organization", entities.ErrSupplyNotFound)
	}
	return out, nil
}

// refreshSupplySnapshots updates the name/unit/unit_cost snapshot of every budget
// item supply belonging to the given items to the current catalog values, in ONE
// statement. It keeps a stored budget's supply snapshots in sync with the prices
// just used by CalculateCosts. The join is NOT filtered by deleted_at (a later
// soft-deleted supply still refreshes consistently with the pricing load), but it IS
// org-scoped on BOTH sides of the join so a (hypothetical) supply ID collision across
// tenants can never pull another org's catalog values into a budget snapshot.
func (r *budgetRepositoryImpl) refreshSupplySnapshots(ctx context.Context, itemIDs []uuid.UUID, organizationID string) error {
	ids := uniqueIDs(itemIDs)
	if len(ids) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Exec(
		`UPDATE budget_item_supplies bis
		 SET name = s.name, unit = s.unit, package_size = s.package_size,
		     package_price = s.package_price, unit_cost = s.unit_cost, updated_at = ?
		 FROM supplies s
		 WHERE bis.supply_id = s.id
		   AND s.organization_id = ?
		   AND bis.organization_id = ?
		   AND bis.budget_item_id IN ?`,
		time.Now(), organizationID, organizationID, ids,
	).Error; err != nil {
		return fmt.Errorf("failed to refresh supply snapshots: %w", err)
	}
	return nil
}
