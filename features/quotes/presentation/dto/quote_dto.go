package dto

import (
	"fmt"

	"github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
)

// CreateQuoteRequest representa a requisição para criar um orçamento
type CreateQuoteRequest struct {
	Title          string                       `json:"title" validate:"required,min=1,max=255"`
	Notes          string                       `json:"notes,omitempty"`
	FilamentLines  []CreateFilamentLineRequest  `json:"filament_lines" validate:"required,min=1"`
	MachineProfile *CreateMachineProfileRequest `json:"machine_profile,omitempty"`
	EnergyProfile  *CreateEnergyProfileRequest  `json:"energy_profile,omitempty"`
	CostProfile    *CreateCostProfileRequest    `json:"cost_profile,omitempty"`
	MarginProfile  *CreateMarginProfileRequest  `json:"margin_profile,omitempty"`
}

// UpdateQuoteRequest representa a requisição para atualizar um orçamento
type UpdateQuoteRequest struct {
	Title          string                       `json:"title" validate:"required,min=1,max=255"`
	Notes          string                       `json:"notes,omitempty"`
	FilamentLines  []UpdateFilamentLineRequest  `json:"filament_lines" validate:"required,min=1"`
	MachineProfile *UpdateMachineProfileRequest `json:"machine_profile,omitempty"`
	EnergyProfile  *UpdateEnergyProfileRequest  `json:"energy_profile,omitempty"`
	CostProfile    *UpdateCostProfileRequest    `json:"cost_profile,omitempty"`
	MarginProfile  *UpdateMarginProfileRequest  `json:"margin_profile,omitempty"`
}

// CreateFilamentLineRequest representa a requisição para criar uma linha de filamento
// Supports both automatic snapshot (via filament_id) and manual snapshot (via filament_snapshot_* fields)
type CreateFilamentLineRequest struct {
	// Option 1: Automatic snapshot from existing filament
	FilamentID *uint `json:"filament_id,omitempty" validate:"omitempty,min=1"`

	// Option 2: Manual snapshot data (required if filament_id not provided)
	FilamentSnapshotName          string   `json:"filament_snapshot_name,omitempty"`
	FilamentSnapshotBrand         string   `json:"filament_snapshot_brand,omitempty"`
	FilamentSnapshotMaterial      string   `json:"filament_snapshot_material,omitempty"`
	FilamentSnapshotColor         string   `json:"filament_snapshot_color,omitempty"`
	FilamentSnapshotColorHex      string   `json:"filament_snapshot_color_hex,omitempty"`
	FilamentSnapshotPricePerKg    float64  `json:"filament_snapshot_price_per_kg,omitempty" validate:"omitempty,min=0"`
	FilamentSnapshotPricePerMeter *float64 `json:"filament_snapshot_price_per_meter,omitempty" validate:"omitempty,min=0"`
	FilamentSnapshotURL           string   `json:"filament_snapshot_url,omitempty"`

	// Required fields
	WeightGrams  float64  `json:"weight_grams" validate:"required,min=0"`
	LengthMeters *float64 `json:"length_meters,omitempty" validate:"omitempty,min=0"`
}

// UpdateFilamentLineRequest representa a requisição para atualizar uma linha de filamento
type UpdateFilamentLineRequest struct {
	ID                            uint     `json:"id,omitempty"`
	FilamentSnapshotName          string   `json:"filament_snapshot_name" validate:"required"`
	FilamentSnapshotBrand         string   `json:"filament_snapshot_brand" validate:"required"`
	FilamentSnapshotMaterial      string   `json:"filament_snapshot_material" validate:"required"`
	FilamentSnapshotColor         string   `json:"filament_snapshot_color" validate:"required"`
	FilamentSnapshotColorHex      string   `json:"filament_snapshot_color_hex,omitempty"`
	FilamentSnapshotPricePerKg    float64  `json:"filament_snapshot_price_per_kg" validate:"required,min=0"`
	FilamentSnapshotPricePerMeter *float64 `json:"filament_snapshot_price_per_meter,omitempty" validate:"omitempty,min=0"`
	FilamentSnapshotURL           string   `json:"filament_snapshot_url,omitempty"`
	WeightGrams                   float64  `json:"weight_grams" validate:"required,min=0"`
	LengthMeters                  *float64 `json:"length_meters,omitempty" validate:"omitempty,min=0"`
}

