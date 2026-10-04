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

// GetByID retrieves a preset by its ID, scoped to the organization.
// Soft-deleted rows are excluded automatically by GORM (gorm.DeletedAt).
func (r *PresetRepositoryImpl) GetByID(id uuid.UUID, organizationID string) (*entities.PresetEntity, error) {
	var model models.PresetModel

	err := r.db.
		Where("id = ? AND organization_id = ?", id, organizationID).
		First(&model).Error
	if err != nil {
		return nil, err
	}

	entity := model.ToEntity()
	return &entity, nil
}

// ListPresets retrieves presets for an organization applying combinable filters.
func (r *PresetRepositoryImpl) ListPresets(organizationID string, filters entities.PresetFilters) ([]*entities.PresetEntity, error) {
	var presetModels []models.PresetModel

	query := r.db.Where("organization_id = ?", organizationID)

	if filters.Type != nil {
		query = query.Where("type = ?", string(*filters.Type))
	}
	if filters.ActiveOnly {
		query = query.Where("is_active = ?", true)
	}
	if filters.DefaultOnly {
		query = query.Where("is_default = ?", true)
	}
	if filters.GlobalOnly {
		query = query.Where("user_id IS NULL")
	}
	if filters.UserID != nil {
		query = query.Where("user_id = ?", *filters.UserID)
	}

	if err := query.Find(&presetModels).Error; err != nil {
		return nil, err
	}

	entitiesList := make([]*entities.PresetEntity, 0, len(presetModels))
	for i := range presetModels {
		entity := presetModels[i].ToEntity()
		entitiesList = append(entitiesList, &entity)
	}

	return entitiesList, nil
}

