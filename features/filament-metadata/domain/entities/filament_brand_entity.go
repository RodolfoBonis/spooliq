package entities

import (
	"time"

	"github.com/jinzhu/gorm"
)

// FilamentBrand representa uma marca de filamento padronizada
// @Description Marca de filamento para padronização do catálogo
// @Example {"id": 1, "name": "SUNLU", "description": "Marca chinesa de filamentos 3D", "active": true}
type FilamentBrand struct {
	ID          uint       `gorm:"primary_key;auto_increment" json:"id"`
	Name        string     `gorm:"type:varchar(100);not null;unique" json:"name" validate:"required,min=1,max=100"`
	Description *string    `gorm:"type:text" json:"description,omitempty"`
	Active      bool       `gorm:"default:true;not null" json:"active"`
	CreatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName especifica o nome da tabela para o GORM
func (FilamentBrand) TableName() string {
	return "filament_brands"
}

// BeforeCreate é um hook do GORM executado antes de criar uma marca
func (fb *FilamentBrand) BeforeCreate(scope *gorm.Scope) error {
	fb.CreatedAt = time.Now()
	fb.UpdatedAt = time.Now()
	if fb.Active == false {
		fb.Active = true // Default para ativo
	}
	return nil
}

// BeforeUpdate é um hook do GORM executado antes de atualizar uma marca
func (fb *FilamentBrand) BeforeUpdate(scope *gorm.Scope) error {
	fb.UpdatedAt = time.Now()
	return nil
}

// IsActive verifica se a marca está ativa
func (fb *FilamentBrand) IsActive() bool {
	return fb.Active && fb.DeletedAt == nil
}