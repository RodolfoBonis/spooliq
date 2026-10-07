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
		Preload("Profile").
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
		"name":                 model.Name,
		"description":          model.Description,
		"customer_id":          model.CustomerID,
		"profile_id":           model.ProfileID,
		"machine_preset_id":    model.MachinePresetID,
		"energy_preset_id":     model.EnergyPresetID,
		"cost_preset_id":       model.CostPresetID,
		"include_energy_cost":  model.IncludeEnergyCost,
		"include_waste_cost":   model.IncludeWasteCost,
		"include_machine_cost": model.IncludeMachineCost,
		"discount_type":        model.DiscountType,
		"discount_value":       model.DiscountValue,
		"include_shipping":     model.IncludeShipping,
		"shipping_override":    model.ShippingOverride,
		"tax_rate":             model.TaxRate,
		"delivery_days":        model.DeliveryDays,
		"payment_terms":        model.PaymentTerms,
		"notes":                model.Notes,
		"pdf_url":              model.PDFUrl,
		"updated_at":           model.UpdatedAt,
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
// organization AND guarded by the expected current status. It is the single owner
// of the status column; keeping status writes out of the generic Update prevents a
// concurrent full edit from reverting a status transition.
//
// The expectedCurrent guard (WHERE status = expectedCurrent) makes the write
// optimistic: if another request already moved the budget off that status, no row
// matches and this returns entities.ErrBudgetStatusConflict so the caller can
// surface a 409 Conflict instead of silently applying a stale transition.
func (r *budgetRepositoryImpl) UpdateStatus(ctx context.Context, budgetID uuid.UUID, organizationID string, expectedCurrent, newStatus entities.BudgetStatus) error {
	result := r.db.WithContext(ctx).
		Model(&models.BudgetModel{}).
		Where("id = ? AND organization_id = ? AND status = ?", budgetID, organizationID, string(expectedCurrent)).
		Updates(map[string]interface{}{
			"status":     string(newStatus),
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("failed to update budget status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return entities.ErrBudgetStatusConflict
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

// SearchBudgets returns a filtered, sorted, org-scoped page of budgets WITHOUT any
// relationship preloads. The list response is assembled by the use case from the
// batch loaders below, so a page of N budgets costs a constant number of queries.
func (r *budgetRepositoryImpl) SearchBudgets(ctx context.Context, organizationID string, filters map[string]interface{}, orderBy string, limit, offset int) ([]*entities.BudgetEntity, int, error) {
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

	// Get total count (before ordering/pagination).
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count budgets: %w", err)
	}

	// orderBy comes from the sort whitelist (injection-safe); fall back to a stable
	// default when empty.
	if strings.TrimSpace(orderBy) == "" {
		orderBy = "created_at DESC"
	}

	if err := query.
		Limit(limit).
		Offset(offset).
		Order(orderBy).
		Find(&budgets).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search budgets: %w", err)
	}

	result := make([]*entities.BudgetEntity, len(budgets))
	for i, model := range budgets {
		result[i] = model.ToEntity()
	}

	return result, int(total), nil
}

// GetCustomersInfo loads the {id, name, ...} of every given customer in ONE
// org-scoped query, returned as a map keyed by customer ID.
func (r *budgetRepositoryImpl) GetCustomersInfo(ctx context.Context, customerIDs []uuid.UUID, organizationID string) (map[uuid.UUID]*entities.CustomerInfo, error) {
	out := make(map[uuid.UUID]*entities.CustomerInfo)
	ids := uniqueIDs(customerIDs)
	if len(ids) == 0 {
		return out, nil
	}

	var rows []struct {
		ID       uuid.UUID `gorm:"column:id"`
		Name     string    `gorm:"column:name"`
		Email    *string   `gorm:"column:email"`
		Phone    *string   `gorm:"column:phone"`
		Document *string   `gorm:"column:document"`
	}
	if err := r.db.WithContext(ctx).
		Table("customers").
		Select("id, name, email, phone, document").
		Where("id IN ? AND organization_id = ?", ids, organizationID).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load customers: %w", err)
	}

	for _, row := range rows {
		out[row.ID] = &entities.CustomerInfo{
			ID:       row.ID.String(),
			Name:     row.Name,
			Email:    row.Email,
			Phone:    row.Phone,
			Document: row.Document,
		}
	}
	return out, nil
}

// GetItemsByBudgetIDs loads the items of every given budget in ONE query, grouped
// by budget ID and ordered by item "order" within each budget.
func (r *budgetRepositoryImpl) GetItemsByBudgetIDs(ctx context.Context, budgetIDs []uuid.UUID, organizationID string) (map[uuid.UUID][]*entities.BudgetItemEntity, error) {
	out := make(map[uuid.UUID][]*entities.BudgetItemEntity)
	ids := uniqueIDs(budgetIDs)
	if len(ids) == 0 {
		return out, nil
	}

	var items []*models.BudgetItemModel
	if err := r.db.WithContext(ctx).
		Where("budget_id IN ? AND organization_id = ?", ids, organizationID).
		Order("budget_id ASC, \"order\" ASC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to load budget items: %w", err)
	}

	for _, item := range items {
		out[item.BudgetID] = append(out[item.BudgetID], item.ToEntity())
	}
	return out, nil
}

