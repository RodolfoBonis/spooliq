package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/google/uuid"
)

type BrandModel struct {
	ID          uuid.UUID  `gorm:"<-:create;type:uuid;PRIMARY_KEY;" json:"id"`
	Name        string     `gorm:"type:varchar(255);not null" json:"name"`
	Description string     `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time  `gorm:"<-:create;type:timestamp;" json:"created_at,omitempty"`
	UpdatedAt   time.Time  `gorm:"<-:update;type:timestamp;" json:"updated_at,omitempty"`
	DeletedAt   *time.Time `sql:"index" json:"deleted_at,omitempty"`
}

func (b *BrandModel) BeforeCreate() (err error) {
	b.ID = uuid.New()
	return
}

func (b *BrandModel) TableName() string { return "brands" }

func (b *BrandModel) FromEntity(entity *entities.BrandEntity) {
	b.ID = entity.ID
	b.Name = entity.Name
	b.Description = entity.Description
	b.CreatedAt = entity.CreatedAt
	b.UpdatedAt = entity.UpdatedAt
	b.DeletedAt = entity.DeletedAt
	return
}

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
