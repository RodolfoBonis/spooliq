package entities

import "time"

// Quote representa um orçamento de impressão 3D (Entity pura de domínio)
type Quote struct {
	ID          uint
	Title       string
	Notes       string
	OwnerUserID string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Relacionamentos
	FilamentLines []QuoteFilamentLine
	MachineProfile *MachineProfile
	EnergyProfile  *EnergyProfile
	CostProfile    *CostProfile
	MarginProfile  *MarginProfile
}

// CanUserAccess verifica se um usuário pode acessar este orçamento (regra de negócio)
func (q *Quote) CanUserAccess(userID string, isAdmin bool) bool {
	// Admin pode acessar tudo
	if isAdmin {
		return true
	}
	// Usuário pode acessar apenas seus próprios orçamentos
	return q.OwnerUserID == userID
}

// IsValid valida se o orçamento está válido (regra de negócio)
func (q *Quote) IsValid() bool {
	if q.Title == "" || len(q.Title) > 255 {
		return false
	}
	if q.OwnerUserID == "" {
		return false
	}
	if len(q.FilamentLines) == 0 {
		return false
	}
	return true
}

// CalculateTotalWeight calcula o peso total dos filamentos (regra de negócio)
func (q *Quote) CalculateTotalWeight() float64 {
	total := 0.0
	for _, line := range q.FilamentLines {
		total += line.WeightGrams
	}
	return total
}

// QuoteFilamentLine representa uma linha de filamento de orçamento (Entity pura)
type QuoteFilamentLine struct {
	ID      uint
	QuoteID uint

	// Snapshot dos dados do filamento (para preservar histórico)
	FilamentSnapshotName          string
	FilamentSnapshotBrand         string
	FilamentSnapshotMaterial      string
	FilamentSnapshotColor         string
	FilamentSnapshotColorHex      string
	FilamentSnapshotPricePerKg    float64
	FilamentSnapshotPricePerMeter *float64
	FilamentSnapshotURL           string
	WeightGrams                   float64
	LengthMeters                  *float64
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
}

// CalculateCost calcula o custo desta linha de filamento (regra de negócio)
func (fl *QuoteFilamentLine) CalculateCost() float64 {
	// Preço por kg convertido para gramas
	pricePerGram := fl.FilamentSnapshotPricePerKg / 1000.0
	return fl.WeightGrams * pricePerGram
}

// MachineProfile representa um perfil de máquina (Entity pura)
type MachineProfile struct {
	ID          uint
	QuoteID     uint
	Name        string
	Brand       string
	Model       string
	Watt        float64
	IdleFactor  float64
	Description string
	URL         string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CalculateEnergyConsumption calcula o consumo de energia por hora (regra de negócio)
func (mp *MachineProfile) CalculateEnergyConsumption(printTimeHours float64) float64 {
	activeTime := printTimeHours
	idleTime := printTimeHours * mp.IdleFactor
	totalTimeHours := activeTime + idleTime

	// Watts para kWh
	return (mp.Watt * totalTimeHours) / 1000.0
}

// EnergyProfile representa um perfil de energia (Entity pura)
type EnergyProfile struct {
	ID            uint
	QuoteID       uint
	BaseTariff    float64
	FlagSurcharge float64
	Location      string
	Year          int
	Description   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CalculateEnergyCost calcula o custo da energia (regra de negócio)
func (ep *EnergyProfile) CalculateEnergyCost(kWhConsumed float64) float64 {
	totalTariff := ep.BaseTariff + ep.FlagSurcharge
	return kWhConsumed * totalTariff
}

// CostProfile representa um perfil de custos (Entity pura)
type CostProfile struct {
	ID               uint
	QuoteID          uint
	WearPercentage   float64
	OverheadAmount   float64
	Description      string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CalculateWearCost calcula o custo de desgaste (regra de negócio)
func (cp *CostProfile) CalculateWearCost(materialCost float64) float64 {
	return materialCost * (cp.WearPercentage / 100.0)
}

// MarginProfile representa um perfil de margens (Entity pura)
type MarginProfile struct {
	ID                    uint
	QuoteID               uint
	PrintingOnlyMargin    float64
	PrintingPlusMargin    float64
	FullServiceMargin     float64
	OperatorRatePerHour   float64
	ModelerRatePerHour    float64
	Description           string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// CalculateLaborCost calcula o custo de mão de obra (regra de negócio)
func (mp *MarginProfile) CalculateLaborCost(operatorMinutes, modelerMinutes float64) float64 {
	operatorHours := operatorMinutes / 60.0
	modelerHours := modelerMinutes / 60.0

	operatorCost := operatorHours * mp.OperatorRatePerHour
	modelerCost := modelerHours * mp.ModelerRatePerHour

	return operatorCost + modelerCost
}

// GetMarginByServiceType retorna a margem baseada no tipo de serviço (regra de negócio)
func (mp *MarginProfile) GetMarginByServiceType(serviceType string) float64 {
	switch serviceType {
	case "printing_only":
		return mp.PrintingOnlyMargin
	case "printing_plus":
		return mp.PrintingPlusMargin
	case "full_service":
		return mp.FullServiceMargin
	default:
		return 0.0
	}
}