// updateBasePreset updates the base preset row within the given db/tx handle,
// scoped to its organization. Returns gorm.ErrRecordNotFound when no row matches
// the id+organization pair. Shared by Update and the atomic *WithPreset methods.
func updateBasePreset(db *gorm.DB, preset *entities.PresetEntity) error {
	model := &models.PresetModel{}
	model.FromEntity(preset)

	result := db.Model(&models.PresetModel{}).
		Where("id = ? AND organization_id = ?", model.ID, model.OrganizationID).
		Select("name", "description", "type", "is_active", "is_default", "user_id", "updated_at").
		Updates(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Update updates an existing preset, scoped to its organization.
// Returns gorm.ErrRecordNotFound when no row matches the id+organization pair.
func (r *PresetRepositoryImpl) Update(preset *entities.PresetEntity) error {
	return updateBasePreset(r.db, preset)
}

// Delete soft deletes a preset, scoped to its organization.
// Returns gorm.ErrRecordNotFound when no row matches the id+organization pair.
func (r *PresetRepositoryImpl) Delete(id uuid.UUID, organizationID string) error {
	result := r.db.
		Where("id = ? AND organization_id = ?", id, organizationID).
		Delete(&models.PresetModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
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

// GetMachineByID retrieves a machine preset by ID, verifying the base preset
// belongs to the organization and is not soft-deleted.
func (r *PresetRepositoryImpl) GetMachineByID(id uuid.UUID, organizationID string) (*entities.MachinePresetEntity, error) {
	var model models.MachinePresetModel

	err := r.db.
		Select("machine_presets.*").
		Joins("INNER JOIN presets ON presets.id = machine_presets.id").
		Where("machine_presets.id = ? AND presets.organization_id = ? AND presets.deleted_at IS NULL", id, organizationID).
		First(&model).Error
	if err != nil {
		return nil, err
	}

	entity := model.ToEntity()
	return &entity, nil
}

// GetMachinesByBrand retrieves machine presets by brand, scoped to the organization.
func (r *PresetRepositoryImpl) GetMachinesByBrand(brand, organizationID string) ([]*entities.MachinePresetEntity, error) {
	var machineModels []models.MachinePresetModel

	err := r.db.
		Select("machine_presets.*").
		Joins("INNER JOIN presets ON presets.id = machine_presets.id").
		Where("machine_presets.brand = ? AND presets.organization_id = ? AND presets.deleted_at IS NULL", brand, organizationID).
		Find(&machineModels).Error
	if err != nil {
		return nil, err
	}

	entitiesList := make([]*entities.MachinePresetEntity, 0, len(machineModels))
	for i := range machineModels {
		entity := machineModels[i].ToEntity()
		entitiesList = append(entitiesList, &entity)
	}

	return entitiesList, nil
}

// updateMachineChild updates a machine child row within the given db/tx handle,
// scoped to its organization. Returns gorm.ErrRecordNotFound when no row matches.
func updateMachineChild(db *gorm.DB, machine *entities.MachinePresetEntity) error {
	model := &models.MachinePresetModel{}
	model.FromEntity(machine)

	result := db.Model(&models.MachinePresetModel{}).
		Where("id = ? AND organization_id = ?", model.ID, model.OrganizationID).
		Select("*").
		Updates(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateMachine updates a machine preset, scoped to its organization.
func (r *PresetRepositoryImpl) UpdateMachine(machine *entities.MachinePresetEntity) error {
	return updateMachineChild(r.db, machine)
}

// UpdateMachineWithPreset atomically updates the base preset and its machine child
// in a single, organization-scoped transaction. Either both rows are updated or
// neither is: if either the base preset or the machine child does not match the
// id+organization pair, the transaction rolls back with gorm.ErrRecordNotFound.
func (r *PresetRepositoryImpl) UpdateMachineWithPreset(preset *entities.PresetEntity, machine *entities.MachinePresetEntity) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := updateBasePreset(tx, preset); err != nil {
			return err
		}
		return updateMachineChild(tx, machine)
	})
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

// GetEnergyByID retrieves an energy preset by ID, verifying the base preset
// belongs to the organization and is not soft-deleted.
func (r *PresetRepositoryImpl) GetEnergyByID(id uuid.UUID, organizationID string) (*entities.EnergyPresetEntity, error) {
	var model models.EnergyPresetModel

	err := r.db.
		Select("energy_presets.*").
		Joins("INNER JOIN presets ON presets.id = energy_presets.id").
		Where("energy_presets.id = ? AND presets.organization_id = ? AND presets.deleted_at IS NULL", id, organizationID).
		First(&model).Error
	if err != nil {
		return nil, err
	}

	entity := model.ToEntity()
	return &entity, nil
}

// GetEnergyByLocation retrieves energy presets by location, scoped to the organization.
func (r *PresetRepositoryImpl) GetEnergyByLocation(country, state, city, organizationID string) ([]*entities.EnergyPresetEntity, error) {
	var energyModels []models.EnergyPresetModel

	query := r.db.
		Select("energy_presets.*").
		Joins("INNER JOIN presets ON presets.id = energy_presets.id").
		Where("presets.organization_id = ? AND presets.deleted_at IS NULL", organizationID)

	if country != "" {
		query = query.Where("energy_presets.country = ?", country)
	}
	if state != "" {
		query = query.Where("energy_presets.state = ?", state)
	}
	if city != "" {
		query = query.Where("energy_presets.city = ?", city)
	}

	if err := query.Find(&energyModels).Error; err != nil {
		return nil, err
	}

	entitiesList := make([]*entities.EnergyPresetEntity, 0, len(energyModels))
	for i := range energyModels {
		entity := energyModels[i].ToEntity()
		entitiesList = append(entitiesList, &entity)
	}

	return entitiesList, nil
}

// GetEnergyByCurrency retrieves energy presets by currency, scoped to the organization.
func (r *PresetRepositoryImpl) GetEnergyByCurrency(currency, organizationID string) ([]*entities.EnergyPresetEntity, error) {
	var energyModels []models.EnergyPresetModel

	err := r.db.
		Select("energy_presets.*").
		Joins("INNER JOIN presets ON presets.id = energy_presets.id").
		Where("energy_presets.currency = ? AND presets.organization_id = ? AND presets.deleted_at IS NULL", currency, organizationID).
		Find(&energyModels).Error
	if err != nil {
		return nil, err
	}

	entitiesList := make([]*entities.EnergyPresetEntity, 0, len(energyModels))
	for i := range energyModels {
		entity := energyModels[i].ToEntity()
		entitiesList = append(entitiesList, &entity)
	}

	return entitiesList, nil
}

// updateEnergyChild updates an energy child row within the given db/tx handle,
// scoped to its organization. Returns gorm.ErrRecordNotFound when no row matches.
func updateEnergyChild(db *gorm.DB, energy *entities.EnergyPresetEntity) error {
	model := &models.EnergyPresetModel{}
	model.FromEntity(energy)

	result := db.Model(&models.EnergyPresetModel{}).
		Where("id = ? AND organization_id = ?", model.ID, model.OrganizationID).
		Select("*").
		Updates(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateEnergy updates an energy preset, scoped to its organization.
func (r *PresetRepositoryImpl) UpdateEnergy(energy *entities.EnergyPresetEntity) error {
	return updateEnergyChild(r.db, energy)
}

// UpdateEnergyWithPreset atomically updates the base preset and its energy child
// in a single, organization-scoped transaction. Either both rows are updated or
// neither is.
func (r *PresetRepositoryImpl) UpdateEnergyWithPreset(preset *entities.PresetEntity, energy *entities.EnergyPresetEntity) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := updateBasePreset(tx, preset); err != nil {
			return err
		}
		return updateEnergyChild(tx, energy)
	})
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

// GetCostByID retrieves a cost preset by ID, verifying the base preset
// belongs to the organization and is not soft-deleted.
func (r *PresetRepositoryImpl) GetCostByID(id uuid.UUID, organizationID string) (*entities.CostPresetEntity, error) {
	var model models.CostPresetModel

	err := r.db.
		Select("cost_presets.*").
		Joins("INNER JOIN presets ON presets.id = cost_presets.id").
		Where("cost_presets.id = ? AND presets.organization_id = ? AND presets.deleted_at IS NULL", id, organizationID).
		First(&model).Error
	if err != nil {
		return nil, err
	}

	entity := model.ToEntity()
	return &entity, nil
}

// updateCostChild updates a cost child row within the given db/tx handle, scoped
// to its organization. Returns gorm.ErrRecordNotFound when no row matches.
func updateCostChild(db *gorm.DB, cost *entities.CostPresetEntity) error {
	model := &models.CostPresetModel{}
	model.FromEntity(cost)

	result := db.Model(&models.CostPresetModel{}).
		Where("id = ? AND organization_id = ?", model.ID, model.OrganizationID).
		Select("*").
		Updates(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateCost updates a cost preset, scoped to its organization.
func (r *PresetRepositoryImpl) UpdateCost(cost *entities.CostPresetEntity) error {
	return updateCostChild(r.db, cost)
}

// UpdateCostWithPreset atomically updates the base preset and its cost child in a
// single, organization-scoped transaction. Either both rows are updated or
// neither is.
func (r *PresetRepositoryImpl) UpdateCostWithPreset(preset *entities.PresetEntity, cost *entities.CostPresetEntity) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := updateBasePreset(tx, preset); err != nil {
			return err
		}
		return updateCostChild(tx, cost)
	})
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