// CreateMachineProfileRequest representa a requisição para criar um perfil de máquina
type CreateMachineProfileRequest struct {
	Name        string  `json:"name" validate:"required"`
	Brand       string  `json:"brand" validate:"required"`
	Model       string  `json:"model" validate:"required"`
	Watt        float64 `json:"watt" validate:"required,min=0"`
	IdleFactor  float64 `json:"idle_factor" validate:"min=0,max=1"`
	Description string  `json:"description,omitempty"`
	URL         string  `json:"url,omitempty" validate:"omitempty,url"`
}

// UpdateMachineProfileRequest representa a requisição para atualizar um perfil de máquina
type UpdateMachineProfileRequest struct {
	Name        string  `json:"name" validate:"required"`
	Brand       string  `json:"brand" validate:"required"`
	Model       string  `json:"model" validate:"required"`
	Watt        float64 `json:"watt" validate:"required,min=0"`
	IdleFactor  float64 `json:"idle_factor" validate:"min=0,max=1"`
	Description string  `json:"description,omitempty"`
	URL         string  `json:"url,omitempty" validate:"omitempty,url"`
}

// CreateEnergyProfileRequest representa a requisição para criar um perfil de energia
// Pode usar um preset existente (via preset_key) OU fornecer dados customizados
type CreateEnergyProfileRequest struct {
	// Opção 1: Referenciar um preset existente
	PresetKey string `json:"preset_key,omitempty" validate:"omitempty"`
	
	// Opção 2: Dados customizados (todos campos abaixo)
	Name          string  `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	BaseTariff    float64 `json:"base_tariff,omitempty" validate:"omitempty,min=0"`
	FlagSurcharge float64 `json:"flag_surcharge,omitempty" validate:"omitempty,min=0"`
	Location      string  `json:"location,omitempty" validate:"omitempty"`
	Year          int     `json:"year,omitempty" validate:"omitempty,min=2020,max=2030"`
	Description   string  `json:"description,omitempty"`
}

// UpdateEnergyProfileRequest representa a requisição para atualizar um perfil de energia
type UpdateEnergyProfileRequest struct {
	Name          string  `json:"name" validate:"required,min=1,max=255"`
	BaseTariff    float64 `json:"base_tariff" validate:"required,min=0"`
	FlagSurcharge float64 `json:"flag_surcharge" validate:"min=0"`
	Location      string  `json:"location" validate:"required"`
	Year          int     `json:"year" validate:"required,min=2020,max=2030"`
	Description   string  `json:"description,omitempty"`
}

// CreateCostProfileRequest representa a requisição para criar um perfil de custos
type CreateCostProfileRequest struct {
	WearPercentage float64 `json:"wear_percentage" validate:"min=0,max=100"`
	OverheadAmount float64 `json:"overhead_amount" validate:"min=0"`
	Description    string  `json:"description,omitempty"`
}

// UpdateCostProfileRequest representa a requisição para atualizar um perfil de custos
type UpdateCostProfileRequest struct {
	WearPercentage float64 `json:"wear_percentage" validate:"min=0,max=100"`
	OverheadAmount float64 `json:"overhead_amount" validate:"min=0"`
	Description    string  `json:"description,omitempty"`
}

// CreateMarginProfileRequest representa a requisição para criar um perfil de margens
type CreateMarginProfileRequest struct {
	PrintingOnlyMargin  float64 `json:"printing_only_margin" validate:"min=0"`
	PrintingPlusMargin  float64 `json:"printing_plus_margin" validate:"min=0"`
	FullServiceMargin   float64 `json:"full_service_margin" validate:"min=0"`
	OperatorRatePerHour float64 `json:"operator_rate_per_hour" validate:"min=0"`
	ModelerRatePerHour  float64 `json:"modeler_rate_per_hour" validate:"min=0"`
	Description         string  `json:"description,omitempty"`
}

// UpdateMarginProfileRequest representa a requisição para atualizar um perfil de margens
type UpdateMarginProfileRequest struct {
	PrintingOnlyMargin  float64 `json:"printing_only_margin" validate:"min=0"`
	PrintingPlusMargin  float64 `json:"printing_plus_margin" validate:"min=0"`
	FullServiceMargin   float64 `json:"full_service_margin" validate:"min=0"`
	OperatorRatePerHour float64 `json:"operator_rate_per_hour" validate:"min=0"`
	ModelerRatePerHour  float64 `json:"modeler_rate_per_hour" validate:"min=0"`
	Description         string  `json:"description,omitempty"`
}

// CalculateQuoteRequest representa a requisição para calcular um orçamento
type CalculateQuoteRequest struct {
	PrintTimeHours  float64 `json:"print_time_hours" validate:"required,min=0"`
	OperatorMinutes float64 `json:"operator_minutes" validate:"min=0"`
	ModelerMinutes  float64 `json:"modeler_minutes" validate:"min=0"`
	ServiceType     string  `json:"service_type" validate:"required,oneof=printing_only printing_plus full_service"`
}

// QuoteResponse representa a resposta de um orçamento
type QuoteResponse struct {
	ID             uint                    `json:"id"`
	Title          string                  `json:"title"`
	Notes          string                  `json:"notes,omitempty"`
	OwnerUserID    string                  `json:"owner_user_id"`
	CreatedAt      string                  `json:"created_at"`
	UpdatedAt      string                  `json:"updated_at"`
	FilamentLines  []FilamentLineResponse  `json:"filament_lines,omitempty"`
	MachineProfile *MachineProfileResponse `json:"machine_profile,omitempty"`
	EnergyProfile  *EnergyProfileResponse  `json:"energy_profile,omitempty"`
	CostProfile    *CostProfileResponse    `json:"cost_profile,omitempty"`
	MarginProfile  *MarginProfileResponse  `json:"margin_profile,omitempty"`
}

// FilamentLineResponse representa a resposta de uma linha de filamento
type FilamentLineResponse struct {
	ID                            uint     `json:"id"`
	QuoteID                       uint     `json:"quote_id"`
	FilamentSnapshotName          string   `json:"filament_snapshot_name"`
	FilamentSnapshotBrand         string   `json:"filament_snapshot_brand"`
	FilamentSnapshotMaterial      string   `json:"filament_snapshot_material"`
	FilamentSnapshotColor         string   `json:"filament_snapshot_color"`
	FilamentSnapshotColorHex      string   `json:"filament_snapshot_color_hex,omitempty"`
	FilamentSnapshotPricePerKg    float64  `json:"filament_snapshot_price_per_kg"`
	FilamentSnapshotPricePerMeter *float64 `json:"filament_snapshot_price_per_meter,omitempty"`
	FilamentSnapshotURL           string   `json:"filament_snapshot_url,omitempty"`
	WeightGrams                   float64  `json:"weight_grams"`
	LengthMeters                  *float64 `json:"length_meters,omitempty"`
	CreatedAt                     string   `json:"created_at"`
	UpdatedAt                     string   `json:"updated_at"`
}

// MachineProfileResponse representa a resposta de um perfil de máquina
type MachineProfileResponse struct {
	ID          uint    `json:"id"`
	QuoteID     uint    `json:"quote_id"`
	Name        string  `json:"name"`
	Brand       string  `json:"brand"`
	Model       string  `json:"model"`
	Watt        float64 `json:"watt"`
	IdleFactor  float64 `json:"idle_factor"`
	Description string  `json:"description,omitempty"`
	URL         string  `json:"url,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// EnergyProfileResponse representa a resposta de um perfil de energia
type EnergyProfileResponse struct {
	ID            uint    `json:"id"`
	QuoteID       uint    `json:"quote_id"`
	Name          string  `json:"name"`
	BaseTariff    float64 `json:"base_tariff"`
	FlagSurcharge float64 `json:"flag_surcharge"`
	Location      string  `json:"location"`
	Year          int     `json:"year"`
	Description   string  `json:"description,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// CostProfileResponse representa a resposta de um perfil de custos
type CostProfileResponse struct {
	ID             uint    `json:"id"`
	QuoteID        uint    `json:"quote_id"`
	WearPercentage float64 `json:"wear_percentage"`
	OverheadAmount float64 `json:"overhead_amount"`
	Description    string  `json:"description,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// MarginProfileResponse representa a resposta de um perfil de margens
type MarginProfileResponse struct {
	ID                  uint    `json:"id"`
	QuoteID             uint    `json:"quote_id"`
	PrintingOnlyMargin  float64 `json:"printing_only_margin"`
	PrintingPlusMargin  float64 `json:"printing_plus_margin"`
	FullServiceMargin   float64 `json:"full_service_margin"`
	OperatorRatePerHour float64 `json:"operator_rate_per_hour"`
	ModelerRatePerHour  float64 `json:"modeler_rate_per_hour"`
	Description         string  `json:"description,omitempty"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

// CalculationResult representa o resultado de um cálculo de orçamento
type CalculationResult struct {
	MaterialCost    float64 `json:"material_cost"`
	EnergyCost      float64 `json:"energy_cost"`
	WearCost        float64 `json:"wear_cost"`
	LaborCost       float64 `json:"labor_cost"`
	DirectCost      float64 `json:"direct_cost"`
	FinalPrice      float64 `json:"final_price"`
	PrintTimeHours  float64 `json:"print_time_hours"`
	OperatorMinutes float64 `json:"operator_minutes"`
	ModelerMinutes  float64 `json:"modeler_minutes"`
	ServiceType     string  `json:"service_type"`
	AppliedMargin   float64 `json:"applied_margin"`
}

// ToQuoteResponse converte uma entidade Quote para QuoteResponse
func ToQuoteResponse(quote *entities.Quote) *QuoteResponse {
	response := &QuoteResponse{
		ID:          quote.ID,
		Title:       quote.Title,
		Notes:       quote.Notes,
		OwnerUserID: quote.OwnerUserID,
		CreatedAt:   quote.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   quote.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Convert filament lines
	if len(quote.FilamentLines) > 0 {
		response.FilamentLines = make([]FilamentLineResponse, 0, len(quote.FilamentLines))
		for _, line := range quote.FilamentLines {
			response.FilamentLines = append(response.FilamentLines, FilamentLineResponse{
				ID:                            line.ID,
				QuoteID:                       line.QuoteID,
				FilamentSnapshotName:          line.FilamentSnapshotName,
				FilamentSnapshotBrand:         line.FilamentSnapshotBrand,
				FilamentSnapshotMaterial:      line.FilamentSnapshotMaterial,
				FilamentSnapshotColor:         line.FilamentSnapshotColor,
				FilamentSnapshotColorHex:      line.FilamentSnapshotColorHex,
				FilamentSnapshotPricePerKg:    line.FilamentSnapshotPricePerKg,
				FilamentSnapshotPricePerMeter: line.FilamentSnapshotPricePerMeter,
				FilamentSnapshotURL:           line.FilamentSnapshotURL,
				WeightGrams:                   line.WeightGrams,
				LengthMeters:                  line.LengthMeters,
				CreatedAt:                     line.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:                     line.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
	}

	// Convert profiles
	if quote.MachineProfile != nil {
		response.MachineProfile = &MachineProfileResponse{
			ID:          quote.MachineProfile.ID,
			QuoteID:     quote.MachineProfile.QuoteID,
			Name:        quote.MachineProfile.Name,
			Brand:       quote.MachineProfile.Brand,
			Model:       quote.MachineProfile.Model,
			Watt:        quote.MachineProfile.Watt,
			IdleFactor:  quote.MachineProfile.IdleFactor,
			Description: quote.MachineProfile.Description,
			URL:         quote.MachineProfile.URL,
			CreatedAt:   quote.MachineProfile.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   quote.MachineProfile.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	if quote.EnergyProfile != nil {
		response.EnergyProfile = &EnergyProfileResponse{
			ID:            quote.EnergyProfile.ID,
			QuoteID:       quote.EnergyProfile.QuoteID,
			Name:          quote.EnergyProfile.Name,
			BaseTariff:    quote.EnergyProfile.BaseTariff,
			FlagSurcharge: quote.EnergyProfile.FlagSurcharge,
			Location:      quote.EnergyProfile.Location,
			Year:          quote.EnergyProfile.Year,
			Description:   quote.EnergyProfile.Description,
			CreatedAt:     quote.EnergyProfile.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:     quote.EnergyProfile.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	if quote.CostProfile != nil {
		response.CostProfile = &CostProfileResponse{
			ID:             quote.CostProfile.ID,
			QuoteID:        quote.CostProfile.QuoteID,
			WearPercentage: quote.CostProfile.WearPercentage,
			OverheadAmount: quote.CostProfile.OverheadAmount,
			Description:    quote.CostProfile.Description,
			CreatedAt:      quote.CostProfile.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:      quote.CostProfile.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	if quote.MarginProfile != nil {
		response.MarginProfile = &MarginProfileResponse{
			ID:                  quote.MarginProfile.ID,
			QuoteID:             quote.MarginProfile.QuoteID,
			PrintingOnlyMargin:  quote.MarginProfile.PrintingOnlyMargin,
			PrintingPlusMargin:  quote.MarginProfile.PrintingPlusMargin,
			FullServiceMargin:   quote.MarginProfile.FullServiceMargin,
			OperatorRatePerHour: quote.MarginProfile.OperatorRatePerHour,
			ModelerRatePerHour:  quote.MarginProfile.ModelerRatePerHour,
			Description:         quote.MarginProfile.Description,
			CreatedAt:           quote.MarginProfile.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:           quote.MarginProfile.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return response
}

// Validate validates the CreateEnergyProfileRequest ensuring either PresetKey or complete data is provided
func (req *CreateEnergyProfileRequest) Validate() error {
	// Option 1: Using a preset
	if req.PresetKey != "" {
		// If using preset, no other fields should be provided
		if req.Name != "" || req.BaseTariff != 0 || req.Location != "" || req.Year != 0 {
			return fmt.Errorf("when using preset_key, other fields should not be provided")
		}
		return nil
	}
	
	// Option 2: Custom data - at least location and year are required
	if req.Location == "" {
		return fmt.Errorf("location is required when not using preset_key")
	}
	if req.Year == 0 {
		return fmt.Errorf("year is required when not using preset_key")
	}
	if req.BaseTariff == 0 {
		return fmt.Errorf("base_tariff is required when not using preset_key")
	}
	
	// Name can be auto-generated if not provided
	return nil
}

// Validate validates the CreateFilamentLineRequest ensuring either FilamentID or manual snapshot data is provided
func (req *CreateFilamentLineRequest) Validate() error {
	// Either filament_id OR manual snapshot data must be provided
	if req.FilamentID != nil {
		// If filament_id is provided, it should be valid
		if *req.FilamentID == 0 {
			return fmt.Errorf("filament_id must be greater than 0")
		}
		// Manual snapshot data is not required when filament_id is provided
		return nil
	}

	// If filament_id is not provided, manual snapshot data is required
	if req.FilamentSnapshotName == "" {
		return fmt.Errorf("filament_snapshot_name is required when filament_id is not provided")
	}
	if req.FilamentSnapshotBrand == "" {
		return fmt.Errorf("filament_snapshot_brand is required when filament_id is not provided")
	}
	if req.FilamentSnapshotMaterial == "" {
		return fmt.Errorf("filament_snapshot_material is required when filament_id is not provided")
	}
	if req.FilamentSnapshotColor == "" {
		return fmt.Errorf("filament_snapshot_color is required when filament_id is not provided")
	}
	if req.FilamentSnapshotPricePerKg <= 0 {
		return fmt.Errorf("filament_snapshot_price_per_kg must be greater than 0 when filament_id is not provided")
	}

	return nil
}
