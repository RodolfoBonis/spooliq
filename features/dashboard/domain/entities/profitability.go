package entities

// ProfitRow is one line of a profitability ranking (money in cents).
type ProfitRow struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Subtitle string  `json:"subtitle,omitempty"`
	ColorHex string  `json:"color_hex,omitempty"`
	Revenue  int64   `json:"revenue"` // net revenue (no tax/shipping)
	Profit   int64   `json:"profit"`
	Margin   float64 `json:"margin"` // profit / revenue * 100
	Count    int     `json:"count"`  // sales (customers/machines) or items (materials/filaments)
	Grams    float64 `json:"grams,omitempty"`
	Hours    float64 `json:"hours,omitempty"`
	// ProfitPerHour is set for machines (profit per print hour, cents).
	ProfitPerHour int64 `json:"profit_per_hour,omitempty"`
	// DiscountRate is set for customers: discount / price before discount * 100.
	DiscountRate float64 `json:"discount_rate,omitempty"`
	// Repeat is set for customers with more than one sale ever.
	Repeat bool `json:"repeat,omitempty"`
}

// ProfitabilityResponse breaks the period's profit down by dimension. Revenue and
// profit of a budget are allocated to its items by item cost share, and to the
// item's filaments by grams share.
type ProfitabilityResponse struct {
	ByMaterial    []ProfitRow `json:"by_material"`
	ByFilament    []ProfitRow `json:"by_filament"`
	ByCustomer    []ProfitRow `json:"by_customer"`
	ByMachine     []ProfitRow `json:"by_machine"`
	ByCostPreset  []ProfitRow `json:"by_cost_preset"`
	AverageMargin float64     `json:"average_margin"`
	Period        string      `json:"period"`
}
