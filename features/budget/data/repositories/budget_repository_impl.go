package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/data/models"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
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

	if err := query.First(model, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("budget not found: %w", err)
	}

	return model.ToEntity(), nil
}

func (r *budgetRepositoryImpl) Update(ctx context.Context, budget *entities.BudgetEntity) error {
	model := &models.BudgetModel{}
	model.FromEntity(budget)

	// Use Updates instead of Save to avoid issues with zero values
	if err := r.db.WithContext(ctx).
		Model(&models.BudgetModel{}).
		Where("id = ?", budget.ID).
		Updates(model).Error; err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
	}

	return nil
}

func (r *budgetRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.BudgetModel{}, "id = ?", id).Error; err != nil {
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

	// Get paginated results
	if err := query.
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

func (r *budgetRepositoryImpl) UpdateItem(ctx context.Context, item *entities.BudgetItemEntity) error {
	model := &models.BudgetItemModel{}
	model.FromEntity(item)

	if err := r.db.WithContext(ctx).
		Model(&models.BudgetItemModel{}).
		Where("id = ?", item.ID).
		Updates(model).Error; err != nil {
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

func (r *budgetRepositoryImpl) DeleteAllItems(ctx context.Context, budgetID uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Where("budget_id = ?", budgetID).
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

// GetFilamentUsageInfo retrieves detailed filament usage info for a budget item
func (r *budgetRepositoryImpl) GetFilamentUsageInfo(ctx context.Context, itemID uuid.UUID) ([]entities.FilamentUsageInfo, error) {
	// Get filaments with all related info via JOIN
	var results []struct {
		FilamentID   uuid.UUID
		Quantity     float64
		Order        int
		FilamentName string
		BrandName    string
		MaterialName string
		Color        string
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
			f.price_per_kg
		`).
		Joins("JOIN filaments f ON f.id = bif.filament_id").
		Joins("JOIN brands b ON b.id = f.brand_id").
		Joins("JOIN materials m ON m.id = f.material_id").
		Where("bif.budget_item_id = ?", itemID).
		Order("bif.\"order\" ASC").
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get filament usage info: %w", err)
	}

	infos := make([]entities.FilamentUsageInfo, len(results))
	for i, r := range results {
		// Quantity já é o total, não precisa multiplicar!
		cost := int64((r.Quantity / 1000.0) * r.PricePerKg * 100) // to cents

		infos[i] = entities.FilamentUsageInfo{
			FilamentID:   r.FilamentID.String(),
			FilamentName: r.FilamentName,
			BrandName:    r.BrandName,
			MaterialName: r.MaterialName,
			Color:        r.Color,
			Quantity:     r.Quantity,
			Cost:         cost,
			Order:        r.Order,
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

// CalculateCosts calculates all costs for a budget (REFACTORED for multi-filament support)
func (r *budgetRepositoryImpl) CalculateCosts(ctx context.Context, budgetID uuid.UUID) error {
	// Get budget
	var budget models.BudgetModel
	if err := r.db.WithContext(ctx).First(&budget, "id = ?", budgetID).Error; err != nil {
		return fmt.Errorf("failed to get budget: %w", err)
	}

	// Get items
	items, err := r.GetItems(ctx, budgetID)
	if err != nil {
		return err
	}

	// Get machine and energy presets (global)
	var machinePreset *entities.PresetInfo
	var energyPreset *entities.PresetInfo

	if budget.MachinePresetID != nil {
		machinePreset, _ = r.GetPresetInfo(ctx, *budget.MachinePresetID, "machine")
	}
	if budget.EnergyPresetID != nil {
		energyPreset, _ = r.GetPresetInfo(ctx, *budget.EnergyPresetID, "energy")
	}

	var totalFilamentCost, totalWasteCost, totalEnergyCost, totalLaborCost int64

	// Process each item (product)
	for _, item := range items {
		var itemFilamentCost, itemWasteCost, itemEnergyCost, itemLaborCost int64

		// 1. Get filaments for this item
		itemFilaments, err := r.GetItemFilaments(ctx, item.ID)
		if err != nil {
			return fmt.Errorf("failed to get item filaments: %w", err)
		}

		// 2. Calculate filament cost (sum of all filaments in this item)
		var totalGrams float64
		var avgPrice float64
		var totalPrice float64

		for _, itemFil := range itemFilaments {
			filament, err := r.GetFilamentInfo(ctx, itemFil.FilamentID)
			if err != nil {
				continue
			}

			// Quantity já é o total (não precisa multiplicar por ProductQuantity!)
			gramsTotal := itemFil.Quantity
			cost := (gramsTotal / 1000.0) * filament.PricePerKg * 100 // to cents
			itemFilamentCost += int64(cost)
			totalGrams += gramsTotal
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
			itemWasteCost = int64((totalWaste / 1000.0) * avgPrice * 100)
		}

		// 4. Calculate energy cost (proportional to this item's print time)
		if budget.IncludeEnergyCost && machinePreset != nil && energyPreset != nil {
			// Get power consumption and energy price from presets
			var powerConsumption, energyPrice float64

			r.db.WithContext(ctx).
				Table("presets").
				Select("CAST(value AS FLOAT) as price").
				Where("id = ? AND key = 'price_per_kwh'", energyPreset.ID).
				Scan(&energyPrice)

			r.db.WithContext(ctx).
				Table("presets").
				Select("CAST(value AS FLOAT) as power").
				Where("id = ? AND key = 'power_consumption'", machinePreset.ID).
				Scan(&powerConsumption)

			itemHours := float64(item.PrintTimeHours) + float64(item.PrintTimeMinutes)/60.0
			kwh := powerConsumption * itemHours / 1000.0
			itemEnergyCost = int64(kwh * energyPrice * 100)
		}

		// 5. Calculate labor cost (base + additional)
		itemHours := float64(item.PrintTimeHours) + float64(item.PrintTimeMinutes)/60.0

		// Base labor cost from preset
		var laborRate float64
		if item.CostPresetID != nil {
			costPreset, err := r.GetPresetInfo(ctx, *item.CostPresetID, "cost")
			if err == nil {
				r.db.WithContext(ctx).
					Table("presets").
					Select("CAST(value AS FLOAT) as rate").
					Where("id = ? AND key = 'labor_cost_per_hour'", costPreset.ID).
					Scan(&laborRate)
			}
		}

		itemLaborCost = int64(itemHours * laborRate * 100)

		// Add additional labor cost (pintura, acabamento, etc)
		if item.AdditionalLaborCost != nil {
			itemLaborCost += *item.AdditionalLaborCost
		}

		// 6. Calculate item total cost
		item.FilamentCost = itemFilamentCost
		item.WasteCost = itemWasteCost
		item.EnergyCost = itemEnergyCost
		item.LaborCost = itemLaborCost
		item.ItemTotalCost = itemFilamentCost + itemWasteCost + itemEnergyCost + itemLaborCost

		// 7. Calculate unit price
		if item.ProductQuantity > 0 {
			item.UnitPrice = item.ItemTotalCost / int64(item.ProductQuantity)
		}

		// 8. Update item in database
		if err := r.UpdateItem(ctx, item); err != nil {
			return fmt.Errorf("failed to update item costs: %w", err)
		}

		// 9. Sum to budget totals
		totalFilamentCost += itemFilamentCost
		totalWasteCost += itemWasteCost
		totalEnergyCost += itemEnergyCost
		totalLaborCost += itemLaborCost
	}

	// Update budget totals
	budget.FilamentCost = totalFilamentCost
	budget.WasteCost = totalWasteCost
	budget.EnergyCost = totalEnergyCost
	budget.LaborCost = totalLaborCost
	budget.TotalCost = totalFilamentCost + totalWasteCost + totalEnergyCost + totalLaborCost

	if err := r.db.WithContext(ctx).
		Model(&models.BudgetModel{}).
		Where("id = ?", budgetID).
		Updates(map[string]interface{}{
			"filament_cost": budget.FilamentCost,
			"waste_cost":    budget.WasteCost,
			"energy_cost":   budget.EnergyCost,
			"labor_cost":    budget.LaborCost,
			"total_cost":    budget.TotalCost,
		}).Error; err != nil {
		return fmt.Errorf("failed to update budget costs: %w", err)
	}

	return nil
}

// GetCustomerInfo fetches customer information by ID
func (r *budgetRepositoryImpl) GetCustomerInfo(ctx context.Context, customerID uuid.UUID) (*entities.CustomerInfo, error) {
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
		Where("id = ?", customerID).
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

// GetFilamentInfo fetches filament information by ID
func (r *budgetRepositoryImpl) GetFilamentInfo(ctx context.Context, filamentID uuid.UUID) (*entities.FilamentInfo, error) {
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
		Where("id = ?", filamentID).
		First(&filament).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch filament: %w", err)
	}

	// Get brand name
	var brandName string
	r.db.WithContext(ctx).
		Table("brands").
		Select("name").
		Where("id = ?", filament.BrandID).
		Scan(&brandName)

	// Get material name
	var materialName string
	r.db.WithContext(ctx).
		Table("materials").
		Select("name").
		Where("id = ?", filament.MaterialID).
		Scan(&materialName)

	return &entities.FilamentInfo{
		ID:           filament.ID.String(),
		Name:         filament.Name,
		BrandName:    brandName,
		MaterialName: materialName,
		Color:        filament.Color,
		PricePerKg:   filament.PricePerKg,
	}, nil
}

// GetPresetInfo fetches preset information by ID
func (r *budgetRepositoryImpl) GetPresetInfo(ctx context.Context, presetID uuid.UUID, presetType string) (*entities.PresetInfo, error) {
	var preset struct {
		ID   uuid.UUID `gorm:"column:id"`
		Name string    `gorm:"column:name"`
		Type string    `gorm:"column:type"`
	}

	table := "presets"
	if presetType == "machine" {
		table = "machine_presets"
	} else if presetType == "energy" {
		table = "energy_presets"
	} else if presetType == "cost" {
		table = "cost_presets"
	}

	if err := r.db.WithContext(ctx).
		Table(table).
		Select("id, name, type").
		Where("id = ?", presetID).
		First(&preset).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch preset: %w", err)
	}

	return &entities.PresetInfo{
		ID:   preset.ID.String(),
		Name: preset.Name,
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
