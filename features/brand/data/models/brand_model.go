package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BrandModel represents the database model for brand entities.
type BrandModel struct {
	ID          uuid.UUID  `gorm:"<-:create;type:uuid;primaryKey" json:"id"`
	Name        string     `gorm:"type:varchar(255);not null" json:"name"`
	Description string     `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time  `gorm:"<-:create;type:timestamp;" json:"created_at,omitempty"`
	UpdatedAt   time.Time  `gorm:"<-:update;type:timestamp;" json:"updated_at,omitempty"`
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate is a GORM hook that sets the UUID before creating a brand record.
func (b *BrandModel) BeforeCreate(tx *gorm.DB) (err error) {
	b.ID = uuid.New()
	return
}

// TableName returns the table name for the brand model.
func (b *BrandModel) TableName() string { return "brands" }

// FromEntity populates the BrandModel from a BrandEntity.
func (b *BrandModel) FromEntity(entity *entities.BrandEntity) {
	b.ID = entity.ID
	b.Name = entity.Name
	b.Description = entity.Description
	b.CreatedAt = entity.CreatedAt
	b.UpdatedAt = entity.UpdatedAt
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
