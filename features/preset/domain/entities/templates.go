package entities

// Preset templates are a STATIC, in-code catalog of realistic starting points
// users can clone into their organization and then adjust. They are intentionally
// approximate: power draw, build volumes and especially energy tariffs vary by
// unit, firmware, print settings and region. Every template description tells the
// user to review and tune the numbers for their own setup.
//
// No template is persisted on its own; `POST /presets/from-template/:key` copies
// one into the caller's organization as a regular preset.

// MachineTemplateFields holds the machine-specific values of a template.
type MachineTemplateFields struct {
	Brand            string  `json:"brand"`
	Model            string  `json:"model"`
	BuildVolumeX     float32 `json:"build_volume_x"`
	BuildVolumeY     float32 `json:"build_volume_y"`
	BuildVolumeZ     float32 `json:"build_volume_z"`
	NozzleDiameter   float32 `json:"nozzle_diameter"`
	FilamentDiameter float32 `json:"filament_diameter"`
	PowerConsumption float32 `json:"power_consumption"`
}

// EnergyTemplateFields holds the energy-specific values of a template.
type EnergyTemplateFields struct {
	Country          string  `json:"country"`
	EnergyCostPerKwh float32 `json:"energy_cost_per_kwh"`
	Currency         string  `json:"currency"`
}

// CostTemplateFields holds the cost-specific values of a template. Money fields
// are in reais; overhead and profit margin are percentages.
type CostTemplateFields struct {
	LaborCostPerHour       float32 `json:"labor_cost_per_hour"`
	OverheadPercentage     float32 `json:"overhead_percentage"`
	ProfitMarginPercentage float32 `json:"profit_margin_percentage"`
}

