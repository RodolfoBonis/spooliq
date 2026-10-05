// Package models contains the GORM database models for the print profile feature.
package models

import (
	"time"

	companyModels "github.com/RodolfoBonis/spooliq/features/company/data/models"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PrintProfileModel represents a saved combination of machine, energy and
// (optional) cost presets used as a reusable printing profile.
type PrintProfileModel struct {
	ID              uuid.UUID      `gorm:"<-:create;type:uuid;primaryKey" json:"id"`
	OrganizationID  string         `gorm:"type:varchar(255);not null;index" json:"organization_id"` // FK: references companies(organization_id) ON DELETE RESTRICT
	Name            string         `gorm:"type:varchar(255);not null" json:"name"`
	Description     string         `gorm:"type:text" json:"description,omitempty"`
	MachinePresetID uuid.UUID      `gorm:"type:uuid;not null;index" json:"machine_preset_id"`
	EnergyPresetID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"energy_preset_id"`
	CostPresetID    *uuid.UUID     `gorm:"type:uuid;index" json:"cost_preset_id,omitempty"`
	IsDefault       bool           `gorm:"type:boolean;default:false" json:"is_default"`
	CreatedBy       string         `gorm:"type:varchar(255)" json:"created_by,omitempty"`
	CreatedAt       time.Time      `gorm:"<-:create;type:timestamp" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"<-:update;type:timestamp" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

	// GORM v2 Relationship to the tenant company (org scoping).
	Organization *companyModels.CompanyModel `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"organization,omitempty"`
}

// TableName returns the table name for the print profile model.
func (p *PrintProfileModel) TableName() string { return "print_profiles" }

// FromEntity populates the PrintProfileModel from a ProfileEntity.
func (p *PrintProfileModel) FromEntity(entity *entities.ProfileEntity) {
	p.ID = entity.ID
	p.OrganizationID = entity.OrganizationID
	p.Name = entity.Name
	p.Description = entity.Description
	p.MachinePresetID = entity.MachinePresetID
	p.EnergyPresetID = entity.EnergyPresetID
	p.CostPresetID = entity.CostPresetID
	p.IsDefault = entity.IsDefault
	p.CreatedBy = entity.CreatedBy
	p.CreatedAt = entity.CreatedAt
	p.UpdatedAt = entity.UpdatedAt
	if entity.DeletedAt != nil {
		p.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	} else {
		p.DeletedAt = gorm.DeletedAt{}
	}
}

// ToEntity converts the PrintProfileModel to a ProfileEntity.
func (p *PrintProfileModel) ToEntity() entities.ProfileEntity {
	var deletedAt *time.Time
	if p.DeletedAt.Valid {
		t := p.DeletedAt.Time
		deletedAt = &t
	}

	return entities.ProfileEntity{
		ID:              p.ID,
		OrganizationID:  p.OrganizationID,
		Name:            p.Name,
		Description:     p.Description,
		MachinePresetID: p.MachinePresetID,
		EnergyPresetID:  p.EnergyPresetID,
		CostPresetID:    p.CostPresetID,
		IsDefault:       p.IsDefault,
		CreatedBy:       p.CreatedBy,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
		DeletedAt:       deletedAt,
	}
}
