package models

import (
	"encoding/json"
	"time"

	"github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	companyModels "github.com/RodolfoBonis/spooliq/features/company/data/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ActivityModel represents the database model for activity tracking.
type ActivityModel struct {
	ID             uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string          `gorm:"type:varchar(255);not null;index:idx_activity_org" json:"organization_id"`
	UserID         string          `gorm:"type:varchar(255);not null" json:"user_id"`
	Action         string          `gorm:"type:varchar(20);not null" json:"action"`
	EntityType     string          `gorm:"type:varchar(20);not null" json:"entity_type"`
	EntityID       string          `gorm:"type:varchar(255);not null" json:"entity_id"`
	EntityName     string          `gorm:"type:varchar(255);not null" json:"entity_name"`
	Description    string          `gorm:"type:text" json:"description,omitempty"`
	Metadata       json.RawMessage `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt      time.Time       `gorm:"autoCreateTime" json:"created_at"`

	// GORM v2 Relationships
	Organization *companyModels.CompanyModel `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"organization,omitempty"`
}

// TableName returns the table name for the activity model.
func (ActivityModel) TableName() string {
	return "activities"
}

// BeforeCreate is a GORM hook executed before creating an activity record.
func (a *ActivityModel) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// ToEntity converts the ActivityModel to an ActivityEntity.
func (a *ActivityModel) ToEntity() entities.ActivityEntity {
	entity := entities.ActivityEntity{
		ID:             a.ID.String(),
		OrganizationID: a.OrganizationID,
		UserID:         a.UserID,
		Action:         entities.ActivityAction(a.Action),
		EntityType:     entities.ActivityEntityType(a.EntityType),
		EntityID:       a.EntityID,
		EntityName:     a.EntityName,
		Description:    a.Description,
		CreatedAt:      a.CreatedAt,
	}

	if len(a.Metadata) > 0 {
		var metadata map[string]any
		if err := json.Unmarshal(a.Metadata, &metadata); err == nil {
			entity.Metadata = metadata
		}
	}

	return entity
}

// FromEntity populates the ActivityModel from an ActivityEntity.
func (a *ActivityModel) FromEntity(entity *entities.ActivityEntity) {
	if entity.ID != "" {
		if parsed, err := uuid.Parse(entity.ID); err == nil {
			a.ID = parsed
		}
	}
	a.OrganizationID = entity.OrganizationID
	a.UserID = entity.UserID
	a.Action = string(entity.Action)
	a.EntityType = string(entity.EntityType)
	a.EntityID = entity.EntityID
	a.EntityName = entity.EntityName
	a.Description = entity.Description

	if entity.Metadata != nil {
		if data, err := json.Marshal(entity.Metadata); err == nil {
			a.Metadata = data
		}
	}

	if !entity.CreatedAt.IsZero() {
		a.CreatedAt = entity.CreatedAt
	}
}
