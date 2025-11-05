package repositories

import (
	"github.com/RodolfoBonis/spooliq/features/preset/data/models"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PresetRepositoryImpl implements the PresetRepository interface
type PresetRepositoryImpl struct {
	db *gorm.DB
}

// NewPresetRepository creates a new instance of PresetRepositoryImpl
func NewPresetRepository(db *gorm.DB) repositories.PresetRepository {
	return &PresetRepositoryImpl{db: db}
}

// Create creates a new preset
func (r *PresetRepositoryImpl) Create(preset *entities.PresetEntity) error {
	model := &models.PresetModel{}
	model.FromEntity(preset)

	return r.db.Create(model).Error
}

// GetByID retrieves a preset by its ID
func (r *PresetRepositoryImpl) GetByID(id uuid.UUID) (*entities.PresetEntity, error) {
	var model models.PresetModel

	err := r.db.
		Where("id = ?", id).
		First(&model).Error
	if err != nil {
		return nil, err
	}

	entity := model.ToEntity()
	return &entity, nil
}

// GetByType retrieves presets by type
func (r *PresetRepositoryImpl) GetByType(presetType entities.PresetType) ([]*entities.PresetEntity, error) {
	var models []models.PresetModel

	err := r.db.
		// For list views, we don't preload relationships to keep queries lightweight
		Where("type = ?", string(presetType)).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	var entities []*entities.PresetEntity
	for _, model := range models {
		entity := model.ToEntity()
		entities = append(entities, &entity)
	}

	return entities, nil
}

// GetByUserID retrieves presets by user ID
func (r *PresetRepositoryImpl) GetByUserID(userID uuid.UUID) ([]*entities.PresetEntity, error) {
	var models []models.PresetModel

	err := r.db.
		// For list views, we don't preload relationships to keep queries lightweight
		Where("user_id = ?", userID).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	var entities []*entities.PresetEntity
	for _, model := range models {
		entity := model.ToEntity()
		entities = append(entities, &entity)
	}

	return entities, nil
}

// GetGlobalPresets retrieves global presets (user_id is null)
func (r *PresetRepositoryImpl) GetGlobalPresets() ([]*entities.PresetEntity, error) {
	var models []models.PresetModel

	err := r.db.Where("user_id IS NULL").Find(&models).Error
	if err != nil {
		return nil, err
	}

	var entities []*entities.PresetEntity
	for _, model := range models {
		entity := model.ToEntity()
		entities = append(entities, &entity)
	}

	return entities, nil
}

// GetActivePresets retrieves active presets
func (r *PresetRepositoryImpl) GetActivePresets() ([]*entities.PresetEntity, error) {
	var models []models.PresetModel

	err := r.db.Where("is_active = ?", true).Find(&models).Error
	if err != nil {
		return nil, err
	}

	var entities []*entities.PresetEntity
	for _, model := range models {
		entity := model.ToEntity()
		entities = append(entities, &entity)
	}

	return entities, nil
}

// GetDefaultPresets retrieves default presets
func (r *PresetRepositoryImpl) GetDefaultPresets() ([]*entities.PresetEntity, error) {
	var models []models.PresetModel

	err := r.db.Where("is_default = ?", true).Find(&models).Error
	if err != nil {
		return nil, err
	}

	var entities []*entities.PresetEntity
	for _, model := range models {
		entity := model.ToEntity()
		entities = append(entities, &entity)
	}

	return entities, nil
}

// Update updates an existing preset
func (r *PresetRepositoryImpl) Update(preset *entities.PresetEntity) error {
	model := &models.PresetModel{}
	model.FromEntity(preset)

	return r.db.Save(model).Error
}

// Delete soft deletes a preset
func (r *PresetRepositoryImpl) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.PresetModel{}).Error
}

// CreateMachine creates a new machine preset with base preset
func (r *PresetRepositoryImpl) CreateMachine(preset *entities.PresetEntity, machine *entities.MachinePresetEntity) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create base preset
		presetModel := &models.PresetModel{}
		presetModel.FromEntity(preset)
		if err := tx.Create(presetModel).Error; err != nil {
			return err
		}

		// Create machine-specific data
		machineModel := &models.MachinePresetModel{}
		machineModel.FromEntity(machine)
		machineModel.ID = presetModel.ID // Use same ID as base preset

		return tx.Create(machineModel).Error
	})
}

