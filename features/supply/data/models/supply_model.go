package models

import (
	"time"

	companyModels "github.com/RodolfoBonis/spooliq/features/company/data/models"
	"github.com/RodolfoBonis/spooliq/features/supply/domain/entities"
	"github.com/google/uuid"
)

// SupplyModel represents the database model for supply (insumo) entities.
type SupplyModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;index:idx_supply_org" json:"organization_id"` // FK: references companies(organization_id) ON DELETE RESTRICT
	Name           string    `gorm:"type:varchar(255);not null" json:"name"`
	Unit           string    `gorm:"type:varchar(20);not null" json:"unit"`
	UnitCost       int64     `gorm:"type:bigint;not null;default:0" json:"unit_cost"` // cents
	Description    *string   `gorm:"type:text" json:"description,omitempty"`
	// IsActive carries NO gorm `default` tag on purpose: a `default` tag makes GORM
	// omit a zero value (false) from the INSERT so the DB default would silently turn
	// an explicitly inactive supply active. The "new supplies default to active" rule
	// is enforced in the create use case (which sets true when the client omits it),
	// so the stored value always reflects the caller's intent.
	IsActive  bool       `gorm:"type:boolean;not null" json:"is_active"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at,omitempty"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`

	// GORM v2 Relationships
	Organization *companyModels.CompanyModel `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"organization,omitempty"`
}

// TableName returns the table name for the supply model.
func (s *SupplyModel) TableName() string { return "supplies" }

// FromEntity populates the SupplyModel from a SupplyEntity.
func (s *SupplyModel) FromEntity(entity *entities.SupplyEntity) {
	// Only set ID if it's not zero (for updates)
	if entity.ID != uuid.Nil {
		s.ID = entity.ID
	}
	s.OrganizationID = entity.OrganizationID
	s.Name = entity.Name
	s.Unit = entity.Unit
	s.UnitCost = entity.UnitCost
	s.Description = entity.Description
	s.IsActive = entity.IsActive
	// Let GORM handle timestamps
	if !entity.CreatedAt.IsZero() {
		s.CreatedAt = entity.CreatedAt
	}
	if !entity.UpdatedAt.IsZero() {
		s.UpdatedAt = entity.UpdatedAt
	}
	s.DeletedAt = entity.DeletedAt
}

// ToEntity converts the SupplyModel to a SupplyEntity.
func (s *SupplyModel) ToEntity() entities.SupplyEntity {
	return entities.SupplyEntity{
		ID:             s.ID,
		OrganizationID: s.OrganizationID,
		Name:           s.Name,
		Unit:           s.Unit,
		UnitCost:       s.UnitCost,
		Description:    s.Description,
		IsActive:       s.IsActive,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
		DeletedAt:      s.DeletedAt,
	}
}
