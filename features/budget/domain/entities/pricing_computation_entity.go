package entities

import "github.com/google/uuid"

// PricingFilamentSpec describes one filament used by an item for a pricing run.
// Grams is the TOTAL grams of this filament for the item.
type PricingFilamentSpec struct {
	FilamentID uuid.UUID
	Quantity   float64
}

// PricingItemSpec describes one product line for a pricing run. It is the
// persistence-agnostic shape consumed by the repository's pricing computation,
// so it is used both for stored budgets (CalculateCosts) and for the stateless
// preview endpoint.
type PricingItemSpec struct {
	ProductQuantity         int
	PrintTimeHours          int
	PrintTimeMinutes        int
	SetupTimeMinutes        int
	ManualLaborMinutesTotal int
	CostPresetID            *uuid.UUID
	Filaments               []PricingFilamentSpec
}

// PricingComputationInput is everything the repository needs to load the
// org-scoped rates (filament prices, machine/energy/cost presets) and run the
// pure pricing engine, WITHOUT persisting anything.
type PricingComputationInput struct {
	OrganizationID     string
	IncludeEnergyCost  bool
	IncludeWasteCost   bool
	MachinePresetID    *uuid.UUID
	EnergyPresetID     *uuid.UUID
	BudgetCostPresetID *uuid.UUID
	Items              []PricingItemSpec
}
