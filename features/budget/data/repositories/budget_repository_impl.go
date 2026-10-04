package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/data/models"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	pricing "github.com/RodolfoBonis/spooliq/features/budget/domain/services"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type budgetRepositoryImpl struct {
	db *gorm.DB
}

// NewBudgetRepository creates a new instance of BudgetRepository
func NewBudgetRepository(db *gorm.DB) repositories.BudgetRepository {
	return &budgetRepositoryImpl{db: db}
}

// WithTransaction runs fn inside a single database transaction. The repository
// handed to fn is bound to the transaction so every write participates in it;
// returning an error rolls everything back.
func (r *budgetRepositoryImpl) WithTransaction(ctx context.Context, fn func(repo repositories.BudgetRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&budgetRepositoryImpl{db: tx})
	})
}

func (r *budgetRepositoryImpl) Create(ctx context.Context, budget *entities.BudgetEntity) error {
	model := &models.BudgetModel{}
	model.FromEntity(budget)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create budget: %w", err)
	}
	return nil
}

func (r *budgetRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID, organizationID string) (*entities.BudgetEntity, error) {
	model := &models.BudgetModel{}

	query := r.db.WithContext(ctx)
	query = query.Where("organization_id = ?", organizationID)

	// Preload all relationships including nested ones
	if err := query.
		Preload("Customer").
		Preload("User").
		Preload("MachinePreset").
		Preload("EnergyPreset").
		Preload("CostPreset").
		Preload("Items").
		Preload("Items.Filament").
		Preload("Items.Filament.Brand").
		Preload("Items.Filament.Material").
		Preload("Items.CostPreset").
		Preload("Items.Filaments").
		Preload("Items.Filaments.Filament").
		Preload("Items.Filaments.Filament.Brand").
		Preload("Items.Filaments.Filament.Material").
		Preload("StatusHistory").
		First(model, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("budget not found: %w", err)
	}

	return model.ToEntity(), nil
}

// Update persists editable budget fields using an explicit column set so that
// zero values (false flags, nil presets, cleared pdf_url) are written back.
// GORM's Updates(struct) skips zero values, which previously made it impossible
// to turn flags off or clear optional references.
//
// The status column is intentionally NOT written here: it is owned exclusively by
// UpdateStatus. Writing status from a full Update caused a race where a PUT that
// read the budget as a draft could silently revert a concurrent PATCH approval.
// Cost columns are owned by CalculateCosts.
//
// The write is guarded to drafts belonging to the organization
// (WHERE id = ? AND organization_id = ? AND status = 'draft'). If no row matches
// (the budget was approved/deleted by a concurrent request, or belongs to another
// tenant) it returns entities.ErrBudgetNotEditable so callers can surface a 409
// Conflict instead of silently succeeding.
func (r *budgetRepositoryImpl) Update(ctx context.Context, budget *entities.BudgetEntity) error {
	model := &models.BudgetModel{}
	model.FromEntity(budget)

	updates := map[string]interface{}{
		"name":                model.Name,
		"description":         model.Description,
		"customer_id":         model.CustomerID,
		"machine_preset_id":   model.MachinePresetID,
		"energy_preset_id":    model.EnergyPresetID,
		"cost_preset_id":      model.CostPresetID,
		"include_energy_cost": model.IncludeEnergyCost,
		"include_waste_cost":  model.IncludeWasteCost,
		"delivery_days":       model.DeliveryDays,
		"payment_terms":       model.PaymentTerms,
		"notes":               model.Notes,
		"pdf_url":             model.PDFUrl,
		"updated_at":          model.UpdatedAt,
	}

	result := r.db.WithContext(ctx).
		Model(&models.BudgetModel{}).
		Where("id = ? AND organization_id = ? AND status = ?", budget.ID, budget.OrganizationID, string(entities.StatusDraft)).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update budget: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return entities.ErrBudgetNotEditable
	}

	return nil
}

