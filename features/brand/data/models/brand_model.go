package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/google/uuid"
)

// BrandModel represents the database model for brand entities.
type BrandModel struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string     `gorm:"type:varchar(255);not null" json:"name"`
	Description string     `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at,omitempty"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName returns the table name for the brand model.
func (b *BrandModel) TableName() string { return "brands" }

// FromEntity populates the BrandModel from a BrandEntity.
func (b *BrandModel) FromEntity(entity *entities.BrandEntity) {
	// Only set ID if it's not zero (for updates)
	if entity.ID != uuid.Nil {
		b.ID = entity.ID
	}
	b.Name = entity.Name
	b.Description = entity.Description
	// Let GORM handle timestamps
	if !entity.CreatedAt.IsZero() {
		b.CreatedAt = entity.CreatedAt
	}
	if !entity.UpdatedAt.IsZero() {
		b.UpdatedAt = entity.UpdatedAt
	}
	b.DeletedAt = entity.DeletedAt
}

// ToEntity converts the BrandModel to a BrandEntity.
func (b *BrandModel) ToEntity() entities.BrandEntity {
	return entities.BrandEntity{
		ID:          b.ID,
		Name:        b.Name,
		Description: b.Description,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
		DeletedAt:   b.DeletedAt,
	}
}