// PresetTemplate is a single catalog entry. Exactly one of Machine/Energy/Cost is
// populated, matching Type.
type PresetTemplate struct {
	Key         string                 `json:"key"`
	Type        PresetType             `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Machine     *MachineTemplateFields `json:"machine,omitempty"`
	Energy      *EnergyTemplateFields  `json:"energy,omitempty"`
	Cost        *CostTemplateFields    `json:"cost,omitempty"`
}

// presetTemplates is the immutable source catalog. Build volumes are the
// manufacturers' published usable areas; all machines use a 0.4 mm nozzle and
// 1.75 mm filament. Power consumption is an APPROXIMATE average draw while
// printing PLA (bed + hotend + electronics), not peak heat-up wattage.
var presetTemplates = []PresetTemplate{
	// ---- Machines --------------------------------------------------------
	{
		Key:         "bambu-a1-mini",
		Type:        PresetTypeMachine,
		Name:        "Bambu Lab A1 mini",
		Description: "Valores aproximados para a Bambu Lab A1 mini. Ajuste conforme seu equipamento e configurações de impressão.",
		Machine: &MachineTemplateFields{
			Brand: "Bambu Lab", Model: "A1 mini",
			BuildVolumeX: 180, BuildVolumeY: 180, BuildVolumeZ: 180,
			NozzleDiameter: 0.4, FilamentDiameter: 1.75,
			// Small bed, compact machine: ~80 W average while printing PLA.
			PowerConsumption: 80,
		},
	},
	{
		Key:         "bambu-a1",
		Type:        PresetTypeMachine,
		Name:        "Bambu Lab A1",
		Description: "Valores aproximados para a Bambu Lab A1. Ajuste conforme seu equipamento e configurações de impressão.",
		Machine: &MachineTemplateFields{
			Brand: "Bambu Lab", Model: "A1",
			BuildVolumeX: 256, BuildVolumeY: 256, BuildVolumeZ: 256,
			NozzleDiameter: 0.4, FilamentDiameter: 1.75,
			// Larger heated bed than the mini: ~100 W average while printing PLA.
			PowerConsumption: 100,
		},
	},
	{
		Key:         "bambu-p1s",
		Type:        PresetTypeMachine,
		Name:        "Bambu Lab P1S",
		Description: "Valores aproximados para a Bambu Lab P1S. Ajuste conforme seu equipamento e configurações de impressão.",
		Machine: &MachineTemplateFields{
			Brand: "Bambu Lab", Model: "P1S",
			BuildVolumeX: 256, BuildVolumeY: 256, BuildVolumeZ: 256,
			NozzleDiameter: 0.4, FilamentDiameter: 1.75,
			// Enclosed CoreXY with chamber: ~120 W average while printing PLA.
			PowerConsumption: 120,
		},
	},
	{
		Key:         "bambu-x1-carbon",
		Type:        PresetTypeMachine,
		Name:        "Bambu Lab X1 Carbon",
		Description: "Valores aproximados para a Bambu Lab X1 Carbon. Ajuste conforme seu equipamento e configurações de impressão.",
		Machine: &MachineTemplateFields{
			Brand: "Bambu Lab", Model: "X1 Carbon",
			BuildVolumeX: 256, BuildVolumeY: 256, BuildVolumeZ: 256,
			NozzleDiameter: 0.4, FilamentDiameter: 1.75,
			// Flagship enclosed CoreXY, extra sensors/AMS: ~130 W average on PLA.
			PowerConsumption: 130,
		},
	},
	{
		Key:         "prusa-mk4",
		Type:        PresetTypeMachine,
		Name:        "Prusa MK4",
		Description: "Valores aproximados para a Original Prusa MK4. Ajuste conforme seu equipamento e configurações de impressão.",
		Machine: &MachineTemplateFields{
			Brand: "Prusa", Model: "MK4",
			BuildVolumeX: 250, BuildVolumeY: 210, BuildVolumeZ: 220,
			NozzleDiameter: 0.4, FilamentDiameter: 1.75,
			// Open-frame bedslinger with a large heated bed: ~120 W average on PLA.
			PowerConsumption: 120,
		},
	},
	{
		Key:         "creality-ender3-v3-se",
		Type:        PresetTypeMachine,
		Name:        "Creality Ender-3 V3 SE",
		Description: "Valores aproximados para a Creality Ender-3 V3 SE. Ajuste conforme seu equipamento e configurações de impressão.",
		Machine: &MachineTemplateFields{
			Brand: "Creality", Model: "Ender-3 V3 SE",
			BuildVolumeX: 220, BuildVolumeY: 220, BuildVolumeZ: 250,
			NozzleDiameter: 0.4, FilamentDiameter: 1.75,
			// Budget open-frame bedslinger: ~110 W average while printing PLA.
			PowerConsumption: 110,
		},
	},

	// ---- Energy ----------------------------------------------------------
	{
		Key:         "energia-residencial-br",
		Type:        PresetTypeEnergy,
		Name:        "Tarifa média residencial (Brasil)",
		Description: "Tarifa residencial média aproximada no Brasil (~R$ 0,85/kWh). Consulte a sua conta de luz e ajuste para a sua distribuidora e estado.",
		Energy: &EnergyTemplateFields{
			Country:          "Brasil",
			EnergyCostPerKwh: 0.85,
			Currency:         "BRL",
		},
	},

	// ---- Cost ------------------------------------------------------------
	{
		Key:         "custo-hobby",
		Type:        PresetTypeCost,
		Name:        "Hobby",
		Description: "Perfil de custos inicial para hobbistas: mão de obra simbólica, baixa sobrecarga e margem modesta. Ajuste à sua realidade.",
		Cost: &CostTemplateFields{
			LaborCostPerHour:       20,
			OverheadPercentage:     5,
			ProfitMarginPercentage: 15,
		},
	},
	{
		Key:         "custo-profissional",
		Type:        PresetTypeCost,
		Name:        "Profissional",
		Description: "Perfil de custos inicial para operação profissional: mão de obra, sobrecarga e margem mais altas. Ajuste à sua realidade.",
		Cost: &CostTemplateFields{
			LaborCostPerHour:       60,
			OverheadPercentage:     15,
			ProfitMarginPercentage: 40,
		},
	},
}

// cloneTemplate returns a deep copy of a template, so the shared pointer fields
// (Machine/Energy/Cost) in the global catalog can never be mutated by callers.
func cloneTemplate(t PresetTemplate) PresetTemplate {
	clone := t
	if t.Machine != nil {
		m := *t.Machine
		clone.Machine = &m
	}
	if t.Energy != nil {
		e := *t.Energy
		clone.Energy = &e
	}
	if t.Cost != nil {
		c := *t.Cost
		clone.Cost = &c
	}
	return clone
}

// Templates returns a deep copy of the full static catalog so callers cannot
// mutate the shared backing data.
func Templates() []PresetTemplate {
	out := make([]PresetTemplate, 0, len(presetTemplates))
	for _, tmpl := range presetTemplates {
		out = append(out, cloneTemplate(tmpl))
	}
	return out
}

// TemplatesByType returns deep copies of the catalog entries matching the given
// type. When presetType is empty, all templates are returned.
func TemplatesByType(presetType PresetType) []PresetTemplate {
	out := make([]PresetTemplate, 0, len(presetTemplates))
	for _, tmpl := range presetTemplates {
		if presetType == "" || tmpl.Type == presetType {
			out = append(out, cloneTemplate(tmpl))
		}
	}
	return out
}

// TemplateByKey returns a deep copy of the template with the given key, or
// (zero, false) when no template matches.
func TemplateByKey(key string) (PresetTemplate, bool) {
	for _, tmpl := range presetTemplates {
		if tmpl.Key == key {
			return cloneTemplate(tmpl), true
		}
	}
	return PresetTemplate{}, false
}
