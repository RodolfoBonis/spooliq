package repositories

import (
	"time"

	"github.com/RodolfoBonis/spooliq/core/helpers"
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

// clearOtherDefaults unsets is_default on every other live preset of the same
// (organization, type), excluding exceptID. It must run inside the mutating
// transaction so the "single default per (organization, type)" invariant — also
// enforced by a partial unique index in the database — is never transiently
// violated by having two defaults at once.
func clearOtherDefaults(db *gorm.DB, organizationID string, presetType entities.PresetType, exceptID uuid.UUID) error {
	return db.Model(&models.PresetModel{}).
		Where("organization_id = ? AND type = ? AND id <> ? AND is_default = ?", organizationID, string(presetType), exceptID, true).
		Update("is_default", false).Error
}

// defaultLockKey builds the advisory-lock key serializing default mutations for
// a given (organization, type) pair.
func defaultLockKey(organizationID string, presetType entities.PresetType) string {
	return organizationID + ":" + string(presetType)
}

// lockDefaults takes a transaction-scoped PostgreSQL advisory lock so concurrent
// default mutations for the same (organization, type) serialize instead of
// racing on the partial unique index. Released automatically at transaction end.
func lockDefaults(tx *gorm.DB, organizationID string, presetType entities.PresetType) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", defaultLockKey(organizationID, presetType)).Error
}

// prepareDefaultOnCreate runs inside a create transaction. It serializes default
// mutations for the (organization, type) pair and then either clears the other
// defaults (when the new preset is the default) or, when the organization has
// no live default of that type yet, promotes the new preset to default so every
// type always has one to fall back to.
func prepareDefaultOnCreate(tx *gorm.DB, preset *entities.PresetEntity, presetType entities.PresetType) error {
	if err := lockDefaults(tx, preset.OrganizationID, presetType); err != nil {
		return err
	}
	if preset.IsDefault {
		return clearOtherDefaults(tx, preset.OrganizationID, presetType, preset.ID)
	}
	var count int64
	if err := tx.Model(&models.PresetModel{}).
		Where("organization_id = ? AND type = ? AND is_default = ?", preset.OrganizationID, string(presetType), true).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		preset.IsDefault = true
	}
	return nil
}