// UpdateStatus writes only the status column (and updated_at), scoped by
// organization. It is the single owner of the status column; keeping status
// writes out of the generic Update prevents a concurrent full edit from
// reverting a status transition.
func (r *budgetRepositoryImpl) UpdateStatus(ctx context.Context, budgetID uuid.UUID, organizationID string, status entities.BudgetStatus) error {
	result := r.db.WithContext(ctx).
		Model(&models.BudgetModel{}).
		Where("id = ? AND organization_id = ?", budgetID, organizationID).
		Updates(map[string]interface{}{
			"status":     string(status),
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("failed to update budget status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return entities.ErrBudgetNotFound
	}
	return nil
}

// UpdatePDFURL writes only the pdf_url column (and updated_at), scoped by
// organization. Unlike Update it is NOT restricted to drafts, because PDFs are
// generated for approved/sent budgets.
func (r *budgetRepositoryImpl) UpdatePDFURL(ctx context.Context, budgetID uuid.UUID, organizationID string, pdfURL *string) error {
	result := r.db.WithContext(ctx).
		Model(&models.BudgetModel{}).
		Where("id = ? AND organization_id = ?", budgetID, organizationID).
		Updates(map[string]interface{}{
			"pdf_url":    pdfURL,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("failed to update budget pdf url: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return entities.ErrBudgetNotFound
	}
	return nil
}

func (r *budgetRepositoryImpl) Delete(ctx context.Context, id uuid.UUID, organizationID string) error {
	if err := r.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Delete(&models.BudgetModel{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete budget: %w", err)
	}
	return nil
}

func (r *budgetRepositoryImpl) FindAll(ctx context.Context, organizationID string, limit, offset int) ([]*entities.BudgetEntity, int, error) {
	var budgets []*models.BudgetModel
	var total int64

	query := r.db.WithContext(ctx).Model(&models.BudgetModel{})

	// Filter by organization
	query = query.Where("organization_id = ?", organizationID)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count budgets: %w", err)
	}

	// Get paginated results with relationships
	// Note: For list views, we preload only essential relationships to avoid performance issues
	// For detailed view, use FindByID which loads everything
	if err := query.
		Preload("Customer").
		Preload("Items").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&budgets).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find budgets: %w", err)
	}

	// Convert to entities
	entities := make([]*entities.BudgetEntity, len(budgets))
	for i, model := range budgets {
		entities[i] = model.ToEntity()
	}

	return entities, int(total), nil
}

func (r *budgetRepositoryImpl) FindByCustomer(ctx context.Context, customerID uuid.UUID, organizationID string) ([]*entities.BudgetEntity, error) {
	var budgets []*models.BudgetModel

	query := r.db.WithContext(ctx).Model(&models.BudgetModel{}).Where("customer_id = ? AND organization_id = ?", customerID, organizationID)

	// Get results
	if err := query.
		Order("created_at DESC").
		Find(&budgets).Error; err != nil {
		return nil, fmt.Errorf("failed to find budgets: %w", err)
	}

	// Convert to entities
	entities := make([]*entities.BudgetEntity, len(budgets))
	for i, model := range budgets {
		entities[i] = model.ToEntity()
	}

	return entities, nil
}

func (r *budgetRepositoryImpl) SearchBudgets(ctx context.Context, organizationID string, filters map[string]interface{}, limit, offset int) ([]*entities.BudgetEntity, int, error) {
	var budgets []*models.BudgetModel
	var total int64

	query := r.db.WithContext(ctx).Model(&models.BudgetModel{})

	// Always scope to organization
	query = query.Where("organization_id = ?", organizationID)

	// Apply filters
	if name, ok := filters["name"].(string); ok && name != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%")
	}
	if customerID, ok := filters["customer_id"].(uuid.UUID); ok && customerID != uuid.Nil {
		query = query.Where("customer_id = ?", customerID)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if startDate, ok := filters["start_date"].(time.Time); ok && !startDate.IsZero() {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate, ok := filters["end_date"].(time.Time); ok && !endDate.IsZero() {
		query = query.Where("created_at <= ?", endDate)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count budgets: %w", err)
	}

	// Get paginated results
	if err := query.
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&budgets).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search budgets: %w", err)
	}

	// Convert to entities
	entities := make([]*entities.BudgetEntity, len(budgets))
	for i, model := range budgets {
		entities[i] = model.ToEntity()
	}

	return entities, int(total), nil
}

func (r *budgetRepositoryImpl) AddItem(ctx context.Context, item *entities.BudgetItemEntity) error {
	model := &models.BudgetItemModel{}
	model.FromEntity(item)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to add budget item: %w", err)
	}
	return nil
}

