package entities

import (
	"time"

	"github.com/jinzhu/gorm"
)

// CostProfile representa um perfil de custos operacionais para cálculos
// @Description Perfil de custos operacionais (desgaste, manutenção, etc.)
type CostProfile struct {
	ID                     uint       `gorm:"primary_key;auto_increment" json:"id"`
	QuoteID                uint       `gorm:"index" json:"quote_id,omitempty"`
	Name                   string     `gorm:"type:varchar(255);not null" json:"name" validate:"required,min=1,max=255"`
	WearPercentage         float64    `gorm:"type:decimal(5,2);default:0" json:"wear_percentage" validate:"min=0,max=100"`    // Percentual de desgaste
	OverheadAmount         float64    `gorm:"type:decimal(10,2);default:0" json:"overhead_amount" validate:"min=0"`           // Valor fixo de overhead
	WearCostPerHour        float64    `gorm:"type:decimal(10,4);default:0" json:"wear_cost_per_hour" validate:"min=0"`        // Custo de desgaste por hora
	MaintenanceCostPerHour float64    `gorm:"type:decimal(10,4);default:0" json:"maintenance_cost_per_hour" validate:"min=0"` // Custo de manutenção por hora
	OverheadCostPerHour    float64    `gorm:"type:decimal(10,4);default:0" json:"overhead_cost_per_hour" validate:"min=0"`    // Custo de overhead por hora
	Description            string     `gorm:"type:text" json:"description"`
	OwnerUserID            *string    `gorm:"type:varchar(255);index" json:"owner_user_id,omitempty"` // null = catálogo global (admin)
	CreatedAt              time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt              time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt              *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName especifica o nome da tabela para o GORM
func (CostProfile) TableName() string {
	return "cost_profiles"
}

// BeforeCreate é um hook do GORM executado antes de criar um perfil de custo
func (cp *CostProfile) BeforeCreate(scope *gorm.Scope) error {
	cp.CreatedAt = time.Now()
	cp.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate é um hook do GORM executado antes de atualizar um perfil de custo
func (cp *CostProfile) BeforeUpdate(scope *gorm.Scope) error {
	cp.UpdatedAt = time.Now()
	return nil
}

// GetTotalCostPerHour calcula o custo total por hora
func (cp *CostProfile) GetTotalCostPerHour() float64 {
	return cp.WearCostPerHour + cp.MaintenanceCostPerHour + cp.OverheadCostPerHour
}

// IsGlobal verifica se o perfil é do catálogo global (sem dono)
func (cp *CostProfile) IsGlobal() bool {
	return cp.OwnerUserID == nil
}

// CanUserAccess verifica se um usuário pode acessar este perfil
func (cp *CostProfile) CanUserAccess(userID string, isAdmin bool) bool {
	// Admin pode acessar tudo
	if isAdmin {
		return true
	}
	// Perfis globais são acessíveis a todos
	if cp.IsGlobal() {
		return true
	}
	// Usuário pode acessar seus próprios perfis
	return cp.OwnerUserID != nil && *cp.OwnerUserID == userID
}
