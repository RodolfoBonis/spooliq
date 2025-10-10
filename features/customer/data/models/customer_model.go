package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CustomerModel represents the customer data model for GORM
type CustomerModel struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string         `gorm:"type:varchar(255);not null;index:idx_customer_org" json:"organization_id"`
	Name           string         `gorm:"type:varchar(255);not null" json:"name"`
	Email          *string        `gorm:"type:varchar(255);uniqueIndex:idx_customer_org_email" json:"email"`
	Phone          *string        `gorm:"type:varchar(50)" json:"phone"`
	Document       *string        `gorm:"type:varchar(50)" json:"document"`
	Address        *string        `gorm:"type:varchar(500)" json:"address"`
	City           *string        `gorm:"type:varchar(255)" json:"city"`
	State          *string        `gorm:"type:varchar(100)" json:"state"`
	ZipCode        *string        `gorm:"type:varchar(20)" json:"zip_code"`
	Notes          *string        `gorm:"type:text" json:"notes"`
	OwnerUserID    string         `gorm:"type:varchar(255);not null;index" json:"owner_user_id"`
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// TableName specifies the table name for GORM
func (CustomerModel) TableName() string {
	return "customers"
}

// BeforeCreate is a GORM hook executed before creating a customer
func (c *CustomerModel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// ToEntity converts the GORM model to domain entity
func (c *CustomerModel) ToEntity() *entities.CustomerEntity {
	return &entities.CustomerEntity{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		Name:           c.Name,
		Email:          c.Email,
		Phone:          c.Phone,
		Document:       c.Document,
		Address:        c.Address,
		City:           c.City,
		State:          c.State,
		ZipCode:        c.ZipCode,
		Notes:          c.Notes,
		OwnerUserID:    c.OwnerUserID,
		IsActive:       c.IsActive,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
		DeletedAt:      getDeletedAt(c.DeletedAt),
	}
}

// getDeletedAt returns nil if deleted_at is not valid, otherwise returns pointer to time
func getDeletedAt(deletedAt gorm.DeletedAt) *time.Time {
	if deletedAt.Valid {
		return &deletedAt.Time
	}
	return nil
}

// FromEntity converts domain entity to GORM model
func (c *CustomerModel) FromEntity(entity *entities.CustomerEntity) {
	c.ID = entity.ID
	c.OrganizationID = entity.OrganizationID
	c.Name = entity.Name
	c.Email = entity.Email
	c.Phone = entity.Phone
	c.Document = entity.Document
	c.Address = entity.Address
	c.City = entity.City
	c.State = entity.State
	c.ZipCode = entity.ZipCode
	c.Notes = entity.Notes
	c.OwnerUserID = entity.OwnerUserID
	c.IsActive = entity.IsActive
	c.CreatedAt = entity.CreatedAt
	c.UpdatedAt = entity.UpdatedAt
	if entity.DeletedAt != nil {
		c.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}
}
