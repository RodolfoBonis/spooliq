package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubscriptionPlanModel represents the subscription plan data model for GORM
type SubscriptionPlanModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Price       float64   `gorm:"type:decimal(10,2);not null" json:"price"`
	Cycle       string    `gorm:"type:varchar(20);not null" json:"cycle"` // MONTHLY, YEARLY
	Features    string    `gorm:"type:jsonb" json:"features"`             // JSON with features
	IsActive    bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// TableName specifies the table name for GORM
func (SubscriptionPlanModel) TableName() string {
	return "subscription_plans"
}

// BeforeCreate is a GORM hook executed before creating a plan
func (s *SubscriptionPlanModel) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// ToEntity converts the GORM model to domain entity
func (s *SubscriptionPlanModel) ToEntity() *entities.SubscriptionPlanEntity {
	return &entities.SubscriptionPlanEntity{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Price:       s.Price,
		Cycle:       s.Cycle,
		Features:    s.Features,
		IsActive:    s.IsActive,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		DeletedAt:   getDeletedAtFromPlan(s.DeletedAt),
	}
}

// FromEntity converts domain entity to GORM model
func (s *SubscriptionPlanModel) FromEntity(entity *entities.SubscriptionPlanEntity) {
	s.ID = entity.ID
	s.Name = entity.Name
	s.Description = entity.Description
	s.Price = entity.Price
	s.Cycle = entity.Cycle
	s.Features = entity.Features
	s.IsActive = entity.IsActive
	s.CreatedAt = entity.CreatedAt
	s.UpdatedAt = entity.UpdatedAt
	if entity.DeletedAt != nil {
		s.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}
}

// getDeletedAtFromPlan returns nil if deleted_at is not valid, otherwise returns pointer to time
func getDeletedAtFromPlan(deletedAt gorm.DeletedAt) *time.Time {
	if deletedAt.Valid {
		return &deletedAt.Time
	}
	return nil
}