func (r *budgetRepositoryImpl) RemoveItem(ctx context.Context, itemID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.BudgetItemModel{}, "id = ?", itemID).Error; err != nil {
		return fmt.Errorf("failed to remove budget item: %w", err)
	}
	return nil
}

// UpdateItem persists the calculated cost columns of a budget item. It writes an
// explicit column set (not Updates(struct)) so that zero-valued costs are
// persisted: GORM's Updates(struct) skips zero fields, which previously left a
// stale EnergyCost on an item after energy was turned off even though the budget
// totals (recomputed from scratch) dropped to 0. Only cost columns are touched;
// product/time fields are owned by the item replacement path. The write is scoped
// by organization as defense-in-depth.
func (r *budgetRepositoryImpl) UpdateItem(ctx context.Context, item *entities.BudgetItemEntity) error {
	updates := map[string]interface{}{
		"filament_cost":     item.FilamentCost,
		"waste_cost":        item.WasteCost,
		"energy_cost":       item.EnergyCost,
		"setup_cost":        item.SetupCost,
		"manual_labor_cost": item.ManualLaborCost,
		"item_total_cost":   item.ItemTotalCost,
		"unit_price":        item.UnitPrice,
		"updated_at":        time.Now(),
	}

	if err := r.db.WithContext(ctx).
		Model(&models.BudgetItemModel{}).
		Where("id = ? AND organization_id = ?", item.ID, item.OrganizationID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update budget item: %w", err)
	}

	return nil
}

func (r *budgetRepositoryImpl) GetItems(ctx context.Context, budgetID uuid.UUID) ([]*entities.BudgetItemEntity, error) {
	var items []*models.BudgetItemModel

	if err := r.db.WithContext(ctx).
		Where("budget_id = ?", budgetID).
		Order("\"order\" ASC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to get budget items: %w", err)
	}

	entities := make([]*entities.BudgetItemEntity, len(items))
	for i, model := range items {
		entities[i] = model.ToEntity()
	}

	return entities, nil
}

func (r *budgetRepositoryImpl) DeleteAllItems(ctx context.Context, budgetID uuid.UUID, organizationID string) error {
	if err := r.db.WithContext(ctx).
		Where("budget_id = ? AND organization_id = ?", budgetID, organizationID).
		Delete(&models.BudgetItemModel{}).Error; err != nil {
		return fmt.Errorf("failed to delete budget items: %w", err)
	}
	return nil
}

// ============================================================================
// Item Filament Operations (NEW - Multi-filament support)
// ============================================================================

func (r *budgetRepositoryImpl) AddItemFilament(ctx context.Context, filament *entities.BudgetItemFilamentEntity) error {
	model := &models.BudgetItemFilamentModel{}
	model.FromEntity(filament)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to add item filament: %w", err)
	}
	return nil
}

func (r *budgetRepositoryImpl) RemoveItemFilament(ctx context.Context, filamentID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.BudgetItemFilamentModel{}, "id = ?", filamentID).Error; err != nil {
		return fmt.Errorf("failed to remove item filament: %w", err)
	}
	return nil
}

func (r *budgetRepositoryImpl) GetItemFilaments(ctx context.Context, itemID uuid.UUID) ([]*entities.BudgetItemFilamentEntity, error) {
	var filaments []*models.BudgetItemFilamentModel

	if err := r.db.WithContext(ctx).
		Where("budget_item_id = ?", itemID).
		Order("\"order\" ASC").
		Find(&filaments).Error; err != nil {
		return nil, fmt.Errorf("failed to get item filaments: %w", err)
	}

	entities := make([]*entities.BudgetItemFilamentEntity, len(filaments))
	for i, f := range filaments {
		entities[i] = f.ToEntity()
	}

	return entities, nil
}

func (r *budgetRepositoryImpl) DeleteAllItemFilaments(ctx context.Context, itemID uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Where("budget_item_id = ?", itemID).
		Delete(&models.BudgetItemFilamentModel{}).Error; err != nil {
		return fmt.Errorf("failed to delete item filaments: %w", err)
	}
	return nil
}

