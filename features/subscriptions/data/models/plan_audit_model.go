package models

import (
	"encoding/json"
	"time"

	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlanAuditModel represents the plan audit log data model for GORM
type PlanAuditModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PlanID    uuid.UUID `gorm:"type:uuid;not null;index" json:"plan_id"`
	PlanName  string    `gorm:"type:varchar(255);not null" json:"plan_name"`
	Action    string    `gorm:"type:varchar(50);not null;index" json:"action"`
	UserID    string    `gorm:"type:varchar(255);not null" json:"user_id"`
	UserEmail string    `gorm:"type:varchar(255);not null" json:"user_email"`
	Changes   string    `gorm:"type:jsonb" json:"changes"` // JSON string
	Reason    string    `gorm:"type:text" json:"reason"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Relationships
	Plan *SubscriptionPlanModel `gorm:"foreignKey:PlanID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"plan,omitempty"`
}

// TableName specifies the table name for GORM
func (PlanAuditModel) TableName() string {
	return "plan_audit_logs"
}

// ToEntity converts model to entity
func (m *PlanAuditModel) ToEntity() *adminEntities.PlanAuditEntry {
	var changes map[string]interface{}
	if m.Changes != "" {
		json.Unmarshal([]byte(m.Changes), &changes)
	}

	return &adminEntities.PlanAuditEntry{
		ID:        m.ID.String(),
		PlanID:    m.PlanID.String(),
		PlanName:  m.PlanName,
		Action:    m.Action,
		UserID:    m.UserID,
		UserEmail: m.UserEmail,
		Changes:   changes,
		Reason:    m.Reason,
		CreatedAt: m.CreatedAt,
	}
}

// FromEntity converts entity to model
func (m *PlanAuditModel) FromEntity(entity *adminEntities.PlanAuditEntry) {
	if entity.ID != "" {
		m.ID = uuid.MustParse(entity.ID)
	}
	m.PlanID = uuid.MustParse(entity.PlanID)
	m.PlanName = entity.PlanName
	m.Action = entity.Action
	m.UserID = entity.UserID
	m.UserEmail = entity.UserEmail
	m.Reason = entity.Reason
	m.CreatedAt = entity.CreatedAt

	if entity.Changes != nil {
		changesBytes, _ := json.Marshal(entity.Changes)
		m.Changes = string(changesBytes)
	}
}

// BeforeCreate hook for GORM
func (m *PlanAuditModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}