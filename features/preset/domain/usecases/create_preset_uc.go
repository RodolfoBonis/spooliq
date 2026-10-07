package usecases

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	"github.com/google/uuid"
)

// CreatePresetUseCase handles creating new presets
type CreatePresetUseCase struct {
	presetRepo repositories.PresetRepository
}

// NewCreatePresetUseCase creates a new instance of CreatePresetUseCase
func NewCreatePresetUseCase(presetRepo repositories.PresetRepository) *CreatePresetUseCase {
	return &CreatePresetUseCase{
		presetRepo: presetRepo,
	}
}

// CreateMachinePresetRequest represents the request to create a machine preset.
// Name is optional: when empty it is auto-generated from the brand, model and
// nozzle. UserID is kept for backwards compatibility but is NOT trusted: the
// owning user is always derived from the authenticated context.
type CreateMachinePresetRequest struct {
	Name                   string     `json:"name"`
	Description            string     `json:"description"`
	IsDefault              bool       `json:"is_default"`
	UserID                 *uuid.UUID `json:"user_id"` // Deprecated: ignored; derived from the authenticated user.
	Brand                  string     `json:"brand"`
	Model                  string     `json:"model"`
	BuildVolumeX           float32    `json:"build_volume_x" binding:"required,gt=0"`
	BuildVolumeY           float32    `json:"build_volume_y" binding:"required,gt=0"`
	BuildVolumeZ           float32    `json:"build_volume_z" binding:"required,gt=0"`
	NozzleDiameter         float32    `json:"nozzle_diameter" binding:"required,gt=0"`
	LayerHeightMin         float32    `json:"layer_height_min" binding:"required,gt=0"`
	LayerHeightMax         float32    `json:"layer_height_max" binding:"required,gt=0"`
	PrintSpeedMax          float32    `json:"print_speed_max" binding:"required,gt=0"`
	PowerConsumption       float32    `json:"power_consumption" binding:"required,gt=0"`
	BedTemperatureMax      float32    `json:"bed_temperature_max"`
	ExtruderTemperatureMax float32    `json:"extruder_temperature_max"`
	FilamentDiameter       float32    `json:"filament_diameter" binding:"required,gt=0"`
	CostPerHour            float32    `json:"cost_per_hour"`
}

// CreateEnergyPresetRequest represents the request to create an energy preset.
// Name is optional (auto-generated from provider/city/state and tariff when
// empty). Peak/off-peak multipliers default to 1.0 when omitted. UserID is kept
// for backwards compatibility but is NOT trusted: the owning user is always
// derived from the authenticated context.
type CreateEnergyPresetRequest struct {
	Name                  string     `json:"name"`
	Description           string     `json:"description"`
	IsDefault             bool       `json:"is_default"`
	UserID                *uuid.UUID `json:"user_id"` // Deprecated: ignored; derived from the authenticated user.
	Country               string     `json:"country"`
	State                 string     `json:"state"`
	City                  string     `json:"city"`
	EnergyCostPerKwh      float32    `json:"energy_cost_per_kwh" binding:"required,gt=0"`
	Currency              string     `json:"currency" binding:"required,len=3"`
	Provider              string     `json:"provider"`
	TariffType            string     `json:"tariff_type"`
	PeakHourMultiplier    float32    `json:"peak_hour_multiplier" binding:"omitempty,gt=0"`
	OffPeakHourMultiplier float32    `json:"off_peak_hour_multiplier" binding:"omitempty,gt=0"`
}

// CreateCostPresetRequest represents the request to create a cost preset.
// Name is optional (auto-generated from labor rate and profit margin when empty).
// UserID is kept for backwards compatibility but is NOT trusted: the owning user
// is always derived from the authenticated context.
type CreateCostPresetRequest struct {
	Name                      string     `json:"name"`
	Description               string     `json:"description"`
	IsDefault                 bool       `json:"is_default"`
	UserID                    *uuid.UUID `json:"user_id"` // Deprecated: ignored; derived from the authenticated user.
	LaborCostPerHour          float32    `json:"labor_cost_per_hour" binding:"min=0"`
	PackagingCostPerItem      float32    `json:"packaging_cost_per_item" binding:"min=0"`
	ShippingCostBase          float32    `json:"shipping_cost_base" binding:"min=0"`
	ShippingCostPerGram       float32    `json:"shipping_cost_per_gram" binding:"min=0"`
	OverheadPercentage        float32    `json:"overhead_percentage" binding:"min=0,max=100"`
	ProfitMarginPercentage    float32    `json:"profit_margin_percentage" binding:"min=0,max=1000"`
	PostProcessingCostPerHour float32    `json:"post_processing_cost_per_hour" binding:"min=0"`
	SupportRemovalCostPerHour float32    `json:"support_removal_cost_per_hour" binding:"min=0"`
	QualityControlCostPerItem float32    `json:"quality_control_cost_per_item" binding:"min=0"`
	FailureRatePercentage     float32    `json:"failure_rate_percentage" binding:"min=0,max=100"`
	// WasteGramsPerColorChange defaults to 15 when omitted (0) to match the engine default.
	WasteGramsPerColorChange float32 `json:"waste_grams_per_color_change" binding:"min=0"`
}

