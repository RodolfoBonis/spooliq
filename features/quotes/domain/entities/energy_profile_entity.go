package entities

import (
	"time"

	"github.com/jinzhu/gorm"
)

// EnergyProfile representa um perfil de energia/tarifa para cálculos de custo
// @Description Perfil de tarifa energética
type EnergyProfile struct {
	ID            uint       `gorm:"primary_key;auto_increment" json:"id"`
	QuoteID       uint       `gorm:"index" json:"quote_id,omitempty"`
	Name          string     `gorm:"type:varchar(255);not null" json:"name" validate:"required,min=1,max=255"`
	BaseTariff    float64    `gorm:"type:decimal(10,4);not null" json:"base_tariff" validate:"required,min=0"` // Tarifa base em R$/kWh
	FlagSurcharge float64    `gorm:"type:decimal(10,4);default:0" json:"flag_surcharge" validate:"min=0"`      // Adicional da bandeira tarifária
	Location      string     `gorm:"type:varchar(255)" json:"location"`                                        // Cidade/Estado
	Year          int        `gorm:"type:integer" json:"year"`                                                 // Ano da tarifa
	Description   string     `gorm:"type:text" json:"description"`
	OwnerUserID   *string    `gorm:"type:varchar(255);index" json:"owner_user_id,omitempty"` // null = catálogo global (admin)
	CreatedAt     time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt     *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName especifica o nome da tabela para o GORM
func (EnergyProfile) TableName() string {
	return "energy_profiles"
}

// BeforeCreate é um hook do GORM executado antes de criar um perfil de energia
func (ep *EnergyProfile) BeforeCreate(scope *gorm.Scope) error {
	ep.CreatedAt = time.Now()
	ep.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate é um hook do GORM executado antes de atualizar um perfil de energia
func (ep *EnergyProfile) BeforeUpdate(scope *gorm.Scope) error {
	ep.UpdatedAt = time.Now()
	return nil
}

// GetTotalTariff calcula a tarifa total (base + bandeira)
func (ep *EnergyProfile) GetTotalTariff() float64 {
	return ep.BaseTariff + ep.FlagSurcharge
}

// IsGlobal verifica se o perfil é do catálogo global (sem dono)
func (ep *EnergyProfile) IsGlobal() bool {
	return ep.OwnerUserID == nil
}

// CanUserAccess verifica se um usuário pode acessar este perfil
func (ep *EnergyProfile) CanUserAccess(userID string, isAdmin bool) bool {
	// Admin pode acessar tudo
	if isAdmin {
		return true
	}
	// Perfis globais são acessíveis a todos
	if ep.IsGlobal() {
		return true
	}
	// Usuário pode acessar seus próprios perfis
	return ep.OwnerUserID != nil && *ep.OwnerUserID == userID
}
