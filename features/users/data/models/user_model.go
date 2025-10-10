package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserModel represents the user data model for GORM
type UserModel struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID string         `gorm:"type:varchar(255);not null;index"`
	KeycloakUserID string         `gorm:"type:varchar(255);unique;not null;index"`
	Email          string         `gorm:"type:varchar(255);unique;not null"`
	Name           string         `gorm:"type:varchar(255);not null"`
	UserType       string         `gorm:"type:varchar(20);not null"` // owner, admin, user
	IsActive       bool           `gorm:"default:true"`
	CreatedAt      time.Time      `gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (UserModel) TableName() string {
	return "users"
}

// BeforeCreate is a GORM hook executed before creating a user
func (u *UserModel) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// ToEntity converts the GORM model to domain entity
func (u *UserModel) ToEntity() *entities.UserEntity {
	return &entities.UserEntity{
		ID:             u.ID,
		OrganizationID: u.OrganizationID,
		KeycloakUserID: u.KeycloakUserID,
		Email:          u.Email,
		Name:           u.Name,
		UserType:       u.UserType,
		IsActive:       u.IsActive,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
		DeletedAt:      getDeletedAt(u.DeletedAt),
	}
}

// FromEntity converts domain entity to GORM model
func (u *UserModel) FromEntity(entity *entities.UserEntity) {
	u.ID = entity.ID
	u.OrganizationID = entity.OrganizationID
	u.KeycloakUserID = entity.KeycloakUserID
	u.Email = entity.Email
	u.Name = entity.Name
	u.UserType = entity.UserType
	u.IsActive = entity.IsActive
	u.CreatedAt = entity.CreatedAt
	u.UpdatedAt = entity.UpdatedAt
	if entity.DeletedAt != nil {
		u.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}
}

// getDeletedAt returns nil if deleted_at is not valid, otherwise returns pointer to time
func getDeletedAt(deletedAt gorm.DeletedAt) *time.Time {
	if deletedAt.Valid {
		return &deletedAt.Time
	}
	return nil
}