// CreateMachinePreset creates a new machine preset. The owning user is taken
// from the authenticated context (userID), never from the request body. When the
// name is empty it is auto-generated from the brand, model and nozzle.
func (uc *CreatePresetUseCase) CreateMachinePreset(req *CreateMachinePresetRequest, organizationID string, userID *uuid.UUID) (*entities.PresetEntity, error) {
	name := req.Name
	if name == "" {
		name = entities.GenerateMachineName(req.Brand, req.Model, req.NozzleDiameter)
	}

	preset := &entities.PresetEntity{
		ID:             uuid.New(),
		Name:           name,
		Description:    req.Description,
		Type:           entities.PresetTypeMachine,
		IsActive:       true,
		IsDefault:      req.IsDefault,
		UserID:         userID,
		OrganizationID: organizationID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := preset.Validate(); err != nil {
		return nil, err
	}

	machine := &entities.MachinePresetEntity{
		ID:                     preset.ID,
		OrganizationID:         organizationID,
		Brand:                  req.Brand,
		Model:                  req.Model,
		BuildVolumeX:           req.BuildVolumeX,
		BuildVolumeY:           req.BuildVolumeY,
		BuildVolumeZ:           req.BuildVolumeZ,
		NozzleDiameter:         req.NozzleDiameter,
		LayerHeightMin:         req.LayerHeightMin,
		LayerHeightMax:         req.LayerHeightMax,
		PrintSpeedMax:          req.PrintSpeedMax,
		PowerConsumption:       req.PowerConsumption,
		BedTemperatureMax:      req.BedTemperatureMax,
		ExtruderTemperatureMax: req.ExtruderTemperatureMax,
		FilamentDiameter:       req.FilamentDiameter,
		CostPerHour:            req.CostPerHour,
	}

	if err := machine.Validate(); err != nil {
		return nil, err
	}

	if err := uc.presetRepo.CreateMachine(preset, machine); err != nil {
		return nil, err
	}

	return preset, nil
}

// CreateEnergyPreset creates a new energy preset. The owning user is taken from
// the authenticated context (userID), never from the request body. Name is
// auto-generated when empty and the peak/off-peak multipliers default to 1.0.
func (uc *CreatePresetUseCase) CreateEnergyPreset(req *CreateEnergyPresetRequest, organizationID string, userID *uuid.UUID) (*entities.PresetEntity, error) {
	name := req.Name
	if name == "" {
		name = entities.GenerateEnergyName(req.Provider, req.City, req.State, req.EnergyCostPerKwh)
	}

	preset := &entities.PresetEntity{
		ID:             uuid.New(),
		Name:           name,
		Description:    req.Description,
		Type:           entities.PresetTypeEnergy,
		IsActive:       true,
		IsDefault:      req.IsDefault,
		UserID:         userID,
		OrganizationID: organizationID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := preset.Validate(); err != nil {
		return nil, err
	}

	// Peak/off-peak multipliers default to 1.0 (no peak/off-peak pricing) when
	// omitted, so a simple flat tariff needs no extra fields.
	peak := req.PeakHourMultiplier
	if peak == 0 {
		peak = 1.0
	}
	offPeak := req.OffPeakHourMultiplier
	if offPeak == 0 {
		offPeak = 1.0
	}

	energy := &entities.EnergyPresetEntity{
		ID:                    preset.ID,
		OrganizationID:        organizationID,
		Country:               req.Country,
		State:                 req.State,
		City:                  req.City,
		EnergyCostPerKwh:      req.EnergyCostPerKwh,
		Currency:              req.Currency,
		Provider:              req.Provider,
		TariffType:            req.TariffType,
		PeakHourMultiplier:    peak,
		OffPeakHourMultiplier: offPeak,
	}

	if err := energy.Validate(); err != nil {
		return nil, err
	}

	if err := uc.presetRepo.CreateEnergy(preset, energy); err != nil {
		return nil, err
	}

	return preset, nil
}

// CreateCostPreset creates a new cost preset. The owning user is taken from the
// authenticated context (userID), never from the request body. Name is
// auto-generated from the labor rate and profit margin when empty.
func (uc *CreatePresetUseCase) CreateCostPreset(req *CreateCostPresetRequest, organizationID string, userID *uuid.UUID) (*entities.PresetEntity, error) {
	name := req.Name
	if name == "" {
		name = entities.GenerateCostName(req.LaborCostPerHour, req.ProfitMarginPercentage)
	}

	preset := &entities.PresetEntity{
		ID:             uuid.New(),
		Name:           name,
		Description:    req.Description,
		Type:           entities.PresetTypeCost,
		IsActive:       true,
		IsDefault:      req.IsDefault,
		UserID:         userID,
		OrganizationID: organizationID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := preset.Validate(); err != nil {
		return nil, err
	}

	// Default waste-per-color-change to the engine constant (15g) when omitted, so a
	// new preset reproduces the legacy waste behaviour unless explicitly configured.
	wasteGrams := req.WasteGramsPerColorChange
	if wasteGrams == 0 {
		wasteGrams = 15
	}

	cost := &entities.CostPresetEntity{
		ID:                        preset.ID,
		OrganizationID:            organizationID,
		LaborCostPerHour:          req.LaborCostPerHour,
		PackagingCostPerItem:      req.PackagingCostPerItem,
		ShippingCostBase:          req.ShippingCostBase,
		ShippingCostPerGram:       req.ShippingCostPerGram,
		OverheadPercentage:        req.OverheadPercentage,
		ProfitMarginPercentage:    req.ProfitMarginPercentage,
		PostProcessingCostPerHour: req.PostProcessingCostPerHour,
		SupportRemovalCostPerHour: req.SupportRemovalCostPerHour,
		QualityControlCostPerItem: req.QualityControlCostPerItem,
		FailureRatePercentage:     req.FailureRatePercentage,
		WasteGramsPerColorChange:  wasteGrams,
	}

	if err := cost.Validate(); err != nil {
		return nil, err
	}

	if err := uc.presetRepo.CreateCost(preset, cost); err != nil {
		return nil, err
	}

	return preset, nil
}

// FromTemplateOverrides holds the optional fields a caller may override when
// instantiating a preset from a template.
type FromTemplateOverrides struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

// CreateFromTemplate instantiates a preset in the organization from a static
// template, applying optional overrides (name and is_default). The owning user
// is taken from the authenticated context. Returns entities.ErrTemplateNotFound
// when the key is unknown.
func (uc *CreatePresetUseCase) CreateFromTemplate(key string, overrides FromTemplateOverrides, organizationID string, userID *uuid.UUID) (*entities.PresetEntity, error) {
	tmpl, ok := entities.TemplateByKey(key)
	if !ok {
		return nil, entities.ErrTemplateNotFound
	}

	name := overrides.Name
	if name == "" {
		name = tmpl.Name
	}

	switch tmpl.Type {
	case entities.PresetTypeMachine:
		m := tmpl.Machine
		return uc.CreateMachinePreset(&CreateMachinePresetRequest{
			Name:             name,
			Description:      tmpl.Description,
			IsDefault:        overrides.IsDefault,
			Brand:            m.Brand,
			Model:            m.Model,
			BuildVolumeX:     m.BuildVolumeX,
			BuildVolumeY:     m.BuildVolumeY,
			BuildVolumeZ:     m.BuildVolumeZ,
			NozzleDiameter:   m.NozzleDiameter,
			LayerHeightMin:   0.1,
			LayerHeightMax:   0.3,
			PrintSpeedMax:    100,
			PowerConsumption: m.PowerConsumption,
			FilamentDiameter: m.FilamentDiameter,
		}, organizationID, userID)
	case entities.PresetTypeEnergy:
		e := tmpl.Energy
		return uc.CreateEnergyPreset(&CreateEnergyPresetRequest{
			Name:             name,
			Description:      tmpl.Description,
			IsDefault:        overrides.IsDefault,
			Country:          e.Country,
			EnergyCostPerKwh: e.EnergyCostPerKwh,
			Currency:         e.Currency,
		}, organizationID, userID)
	case entities.PresetTypeCost:
		c := tmpl.Cost
		return uc.CreateCostPreset(&CreateCostPresetRequest{
			Name:                   name,
			Description:            tmpl.Description,
			IsDefault:              overrides.IsDefault,
			LaborCostPerHour:       c.LaborCostPerHour,
			OverheadPercentage:     c.OverheadPercentage,
			ProfitMarginPercentage: c.ProfitMarginPercentage,
		}, organizationID, userID)
	default:
		return nil, entities.ErrInvalidPresetType
	}
}