// GetFilamentUsageInfoByItemIDs loads the filament usage of every given item in ONE
// org-scoped JOIN query, grouped by item ID and ordered by the color-change order.
func (r *budgetRepositoryImpl) GetFilamentUsageInfoByItemIDs(ctx context.Context, itemIDs []uuid.UUID, organizationID string) (map[uuid.UUID][]entities.FilamentUsageInfo, error) {
	out := make(map[uuid.UUID][]entities.FilamentUsageInfo)
	ids := uniqueIDs(itemIDs)
	if len(ids) == 0 {
		return out, nil
	}

	var results []struct {
		BudgetItemID uuid.UUID
		FilamentID   uuid.UUID
		Quantity     float64
		Order        int
		FilamentName string
		BrandName    string
		MaterialName string
		Color        string
		ColorType    string
		ColorData    []byte
		ColorHex     string
		ColorPreview string
		PricePerKg   float64
	}

	err := r.db.WithContext(ctx).
		Table("budget_item_filaments bif").
		Select(`
			bif.budget_item_id,
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
		Where("bif.budget_item_id IN ? AND bif.organization_id = ?", ids, organizationID).
		Order("bif.budget_item_id ASC, bif.\"order\" ASC").
		Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get filament usage info: %w", err)
	}

	for _, res := range results {
		// Quantity is the total grams for the item. price_per_kg is in CENTS/kg,
		// costs are stored in CENTS => convert and round to the nearest cent.
		cost := pricing.FilamentCostCents(res.Quantity, res.PricePerKg)
		out[res.BudgetItemID] = append(out[res.BudgetItemID], entities.FilamentUsageInfo{
			FilamentID:   res.FilamentID.String(),
			FilamentName: res.FilamentName,
			BrandName:    res.BrandName,
			MaterialName: res.MaterialName,
			Color:        res.Color,
			ColorType:    res.ColorType,
			ColorData:    json.RawMessage(res.ColorData),
			ColorHex:     res.ColorHex,
			ColorPreview: res.ColorPreview,
			Quantity:     res.Quantity,
			Cost:         cost,
			Order:        res.Order,
		})
	}
	return out, nil
}

// GetCostPresetNames loads the display name of every given cost preset in ONE
// org-scoped query. Names are resolved WITHOUT a deleted_at filter for historical
// consistency with budgets referencing a since-soft-deleted preset.
func (r *budgetRepositoryImpl) GetCostPresetNames(ctx context.Context, presetIDs []uuid.UUID, organizationID string) (map[uuid.UUID]string, error) {
	out := make(map[uuid.UUID]string)
	ids := uniqueIDs(presetIDs)
	if len(ids) == 0 {
		return out, nil
	}

	var rows []struct {
		ID   uuid.UUID `gorm:"column:id"`
		Name string    `gorm:"column:name"`
	}
	if err := r.db.WithContext(ctx).
		Table("presets").
		Select("id, name").
		Where("id IN ? AND organization_id = ? AND type = ?", ids, organizationID, "cost").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load cost preset names: %w", err)
	}
	for _, row := range rows {
		out[row.ID] = row.Name
	}
	return out, nil
}

// GetProfileNames loads the display name of every given live print profile in ONE
// org-scoped query (soft-deleted profiles are excluded so the list matches the
// detail response, which leaves the name empty for a soft-deleted profile).
func (r *budgetRepositoryImpl) GetProfileNames(ctx context.Context, profileIDs []uuid.UUID, organizationID string) (map[uuid.UUID]string, error) {
	out := make(map[uuid.UUID]string)
	ids := uniqueIDs(profileIDs)
	if len(ids) == 0 {
		return out, nil
	}

	var rows []struct {
		ID   uuid.UUID `gorm:"column:id"`
		Name string    `gorm:"column:name"`
	}
	if err := r.db.WithContext(ctx).
		Table("print_profiles").
		Select("id, name").
		Where("id IN ? AND organization_id = ? AND deleted_at IS NULL", ids, organizationID).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load profile names: %w", err)
	}
	for _, row := range rows {
		out[row.ID] = row.Name
	}
	return out, nil
}

// uniqueIDs returns the distinct, non-nil UUIDs from ids, preserving first-seen order.
func uniqueIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
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
		"filament_cost":        item.FilamentCost,
		"waste_cost":           item.WasteCost,
		"energy_cost":          item.EnergyCost,
		"machine_cost":         item.MachineCost,
		"setup_cost":           item.SetupCost,
		"manual_labor_cost":    item.ManualLaborCost,
		"post_processing_cost": item.PostProcessingCost,
		"support_removal_cost": item.SupportRemovalCost,
		"packaging_cost":       item.PackagingCost,
		"quality_control_cost": item.QualityControlCost,
		"failure_cost":         item.FailureCost,
		"item_total_cost":      item.ItemTotalCost,
		"unit_price":           item.UnitPrice,
		"updated_at":           time.Now(),
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

	entitiesOut := make([]*entities.BudgetItemEntity, len(items))
	for i, model := range items {
		entitiesOut[i] = model.ToEntity()
	}

	return entitiesOut, nil
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

	entitiesOut := make([]*entities.BudgetItemFilamentEntity, len(filaments))
	for i, f := range filaments {
		entitiesOut[i] = f.ToEntity()
	}

	return entitiesOut, nil
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

// GetStatusHistory returns a page of a budget's status history (newest first) plus
// the total row count, so the handler can build a paginated envelope.
func (r *budgetRepositoryImpl) GetStatusHistory(ctx context.Context, budgetID uuid.UUID, limit, offset int) ([]entities.BudgetStatusHistoryEntity, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&models.BudgetStatusHistoryModel{}).
		Where("budget_id = ?", budgetID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count status history: %w", err)
	}

	var history []*models.BudgetStatusHistoryModel
	if err := r.db.WithContext(ctx).
		Where("budget_id = ?", budgetID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&history).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get status history: %w", err)
	}

	out := make([]entities.BudgetStatusHistoryEntity, len(history))
	for i, model := range history {
		out[i] = *model.ToEntity()
	}

	return out, total, nil
}

// CalculateCosts recomputes and persists all costs for a stored budget.
//
// It loads everything org-scoped (the budget itself, its items, every item's
// filaments in ONE batched query and each distinct preset once), delegates the
// arithmetic to the pure pricing engine via ComputeBudgetPricing, and then
// persists the per-item cost columns and the budget totals using explicit column
// maps inside the caller's transaction. All monetary results are in CENTS.
//
// The initial budget fetch is scoped by organization (defense-in-depth against a
// cross-tenant budget ID). Historical consistency for presets/filaments is handled
// inside ComputeBudgetPricing (references are resolved by (id, organization) so a
// since-soft-deleted preset referenced by this budget is still usable).
func (r *budgetRepositoryImpl) CalculateCosts(ctx context.Context, budgetID uuid.UUID, organizationID string) error {
	var budget models.BudgetModel
	if err := r.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		First(&budget, "id = ?", budgetID).Error; err != nil {
		return fmt.Errorf("failed to get budget: %w", err)
	}

	items, err := r.GetItems(ctx, budgetID)
	if err != nil {
		return err
	}

	// Batch-load every item's filaments (grams) in ONE query, grouped by item.
	gramsByItem, err := r.loadItemFilamentSpecs(ctx, items, organizationID)
	if err != nil {
		return err
	}

	specs := make([]entities.PricingItemSpec, len(items))
	for i, item := range items {
		specs[i] = entities.PricingItemSpec{
			ProductQuantity:         item.ProductQuantity,
			PrintTimeHours:          item.PrintTimeHours,
			PrintTimeMinutes:        item.PrintTimeMinutes,
			SetupTimeMinutes:        item.SetupTimeMinutes,
			ManualLaborMinutesTotal: item.ManualLaborMinutesTotal,
			PostProcessingMinutes:   item.PostProcessingMinutes,
			SupportRemovalMinutes:   item.SupportRemovalMinutes,
			CostPresetID:            item.CostPresetID,
			Filaments:               gramsByItem[item.ID],
		}
	}

	result, err := r.ComputeBudgetPricing(ctx, entities.PricingComputationInput{
		OrganizationID:        organizationID,
		IncludeEnergyCost:     budget.IncludeEnergyCost,
		IncludeWasteCost:      budget.IncludeWasteCost,
		IncludeMachineCost:    budget.IncludeMachineCost,
		MachinePresetID:       budget.MachinePresetID,
		EnergyPresetID:        budget.EnergyPresetID,
		BudgetCostPresetID:    budget.CostPresetID,
		DiscountType:          strValue(budget.DiscountType),
		DiscountValue:         floatValue(budget.DiscountValue),
		IncludeShipping:       budget.IncludeShipping,
		ShippingOverrideCents: budget.ShippingOverride,
		TaxRate:               budget.TaxRate,
		Items:                 specs,
	})
	if err != nil {
		return err
	}

	// Persist per-item costs (UpdateItem uses an explicit column set so zero-valued
	// costs are written back).
	for i, item := range items {
		res := result.Items[i]
		item.FilamentCost = res.FilamentCost
		item.WasteCost = res.WasteCost
		item.EnergyCost = res.EnergyCost
		item.MachineCost = res.MachineCost
		item.SetupCost = res.SetupCost
		item.ManualLaborCost = res.ManualLaborCost
		item.PostProcessingCost = res.PostProcessingCost
		item.SupportRemovalCost = res.SupportRemovalCost
		item.PackagingCost = res.PackagingCost
		item.QualityControlCost = res.QualityControlCost
		item.FailureCost = res.FailureCost
		item.ItemTotalCost = res.ItemTotalCost
		item.UnitPrice = res.UnitCost
		if err := r.UpdateItem(ctx, item); err != nil {
			return fmt.Errorf("failed to update item costs: %w", err)
		}
	}

	// Persist budget totals (explicit column map, org-scoped).
	if err := r.db.WithContext(ctx).
		Model(&models.BudgetModel{}).
		Where("id = ? AND organization_id = ?", budgetID, organizationID).
		Updates(map[string]interface{}{
			"filament_cost":        result.FilamentCost,
			"waste_cost":           result.WasteCost,
			"energy_cost":          result.EnergyCost,
			"machine_cost":         result.MachineCost,
			"setup_cost":           result.SetupCost,
			"labor_cost":           result.LaborCost,
			"post_processing_cost": result.PostProcessingCost,
			"packaging_cost":       result.PackagingCost,
			"quality_control_cost": result.QualityControlCost,
			"failure_cost":         result.FailureCost,
			"overhead_cost":        result.Overhead,
			"profit_amount":        result.Profit,
			"discount_amount":      result.DiscountAmount,
			"shipping_cost":        result.ShippingCost,
			"tax_amount":           result.TaxAmount,
			"tax_rate_applied":     result.TaxRateApplied,
			"total_cost":           result.Total,
		}).Error; err != nil {
		return fmt.Errorf("failed to update budget costs: %w", err)
	}

	return nil
}

// costPresetRates carries every rate loaded from a cost preset that the pricing
// engine needs: the per-item CostPresetInput plus the budget-level shipping rates
// (the shipping rates are only consumed from the BUDGET cost preset).
type costPresetRates struct {
	preset          pricing.CostPresetInput
	shippingBase    float64
	shippingPerGram float64
}

// ComputeBudgetPricing loads the org-scoped rates for the given input (filament
// prices in ONE batched query, distinct cost presets in ONE batched query, the
// machine/energy presets and the company default tax rate) and runs the pure
// pricing engine WITHOUT persisting anything. It is shared by CalculateCosts (which
// persists afterwards) and the stateless preview endpoint.
//
// All lookups are scoped by organization so cross-tenant references cannot leak in;
// a referenced filament or cost preset that does not resolve inside the org is an
// error rather than a silent zero. Presets/filaments are resolved WITHOUT a
// deleted_at filter for historical consistency with already-saved budgets.
func (r *budgetRepositoryImpl) ComputeBudgetPricing(ctx context.Context, in entities.PricingComputationInput) (pricing.PricingResult, error) {
	var zero pricing.PricingResult

	priceByFilament, err := r.loadFilamentPrices(ctx, in.Items, in.OrganizationID)
	if err != nil {
		return zero, err
	}

	costPresetByID, err := r.loadCostPresets(ctx, in)
	if err != nil {
		return zero, err
	}

	// Energy inputs are budget-level and constant across items; load them once.
	energyEnabled := in.IncludeEnergyCost && in.MachinePresetID != nil && in.EnergyPresetID != nil
	var powerWatts, energyPrice float64
	if energyEnabled {
		res := r.db.WithContext(ctx).
			Table("machine_presets").
			Select("power_consumption").
			Where("id = ? AND organization_id = ?", *in.MachinePresetID, in.OrganizationID).
			Scan(&powerWatts)
		if res.Error != nil {
			return zero, fmt.Errorf("failed to load machine power consumption: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return zero, fmt.Errorf("%w: machine preset %s not found for organization", entities.ErrInvalidPresetReference, *in.MachinePresetID)
		}

		res = r.db.WithContext(ctx).
			Table("energy_presets").
			Select("energy_cost_per_kwh").
			Where("id = ? AND organization_id = ?", *in.EnergyPresetID, in.OrganizationID).
			Scan(&energyPrice)
		if res.Error != nil {
			return zero, fmt.Errorf("failed to load energy price: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return zero, fmt.Errorf("%w: energy preset %s not found for organization", entities.ErrInvalidPresetReference, *in.EnergyPresetID)
		}
	}

	// Machine cost per hour is only needed when the budget charges machine time.
	var machineCostPerHour float64
	if in.IncludeMachineCost && in.MachinePresetID != nil {
		res := r.db.WithContext(ctx).
			Table("machine_presets").
			Select("cost_per_hour").
			Where("id = ? AND organization_id = ?", *in.MachinePresetID, in.OrganizationID).
			Scan(&machineCostPerHour)
		if res.Error != nil {
			return zero, fmt.Errorf("failed to load machine cost per hour: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return zero, fmt.Errorf("%w: machine preset %s not found for organization", entities.ErrInvalidPresetReference, *in.MachinePresetID)
		}
	}

	// Resolve the effective tax rate: the budget-level rate when provided, else the
	// company default_tax_rate (0 when the company has no default or no row).
	taxRate, err := r.resolveTaxRate(ctx, in)
	if err != nil {
		return zero, err
	}

	pin := pricing.PricingInput{
		IncludeWasteCost:      in.IncludeWasteCost,
		EnergyEnabled:         energyEnabled,
		MachinePowerWatts:     powerWatts,
		EnergyPricePerKwh:     energyPrice,
		IncludeMachineCost:    in.IncludeMachineCost,
		MachineCostPerHour:    machineCostPerHour,
		DiscountType:          in.DiscountType,
		DiscountValue:         in.DiscountValue,
		IncludeShipping:       in.IncludeShipping,
		ShippingOverrideCents: in.ShippingOverrideCents,
		TaxRatePercent:        taxRate,
	}
	if in.BudgetCostPresetID != nil {
		rates := costPresetByID[*in.BudgetCostPresetID]
		cp := rates.preset
		pin.BudgetCostPreset = &cp
		pin.ShippingCostBase = rates.shippingBase
		pin.ShippingCostPerGram = rates.shippingPerGram
	}
	pin.Items = make([]pricing.PricingItemInput, len(in.Items))
	for i, item := range in.Items {
		filaments := make([]pricing.PricingFilamentInput, len(item.Filaments))
		for j, f := range item.Filaments {
			filaments[j] = pricing.PricingFilamentInput{
				Grams:           f.Quantity,
				PricePerKgCents: priceByFilament[f.FilamentID],
			}
		}
		// Item cost preset fallback: when an item has no cost preset of its own, it
		// uses the budget-level cost preset (resolved from the request/profile/org
		// defaults) for its setup + manual labor rates. This is applied here, in the
		// pricing input mapping, so stored items are never mutated — an item row with
		// a NULL cost_preset_id keeps that NULL while still being priced with the
		// budget-level labor rate.
		var cp *pricing.CostPresetInput
		effectiveCostPresetID := item.CostPresetID
		if effectiveCostPresetID == nil {
			effectiveCostPresetID = in.BudgetCostPresetID
		}
		if effectiveCostPresetID != nil {
			v := costPresetByID[*effectiveCostPresetID].preset
			cp = &v
		}
		pin.Items[i] = pricing.PricingItemInput{
			Quantity:                item.ProductQuantity,
			PrintTimeHours:          item.PrintTimeHours,
			PrintTimeMinutes:        item.PrintTimeMinutes,
			SetupTimeMinutes:        item.SetupTimeMinutes,
			ManualLaborMinutesTotal: item.ManualLaborMinutesTotal,
			PostProcessingMinutes:   item.PostProcessingMinutes,
			SupportRemovalMinutes:   item.SupportRemovalMinutes,
			CostPreset:              cp,
			Filaments:               filaments,
		}
	}

	return pricing.Calculate(pin)
}

// resolveTaxRate returns the effective "por dentro" tax rate for the computation:
// the budget-level rate when provided, otherwise the organization's company default
// (0 when the company has no default or no row). Scoped by organization.
func (r *budgetRepositoryImpl) resolveTaxRate(ctx context.Context, in entities.PricingComputationInput) (float64, error) {
	if in.TaxRate != nil {
		return *in.TaxRate, nil
	}
	var rate float64
	res := r.db.WithContext(ctx).
		Table("companies").
		Select("default_tax_rate").
		Where("organization_id = ? AND deleted_at IS NULL", in.OrganizationID).
		Scan(&rate)
	if res.Error != nil {
		return 0, fmt.Errorf("failed to load company default tax rate: %w", res.Error)
	}
	return rate, nil
}

// loadItemFilamentSpecs batch-loads the filaments (grams) of every given item in a
// single org-scoped query, grouped by item ID and ordered by the color-change order.
func (r *budgetRepositoryImpl) loadItemFilamentSpecs(ctx context.Context, items []*entities.BudgetItemEntity, organizationID string) (map[uuid.UUID][]entities.PricingFilamentSpec, error) {
	out := make(map[uuid.UUID][]entities.PricingFilamentSpec, len(items))
	if len(items) == 0 {
		return out, nil
	}

	itemIDs := make([]uuid.UUID, len(items))
	for i, it := range items {
		itemIDs[i] = it.ID
	}

	var rows []struct {
		BudgetItemID uuid.UUID
		FilamentID   uuid.UUID
		Quantity     float64
	}
	if err := r.db.WithContext(ctx).
		Table("budget_item_filaments").
		Select("budget_item_id, filament_id, quantity").
		Where("budget_item_id IN ? AND organization_id = ?", itemIDs, organizationID).
		Order("\"order\" ASC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load item filaments: %w", err)
	}

	for _, row := range rows {
		out[row.BudgetItemID] = append(out[row.BudgetItemID], entities.PricingFilamentSpec{
			FilamentID: row.FilamentID,
			Quantity:   row.Quantity,
		})
	}
	return out, nil
}

// loadFilamentPrices batch-loads the price (CENTS/kg) of every distinct filament
// referenced by the items in a single org-scoped query. A referenced filament that
// does not resolve inside the organization is an error (not a silent zero cost).
func (r *budgetRepositoryImpl) loadFilamentPrices(ctx context.Context, items []entities.PricingItemSpec, organizationID string) (map[uuid.UUID]float64, error) {
	seen := make(map[uuid.UUID]struct{})
	ids := make([]uuid.UUID, 0)
	for _, item := range items {
		for _, f := range item.Filaments {
			if _, ok := seen[f.FilamentID]; ok {
				continue
			}
			seen[f.FilamentID] = struct{}{}
			ids = append(ids, f.FilamentID)
		}
	}

	out := make(map[uuid.UUID]float64, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	var rows []struct {
		ID         uuid.UUID
		PricePerKg float64
	}
	if err := r.db.WithContext(ctx).
		Table("filaments").
		Select("id, price_per_kg").
		Where("id IN ? AND organization_id = ?", ids, organizationID).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load filament prices: %w", err)
	}
	for _, row := range rows {
		out[row.ID] = row.PricePerKg
	}
	if len(out) != len(ids) {
		return nil, fmt.Errorf("%w: one or more referenced filaments do not belong to your organization", entities.ErrFilamentNotFound)
	}
	return out, nil
}

// loadCostPresets batch-loads every rate of every distinct cost preset referenced by
// the input (budget-level + each item) in a single org-scoped query. A referenced
// preset that does not resolve inside the organization is an error (not a silent
// zero). The returned map carries the full per-item rate set plus the budget-level
// shipping rates.
func (r *budgetRepositoryImpl) loadCostPresets(ctx context.Context, in entities.PricingComputationInput) (map[uuid.UUID]costPresetRates, error) {
	seen := make(map[uuid.UUID]struct{})
	ids := make([]uuid.UUID, 0)
	add := func(id *uuid.UUID) {
		if id == nil {
			return
		}
		if _, ok := seen[*id]; ok {
			return
		}
		seen[*id] = struct{}{}
		ids = append(ids, *id)
	}
	add(in.BudgetCostPresetID)
	for _, item := range in.Items {
		add(item.CostPresetID)
	}

	out := make(map[uuid.UUID]costPresetRates, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	var rows []struct {
		ID                        uuid.UUID
		LaborCostPerHour          float64
		OverheadPercentage        float64
		ProfitMarginPercentage    float64
		PostProcessingCostPerHour float64
		SupportRemovalCostPerHour float64
		PackagingCostPerItem      float64
		QualityControlCostPerItem float64
		FailureRatePercentage     float64
		WasteGramsPerColorChange  float64
		ShippingCostBase          float64
		ShippingCostPerGram       float64
	}
	if err := r.db.WithContext(ctx).
		Table("cost_presets").
		Select(`id, labor_cost_per_hour, overhead_percentage, profit_margin_percentage,
			post_processing_cost_per_hour, support_removal_cost_per_hour,
			packaging_cost_per_item, quality_control_cost_per_item,
			failure_rate_percentage, waste_grams_per_color_change,
			shipping_cost_base, shipping_cost_per_gram`).
		Where("id IN ? AND organization_id = ?", ids, in.OrganizationID).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load cost presets: %w", err)
	}
	for _, row := range rows {
		out[row.ID] = costPresetRates{
			preset: pricing.CostPresetInput{
				LaborRatePerHour:          row.LaborCostPerHour,
				OverheadPercentage:        row.OverheadPercentage,
				ProfitMarginPercentage:    row.ProfitMarginPercentage,
				PostProcessingCostPerHour: row.PostProcessingCostPerHour,
				SupportRemovalCostPerHour: row.SupportRemovalCostPerHour,
				PackagingCostPerItem:      row.PackagingCostPerItem,
				QualityControlCostPerItem: row.QualityControlCostPerItem,
				FailureRatePercentage:     row.FailureRatePercentage,
				WasteGramsPerColorChange:  row.WasteGramsPerColorChange,
			},
			shippingBase:    row.ShippingCostBase,
			shippingPerGram: row.ShippingCostPerGram,
		}
	}
	if len(out) != len(ids) {
		return nil, fmt.Errorf("%w: one or more referenced cost presets do not belong to your organization", entities.ErrInvalidPresetReference)
	}
	return out, nil
}

// strValue dereferences a *string, returning "" when nil.
func strValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// floatValue dereferences a *float64, returning 0 when nil.
func floatValue(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
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
		return fmt.Errorf("%w: one or more referenced filaments do not belong to your organization", entities.ErrFilamentNotFound)
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
		return fmt.Errorf("%w: referenced %s preset %s does not belong to your organization", entities.ErrInvalidPresetReference, presetType, presetID)
	}

	return nil
}

// ValidateModel3DsInOrg ensures every referenced 3D model belongs to the given
// organization. Models are strictly tenant-scoped (models_3d.organization_id is
// NOT NULL), so any model outside the caller's organization is rejected. Only live
// rows (deleted_at IS NULL) count, since this validates freshly provided references.
func (r *budgetRepositoryImpl) ValidateModel3DsInOrg(ctx context.Context, model3dIDs []uuid.UUID, organizationID string) error {
	if len(model3dIDs) == 0 {
		return nil
	}

	seen := make(map[uuid.UUID]struct{}, len(model3dIDs))
	unique := make([]uuid.UUID, 0, len(model3dIDs))
	for _, id := range model3dIDs {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return nil
	}

	var count int64
	if err := r.db.WithContext(ctx).
		Table("models_3d").
		Where("id IN ? AND organization_id = ? AND deleted_at IS NULL", unique, organizationID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate 3D models: %w", err)
	}

	if int(count) != len(unique) {
		return fmt.Errorf("%w: one or more referenced 3D models do not belong to your organization", entities.ErrInvalidModel3DReference)
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

	entitiesOut := make([]*entities.BudgetItemEntity, 0, len(items))
	for _, item := range items {
		entitiesOut = append(entitiesOut, item.ToEntity())
	}

	return entitiesOut, nil
}
