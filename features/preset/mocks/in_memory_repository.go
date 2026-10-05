// Package mocks provides test doubles for the preset feature.
package mocks

import (
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InMemoryPresetRepository is a stateful, in-memory implementation of
// repositories.PresetRepository used in tests.
//
// It faithfully reproduces the tenant-isolation contract of the real
// repository: every read, update and delete is scoped by organizationID and
// cross-organization access returns gorm.ErrRecordNotFound (never leaking the
// existence of another tenant's data). It also records the organization IDs it
// was invoked with so tests can assert the scope is threaded through.
type InMemoryPresetRepository struct {
	presets  map[uuid.UUID]*entities.PresetEntity
	machines map[uuid.UUID]*entities.MachinePresetEntity
	energies map[uuid.UUID]*entities.EnergyPresetEntity
	costs    map[uuid.UUID]*entities.CostPresetEntity

	// OrgScopes records every organizationID the repository was asked to scope
	// a read/update/delete by. Tests use it to assert pass-through.
	OrgScopes []string

	// ProfileReferenced lets tests mark preset IDs as referenced by a live
	// profile, so IsReferencedByProfile returns true for them.
	ProfileReferenced map[uuid.UUID]bool
}

// NewInMemoryPresetRepository creates an empty in-memory repository.
func NewInMemoryPresetRepository() *InMemoryPresetRepository {
	return &InMemoryPresetRepository{
		presets:  make(map[uuid.UUID]*entities.PresetEntity),
		machines: make(map[uuid.UUID]*entities.MachinePresetEntity),
		energies: make(map[uuid.UUID]*entities.EnergyPresetEntity),
		costs:    make(map[uuid.UUID]*entities.CostPresetEntity),
	}
}

var _ repositories.PresetRepository = (*InMemoryPresetRepository)(nil)

func (r *InMemoryPresetRepository) record(organizationID string) {
	r.OrgScopes = append(r.OrgScopes, organizationID)
}

// liveInOrg returns the stored preset only when it exists, is not soft-deleted
// and belongs to organizationID.
func (r *InMemoryPresetRepository) liveInOrg(id uuid.UUID, organizationID string) *entities.PresetEntity {
	preset, ok := r.presets[id]
	if !ok || preset.DeletedAt != nil || preset.OrganizationID != organizationID {
		return nil
	}
	return preset
}

// Seed inserts a preset together with its type-specific child entity directly,
// bypassing scoping, so tests can set up fixtures for multiple organizations.
func (r *InMemoryPresetRepository) Seed(preset *entities.PresetEntity, child interface{}) {
	clone := *preset
	r.presets[preset.ID] = &clone
	switch c := child.(type) {
	case *entities.MachinePresetEntity:
		cc := *c
		r.machines[preset.ID] = &cc
	case *entities.EnergyPresetEntity:
		cc := *c
		r.energies[preset.ID] = &cc
	case *entities.CostPresetEntity:
		cc := *c
		r.costs[preset.ID] = &cc
	}
}

// StoredPreset returns a copy of the raw stored preset (ignoring scope), or nil.
func (r *InMemoryPresetRepository) StoredPreset(id uuid.UUID) *entities.PresetEntity {
	preset, ok := r.presets[id]
	if !ok {
		return nil
	}
	clone := *preset
	return &clone
}

// Create stores a new base preset.
func (r *InMemoryPresetRepository) Create(preset *entities.PresetEntity) error {
	clone := *preset
	r.presets[preset.ID] = &clone
	return nil
}

// GetByID retrieves a preset scoped to organizationID.
func (r *InMemoryPresetRepository) GetByID(id uuid.UUID, organizationID string) (*entities.PresetEntity, error) {
	r.record(organizationID)
	preset := r.liveInOrg(id, organizationID)
	if preset == nil {
		return nil, gorm.ErrRecordNotFound
	}
	clone := *preset
	return &clone, nil
}

// ListPresets lists presets for an organization applying combinable filters.
func (r *InMemoryPresetRepository) ListPresets(organizationID string, filters entities.PresetFilters) ([]*entities.PresetEntity, error) {
	r.record(organizationID)
	var result []*entities.PresetEntity
	for _, preset := range r.presets {
		if preset.DeletedAt != nil || preset.OrganizationID != organizationID {
			continue
		}
		if filters.Type != nil && preset.Type != *filters.Type {
			continue
		}
		if filters.ActiveOnly && !preset.IsActive {
			continue
		}
		if filters.DefaultOnly && !preset.IsDefault {
			continue
		}
		if filters.GlobalOnly && preset.UserID != nil {
			continue
		}
		if filters.UserID != nil {
			if preset.UserID == nil || *preset.UserID != *filters.UserID {
				continue
			}
		}
		clone := *preset
		result = append(result, &clone)
	}
	return result, nil
}

// Update updates a preset scoped to its organization.
func (r *InMemoryPresetRepository) Update(preset *entities.PresetEntity) error {
	r.record(preset.OrganizationID)
	existing := r.liveInOrg(preset.ID, preset.OrganizationID)
	if existing == nil {
		return gorm.ErrRecordNotFound
	}
	clone := *preset
	r.presets[preset.ID] = &clone
	return nil
}

// Delete soft-deletes a preset scoped to its organization.
func (r *InMemoryPresetRepository) Delete(id uuid.UUID, organizationID string) error {
	r.record(organizationID)
	existing := r.liveInOrg(id, organizationID)
	if existing == nil {
		return gorm.ErrRecordNotFound
	}
	now := existing.UpdatedAt
	existing.DeletedAt = &now
	return nil
}

// CreateMachine stores a base preset and its machine child.
func (r *InMemoryPresetRepository) CreateMachine(preset *entities.PresetEntity, machine *entities.MachinePresetEntity) error {
	if preset.IsDefault {
		r.clearOtherDefaults(preset.OrganizationID, entities.PresetTypeMachine, preset.ID)
	}
	r.Seed(preset, machine)
	return nil
}

// GetMachineByID retrieves a machine child verified against its base preset's org.
func (r *InMemoryPresetRepository) GetMachineByID(id uuid.UUID, organizationID string) (*entities.MachinePresetEntity, error) {
	r.record(organizationID)
	if r.liveInOrg(id, organizationID) == nil {
		return nil, gorm.ErrRecordNotFound
	}
	machine, ok := r.machines[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	clone := *machine
	return &clone, nil
}

// GetMachinesByBrand lists machine children for an organization filtered by brand.
func (r *InMemoryPresetRepository) GetMachinesByBrand(brand, organizationID string) ([]*entities.MachinePresetEntity, error) {
	r.record(organizationID)
	var result []*entities.MachinePresetEntity
	for id, machine := range r.machines {
		if r.liveInOrg(id, organizationID) == nil || machine.Brand != brand {
			continue
		}
		clone := *machine
		result = append(result, &clone)
	}
	return result, nil
}

// UpdateMachine updates a machine child scoped to its organization.
func (r *InMemoryPresetRepository) UpdateMachine(machine *entities.MachinePresetEntity) error {
	r.record(machine.OrganizationID)
	if r.liveInOrg(machine.ID, machine.OrganizationID) == nil {
		return gorm.ErrRecordNotFound
	}
	clone := *machine
	r.machines[machine.ID] = &clone
	return nil
}

// CreateEnergy stores a base preset and its energy child.
func (r *InMemoryPresetRepository) CreateEnergy(preset *entities.PresetEntity, energy *entities.EnergyPresetEntity) error {
	if preset.IsDefault {
		r.clearOtherDefaults(preset.OrganizationID, entities.PresetTypeEnergy, preset.ID)
	}
	r.Seed(preset, energy)
	return nil
}

// GetEnergyByID retrieves an energy child verified against its base preset's org.
func (r *InMemoryPresetRepository) GetEnergyByID(id uuid.UUID, organizationID string) (*entities.EnergyPresetEntity, error) {
	r.record(organizationID)
	if r.liveInOrg(id, organizationID) == nil {
		return nil, gorm.ErrRecordNotFound
	}
	energy, ok := r.energies[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	clone := *energy
	return &clone, nil
}

// GetEnergyByLocation lists energy children for an organization filtered by location.
func (r *InMemoryPresetRepository) GetEnergyByLocation(country, state, city, organizationID string) ([]*entities.EnergyPresetEntity, error) {
	r.record(organizationID)
	var result []*entities.EnergyPresetEntity
	for id, energy := range r.energies {
		if r.liveInOrg(id, organizationID) == nil {
			continue
		}
		if country != "" && energy.Country != country {
			continue
		}
		if state != "" && energy.State != state {
			continue
		}
		if city != "" && energy.City != city {
			continue
		}
		clone := *energy
		result = append(result, &clone)
	}
	return result, nil
}

// GetEnergyByCurrency lists energy children for an organization filtered by currency.
func (r *InMemoryPresetRepository) GetEnergyByCurrency(currency, organizationID string) ([]*entities.EnergyPresetEntity, error) {
	r.record(organizationID)
	var result []*entities.EnergyPresetEntity
	for id, energy := range r.energies {
		if r.liveInOrg(id, organizationID) == nil || energy.Currency != currency {
			continue
		}
		clone := *energy
		result = append(result, &clone)
	}
	return result, nil
}

// UpdateEnergy updates an energy child scoped to its organization.
func (r *InMemoryPresetRepository) UpdateEnergy(energy *entities.EnergyPresetEntity) error {
	r.record(energy.OrganizationID)
	if r.liveInOrg(energy.ID, energy.OrganizationID) == nil {
		return gorm.ErrRecordNotFound
	}
	clone := *energy
	r.energies[energy.ID] = &clone
	return nil
}

// CreateCost stores a base preset and its cost child.
func (r *InMemoryPresetRepository) CreateCost(preset *entities.PresetEntity, cost *entities.CostPresetEntity) error {
	if preset.IsDefault {
		r.clearOtherDefaults(preset.OrganizationID, entities.PresetTypeCost, preset.ID)
	}
	r.Seed(preset, cost)
	return nil
}

// GetCostByID retrieves a cost child verified against its base preset's org.
func (r *InMemoryPresetRepository) GetCostByID(id uuid.UUID, organizationID string) (*entities.CostPresetEntity, error) {
	r.record(organizationID)
	if r.liveInOrg(id, organizationID) == nil {
		return nil, gorm.ErrRecordNotFound
	}
	cost, ok := r.costs[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	clone := *cost
	return &clone, nil
}

// UpdateCost updates a cost child scoped to its organization.
func (r *InMemoryPresetRepository) UpdateCost(cost *entities.CostPresetEntity) error {
	r.record(cost.OrganizationID)
	if r.liveInOrg(cost.ID, cost.OrganizationID) == nil {
		return gorm.ErrRecordNotFound
	}
	clone := *cost
	r.costs[cost.ID] = &clone
	return nil
}

func (r *InMemoryPresetRepository) machineResponse(id uuid.UUID) *repositories.MachinePresetResponse {
	preset := r.presets[id]
	machine := r.machines[id]
	return &repositories.MachinePresetResponse{
		ID:                     preset.ID.String(),
		Name:                   preset.Name,
		Description:            preset.Description,
		Type:                   string(preset.Type),
		IsActive:               preset.IsActive,
		IsDefault:              preset.IsDefault,
		CreatedAt:              preset.CreatedAt,
		UpdatedAt:              preset.UpdatedAt,
		Brand:                  machine.Brand,
		Model:                  machine.Model,
		BuildVolumeX:           machine.BuildVolumeX,
		BuildVolumeY:           machine.BuildVolumeY,
		BuildVolumeZ:           machine.BuildVolumeZ,
		NozzleDiameter:         machine.NozzleDiameter,
		LayerHeightMin:         machine.LayerHeightMin,
		LayerHeightMax:         machine.LayerHeightMax,
		PrintSpeedMax:          machine.PrintSpeedMax,
		PowerConsumption:       machine.PowerConsumption,
		BedTemperatureMax:      machine.BedTemperatureMax,
		ExtruderTemperatureMax: machine.ExtruderTemperatureMax,
		FilamentDiameter:       machine.FilamentDiameter,
		CostPerHour:            machine.CostPerHour,
	}
}

// GetMachinePresets lists active machine responses scoped to an organization.
func (r *InMemoryPresetRepository) GetMachinePresets(organizationID string) ([]*repositories.MachinePresetResponse, error) {
	r.record(organizationID)
	var result []*repositories.MachinePresetResponse
	for id := range r.machines {
		preset := r.liveInOrg(id, organizationID)
		if preset == nil || !preset.IsActive {
			continue
		}
		result = append(result, r.machineResponse(id))
	}
	return result, nil
}

// GetMachinePresetsByBrand lists active machine responses scoped to org and brand.
func (r *InMemoryPresetRepository) GetMachinePresetsByBrand(brand, organizationID string) ([]*repositories.MachinePresetResponse, error) {
	r.record(organizationID)
	var result []*repositories.MachinePresetResponse
	for id, machine := range r.machines {
		preset := r.liveInOrg(id, organizationID)
		if preset == nil || !preset.IsActive || machine.Brand != brand {
			continue
		}
		result = append(result, r.machineResponse(id))
	}
	return result, nil
}

func (r *InMemoryPresetRepository) energyResponse(id uuid.UUID) *repositories.EnergyPresetResponse {
	preset := r.presets[id]
	energy := r.energies[id]
	return &repositories.EnergyPresetResponse{
		ID:                    preset.ID.String(),
		Name:                  preset.Name,
		Description:           preset.Description,
		Type:                  string(preset.Type),
		IsActive:              preset.IsActive,
		IsDefault:             preset.IsDefault,
		CreatedAt:             preset.CreatedAt,
		UpdatedAt:             preset.UpdatedAt,
		Country:               energy.Country,
		State:                 energy.State,
		City:                  energy.City,
		EnergyCostPerKwh:      energy.EnergyCostPerKwh,
		Currency:              energy.Currency,
		Provider:              energy.Provider,
		TariffType:            energy.TariffType,
		PeakHourMultiplier:    energy.PeakHourMultiplier,
		OffPeakHourMultiplier: energy.OffPeakHourMultiplier,
	}
}

// GetEnergyPresets lists active energy responses scoped to an organization.
func (r *InMemoryPresetRepository) GetEnergyPresets(organizationID string) ([]*repositories.EnergyPresetResponse, error) {
	r.record(organizationID)
	var result []*repositories.EnergyPresetResponse
	for id := range r.energies {
		preset := r.liveInOrg(id, organizationID)
		if preset == nil || !preset.IsActive {
			continue
		}
		result = append(result, r.energyResponse(id))
	}
	return result, nil
}

// GetEnergyPresetsByLocation lists active energy responses scoped to org and location.
func (r *InMemoryPresetRepository) GetEnergyPresetsByLocation(country, state, city, organizationID string) ([]*repositories.EnergyPresetResponse, error) {
	r.record(organizationID)
	var result []*repositories.EnergyPresetResponse
	for id, energy := range r.energies {
		preset := r.liveInOrg(id, organizationID)
		if preset == nil || !preset.IsActive {
			continue
		}
		if country != "" && energy.Country != country {
			continue
		}
		if state != "" && energy.State != state {
			continue
		}
		if city != "" && energy.City != city {
			continue
		}
		result = append(result, r.energyResponse(id))
	}
	return result, nil
}

// GetEnergyPresetsByCurrency lists active energy responses scoped to org and currency.
func (r *InMemoryPresetRepository) GetEnergyPresetsByCurrency(currency, organizationID string) ([]*repositories.EnergyPresetResponse, error) {
	r.record(organizationID)
	var result []*repositories.EnergyPresetResponse
	for id, energy := range r.energies {
		preset := r.liveInOrg(id, organizationID)
		if preset == nil || !preset.IsActive || energy.Currency != currency {
			continue
		}
		result = append(result, r.energyResponse(id))
	}
	return result, nil
}

// GetCostPresets lists active cost responses scoped to an organization.
func (r *InMemoryPresetRepository) GetCostPresets(organizationID string) ([]*repositories.CostPresetResponse, error) {
	r.record(organizationID)
	var result []*repositories.CostPresetResponse
	for id := range r.costs {
		preset := r.liveInOrg(id, organizationID)
		if preset == nil || !preset.IsActive {
			continue
		}
		cost := r.costs[id]
		result = append(result, &repositories.CostPresetResponse{
			ID:                        preset.ID.String(),
			Name:                      preset.Name,
			Description:               preset.Description,
			Type:                      string(preset.Type),
			IsActive:                  preset.IsActive,
			IsDefault:                 preset.IsDefault,
			CreatedAt:                 preset.CreatedAt,
			UpdatedAt:                 preset.UpdatedAt,
			LaborCostPerHour:          cost.LaborCostPerHour,
			PackagingCostPerItem:      cost.PackagingCostPerItem,
			ShippingCostBase:          cost.ShippingCostBase,
			ShippingCostPerGram:       cost.ShippingCostPerGram,
			OverheadPercentage:        cost.OverheadPercentage,
			ProfitMarginPercentage:    cost.ProfitMarginPercentage,
			PostProcessingCostPerHour: cost.PostProcessingCostPerHour,
			SupportRemovalCostPerHour: cost.SupportRemovalCostPerHour,
			QualityControlCostPerItem: cost.QualityControlCostPerItem,
		})
	}
	return result, nil
}
