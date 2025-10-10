package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/material/domain/entities"
	"github.com/google/uuid"
)

// MaterialModel represents a 3D printing material in the database
type MaterialModel struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name         string     `gorm:"type:varchar(255);not null" json:"name"`
	Description  string     `gorm:"type:text" json:"description,omitempty"`
	TempTable    float32    `gorm:"type:float" json:"tempTable,omitempty"`
	TempExtruder float32    `gorm:"type:float" json:"tempExtruder,omitempty"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at,omitempty"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
	DeletedAt    *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName returns the database table name for MaterialModel
func (m *MaterialModel) TableName() string { return "materials" }

// FromEntity populates the MaterialModel from a MaterialEntity.
func (m *MaterialModel) FromEntity(entity *entities.MaterialEntity) {
	// Only set ID if it's not zero (for updates)
	if entity.ID != uuid.Nil {
		m.ID = entity.ID
	}
	m.Name = entity.Name
	m.Description = entity.Description
	m.TempTable = entity.TempTable
	m.TempExtruder = entity.TempExtruder
	// Let GORM handle timestamps
	if !entity.CreatedAt.IsZero() {
		m.CreatedAt = entity.CreatedAt
	}
	if !entity.UpdatedAt.IsZero() {
		m.UpdatedAt = entity.UpdatedAt
	}
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
