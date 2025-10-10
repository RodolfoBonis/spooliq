package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/company/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CompanyModel represents the company data model for GORM
type CompanyModel struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"organization_id"`
	Name           string         `gorm:"type:varchar(255);not null" json:"name"`
	TradeName      *string        `gorm:"type:varchar(255)" json:"trade_name"`
	Document       *string        `gorm:"type:varchar(20)" json:"document"`
	Email          *string        `gorm:"type:varchar(255)" json:"email"`
	Phone          *string        `gorm:"type:varchar(50)" json:"phone"`
	WhatsApp       *string        `gorm:"type:varchar(50)" json:"whatsapp"`
	Instagram      *string        `gorm:"type:varchar(255)" json:"instagram"`
	Website        *string        `gorm:"type:varchar(255)" json:"website"`
	LogoURL        *string        `gorm:"type:varchar(500)" json:"logo_url"`
	Address        *string        `gorm:"type:text" json:"address"`
	City           *string        `gorm:"type:varchar(100)" json:"city"`
	State          *string        `gorm:"type:varchar(100)" json:"state"`
	ZipCode        *string        `gorm:"type:varchar(20)" json:"zip_code"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// TableName specifies the table name for GORM
func (CompanyModel) TableName() string {
	return "companies"
}

// BeforeCreate is a GORM hook executed before creating a company
func (c *CompanyModel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// ToEntity converts the GORM model to domain entity
func (c *CompanyModel) ToEntity() *entities.CompanyEntity {
	return &entities.CompanyEntity{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		Name:           c.Name,
		TradeName:      c.TradeName,
		Document:       c.Document,
		Email:          c.Email,
		Phone:          c.Phone,
		WhatsApp:       c.WhatsApp,
		Instagram:      c.Instagram,
		Website:        c.Website,
		LogoURL:        c.LogoURL,
		Address:        c.Address,
		City:           c.City,
		State:          c.State,
		ZipCode:        c.ZipCode,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
		DeletedAt:      getDeletedAt(c.DeletedAt),
	}
}

// FromEntity converts domain entity to GORM model
func (c *CompanyModel) FromEntity(entity *entities.CompanyEntity) {
	c.ID = entity.ID
	c.OrganizationID = entity.OrganizationID
	c.Name = entity.Name
	c.TradeName = entity.TradeName
	c.Document = entity.Document
	c.Email = entity.Email
	c.Phone = entity.Phone
	c.WhatsApp = entity.WhatsApp
	c.Instagram = entity.Instagram
	c.Website = entity.Website
	c.LogoURL = entity.LogoURL
	c.Address = entity.Address
	c.City = entity.City
	c.State = entity.State
	c.ZipCode = entity.ZipCode
	c.CreatedAt = entity.CreatedAt
	c.UpdatedAt = entity.UpdatedAt
	if entity.DeletedAt != nil {
		c.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}
}

// getDeletedAt returns nil if deleted_at is not valid, otherwise returns pointer to time
func getDeletedAt(deletedAt gorm.DeletedAt) *time.Time {
	if deletedAt.Valid {
		return &deletedAt.Time
	}
	return nil
}