// GetMachineByID retrieves a machine preset by ID
func (r *PresetRepositoryImpl) GetMachineByID(id uuid.UUID) (*entities.MachinePresetEntity, error) {
	var model models.MachinePresetModel

	err := r.db.
		Where("id = ?", id).
		First(&model).Error
	if err != nil {
		return nil, err
	}

	entity := model.ToEntity()
	return &entity, nil
}

// GetMachinesByBrand retrieves machine presets by brand
func (r *PresetRepositoryImpl) GetMachinesByBrand(brand string) ([]*entities.MachinePresetEntity, error) {
	var models []models.MachinePresetModel

	err := r.db.Where("brand = ?", brand).Find(&models).Error
	if err != nil {
		return nil, err
	}

	var entities []*entities.MachinePresetEntity
	for _, model := range models {
		entity := model.ToEntity()
		entities = append(entities, &entity)
	}

	return entities, nil
}

// UpdateMachine updates a machine preset
func (r *PresetRepositoryImpl) UpdateMachine(machine *entities.MachinePresetEntity) error {
	model := &models.MachinePresetModel{}
	model.FromEntity(machine)

	return r.db.Save(model).Error
}

// CreateEnergy creates a new energy preset with base preset
func (r *PresetRepositoryImpl) CreateEnergy(preset *entities.PresetEntity, energy *entities.EnergyPresetEntity) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create base preset
		presetModel := &models.PresetModel{}
		presetModel.FromEntity(preset)
		if err := tx.Create(presetModel).Error; err != nil {
			return err
		}

		// Create energy-specific data
		energyModel := &models.EnergyPresetModel{}
		energyModel.FromEntity(energy)
		energyModel.ID = presetModel.ID // Use same ID as base preset

		return tx.Create(energyModel).Error
	})
}

// GetEnergyByID retrieves an energy preset by ID
func (r *PresetRepositoryImpl) GetEnergyByID(id uuid.UUID) (*entities.EnergyPresetEntity, error) {
	var model models.EnergyPresetModel

	err := r.db.
		Where("id = ?", id).
		First(&model).Error
	if err != nil {
		return nil, err
	}

	entity := model.ToEntity()
	return &entity, nil
}

// GetEnergyByLocation retrieves energy presets by location
func (r *PresetRepositoryImpl) GetEnergyByLocation(country, state, city string) ([]*entities.EnergyPresetEntity, error) {
	var models []models.EnergyPresetModel
	query := r.db

	if country != "" {
		query = query.Where("country = ?", country)
	}
	if state != "" {
		query = query.Where("state = ?", state)
	}
	if city != "" {
		query = query.Where("city = ?", city)
	}

	err := query.Find(&models).Error
	if err != nil {
		return nil, err
	}

	var entities []*entities.EnergyPresetEntity
	for _, model := range models {
		entity := model.ToEntity()
		entities = append(entities, &entity)
	}

	return entities, nil
}

// GetEnergyByCurrency retrieves energy presets by currency
func (r *PresetRepositoryImpl) GetEnergyByCurrency(currency string) ([]*entities.EnergyPresetEntity, error) {
	var models []models.EnergyPresetModel

	err := r.db.Where("currency = ?", currency).Find(&models).Error
	if err != nil {
		return nil, err
	}

	var entities []*entities.EnergyPresetEntity
	for _, model := range models {
		entity := model.ToEntity()
		entities = append(entities, &entity)
	}

	return entities, nil
}

// UpdateEnergy updates an energy preset
func (r *PresetRepositoryImpl) UpdateEnergy(energy *entities.EnergyPresetEntity) error {
	model := &models.EnergyPresetModel{}
	model.FromEntity(energy)

	return r.db.Save(model).Error
}