// translateDefaultErr maps a partial-unique-index violation (a losing race that
// the advisory lock did not cover) to the domain ErrDefaultConflict so callers
// return HTTP 409 with a friendly message instead of a raw DB error / 500.
func translateDefaultErr(err error) error {
	if isUniqueViolation(err) {
		return entities.ErrDefaultConflict
	}
	return err
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

// ListPresetsPage is the paginated variant used by the GET /presets endpoint.
func (r *PresetRepositoryImpl) ListPresetsPage(organizationID string, filters entities.PresetFilters, q helpers.ListQuery) ([]*entities.PresetEntity, int64, error) {
	build := func() *gorm.DB {
		query := r.db.Model(&models.PresetModel{}).Where("organization_id = ?", organizationID)
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
		if q.Search != "" {
			query = query.Where("presets.name ILIKE ?", "%"+q.Search+"%")
		}
		return query
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := build()
	if order := q.OrderClause(); order != "" {
		query = query.Order(order)
	}

	var presetModels []models.PresetModel
	if err := query.Offset(q.Offset()).Limit(q.Limit()).Find(&presetModels).Error; err != nil {
		return nil, 0, err
	}

	entitiesList := make([]*entities.PresetEntity, 0, len(presetModels))
	for i := range presetModels {
		entity := presetModels[i].ToEntity()
		entitiesList = append(entitiesList, &entity)
	}

	return entitiesList, total, nil
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

// SetDefault marks the preset as the single default for its (organization, type)
// pair, clearing any other default of the same type in the same transaction.
func (r *PresetRepositoryImpl) SetDefault(id uuid.UUID, organizationID string) (*entities.PresetEntity, error) {
	var result *entities.PresetEntity
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var model models.PresetModel
		if err := tx.Where("id = ? AND organization_id = ?", id, organizationID).First(&model).Error; err != nil {
			return err
		}

		// Serialize concurrent default mutations for this (org, type).
		if err := lockDefaults(tx, organizationID, entities.PresetType(model.Type)); err != nil {
			return err
		}

		// Clear sibling defaults first so the partial unique index never sees two.
		if err := clearOtherDefaults(tx, organizationID, entities.PresetType(model.Type), id); err != nil {
			return err
		}

		now := time.Now()
		if err := tx.Model(&models.PresetModel{}).
			Where("id = ? AND organization_id = ?", id, organizationID).
			Updates(map[string]interface{}{"is_default": true, "updated_at": now}).Error; err != nil {
			return err
		}

		model.IsDefault = true
		model.UpdatedAt = now
		entity := model.ToEntity()
		result = &entity
		return nil
	})
	if err != nil {
		return nil, translateDefaultErr(err)
	}
	return result, nil
}

// Duplicate copies a preset (base + type-specific child) within the same
// organization in a single transaction. The copy is never a default.
func (r *PresetRepositoryImpl) Duplicate(id uuid.UUID, organizationID string, newName string) (*entities.PresetEntity, error) {
	var result *entities.PresetEntity
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var base models.PresetModel
		if err := tx.Where("id = ? AND organization_id = ?", id, organizationID).First(&base).Error; err != nil {
			return err
		}

		now := time.Now()
		newID := uuid.New()

		newPreset := base
		newPreset.ID = newID
		newPreset.Name = newName
		newPreset.IsDefault = false
		newPreset.CreatedAt = now
		newPreset.UpdatedAt = now
		newPreset.DeletedAt = gorm.DeletedAt{}
		newPreset.Organization = nil
		if err := tx.Create(&newPreset).Error; err != nil {
			return err
		}

		switch entities.PresetType(base.Type) {
		case entities.PresetTypeMachine:
			var child models.MachinePresetModel
			if err := tx.Where("id = ?", id).First(&child).Error; err != nil {
				return err
			}
			child.ID = newID
			if err := tx.Create(&child).Error; err != nil {
				return err
			}
		case entities.PresetTypeEnergy:
			var child models.EnergyPresetModel
			if err := tx.Where("id = ?", id).First(&child).Error; err != nil {
				return err
			}
			child.ID = newID
			if err := tx.Create(&child).Error; err != nil {
				return err
			}
		case entities.PresetTypeCost:
			var child models.CostPresetModel
			if err := tx.Where("id = ?", id).First(&child).Error; err != nil {
				return err
			}
			child.ID = newID
			if err := tx.Create(&child).Error; err != nil {
				return err
			}
		default:
			return entities.ErrInvalidPresetType
		}

		entity := newPreset.ToEntity()
		result = &entity
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CreateMachine creates a new machine preset with base preset
func (r *PresetRepositoryImpl) CreateMachine(preset *entities.PresetEntity, machine *entities.MachinePresetEntity) error {
	return translateDefaultErr(r.db.Transaction(func(tx *gorm.DB) error {
		if err := prepareDefaultOnCreate(tx, preset, entities.PresetTypeMachine); err != nil {
			return err
		}

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
	}))
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
	return translateDefaultErr(r.db.Transaction(func(tx *gorm.DB) error {
		if preset.IsDefault {
			if err := lockDefaults(tx, preset.OrganizationID, entities.PresetTypeMachine); err != nil {
				return err
			}
			if err := clearOtherDefaults(tx, preset.OrganizationID, entities.PresetTypeMachine, preset.ID); err != nil {
				return err
			}
		}
		if err := updateBasePreset(tx, preset); err != nil {
			return err
		}
		return updateMachineChild(tx, machine)
	}))
}

// CreateEnergy creates a new energy preset with base preset
func (r *PresetRepositoryImpl) CreateEnergy(preset *entities.PresetEntity, energy *entities.EnergyPresetEntity) error {
	return translateDefaultErr(r.db.Transaction(func(tx *gorm.DB) error {
		if err := prepareDefaultOnCreate(tx, preset, entities.PresetTypeEnergy); err != nil {
			return err
		}

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
	}))
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
	return translateDefaultErr(r.db.Transaction(func(tx *gorm.DB) error {
		if preset.IsDefault {
			if err := lockDefaults(tx, preset.OrganizationID, entities.PresetTypeEnergy); err != nil {
				return err
			}
			if err := clearOtherDefaults(tx, preset.OrganizationID, entities.PresetTypeEnergy, preset.ID); err != nil {
				return err
			}
		}
		if err := updateBasePreset(tx, preset); err != nil {
			return err
		}
		return updateEnergyChild(tx, energy)
	}))
}

// CreateCost creates a new cost preset with base preset
func (r *PresetRepositoryImpl) CreateCost(preset *entities.PresetEntity, cost *entities.CostPresetEntity) error {
	return translateDefaultErr(r.db.Transaction(func(tx *gorm.DB) error {
		if err := prepareDefaultOnCreate(tx, preset, entities.PresetTypeCost); err != nil {
			return err
		}

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
	}))
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
	return translateDefaultErr(r.db.Transaction(func(tx *gorm.DB) error {
		if preset.IsDefault {
			if err := lockDefaults(tx, preset.OrganizationID, entities.PresetTypeCost); err != nil {
				return err
			}
			if err := clearOtherDefaults(tx, preset.OrganizationID, entities.PresetTypeCost, preset.ID); err != nil {
				return err
			}
		}
		if err := updateBasePreset(tx, preset); err != nil {
			return err
		}
		return updateCostChild(tx, cost)
	}))
}

// OPTIMIZED METHODS WITH ORGANIZATION FILTERING AND JOINS

// GetMachinePresets retrieves a page of machine presets with base data in a
// single query, applying free-text search (on name), sorting and pagination.
func (r *PresetRepositoryImpl) GetMachinePresets(organizationID string, q helpers.ListQuery) ([]*repositories.MachinePresetResponse, int64, error) {
	build := func() *gorm.DB {
		query := r.db.Table("machine_presets").
			Joins("INNER JOIN presets ON machine_presets.id = presets.id").
			Where("presets.organization_id = ? AND presets.deleted_at IS NULL AND presets.is_active = ?", organizationID, true)
		if q.Search != "" {
			query = query.Where("presets.name ILIKE ?", "%"+q.Search+"%")
		}
		return query
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := build().Select(`
			presets.id,
			presets.name,
			presets.description,
			presets.type,
			presets.is_active,
			presets.is_default,
			presets.created_at,
			presets.updated_at,
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
		`)
	if order := q.OrderClause(); order != "" {
		query = query.Order(order)
	}

	var results []*repositories.MachinePresetResponse
	err := query.Offset(q.Offset()).Limit(q.Limit()).Scan(&results).Error
	return results, total, err
}

// GetMachinePresetsByBrand retrieves a page of machine presets by brand with
// base data, applying free-text search (on name), sorting and pagination.
func (r *PresetRepositoryImpl) GetMachinePresetsByBrand(brand, organizationID string, q helpers.ListQuery) ([]*repositories.MachinePresetResponse, int64, error) {
	build := func() *gorm.DB {
		query := r.db.Table("machine_presets").
			Joins("INNER JOIN presets ON machine_presets.id = presets.id").
			Where("presets.organization_id = ? AND presets.deleted_at IS NULL AND presets.is_active = ? AND machine_presets.brand = ?", organizationID, true, brand)
		if q.Search != "" {
			query = query.Where("presets.name ILIKE ?", "%"+q.Search+"%")
		}
		return query
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := build().Select(`
			presets.id,
			presets.name,
			presets.description,
			presets.type,
			presets.is_active,
			presets.is_default,
			presets.created_at,
			presets.updated_at,
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
		`)
	if order := q.OrderClause(); order != "" {
		query = query.Order(order)
	}

	var results []*repositories.MachinePresetResponse
	err := query.Offset(q.Offset()).Limit(q.Limit()).Scan(&results).Error
	return results, total, err
}

// GetEnergyPresets retrieves a page of energy presets with base data, applying
// free-text search (on name), sorting and pagination.
func (r *PresetRepositoryImpl) GetEnergyPresets(organizationID string, q helpers.ListQuery) ([]*repositories.EnergyPresetResponse, int64, error) {
	build := func() *gorm.DB {
		query := r.db.Table("energy_presets").
			Joins("INNER JOIN presets ON energy_presets.id = presets.id").
			Where("presets.organization_id = ? AND presets.deleted_at IS NULL AND presets.is_active = ?", organizationID, true)
		if q.Search != "" {
			query = query.Where("presets.name ILIKE ?", "%"+q.Search+"%")
		}
		return query
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := build().Select(`
			presets.id,
			presets.name,
			presets.description,
			presets.type,
			presets.is_active,
			presets.is_default,
			presets.created_at,
			presets.updated_at,
			energy_presets.country,
			energy_presets.state,
			energy_presets.city,
			energy_presets.energy_cost_per_kwh,
			energy_presets.currency,
			energy_presets.provider,
			energy_presets.tariff_type,
			energy_presets.peak_hour_multiplier,
			energy_presets.off_peak_hour_multiplier
		`)
	if order := q.OrderClause(); order != "" {
		query = query.Order(order)
	}

	var results []*repositories.EnergyPresetResponse
	err := query.Offset(q.Offset()).Limit(q.Limit()).Scan(&results).Error
	return results, total, err
}

// GetEnergyPresetsByLocation retrieves a page of energy presets by location with
// base data, applying free-text search (on name), sorting and pagination.
func (r *PresetRepositoryImpl) GetEnergyPresetsByLocation(country, state, city, organizationID string, q helpers.ListQuery) ([]*repositories.EnergyPresetResponse, int64, error) {
	build := func() *gorm.DB {
		query := r.db.Table("energy_presets").
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
		if q.Search != "" {
			query = query.Where("presets.name ILIKE ?", "%"+q.Search+"%")
		}
		return query
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := build().Select(`
			presets.id,
			presets.name,
			presets.description,
			presets.type,
			presets.is_active,
			presets.is_default,
			presets.created_at,
			presets.updated_at,
			energy_presets.country,
			energy_presets.state,
			energy_presets.city,
			energy_presets.energy_cost_per_kwh,
			energy_presets.currency,
			energy_presets.provider,
			energy_presets.tariff_type,
			energy_presets.peak_hour_multiplier,
			energy_presets.off_peak_hour_multiplier
		`)
	if order := q.OrderClause(); order != "" {
		query = query.Order(order)
	}

	var results []*repositories.EnergyPresetResponse
	err := query.Offset(q.Offset()).Limit(q.Limit()).Scan(&results).Error
	return results, total, err
}

// GetEnergyPresetsByCurrency retrieves a page of energy presets by currency with
// base data, applying free-text search (on name), sorting and pagination.
func (r *PresetRepositoryImpl) GetEnergyPresetsByCurrency(currency, organizationID string, q helpers.ListQuery) ([]*repositories.EnergyPresetResponse, int64, error) {
	build := func() *gorm.DB {
		query := r.db.Table("energy_presets").
			Joins("INNER JOIN presets ON energy_presets.id = presets.id").
			Where("presets.organization_id = ? AND presets.deleted_at IS NULL AND presets.is_active = ? AND energy_presets.currency = ?", organizationID, true, currency)
		if q.Search != "" {
			query = query.Where("presets.name ILIKE ?", "%"+q.Search+"%")
		}
		return query
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := build().Select(`
			presets.id,
			presets.name,
			presets.description,
			presets.type,
			presets.is_active,
			presets.is_default,
			presets.created_at,
			presets.updated_at,
			energy_presets.country,
			energy_presets.state,
			energy_presets.city,
			energy_presets.energy_cost_per_kwh,
			energy_presets.currency,
			energy_presets.provider,
			energy_presets.tariff_type,
			energy_presets.peak_hour_multiplier,
			energy_presets.off_peak_hour_multiplier
		`)
	if order := q.OrderClause(); order != "" {
		query = query.Order(order)
	}

	var results []*repositories.EnergyPresetResponse
	err := query.Offset(q.Offset()).Limit(q.Limit()).Scan(&results).Error
	return results, total, err
}

// GetCostPresets retrieves a page of cost presets with base data, applying
// free-text search (on name), sorting and pagination.
func (r *PresetRepositoryImpl) GetCostPresets(organizationID string, q helpers.ListQuery) ([]*repositories.CostPresetResponse, int64, error) {
	build := func() *gorm.DB {
		query := r.db.Table("cost_presets").
			Joins("INNER JOIN presets ON cost_presets.id = presets.id").
			Where("presets.organization_id = ? AND presets.deleted_at IS NULL AND presets.is_active = ?", organizationID, true)
		if q.Search != "" {
			query = query.Where("presets.name ILIKE ?", "%"+q.Search+"%")
		}
		return query
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := build().Select(`
			presets.id,
			presets.name,
			presets.description,
			presets.type,
			presets.is_active,
			presets.is_default,
			presets.created_at,
			presets.updated_at,
			cost_presets.labor_cost_per_hour,
			cost_presets.packaging_cost_per_item,
			cost_presets.shipping_cost_base,
			cost_presets.shipping_cost_per_gram,
			cost_presets.overhead_percentage,
			cost_presets.profit_margin_percentage,
			cost_presets.post_processing_cost_per_hour,
			cost_presets.support_removal_cost_per_hour,
			cost_presets.quality_control_cost_per_item
		`)
	if order := q.OrderClause(); order != "" {
		query = query.Order(order)
	}

	var results []*repositories.CostPresetResponse
	err := query.Offset(q.Offset()).Limit(q.Limit()).Scan(&results).Error
	return results, total, err
}

// IsReferencedByProfile reports whether any non-deleted print profile in the
// organization references the given preset (as machine, energy or cost). It
// queries the print_profiles table by name to avoid a preset -> profile import
// cycle.
func (r *PresetRepositoryImpl) IsReferencedByProfile(presetID uuid.UUID, organizationID string) (bool, error) {
	var exists bool
	err := r.db.Raw(`
		SELECT EXISTS(
			SELECT 1 FROM print_profiles
			WHERE organization_id = ? AND deleted_at IS NULL
			  AND (machine_preset_id = ? OR energy_preset_id = ? OR cost_preset_id = ?)
		)
	`, organizationID, presetID, presetID, presetID).Scan(&exists).Error
	if err != nil {
		return false, err
	}
	return exists, nil
}
