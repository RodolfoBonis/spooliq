package entities

import (
	"time"

	"github.com/jinzhu/gorm"
)

// MachineProfile representa um perfil de máquina para cálculos de custo
// @Description Perfil de máquina/impressora 3D
type MachineProfile struct {
	ID          uint       `gorm:"primary_key;auto_increment" json:"id"`
	QuoteID     uint       `gorm:"index" json:"quote_id,omitempty"`
	Name        string     `gorm:"type:varchar(255);not null" json:"name" validate:"required,min=1,max=255"`
	Brand       string     `gorm:"type:varchar(255);not null" json:"brand" validate:"required,min=1,max=255"`
	Model       string     `gorm:"type:varchar(255);not null" json:"model" validate:"required,min=1,max=255"`
	Watt        float64    `gorm:"type:decimal(10,2);not null" json:"watt" validate:"required,min=1"`     // Consumo em watts
	IdleFactor  float64    `gorm:"type:decimal(3,2);default:0" json:"idle_factor" validate:"min=0,max=1"` // Fator de consumo em idle (0-1)
	Description string     `gorm:"type:text" json:"description"`
	URL         string     `gorm:"type:text" json:"url" validate:"omitempty,url"`
	OwnerUserID *string    `gorm:"type:varchar(255);index" json:"owner_user_id,omitempty"` // null = catálogo global (admin)
	CreatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName especifica o nome da tabela para o GORM
func (MachineProfile) TableName() string {
	return "machine_profiles"
}

// BeforeCreate é um hook do GORM executado antes de criar um perfil de máquina
func (mp *MachineProfile) BeforeCreate(scope *gorm.Scope) error {
	mp.CreatedAt = time.Now()
	mp.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate é um hook do GORM executado antes de atualizar um perfil de máquina
func (mp *MachineProfile) BeforeUpdate(scope *gorm.Scope) error {
	mp.UpdatedAt = time.Now()
	return nil
}

// IsGlobal verifica se o perfil é do catálogo global (sem dono)
func (mp *MachineProfile) IsGlobal() bool {
	return mp.OwnerUserID == nil
}

// CanUserAccess verifica se um usuário pode acessar este perfil
func (mp *MachineProfile) CanUserAccess(userID string, isAdmin bool) bool {
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