// CreateCost creates a new cost preset with base preset
func (r *PresetRepositoryImpl) CreateCost(preset *entities.PresetEntity, cost *entities.CostPresetEntity) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create base preset
		presetModel := &models.PresetModel{}
		presetModel.FromEntity(preset)
		if err := tx.Create(presetModel).Error; err != nil {
			return err
		}

		// Create cost-specific data
		costModel := &models.CostPresetModel{}
		costModel.FromEntity(cost)
		costModel.ID = presetModel.ID // Use same ID as base preset

		return tx.Create(costModel).Error
	})
}

// GetCostByID retrieves a cost preset by ID
func (r *PresetRepositoryImpl) GetCostByID(id uuid.UUID) (*entities.CostPresetEntity, error) {
	var model models.CostPresetModel

	err := r.db.
		Where("id = ?", id).
		First(&model).Error
	if err != nil {
		return nil, err
	}

	entity := model.ToEntity()
	return &entity, nil
}

// UpdateCost updates a cost preset
func (r *PresetRepositoryImpl) UpdateCost(cost *entities.CostPresetEntity) error {
	model := &models.CostPresetModel{}
	model.FromEntity(cost)

	return r.db.Save(model).Error
}

// OPTIMIZED METHODS WITH ORGANIZATION FILTERING AND JOINS

// GetMachinePresets retrieves all machine presets with base data in a single query
func (r *PresetRepositoryImpl) GetMachinePresets(organizationID string) ([]*repositories.MachinePresetResponse, error) {
	var results []*repositories.MachinePresetResponse

	err := r.db.Table("machine_presets").
		Select(`
			presets.id,
			presets.name,
			presets.description,
			presets.type,
			presets.is_active,
			presets.is_default,
			machine_presets.brand,
			machine_presets.model,
			machine_presets.build_volume_x,
			machine_presets.build_volume_y,
			machine_presets.build_volume_z,
			machine_presets.nozzle_diameter,
			machine_presets.layer_height_min,
			machine_presets.layer_height_max,
			machine_presets.print_speed_max,
			machine_presets.power_consumption,
			machine_presets.bed_temperature_max,
			machine_presets.extruder_temperature_max,
			machine_presets.filament_diameter,
			machine_presets.cost_per_hour
		`).
		Joins("INNER JOIN presets ON machine_presets.id = presets.id").
		Where("presets.organization_id = ? AND presets.deleted_at IS NULL AND presets.is_active = ?", organizationID, true).
		Scan(&results).Error

	return results, err
}

// GetMachinePresetsByBrand retrieves machine presets by brand with base data
func (r *PresetRepositoryImpl) GetMachinePresetsByBrand(brand, organizationID string) ([]*repositories.MachinePresetResponse, error) {
	var results []*repositories.MachinePresetResponse

	err := r.db.Table("machine_presets").
		Select(`
			presets.id,
			presets.name,
			presets.description,
			presets.type,
			presets.is_active,
			presets.is_default,
			machine_presets.brand,
			machine_presets.model,
			machine_presets.build_volume_x,
			machine_presets.build_volume_y,
			machine_presets.build_volume_z,
			machine_presets.nozzle_diameter,
			machine_presets.layer_height_min,
			machine_presets.layer_height_max,
			machine_presets.print_speed_max,
			machine_presets.power_consumption,
			machine_presets.bed_temperature_max,
			machine_presets.extruder_temperature_max,
			machine_presets.filament_diameter,
			machine_presets.cost_per_hour
		`).
		Joins("INNER JOIN presets ON machine_presets.id = presets.id").
		Where("presets.organization_id = ? AND presets.deleted_at IS NULL AND presets.is_active = ? AND machine_presets.brand = ?", organizationID, true, brand).
		Scan(&results).Error

	return results, err
}

