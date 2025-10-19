package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubscriptionPlanModel represents the subscription plan data model for GORM
type SubscriptionPlanModel struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Price       float64        `gorm:"type:decimal(10,2);not null" json:"price"`
	Cycle       string         `gorm:"type:varchar(20);not null" json:"cycle"` // MONTHLY, YEARLY
	IsActive    bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// Relacionamento HasMany
	Features []PlanFeatureModel `gorm:"foreignKey:SubscriptionPlanID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"features"`
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
	features := make([]entities.PlanFeatureEntity, len(s.Features))
	for i, f := range s.Features {
		features[i] = entities.PlanFeatureEntity{
			ID:          f.ID,
			Name:        f.Name,
			Description: f.Description,
			IsActive:    f.IsActive,
			CreatedAt:   f.CreatedAt,
			UpdatedAt:   f.UpdatedAt,
			DeletedAt:   getDeletedAtFromPlanFeature(f.DeletedAt),
		}
	}

	return &entities.SubscriptionPlanEntity{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Price:       s.Price,
		Cycle:       s.Cycle,
		Features:    features,
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
	s.IsActive = entity.IsActive
	s.CreatedAt = entity.CreatedAt
	s.UpdatedAt = entity.UpdatedAt

	// Convert features
	features := make([]PlanFeatureModel, len(entity.Features))
	for i, f := range entity.Features {
		features[i] = PlanFeatureModel{
			ID:                 f.ID,
			SubscriptionPlanID: entity.ID,
			Name:               f.Name,
			Description:        f.Description,
			IsActive:           f.IsActive,
			CreatedAt:          f.CreatedAt,
			UpdatedAt:          f.UpdatedAt,
		}
		if f.DeletedAt != nil {
			features[i].DeletedAt = gorm.DeletedAt{Time: *f.DeletedAt, Valid: true}
		}
	}
	s.Features = features

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

// getDeletedAtFromPlanFeature returns nil if deleted_at is not valid, otherwise returns pointer to time
func getDeletedAtFromPlanFeature(deletedAt gorm.DeletedAt) *time.Time {
	if deletedAt.Valid {
		return &deletedAt.Time
	}
	return nil
}
