package models

import (
	"time"

	companyModels "github.com/RodolfoBonis/spooliq/features/company/data/models"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/google/uuid"
)

// PresetModel represents the base preset model in the database
type PresetModel struct {
	ID             uuid.UUID  `gorm:"<-:create;type:uuid;primaryKey" json:"id"`
	Name           string     `gorm:"type:varchar(255);not null" json:"name"`
	Description    string     `gorm:"type:text" json:"description,omitempty"`
	Type           string     `gorm:"type:varchar(50);not null" json:"type"`
	IsActive       bool       `gorm:"type:boolean;default:true" json:"is_active"`
	IsDefault      bool       `gorm:"type:boolean;default:false" json:"is_default"`
	UserID         *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	OrganizationID string     `gorm:"type:varchar(255);not null;index" json:"organization_id"` // FK: references companies(organization_id) ON DELETE RESTRICT
	CreatedAt      time.Time  `gorm:"<-:create;type:timestamp" json:"created_at,omitempty"`
	UpdatedAt      time.Time  `gorm:"<-:update;type:timestamp" json:"updated_at,omitempty"`
	DeletedAt      *time.Time `gorm:"index" json:"deleted_at,omitempty"`

	// GORM v2 Relationships
	Organization *companyModels.CompanyModel `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"organization,omitempty"`
	// 1:1 with specific preset types (based on Type field)
	// User           *UserModel           `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"user,omitempty"`
	// MachinePreset  *MachinePresetModel  `gorm:"foreignKey:ID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"machine_preset,omitempty"`
	// EnergyPreset   *EnergyPresetModel   `gorm:"foreignKey:ID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"energy_preset,omitempty"`
	// CostPreset     *CostPresetModel     `gorm:"foreignKey:ID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"cost_preset,omitempty"`
	// Commented out to avoid circular imports - relationships defined in specific preset models
}

// TableName returns the table name for the preset model
func (p *PresetModel) TableName() string { return "presets" }

// FromEntity populates the PresetModel from a PresetEntity
func (p *PresetModel) FromEntity(entity *entities.PresetEntity) {
	p.ID = entity.ID
	p.Name = entity.Name
	p.Description = entity.Description
	p.Type = string(entity.Type)
	p.IsActive = entity.IsActive
	p.IsDefault = entity.IsDefault
	p.UserID = entity.UserID
	p.OrganizationID = entity.OrganizationID
	p.CreatedAt = entity.CreatedAt
	p.UpdatedAt = entity.UpdatedAt
	p.DeletedAt = entity.DeletedAt
}

// ToEntity converts the PresetModel to a PresetEntity
func (p *PresetModel) ToEntity() entities.PresetEntity {
	return entities.PresetEntity{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		Type:           entities.PresetType(p.Type),
		IsActive:       p.IsActive,
		IsDefault:      p.IsDefault,
		UserID:         p.UserID,
		OrganizationID: p.OrganizationID,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
		DeletedAt:      p.DeletedAt,
	}
}
