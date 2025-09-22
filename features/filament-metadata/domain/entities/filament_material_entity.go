package entities

import (
	"time"

	"github.com/jinzhu/gorm"
)

// FilamentMaterial representa um material de filamento padronizado
// @Description Material de filamento para padronização do catálogo
// @Example {"id": 1, "name": "PLA", "description": "Ácido Polilático - biodegradável", "properties": "{\"melting_temp\": \"180-220°C\", \"bed_temp\": \"0-60°C\"}", "active": true}
type FilamentMaterial struct {
	ID          uint       `gorm:"primary_key;auto_increment" json:"id"`
	Name        string     `gorm:"type:varchar(50);not null;unique" json:"name" validate:"required,min=1,max=50"`
	Description *string    `gorm:"type:text" json:"description,omitempty"`
	Properties  *string    `gorm:"type:json" json:"properties,omitempty"` // JSON com propriedades como temp. de extrusão, mesa aquecida, etc.
	Active      bool       `gorm:"default:true;not null" json:"active"`
	CreatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName especifica o nome da tabela para o GORM
func (FilamentMaterial) TableName() string {
	return "filament_materials"
}

// BeforeCreate é um hook do GORM executado antes de criar um material
func (fm *FilamentMaterial) BeforeCreate(scope *gorm.Scope) error {
	fm.CreatedAt = time.Now()
	fm.UpdatedAt = time.Now()
	if fm.Active == false {
		fm.Active = true // Default para ativo
	}
	return nil
}

// BeforeUpdate é um hook do GORM executado antes de atualizar um material
func (fm *FilamentMaterial) BeforeUpdate(scope *gorm.Scope) error {
	fm.UpdatedAt = time.Now()
	return nil
}

// IsActive verifica se o material está ativo
func (fm *FilamentMaterial) IsActive() bool {
	return fm.Active && fm.DeletedAt == nil
}