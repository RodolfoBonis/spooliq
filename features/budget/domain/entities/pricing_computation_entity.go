package entities

import "github.com/google/uuid"

// PricingFilamentSpec describes one filament used by an item for a pricing run.
// Grams is the TOTAL grams of this filament for the item.
type PricingFilamentSpec struct {
	FilamentID uuid.UUID
	Quantity   float64
}

// PricingSupplySpec describes one non-printed supply (insumo) used by an item for a
// pricing run. The unit cost is resolved from the org-scoped catalog by SupplyID at
// compute time; QuantityPerPiece is how many units each printed piece consumes.
type PricingSupplySpec struct {
	SupplyID         uuid.UUID
	QuantityPerPiece float64
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
	PostProcessingMinutes   int
	SupportRemovalMinutes   int
	CostPresetID            *uuid.UUID
	Filaments               []PricingFilamentSpec
	Supplies                []PricingSupplySpec
}

// PricingComputationInput is everything the repository needs to load the
// org-scoped rates (filament prices, supply unit costs, machine/energy/cost
// presets, company default tax rate) and run the pure pricing engine, WITHOUT
// persisting anything.
type PricingComputationInput struct {
	OrganizationID     string
	IncludeEnergyCost  bool
	IncludeWasteCost   bool
	IncludeMachineCost bool
	MachinePresetID    *uuid.UUID
	EnergyPresetID     *uuid.UUID
	BudgetCostPresetID *uuid.UUID
	Items              []PricingItemSpec

	// Discount / shipping / tax inputs (budget-level).
	DiscountType          string
	DiscountValue         float64
	IncludeShipping       bool
	ShippingOverrideCents *int64
	// TaxRate is the budget-level tax rate (percent). When nil the repository uses the
	// company's default_tax_rate.
	TaxRate *float64
}