// GetFilamentUsageInfo retrieves detailed filament usage info for a budget item,
// scoped by organization (the budget_item_filaments rows carry organization_id).
func (r *budgetRepositoryImpl) GetFilamentUsageInfo(ctx context.Context, itemID uuid.UUID, organizationID string) ([]entities.FilamentUsageInfo, error) {
	// Get filaments with all related info via JOIN
	var results []struct {
		FilamentID   uuid.UUID
		Quantity     float64
		Order        int
		FilamentName string
		BrandName    string
		MaterialName string
		Color        string
		ColorType    string
		ColorData    []byte // Changed to []byte to handle JSON properly
		ColorHex     string
		ColorPreview string
		PricePerKg   float64
	}

	err := r.db.WithContext(ctx).
		Table("budget_item_filaments bif").
		Select(`
			bif.filament_id,
			bif.quantity,
			bif."order",
			f.name as filament_name,
			b.name as brand_name,
			m.name as material_name,
			f.color,
			f.color_type,
			f.color_data,
			f.color_hex,
			f.color_preview,
			f.price_per_kg
		`).
		Joins("JOIN filaments f ON f.id = bif.filament_id").
		Joins("JOIN brands b ON b.id = f.brand_id").
		Joins("JOIN materials m ON m.id = f.material_id").
		Where("bif.budget_item_id = ? AND bif.organization_id = ?", itemID, organizationID).
		Order("bif.\"order\" ASC").
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get filament usage info: %w", err)
	}

	infos := make([]entities.FilamentUsageInfo, len(results))
	for i, res := range results {
		// Quantity is the total grams for the item. price_per_kg is in CENTS/kg,
		// costs are stored in CENTS => convert and round to the nearest cent.
		cost := pricing.FilamentCostCents(res.Quantity, res.PricePerKg)

		infos[i] = entities.FilamentUsageInfo{
			FilamentID:   res.FilamentID.String(),
			FilamentName: res.FilamentName,
			BrandName:    res.BrandName,
			MaterialName: res.MaterialName,
			Color:        res.Color,
			ColorType:    res.ColorType,
			ColorData:    json.RawMessage(res.ColorData), // Convert []byte to json.RawMessage
			ColorHex:     res.ColorHex,
			ColorPreview: res.ColorPreview,
			Quantity:     res.Quantity,
			Cost:         cost,
			Order:        res.Order,
		}
	}

	return infos, nil
}

// ============================================================================
// Status History Operations
// ============================================================================

func (r *budgetRepositoryImpl) AddStatusHistory(ctx context.Context, history *entities.BudgetStatusHistoryEntity) error {
	model := &models.BudgetStatusHistoryModel{}
	model.FromEntity(history)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to add status history: %w", err)
	}
	return nil
}

func (r *budgetRepositoryImpl) GetStatusHistory(ctx context.Context, budgetID uuid.UUID) ([]entities.BudgetStatusHistoryEntity, error) {
	var history []*models.BudgetStatusHistoryModel

	if err := r.db.WithContext(ctx).
		Where("budget_id = ?", budgetID).
		Order("created_at DESC").
		Find(&history).Error; err != nil {
		return nil, fmt.Errorf("failed to get status history: %w", err)
	}

	entities := make([]entities.BudgetStatusHistoryEntity, len(history))
	for i, model := range history {
		entities[i] = *model.ToEntity()
	}

	return entities, nil
}