// GetEnergyPresets retrieves all energy presets with base data in a single query
func (r *PresetRepositoryImpl) GetEnergyPresets(organizationID string) ([]*repositories.EnergyPresetResponse, error) {
	var results []*repositories.EnergyPresetResponse

	err := r.db.Table("energy_presets").
		Select(`
			presets.id,
			presets.name,
			presets.description,
			presets.type,
			presets.is_active,
			presets.is_default,
			energy_presets.country,
			energy_presets.state,
			energy_presets.city,
			energy_presets.energy_cost_per_kwh,
			energy_presets.currency,
			energy_presets.provider,
			energy_presets.tariff_type,
			energy_presets.peak_hour_multiplier,
			energy_presets.off_peak_hour_multiplier
		`).
		Joins("INNER JOIN presets ON energy_presets.id = presets.id").
		Where("presets.organization_id = ? AND presets.deleted_at IS NULL AND presets.is_active = ?", organizationID, true).
		Scan(&results).Error

	return results, err
}

// GetEnergyPresetsByLocation retrieves energy presets by location with base data
func (r *PresetRepositoryImpl) GetEnergyPresetsByLocation(country, state, city, organizationID string) ([]*repositories.EnergyPresetResponse, error) {
	var results []*repositories.EnergyPresetResponse

	query := r.db.Table("energy_presets").
		Select(`
			presets.id,
			presets.name,
			presets.description,
			presets.type,
			presets.is_active,
			presets.is_default,
			energy_presets.country,
			energy_presets.state,
			energy_presets.city,
			energy_presets.energy_cost_per_kwh,
			energy_presets.currency,
			energy_presets.provider,
			energy_presets.tariff_type,
			energy_presets.peak_hour_multiplier,
			energy_presets.off_peak_hour_multiplier
		`).
		Joins("INNER JOIN presets ON energy_presets.id = presets.id").
		Where("presets.organization_id = ? AND presets.deleted_at IS NULL AND presets.is_active = ?", organizationID, true)

	if country != "" {
		query = query.Where("energy_presets.country = ?", country)
	}
	if state != "" {
		query = query.Where("energy_presets.state = ?", state)
	}
	if city != "" {
		query = query.Where("energy_presets.city = ?", city)
	}

	err := query.Scan(&results).Error
	return results, err
}

// GetEnergyPresetsByCurrency retrieves energy presets by currency with base data
func (r *PresetRepositoryImpl) GetEnergyPresetsByCurrency(currency, organizationID string) ([]*repositories.EnergyPresetResponse, error) {
	var results []*repositories.EnergyPresetResponse

	err := r.db.Table("energy_presets").
		Select(`
			presets.id,
			presets.name,
			presets.description,
			presets.type,
			presets.is_active,
			presets.is_default,
			energy_presets.country,
			energy_presets.state,
			energy_presets.city,
			energy_presets.energy_cost_per_kwh,
			energy_presets.currency,
			energy_presets.provider,
			energy_presets.tariff_type,
			energy_presets.peak_hour_multiplier,
			energy_presets.off_peak_hour_multiplier
		`).
		Joins("INNER JOIN presets ON energy_presets.id = presets.id").
		Where("presets.organization_id = ? AND presets.deleted_at IS NULL AND presets.is_active = ? AND energy_presets.currency = ?", organizationID, true, currency).
		Scan(&results).Error

	return results, err
}

// GetCostPresets retrieves all cost presets with base data in a single query
func (r *PresetRepositoryImpl) GetCostPresets(organizationID string) ([]*repositories.CostPresetResponse, error) {
	var results []*repositories.CostPresetResponse

	err := r.db.Table("cost_presets").
		Select(`
			presets.id,
			presets.name,
			presets.description,
			presets.type,
			presets.is_active,
			presets.is_default,
			cost_presets.labor_cost_per_hour,
			cost_presets.packaging_cost_per_item,
			cost_presets.shipping_cost_base,
			cost_presets.shipping_cost_per_gram,
			cost_presets.overhead_percentage,
			cost_presets.profit_margin_percentage,
			cost_presets.post_processing_cost_per_hour,
			cost_presets.support_removal_cost_per_hour,
			cost_presets.quality_control_cost_per_item
		`).
		Joins("INNER JOIN presets ON cost_presets.id = presets.id").
		Where("presets.organization_id = ? AND presets.deleted_at IS NULL AND presets.is_active = ?", organizationID, true).
		Scan(&results).Error

	return results, err
}
