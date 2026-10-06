package services

import "testing"

func TestFilamentCostCents(t *testing.T) {
	tests := []struct {
		name       string
		grams      float64
		pricePerKg float64 // cents per kg
		want       int64   // cents
	}{
		{name: "100g at R$120/kg = R$12,00", grams: 100, pricePerKg: 12000, want: 1200},
		{name: "1kg at R$120/kg = R$120,00", grams: 1000, pricePerKg: 12000, want: 12000},
		{name: "250g at R$80/kg = R$20,00", grams: 250, pricePerKg: 8000, want: 2000},
		{name: "rounds to nearest cent", grams: 33.333, pricePerKg: 10000, want: 333}, // 333.33 cents -> 333
		{name: "rounds up", grams: 1, pricePerKg: 15500, want: 16},                    // 15.5 cents -> 16
		{name: "zero grams", grams: 0, pricePerKg: 12000, want: 0},
		{name: "zero price", grams: 100, pricePerKg: 0, want: 0},
		{name: "negative guarded", grams: -100, pricePerKg: 12000, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilamentCostCents(tt.grams, tt.pricePerKg); got != tt.want {
				t.Errorf("FilamentCostCents(%v, %v) = %d, want %d", tt.grams, tt.pricePerKg, got, tt.want)
			}
		})
	}
}

func TestFilamentCostCents_MultiFilamentSum(t *testing.T) {
	// A single item with multiple filaments: cost is the sum of each filament's cost.
	type filament struct {
		grams      float64
		pricePerKg float64
	}
	filaments := []filament{
		{grams: 100, pricePerKg: 12000}, // 1200
		{grams: 50, pricePerKg: 20000},  // 1000
		{grams: 25, pricePerKg: 8000},   // 200
	}

	var total int64
	for _, f := range filaments {
		total += FilamentCostCents(f.grams, f.pricePerKg)
	}

	const want int64 = 2400
	if total != want {
		t.Errorf("multi-filament total = %d, want %d", total, want)
	}
}

func TestWasteCostCents(t *testing.T) {
	// 15g waste per color change; 2 changes => 30g at avg 12000 cents/kg (R$120) => R$3,60 => 360 cents.
	if got := WasteCostCents(30, 12000); got != 360 {
		t.Errorf("WasteCostCents(30, 12000) = %d, want 360", got)
	}
}

func TestEnergyCostCents(t *testing.T) {
	tests := []struct {
		name       string
		powerWatts float64
		hours      float64
		pricePerKw float64
		want       int64
	}{
		// 200W for 10h = 2kWh; at R$1,00/kWh => R$2,00 => 200 cents.
		{name: "2 kWh at R$1/kWh", powerWatts: 200, hours: 10, pricePerKw: 1.00, want: 200},
		// 1000W for 1h = 1kWh; at R$0,75/kWh => R$0,75 => 75 cents.
		{name: "1 kWh at R$0,75", powerWatts: 1000, hours: 1, pricePerKw: 0.75, want: 75},
		{name: "zero power", powerWatts: 0, hours: 10, pricePerKw: 1, want: 0},
		{name: "zero hours", powerWatts: 200, hours: 0, pricePerKw: 1, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EnergyCostCents(tt.powerWatts, tt.hours, tt.pricePerKw); got != tt.want {
				t.Errorf("EnergyCostCents(%v, %v, %v) = %d, want %d", tt.powerWatts, tt.hours, tt.pricePerKw, got, tt.want)
			}
		})
	}
}

func TestLaborCostCents(t *testing.T) {
	tests := []struct {
		name      string
		minutes   int
		ratePerHr float64
		want      int64
	}{
		// 30 min at R$60/h => R$30,00 => 3000 cents.
		{name: "30 min at R$60/h", minutes: 30, ratePerHr: 60.00, want: 3000},
		// 90 min at R$40/h => R$60,00 => 6000 cents.
		{name: "90 min at R$40/h", minutes: 90, ratePerHr: 40.00, want: 6000},
		{name: "zero minutes", minutes: 0, ratePerHr: 60, want: 0},
		{name: "zero rate", minutes: 30, ratePerHr: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LaborCostCents(tt.minutes, tt.ratePerHr); got != tt.want {
				t.Errorf("LaborCostCents(%d, %v) = %d, want %d", tt.minutes, tt.ratePerHr, got, tt.want)
			}
		})
	}
}

func TestPercentageCents(t *testing.T) {
	tests := []struct {
		name    string
		base    int64
		percent float64
		want    int64
	}{
		{name: "15% of 10000 cents", base: 10000, percent: 15, want: 1500},
		{name: "10% of 1250 cents rounds", base: 1250, percent: 10, want: 125},
		{name: "33% of 100 cents rounds", base: 100, percent: 33, want: 33},
		{name: "zero percent", base: 10000, percent: 0, want: 0},
		{name: "zero base", base: 0, percent: 15, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PercentageCents(tt.base, tt.percent); got != tt.want {
				t.Errorf("PercentageCents(%d, %v) = %d, want %d", tt.base, tt.percent, got, tt.want)
			}
		})
	}
}

func TestUnitPriceCents(t *testing.T) {
	tests := []struct {
		name     string
		total    int64
		quantity int
		want     int64
	}{
		{name: "even division", total: 1200, quantity: 4, want: 300},
		{name: "rounds to nearest cent", total: 1000, quantity: 3, want: 333},
		{name: "zero quantity guarded", total: 1000, quantity: 0, want: 0},
		{name: "negative quantity guarded", total: 1000, quantity: -2, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnitPriceCents(tt.total, tt.quantity); got != tt.want {
				t.Errorf("UnitPriceCents(%d, %d) = %d, want %d", tt.total, tt.quantity, got, tt.want)
			}
		})
	}
}

// TestOverheadAndProfitChain verifies the overhead-then-profit chaining used by
// CalculateCosts: overhead is applied to the subtotal, profit to (subtotal + overhead).
func TestOverheadAndProfitChain(t *testing.T) {
	const subtotal int64 = 10000 // R$100,00

	overhead := PercentageCents(subtotal, 10)        // R$10,00 => 1000
	profit := PercentageCents(subtotal+overhead, 20) // 20% of 11000 => 2200
	total := subtotal + overhead + profit            // 13200

	if overhead != 1000 {
		t.Errorf("overhead = %d, want 1000", overhead)
	}
	if profit != 2200 {
		t.Errorf("profit = %d, want 2200", profit)
	}
	if total != 13200 {
		t.Errorf("total = %d, want 13200", total)
	}
}
