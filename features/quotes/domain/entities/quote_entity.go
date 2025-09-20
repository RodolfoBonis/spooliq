package entities

import (
	"time"

	"github.com/jinzhu/gorm"
)

// Quote representa um orçamento de impressão 3D
// @Description Orçamento de impressão 3D
type Quote struct {
	ID               uint                `gorm:"primary_key;auto_increment" json:"id"`
	Title            string              `gorm:"type:varchar(255);not null" json:"title" validate:"required,min=1,max=255"`
	Notes            string              `gorm:"type:text" json:"notes"`
	OwnerUserID      string              `gorm:"type:varchar(255);not null;index" json:"owner_user_id" validate:"required"`
	TotalPrintTime   int                 `gorm:"type:integer;default:0" json:"total_print_time"` // em segundos
	TotalFilamentG   float64             `gorm:"type:decimal(10,2);default:0" json:"total_filament_g"`
	TotalCost        float64             `gorm:"type:decimal(10,2);default:0" json:"total_cost"`
	MachineProfileID *uint               `gorm:"index" json:"machine_profile_id,omitempty"`
	EnergyProfileID  *uint               `gorm:"index" json:"energy_profile_id,omitempty"`
	CostProfileID    *uint               `gorm:"index" json:"cost_profile_id,omitempty"`
	MarginProfileID  *uint               `gorm:"index" json:"margin_profile_id,omitempty"`
	FilamentLines    []QuoteFilamentLine `gorm:"foreignkey:QuoteID" json:"filament_lines"`
	MachineProfile   *MachineProfile     `gorm:"foreignkey:MachineProfileID" json:"machine_profile,omitempty"`
	EnergyProfile    *EnergyProfile      `gorm:"foreignkey:EnergyProfileID" json:"energy_profile,omitempty"`
	CostProfile      *CostProfile        `gorm:"foreignkey:CostProfileID" json:"cost_profile,omitempty"`
	MarginProfile    *MarginProfile      `gorm:"foreignkey:MarginProfileID" json:"margin_profile,omitempty"`
	CreatedAt        time.Time           `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt        time.Time           `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt        *time.Time          `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName especifica o nome da tabela para o GORM
func (Quote) TableName() string {
	return "quotes"
}

// BeforeCreate é um hook do GORM executado antes de criar um orçamento
func (q *Quote) BeforeCreate(scope *gorm.Scope) error {
	q.CreatedAt = time.Now()
	q.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate é um hook do GORM executado antes de atualizar um orçamento
func (q *Quote) BeforeUpdate(scope *gorm.Scope) error {
	q.UpdatedAt = time.Now()
	return nil
}

// CanUserAccess verifica se um usuário pode acessar este orçamento
func (q *Quote) CanUserAccess(userID string, isAdmin bool) bool {
	// Admin pode acessar tudo
	if isAdmin {
		return true
	}
	// Usuário pode acessar seus próprios orçamentos
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
