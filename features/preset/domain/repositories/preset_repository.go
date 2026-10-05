package repositories

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/google/uuid"
)

// MachinePresetResponse represents a complete machine preset response
type MachinePresetResponse struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Description            string    `json:"description,omitempty"`
	Type                   string    `json:"type"`
	IsActive               bool      `json:"is_active"`
	IsDefault              bool      `json:"is_default"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
	Brand                  string    `json:"brand,omitempty"`
	Model                  string    `json:"model,omitempty"`
	BuildVolumeX           float32   `json:"build_volume_x"`
	BuildVolumeY           float32   `json:"build_volume_y"`
	BuildVolumeZ           float32   `json:"build_volume_z"`
	NozzleDiameter         float32   `json:"nozzle_diameter"`
	LayerHeightMin         float32   `json:"layer_height_min"`
	LayerHeightMax         float32   `json:"layer_height_max"`
	PrintSpeedMax          float32   `json:"print_speed_max"`
	PowerConsumption       float32   `json:"power_consumption"`
	BedTemperatureMax      float32   `json:"bed_temperature_max"`
	ExtruderTemperatureMax float32   `json:"extruder_temperature_max"`
	FilamentDiameter       float32   `json:"filament_diameter"`
	CostPerHour            float32   `json:"cost_per_hour"`
}

// EnergyPresetResponse represents a complete energy preset response
type EnergyPresetResponse struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	Description           string    `json:"description,omitempty"`
	Type                  string    `json:"type"`
	IsActive              bool      `json:"is_active"`
	IsDefault             bool      `json:"is_default"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	Country               string    `json:"country,omitempty"`
	State                 string    `json:"state,omitempty"`
	City                  string    `json:"city,omitempty"`
	EnergyCostPerKwh      float32   `json:"energy_cost_per_kwh"`
	Currency              string    `json:"currency"`
	Provider              string    `json:"provider,omitempty"`
	TariffType            string    `json:"tariff_type,omitempty"`
	PeakHourMultiplier    float32   `json:"peak_hour_multiplier"`
	OffPeakHourMultiplier float32   `json:"off_peak_hour_multiplier"`
}

// CostPresetResponse represents a complete cost preset response
type CostPresetResponse struct {
	ID                        string    `json:"id"`
	Name                      string    `json:"name"`
	Description               string    `json:"description,omitempty"`
	Type                      string    `json:"type"`
	IsActive                  bool      `json:"is_active"`
	IsDefault                 bool      `json:"is_default"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
	LaborCostPerHour          float32   `json:"labor_cost_per_hour"`
	PackagingCostPerItem      float32   `json:"packaging_cost_per_item"`
	ShippingCostBase          float32   `json:"shipping_cost_base"`
	ShippingCostPerGram       float32   `json:"shipping_cost_per_gram"`
	OverheadPercentage        float32   `json:"overhead_percentage"`
	ProfitMarginPercentage    float32   `json:"profit_margin_percentage"`
	PostProcessingCostPerHour float32   `json:"post_processing_cost_per_hour"`
	SupportRemovalCostPerHour float32   `json:"support_removal_cost_per_hour"`
	QualityControlCostPerItem float32   `json:"quality_control_cost_per_item"`
}

// PresetRepository defines the contract for preset data operations.
//
// Every read, update and delete operation is scoped by organizationID to
// guarantee tenant isolation. Cross-organization access must surface as a
// "record not found" error so callers can map it to HTTP 404 (never 403),
// avoiding leaking the existence of other tenants' presets.
type PresetRepository interface {
	// Base preset operations
	Create(preset *entities.PresetEntity) error
	GetByID(id uuid.UUID, organizationID string) (*entities.PresetEntity, error)
	// ListPresets returns presets for an organization, applying the given
	// combinable filters (type, active, default, global, user) in a single query.
	ListPresets(organizationID string, filters entities.PresetFilters) ([]*entities.PresetEntity, error)
	Update(preset *entities.PresetEntity) error
	Delete(id uuid.UUID, organizationID string) error

	// SetDefault marks the preset as the single default for its (organization,
	// type) pair, clearing any other default of the same type in the same
	// transaction. Returns the updated base preset, or gorm.ErrRecordNotFound
	// when the preset does not exist within the organization.
	SetDefault(id uuid.UUID, organizationID string) (*entities.PresetEntity, error)
	// Duplicate copies a preset (base + type-specific child) within the same
	// organization in a single transaction. The copy is never a default and
	// takes the given name. Returns the new base preset.
	Duplicate(id uuid.UUID, organizationID string, newName string) (*entities.PresetEntity, error)

	// Machine preset operations
	CreateMachine(preset *entities.PresetEntity, machine *entities.MachinePresetEntity) error
	GetMachineByID(id uuid.UUID, organizationID string) (*entities.MachinePresetEntity, error)
	GetMachinesByBrand(brand, organizationID string) ([]*entities.MachinePresetEntity, error)
	UpdateMachine(machine *entities.MachinePresetEntity) error
	// UpdateMachineWithPreset atomically updates the base preset and its machine
	// child in a single, organization-scoped transaction.
	UpdateMachineWithPreset(preset *entities.PresetEntity, machine *entities.MachinePresetEntity) error
	// Optimized methods with organization filtering - return ready-to-use responses
	GetMachinePresets(organizationID string) ([]*MachinePresetResponse, error)
	GetMachinePresetsByBrand(brand, organizationID string) ([]*MachinePresetResponse, error)

	// Energy preset operations
	CreateEnergy(preset *entities.PresetEntity, energy *entities.EnergyPresetEntity) error
	GetEnergyByID(id uuid.UUID, organizationID string) (*entities.EnergyPresetEntity, error)
	GetEnergyByLocation(country, state, city, organizationID string) ([]*entities.EnergyPresetEntity, error)
	GetEnergyByCurrency(currency, organizationID string) ([]*entities.EnergyPresetEntity, error)
	UpdateEnergy(energy *entities.EnergyPresetEntity) error
	// UpdateEnergyWithPreset atomically updates the base preset and its energy
	// child in a single, organization-scoped transaction.
	UpdateEnergyWithPreset(preset *entities.PresetEntity, energy *entities.EnergyPresetEntity) error
	// Optimized methods with organization filtering - return ready-to-use responses
	GetEnergyPresets(organizationID string) ([]*EnergyPresetResponse, error)
	GetEnergyPresetsByLocation(country, state, city, organizationID string) ([]*EnergyPresetResponse, error)
	GetEnergyPresetsByCurrency(currency, organizationID string) ([]*EnergyPresetResponse, error)

	// Cost preset operations
	CreateCost(preset *entities.PresetEntity, cost *entities.CostPresetEntity) error
	GetCostByID(id uuid.UUID, organizationID string) (*entities.CostPresetEntity, error)
	UpdateCost(cost *entities.CostPresetEntity) error
	// UpdateCostWithPreset atomically updates the base preset and its cost child
	// in a single, organization-scoped transaction.
	UpdateCostWithPreset(preset *entities.PresetEntity, cost *entities.CostPresetEntity) error
	// Optimized methods with organization filtering - return ready-to-use responses
	GetCostPresets(organizationID string) ([]*CostPresetResponse, error)
}