// CalculateCosts calculates all costs for a budget (multi-filament aware).
//
// All monetary results are stored in CENTS; filament price_per_kg comes in CENTS,
// energy/labor preset rates come in REAIS. Every conversion rounds to the nearest cent via
// the pure helpers in the budget domain services package. All lookups are scoped
// by the budget's organization so cross-tenant references cannot leak into the
// calculation, and every error is propagated instead of being silently ignored.
//
// Historical consistency: presets are resolved by (id, organization) WITHOUT a
// deleted_at filter, so a preset that was referenced by this budget and later
// soft-deleted is still usable here. Rejecting soft-deleted presets would make an
// already-saved budget impossible to recalculate (or even rename). New references
// are validated separately (see ValidatePresetInOrg, which does require the preset
// to be live). If a referenced preset ID is set but its typed child row cannot be
// found, this returns an error instead of silently charging 0.
func (r *budgetRepositoryImpl) CalculateCosts(ctx context.Context, budgetID uuid.UUID) error {
	// Get budget
	var budget models.BudgetModel
	if err := r.db.WithContext(ctx).First(&budget, "id = ?", budgetID).Error; err != nil {
		return fmt.Errorf("failed to get budget: %w", err)
	}

	organizationID := budget.OrganizationID

	// Get items
	items, err := r.GetItems(ctx, budgetID)
	if err != nil {
		return err
	}

	// Get machine and energy presets (scoped by organization)
	var machinePreset *entities.PresetInfo
	var energyPreset *entities.PresetInfo

	if budget.MachinePresetID != nil {
		machinePreset, err = r.GetPresetInfo(ctx, *budget.MachinePresetID, "machine", organizationID)
		if err != nil {
			return fmt.Errorf("failed to load machine preset: %w", err)
		}
	}
	if budget.EnergyPresetID != nil {
		energyPreset, err = r.GetPresetInfo(ctx, *budget.EnergyPresetID, "energy", organizationID)
		if err != nil {
			return fmt.Errorf("failed to load energy preset: %w", err)
		}
	}

	// Hoist the energy inputs out of the per-item loop: the machine power draw and
	// the energy price per kWh are budget-level and constant across items, so they
	// are loaded once (if energy is enabled) instead of once per item.
	var powerConsumption float64
	var energyPrice float64
	energyEnabled := budget.IncludeEnergyCost && machinePreset != nil && energyPreset != nil
	if energyEnabled {
		res := r.db.WithContext(ctx).
			Table("machine_presets").
			Select("power_consumption").
			Where("id = ? AND organization_id = ?", *budget.MachinePresetID, organizationID).
			Scan(&powerConsumption)
		if res.Error != nil {
			return fmt.Errorf("failed to load machine power consumption: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("machine preset %s not found for organization", *budget.MachinePresetID)
		}

		res = r.db.WithContext(ctx).
			Table("energy_presets").
			Select("energy_cost_per_kwh").
			Where("id = ? AND organization_id = ?", *budget.EnergyPresetID, organizationID).
			Scan(&energyPrice)
		if res.Error != nil {
			return fmt.Errorf("failed to load energy price: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("energy preset %s not found for organization", *budget.EnergyPresetID)
		}
	}

	// Cache cost-preset labor rates so each distinct preset is loaded at most once
	// across all items (and the budget-level fallback) instead of once per item.
	laborRateCache := make(map[uuid.UUID]float64)
	laborRateFor := func(presetID uuid.UUID) (float64, error) {
		if rate, ok := laborRateCache[presetID]; ok {
			return rate, nil
		}
		var rate float64
		res := r.db.WithContext(ctx).
			Table("cost_presets").
			Select("labor_cost_per_hour").
			Where("id = ? AND organization_id = ?", presetID, organizationID).
			Scan(&rate)
		if res.Error != nil {
			return 0, fmt.Errorf("failed to load labor rate: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return 0, fmt.Errorf("cost preset %s not found for organization", presetID)
		}
		laborRateCache[presetID] = rate
		return rate, nil
	}

	var totalFilamentCost, totalWasteCost, totalEnergyCost, totalSetupCost, totalManualLaborCost int64

	// Process each item (product)
	for _, item := range items {
		var itemFilamentCost, itemWasteCost, itemEnergyCost, itemSetupCost, itemManualLaborCost int64

		// 1. Get filaments for this item
		itemFilaments, err := r.GetItemFilaments(ctx, item.ID)
		if err != nil {
			return fmt.Errorf("failed to get item filaments: %w", err)
		}

		// 2. Calculate filament cost (sum of all filaments in this item)
		var avgPrice float64
		var totalPrice float64

		for _, itemFil := range itemFilaments {
			filament, err := r.GetFilamentInfo(ctx, itemFil.FilamentID, organizationID)
			if err != nil {
				return fmt.Errorf("failed to load filament %s: %w", itemFil.FilamentID, err)
			}

			itemFilamentCost += pricing.FilamentCostCents(itemFil.Quantity, filament.PricePerKg)
			totalPrice += filament.PricePerKg
		}

		if len(itemFilaments) > 0 {
			avgPrice = totalPrice / float64(len(itemFilaments))
		}

		// 3. Calculate waste cost (AMS multi-color)
		if budget.IncludeWasteCost && len(itemFilaments) > 1 {
			wastePerChange := 15.0 // grams
			numChanges := len(itemFilaments) - 1
			totalWaste := wastePerChange * float64(numChanges)
			itemWasteCost = pricing.WasteCostCents(totalWaste, avgPrice)
		}

		// 4. Calculate energy cost (proportional to this item's print time)
		if energyEnabled {
			itemHours := float64(item.PrintTimeHours) + float64(item.PrintTimeMinutes)/60.0
			itemEnergyCost = pricing.EnergyCostCents(powerConsumption, itemHours, energyPrice)
		}

		// 5 & 6. Calculate setup cost and manual labor cost
		var laborRate float64
		if item.CostPresetID != nil {
			laborRate, err = laborRateFor(*item.CostPresetID)
			if err != nil {
				return err
			}
		}

		itemSetupCost = pricing.LaborCostCents(item.SetupTimeMinutes, laborRate)
		itemManualLaborCost = pricing.LaborCostCents(item.ManualLaborMinutesTotal, laborRate)

		// 7. Calculate item total cost
		item.FilamentCost = itemFilamentCost
		item.WasteCost = itemWasteCost
		item.EnergyCost = itemEnergyCost
		item.SetupCost = itemSetupCost
		item.ManualLaborCost = itemManualLaborCost
		item.ItemTotalCost = itemFilamentCost + itemWasteCost + itemEnergyCost + itemSetupCost + itemManualLaborCost

		// 8. Calculate unit price
		item.UnitPrice = pricing.UnitPriceCents(item.ItemTotalCost, item.ProductQuantity)

		// 9. Update item in database
		if err := r.UpdateItem(ctx, item); err != nil {
			return fmt.Errorf("failed to update item costs: %w", err)
		}

		// 10. Sum to budget totals
		totalFilamentCost += itemFilamentCost
		totalWasteCost += itemWasteCost
		totalEnergyCost += itemEnergyCost
		totalSetupCost += itemSetupCost
		totalManualLaborCost += itemManualLaborCost
	}

	// Calculate budget subtotal (before overhead and profit)
	budgetSubtotal := totalFilamentCost + totalWasteCost + totalEnergyCost + totalSetupCost + totalManualLaborCost

	// Calculate overhead and profit
	var overheadCost, profitAmount int64

	var costPreset struct {
		OverheadPercentage     float64
		ProfitMarginPercentage float64
	}

	// Try to get from budget-level CostPreset first, fall back to first item's CostPreset.
	var costPresetID *uuid.UUID
	if budget.CostPresetID != nil {
		costPresetID = budget.CostPresetID
	} else if len(items) > 0 && items[0].CostPresetID != nil {
		costPresetID = items[0].CostPresetID
	}

	if costPresetID != nil {
		res := r.db.WithContext(ctx).
			Table("cost_presets").
			Select("overhead_percentage, profit_margin_percentage").
			Where("id = ? AND organization_id = ?", *costPresetID, organizationID).
			Scan(&costPreset)
		if res.Error != nil {
			return fmt.Errorf("failed to load cost preset percentages: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("cost preset %s not found for organization", *costPresetID)
		}
	}

	// Overhead: Subtotal * (OverheadPercentage / 100)
	overheadCost = pricing.PercentageCents(budgetSubtotal, costPreset.OverheadPercentage)

	// Profit: (Subtotal + Overhead) * (ProfitMarginPercentage / 100)
	profitAmount = pricing.PercentageCents(budgetSubtotal+overheadCost, costPreset.ProfitMarginPercentage)

	// Update budget totals
	budget.FilamentCost = totalFilamentCost
	budget.WasteCost = totalWasteCost
	budget.EnergyCost = totalEnergyCost
	budget.SetupCost = totalSetupCost
	budget.LaborCost = totalManualLaborCost
	budget.OverheadCost = overheadCost
	budget.ProfitAmount = profitAmount
	budget.TotalCost = budgetSubtotal + overheadCost + profitAmount

	if err := r.db.WithContext(ctx).
		Model(&models.BudgetModel{}).
		Where("id = ? AND organization_id = ?", budgetID, organizationID).
		Updates(map[string]interface{}{
			"filament_cost": budget.FilamentCost,
			"waste_cost":    budget.WasteCost,
			"energy_cost":   budget.EnergyCost,
			"setup_cost":    budget.SetupCost,
			"labor_cost":    budget.LaborCost,
			"overhead_cost": budget.OverheadCost,
			"profit_amount": budget.ProfitAmount,
			"total_cost":    budget.TotalCost,
		}).Error; err != nil {
		return fmt.Errorf("failed to update budget costs: %w", err)
	}

	return nil
}

// ValidateFilamentsInOrg ensures every filament ID belongs to the given
// organization. Filaments are strictly tenant-scoped (the filament model has a
// NOT NULL organization_id, there is no global/shared filament), so any filament
// outside the caller's organization is rejected.
func (r *budgetRepositoryImpl) ValidateFilamentsInOrg(ctx context.Context, filamentIDs []uuid.UUID, organizationID string) error {
	if len(filamentIDs) == 0 {
		return nil
	}

	// De-duplicate so the count comparison is accurate.
	seen := make(map[uuid.UUID]struct{}, len(filamentIDs))
	unique := make([]uuid.UUID, 0, len(filamentIDs))
	for _, id := range filamentIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}

	var count int64
	if err := r.db.WithContext(ctx).
		Table("filaments").
		Where("id IN ? AND organization_id = ? AND deleted_at IS NULL", unique, organizationID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate filaments: %w", err)
	}

	if int(count) != len(unique) {
		return fmt.Errorf("one or more referenced filaments do not belong to your organization")
	}

	return nil
}

// ValidatePresetInOrg ensures the preset ID belongs to the given organization AND
// has the expected type (machine/energy/cost). The type check prevents, for
// example, a cost preset being accepted as a machine_preset_id, which would later
// cause the typed child-row lookup in CalculateCosts to find nothing and silently
// charge 0. Only live presets (deleted_at IS NULL) are accepted: this validates
// freshly provided references, so a soft-deleted preset must not be re-attachable.
func (r *budgetRepositoryImpl) ValidatePresetInOrg(ctx context.Context, presetID uuid.UUID, presetType string, organizationID string) error {
	var count int64
	if err := r.db.WithContext(ctx).
		Table("presets").
		Where("id = ? AND organization_id = ? AND type = ? AND deleted_at IS NULL", presetID, organizationID, presetType).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate preset: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("referenced %s preset %s does not belong to your organization", presetType, presetID)
	}

	return nil
}

// GetCustomerInfo fetches customer information by ID, scoped by organization.
func (r *budgetRepositoryImpl) GetCustomerInfo(ctx context.Context, customerID uuid.UUID, organizationID string) (*entities.CustomerInfo, error) {
	var customer struct {
		ID       uuid.UUID `gorm:"column:id"`
		Name     string    `gorm:"column:name"`
		Email    *string   `gorm:"column:email"`
		Phone    *string   `gorm:"column:phone"`
		Document *string   `gorm:"column:document"`
	}

	if err := r.db.WithContext(ctx).
		Table("customers").
		Select("id, name, email, phone, document").
		Where("id = ? AND organization_id = ?", customerID, organizationID).
		First(&customer).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch customer: %w", err)
	}

	return &entities.CustomerInfo{
		ID:       customer.ID.String(),
		Name:     customer.Name,
		Email:    customer.Email,
		Phone:    customer.Phone,
		Document: customer.Document,
	}, nil
}

// GetFilamentInfo fetches filament information by ID, scoped by organization.
func (r *budgetRepositoryImpl) GetFilamentInfo(ctx context.Context, filamentID uuid.UUID, organizationID string) (*entities.FilamentInfo, error) {
	var filament struct {
		ID         uuid.UUID `gorm:"column:id"`
		Name       string    `gorm:"column:name"`
		Color      string    `gorm:"column:color"`
		PricePerKg float64   `gorm:"column:price_per_kg"`
		BrandID    uuid.UUID `gorm:"column:brand_id"`
		MaterialID uuid.UUID `gorm:"column:material_id"`
	}

	if err := r.db.WithContext(ctx).
		Table("filaments").
		Select("id, name, color, price_per_kg, brand_id, material_id").
		Where("id = ? AND organization_id = ?", filamentID, organizationID).
		First(&filament).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch filament: %w", err)
	}

	// Get brand name
	var brandName string
	if err := r.db.WithContext(ctx).
		Table("brands").
		Select("name").
		Where("id = ?", filament.BrandID).
		Scan(&brandName).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch filament brand: %w", err)
	}

	// Get material name
	var materialName string
	if err := r.db.WithContext(ctx).
		Table("materials").
		Select("name").
		Where("id = ?", filament.MaterialID).
		Scan(&materialName).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch filament material: %w", err)
	}

	return &entities.FilamentInfo{
		ID:           filament.ID.String(),
		Name:         filament.Name,
		BrandName:    brandName,
		MaterialName: materialName,
		Color:        filament.Color,
		PricePerKg:   filament.PricePerKg,
	}, nil
}

