package entities

import "testing"

func TestGenerateMachineName(t *testing.T) {
	tests := []struct {
		name     string
		brand    string
		model    string
		nozzle   float32
		expected string
	}{
		{"full", "Bambu Lab", "P1S", 0.4, "Bambu Lab P1S · bico 0.4mm"},
		{"brand and nozzle only", "Prusa", "", 0.6, "Prusa · bico 0.6mm"},
		{"model and nozzle only", "", "MK4", 0.4, "MK4 · bico 0.4mm"},
		{"identity only", "Creality", "Ender-3", 0, "Creality Ender-3"},
		{"nozzle only", "", "", 0.4, "bico 0.4mm"},
		{"trims spaces", "  Bambu Lab  ", "  A1  ", 0.4, "Bambu Lab A1 · bico 0.4mm"},
		{"empty -> fallback", "", "", 0, FallbackMachineName},
		{"negative nozzle dropped", "Anycubic", "Kobra", -1, "Anycubic Kobra"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := GenerateMachineName(tc.brand, tc.model, tc.nozzle); got != tc.expected {
				t.Fatalf("GenerateMachineName(%q,%q,%v) = %q, want %q", tc.brand, tc.model, tc.nozzle, got, tc.expected)
			}
		})
	}
}

func TestGenerateEnergyName(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		city     string
		state    string
		kwh      float32
		expected string
	}{
		{"provider state tariff", "CEMIG", "", "MG", 0.95, "CEMIG/MG · R$ 0,95/kWh"},
		{"city fallback", "", "Belo Horizonte", "MG", 0.75, "Belo Horizonte/MG · R$ 0,75/kWh"},
		{"provider preferred over city", "Enel", "São Paulo", "SP", 0.8, "Enel/SP · R$ 0,8/kWh"},
		{"state missing", "Light", "", "", 1.2, "Light · R$ 1,2/kWh"},
		{"tariff only", "", "", "", 0.6, "R$ 0,6/kWh"},
		{"location only", "Copel", "", "PR", 0, "Copel/PR"},
		{"state only", "", "", "RS", 0, "RS"},
		{"empty -> fallback", "", "", "", 0, FallbackEnergyName},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := GenerateEnergyName(tc.provider, tc.city, tc.state, tc.kwh); got != tc.expected {
				t.Fatalf("GenerateEnergyName(%q,%q,%q,%v) = %q, want %q", tc.provider, tc.city, tc.state, tc.kwh, got, tc.expected)
			}
		})
	}
}

func TestGenerateCostName(t *testing.T) {
	tests := []struct {
		name     string
		labor    float32
		profit   float32
		expected string
	}{
		{"full", 50, 20, "Mão de obra R$ 50/h · margem 20%"},
		{"labor with decimals", 35.5, 15, "Mão de obra R$ 35,5/h · margem 15%"},
		{"labor only", 40, 0, "Mão de obra R$ 40/h"},
		{"margin only", 0, 30, "margem 30%"},
		{"margin above 100", 60, 150, "Mão de obra R$ 60/h · margem 150%"},
		{"empty -> fallback", 0, 0, FallbackCostName},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := GenerateCostName(tc.labor, tc.profit); got != tc.expected {
				t.Fatalf("GenerateCostName(%v,%v) = %q, want %q", tc.labor, tc.profit, got, tc.expected)
			}
		})
	}
}
