package entities

import (
	"time"

	"github.com/jinzhu/gorm"
)

// MarginProfile representa um perfil de margem/lucro para cálculos
// @Description Perfil de margem de lucro e mão-de-obra
type MarginProfile struct {
	ID                  uint       `gorm:"primary_key;auto_increment" json:"id"`
	QuoteID             uint       `gorm:"index" json:"quote_id,omitempty"`
	Name                string     `gorm:"type:varchar(255);not null" json:"name" validate:"required,min=1,max=255"`
	PrintingOnlyMargin  float64    `gorm:"type:decimal(5,2);default:0" json:"printing_only_margin" validate:"min=0,max=100"` // Margem só impressão
	PrintingPlusMargin  float64    `gorm:"type:decimal(5,2);default:0" json:"printing_plus_margin" validate:"min=0,max=100"` // Margem impressão + ajustes
	FullServiceMargin   float64    `gorm:"type:decimal(5,2);default:0" json:"full_service_margin" validate:"min=0,max=100"`  // Margem serviço completo
	OperatorRatePerHour float64    `gorm:"type:decimal(10,2);default:0" json:"operator_rate_per_hour" validate:"min=0"`      // Taxa operador por hora
	ModelerRatePerHour  float64    `gorm:"type:decimal(10,2);default:0" json:"modeler_rate_per_hour" validate:"min=0"`       // Taxa modelador por hora
	LaborCostPerHour    float64    `gorm:"type:decimal(10,2);default:0" json:"labor_cost_per_hour" validate:"min=0"`         // Custo de mão-de-obra por hora
	ProfitMargin        float64    `gorm:"type:decimal(5,2);default:0" json:"profit_margin" validate:"min=0,max=100"`        // Margem de lucro em % (0-100)
	Description         string     `gorm:"type:text" json:"description"`
	OwnerUserID         *string    `gorm:"type:varchar(255);index" json:"owner_user_id,omitempty"` // null = catálogo global (admin)
	CreatedAt           time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt           *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName especifica o nome da tabela para o GORM
func (MarginProfile) TableName() string {
	return "margin_profiles"
}

// BeforeCreate é um hook do GORM executado antes de criar um perfil de margem
func (mp *MarginProfile) BeforeCreate(scope *gorm.Scope) error {
	mp.CreatedAt = time.Now()
	mp.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate é um hook do GORM executado antes de atualizar um perfil de margem
func (mp *MarginProfile) BeforeUpdate(scope *gorm.Scope) error {
	mp.UpdatedAt = time.Now()
	return nil
}

// CalculateFinalPrice calcula o preço final com base no custo e margem
func (mp *MarginProfile) CalculateFinalPrice(baseCost float64, printTimeHours float64) float64 {
	laborCost := mp.LaborCostPerHour * printTimeHours
	totalCost := baseCost + laborCost
	marginMultiplier := 1 + (mp.ProfitMargin / 100)
	return totalCost * marginMultiplier
}

// IsGlobal verifica se o perfil é do catálogo global (sem dono)
func (mp *MarginProfile) IsGlobal() bool {
	return mp.OwnerUserID == nil
}

// CanUserAccess verifica se um usuário pode acessar este perfil
func (mp *MarginProfile) CanUserAccess(userID string, isAdmin bool) bool {
	// Admin pode acessar tudo
	if isAdmin {
		return true
	}
	// Perfis globais são acessíveis a todos
	if mp.IsGlobal() {
		return true
	}
	// Usuário pode acessar seus próprios perfis
	return mp.OwnerUserID != nil && *mp.OwnerUserID == userID
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
