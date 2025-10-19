package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
)

// PlanModel represents the subscription plan data model for GORM
type PlanModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string    `gorm:"type:varchar(100);not null"`
	Slug        string    `gorm:"type:varchar(50);not null;uniqueIndex"`
	Description string    `gorm:"type:text"`
	Price       float64   `gorm:"type:decimal(10,2);not null"`
	Currency    string    `gorm:"type:varchar(3);not null"`
	Interval    string    `gorm:"type:varchar(20);not null"` // MONTHLY, YEARLY
	Active      bool      `gorm:"type:boolean;not null"`
	Popular     bool      `gorm:"type:boolean;not null"`
	Recommended bool      `gorm:"type:boolean;not null"`
	SortOrder   int       `gorm:"type:integer;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (PlanModel) TableName() string {
	return "subscription_plans"
}

// PlanFeatureModel represents a plan feature data model for GORM
type PlanFeatureModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	PlanID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Name        string    `gorm:"type:varchar(100);not null"`
	Key         string    `gorm:"type:varchar(50);not null"` // max_users, storage_gb, etc
	Description string    `gorm:"type:text"`
	Value       string    `gorm:"type:varchar(255);not null"` // "5", "true", "unlimited"
	ValueType   string    `gorm:"type:varchar(20);not null"`  // number, boolean, text
	Available   bool      `gorm:"type:boolean;not null"`
	SortOrder   int       `gorm:"type:integer;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (PlanFeatureModel) TableName() string {
	return "plan_features"
}

// ToEntity converts the GORM model to domain entity
// Features must be loaded separately and passed as parameter
func (p *PlanModel) ToEntity(features []entities.PlanFeature) *entities.PlanEntity {
	return &entities.PlanEntity{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Price:       p.Price,
		Currency:    p.Currency,
		Interval:    p.Interval,
		Active:      p.Active,
		Popular:     p.Popular,
		Recommended: p.Recommended,
		SortOrder:   p.SortOrder,
		Features:    features,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// FromEntity converts domain entity to GORM model
func (p *PlanModel) FromEntity(entity *entities.PlanEntity) {
	p.ID = entity.ID
	p.Name = entity.Name
	p.Slug = entity.Slug
	p.Description = entity.Description
	p.Price = entity.Price
	p.Currency = entity.Currency
	p.Interval = entity.Interval
	p.Active = entity.Active
	p.Popular = entity.Popular
	p.Recommended = entity.Recommended
	p.SortOrder = entity.SortOrder
	p.CreatedAt = entity.CreatedAt
	p.UpdatedAt = entity.UpdatedAt
}

// ToEntity converts the GORM model to domain entity
func (f *PlanFeatureModel) ToEntity() *entities.PlanFeature {
	return &entities.PlanFeature{
		ID:          f.ID,
		PlanID:      f.PlanID,
		Name:        f.Name,
		Key:         f.Key,
		Description: f.Description,
		Value:       f.Value,
		ValueType:   f.ValueType,
		Available:   f.Available,
		SortOrder:   f.SortOrder,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}
}

// FromEntity converts domain entity to GORM model
func (f *PlanFeatureModel) FromEntity(entity *entities.PlanFeature) {
	f.ID = entity.ID
	f.PlanID = entity.PlanID
	f.Name = entity.Name
	f.Key = entity.Key
	f.Description = entity.Description
	f.Value = entity.Value
	f.ValueType = entity.ValueType
	f.Available = entity.Available
	f.SortOrder = entity.SortOrder
	f.CreatedAt = entity.CreatedAt
	f.UpdatedAt = entity.UpdatedAt
}