// GetPresetInfo fetches preset information by ID, scoped by organization and type.
//
// Historical consistency: this intentionally does NOT filter on deleted_at, so a
// preset referenced by an existing budget that was later soft-deleted is still
// resolvable here (used by CalculateCosts). Freshly provided references are vetted
// by ValidatePresetInOrg, which does require the preset to be live.
func (r *budgetRepositoryImpl) GetPresetInfo(ctx context.Context, presetID uuid.UUID, presetType string, organizationID string) (*entities.PresetInfo, error) {
	// First get the basic preset info from the main presets table
	var basePreset struct {
		ID   uuid.UUID `gorm:"column:id"`
		Name string    `gorm:"column:name"`
		Type string    `gorm:"column:type"`
	}

	if err := r.db.WithContext(ctx).
		Table("presets").
		Select("id, name, type").
		Where("id = ? AND organization_id = ? AND type = ?", presetID, organizationID, presetType).
		First(&basePreset).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch %s preset: %w", presetType, err)
	}

	return &entities.PresetInfo{
		ID:   basePreset.ID.String(),
		Name: basePreset.Name,
		Type: presetType,
	}, nil
}

// GetCompanyByOrganizationID retrieves company information by organization ID
func (r *budgetRepositoryImpl) GetCompanyByOrganizationID(ctx context.Context, organizationID string) (*entities.CompanyInfo, error) {
	var company struct {
		ID        uuid.UUID `gorm:"column:id"`
		Name      string    `gorm:"column:name"`
		Email     *string   `gorm:"column:email"`
		Phone     *string   `gorm:"column:phone"`
		WhatsApp  *string   `gorm:"column:whats_app"`
		Instagram *string   `gorm:"column:instagram"`
		Website   *string   `gorm:"column:website"`
		LogoURL   *string   `gorm:"column:logo_url"`
	}

	if err := r.db.WithContext(ctx).
		Table("companies").
		Select("id, name, email, phone, whats_app, instagram, website, logo_url").
		Where("organization_id = ?", organizationID).
		Where("deleted_at IS NULL").
		First(&company).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("company not found for organization %s (please configure your company settings)", organizationID)
		}
		return nil, fmt.Errorf("failed to fetch company: %w", err)
	}

	return &entities.CompanyInfo{
		ID:        company.ID.String(),
		Name:      company.Name,
		Email:     company.Email,
		Phone:     company.Phone,
		WhatsApp:  company.WhatsApp,
		Instagram: company.Instagram,
		Website:   company.Website,
		LogoURL:   company.LogoURL,
	}, nil
}

// FindItemsByBudgetID retrieves all items for a budget
func (r *budgetRepositoryImpl) FindItemsByBudgetID(ctx context.Context, budgetID uuid.UUID) ([]*entities.BudgetItemEntity, error) {
	var items []*models.BudgetItemModel

	if err := r.db.WithContext(ctx).
		Where("budget_id = ?", budgetID).
		Order("\"order\" ASC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch budget items: %w", err)
	}

	entities := make([]*entities.BudgetItemEntity, 0, len(items))
	for _, item := range items {
		entities = append(entities, item.ToEntity())
	}

	return entities, nil
}
