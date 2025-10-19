package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlanFeatureModel represents a feature of a subscription plan
type PlanFeatureModel struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SubscriptionPlanID uuid.UUID      `gorm:"type:uuid;not null;index" json:"subscription_plan_id"`
	Name               string         `gorm:"type:varchar(255);not null" json:"name"`
	Description        string         `gorm:"type:text" json:"description"`
	IsActive           bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt          time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// Relacionamento BelongsTo
	SubscriptionPlan *SubscriptionPlanModel `gorm:"foreignKey:SubscriptionPlanID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

// TableName specifies the table name for GORM
func (PlanFeatureModel) TableName() string {
	return "plan_features"
}

// BeforeCreate is a GORM hook executed before creating a feature
func (f *PlanFeatureModel) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}
