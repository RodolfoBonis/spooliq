package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/material/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MaterialModel represents a 3D printing material in the database
type MaterialModel struct {
	ID           uuid.UUID  `gorm:"<-:create;type:uuid;primaryKey" json:"id"`
	Name         string     `gorm:"type:varchar(255);not null" json:"name"`
	Description  string     `gorm:"type:text" json:"description,omitempty"`
	TempTable    float32    `gorm:"type:float" json:"tempTable,omitempty"`
	TempExtruder float32    `gorm:"type:float" json:"tempExtruder,omitempty"`
	CreatedAt    time.Time  `gorm:"<-:create;type:timestamp;" json:"created_at,omitempty"`
	UpdatedAt    time.Time  `gorm:"<-:update;type:timestamp;" json:"updated_at,omitempty"`
	DeletedAt    *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName returns the database table name for MaterialModel
func (m *MaterialModel) TableName() string { return "materials" }

// BeforeCreate é um hook do GORM executado antes de criar um material
func (m *MaterialModel) BeforeCreate(tx *gorm.DB) (err error) {
	m.ID = uuid.New()
	return
}

// FromEntity populates the MaterialModel from a MaterialEntity.
func (m *MaterialModel) FromEntity(entity *entities.MaterialEntity) {
	m.ID = entity.ID
	m.Name = entity.Name
	m.Description = entity.Description
	m.TempTable = entity.TempTable
	m.TempExtruder = entity.TempExtruder
	m.CreatedAt = entity.CreatedAt
	m.UpdatedAt = entity.UpdatedAt
	m.DeletedAt = entity.DeletedAt
}

// ToEntity converts the MaterialModel to a MaterialEntity.
func (m *MaterialModel) ToEntity() entities.MaterialEntity {
	return entities.MaterialEntity{
		ID:           m.ID,
		Name:         m.Name,
		Description:  m.Description,
		TempTable:    m.TempTable,
		TempExtruder: m.TempExtruder,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		DeletedAt:    m.DeletedAt,
	}
}
