package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/company/domain/entities"
	subscriptionModels "github.com/RodolfoBonis/spooliq/features/subscriptions/data/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CompanyModel represents the company data model for GORM
type CompanyModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"organization_id"`
	Name           string    `gorm:"type:varchar(255);not null" json:"name"`
	TradeName      *string   `gorm:"type:varchar(255)" json:"trade_name"`
	Document       *string   `gorm:"type:varchar(20)" json:"document"`
	Email          *string   `gorm:"type:varchar(255)" json:"email"`
	Phone          *string   `gorm:"type:varchar(50)" json:"phone"`
	WhatsApp       *string   `gorm:"type:varchar(50)" json:"whatsapp"`
	Instagram      *string   `gorm:"type:varchar(255)" json:"instagram"`
	Website        *string   `gorm:"type:varchar(255)" json:"website"`
	LogoURL        *string   `gorm:"type:varchar(500)" json:"logo_url"`
	Address        *string   `gorm:"type:text" json:"address"`
	City           *string   `gorm:"type:varchar(100)" json:"city"`
	State          *string   `gorm:"type:varchar(100)" json:"state"`
	ZipCode        *string   `gorm:"type:varchar(20)" json:"zip_code"`

	// Subscription fields
	SubscriptionStatus    string     `gorm:"type:varchar(20);not null;default:'trial';index" json:"subscription_status"`
	SubscriptionPlanID    *uuid.UUID `gorm:"type:uuid;index" json:"subscription_plan_id"` // FK to subscription_plans(id) - Nullable for trial
	StatusUpdatedAt       time.Time  `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP" json:"status_updated_at"`
	IsPlatformCompany     bool       `gorm:"not null;default:false" json:"is_platform_company"`
	TrialEndsAt           *time.Time `gorm:"type:timestamp" json:"trial_ends_at"`
	SubscriptionStartedAt *time.Time `gorm:"type:timestamp" json:"subscription_started_at"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// GORM v2 Relationships
	// 1:1 - Company has one Branding
	Branding *CompanyBrandingModel `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"branding,omitempty"`

	// N:1 - Company belongs to one Subscription Plan
	CurrentPlan *subscriptionModels.SubscriptionPlanModel `gorm:"foreignKey:SubscriptionPlanID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"current_plan,omitempty"`

	// 1:1 - Company has one Payment Gateway Link
	PaymentGateway *subscriptionModels.PaymentGatewayLinkModel `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;-:migration" json:"payment_gateway,omitempty"`

	// 1:N - Company has many Payment Methods
	PaymentMethods []subscriptionModels.PaymentMethodModel `gorm:"foreignKey:OrganizationID;references:OrganizationID;-:migration" json:"payment_methods,omitempty"`

	// 1:N - Company has many Subscription Payments
	Payments []subscriptionModels.SubscriptionModel `gorm:"foreignKey:OrganizationID;references:OrganizationID;-:migration" json:"payments,omitempty"`
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
	entity := &entities.CompanyEntity{
		ID:                    c.ID,
		OrganizationID:        c.OrganizationID,
		Name:                  c.Name,
		TradeName:             c.TradeName,
		Document:              c.Document,
		Email:                 c.Email,
		Phone:                 c.Phone,
		WhatsApp:              c.WhatsApp,
		Instagram:             c.Instagram,
		Website:               c.Website,
		LogoURL:               c.LogoURL,
		Address:               c.Address,
		City:                  c.City,
		State:                 c.State,
		ZipCode:               c.ZipCode,
		SubscriptionStatus:    c.SubscriptionStatus,
		SubscriptionPlanID:    c.SubscriptionPlanID,
		StatusUpdatedAt:       c.StatusUpdatedAt,
		IsPlatformCompany:     c.IsPlatformCompany,
		TrialEndsAt:           c.TrialEndsAt,
		SubscriptionStartedAt: c.SubscriptionStartedAt,
		CreatedAt:             c.CreatedAt,
		UpdatedAt:             c.UpdatedAt,
		DeletedAt:             getDeletedAt(c.DeletedAt),
	}

	// Convert CurrentPlan if available
	if c.CurrentPlan != nil {
		entity.CurrentPlan = c.CurrentPlan.ToEntity()
	}

	return entity
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
	c.SubscriptionStatus = entity.SubscriptionStatus
	c.SubscriptionPlanID = entity.SubscriptionPlanID
	c.StatusUpdatedAt = entity.StatusUpdatedAt
	c.IsPlatformCompany = entity.IsPlatformCompany
	c.TrialEndsAt = entity.TrialEndsAt
	c.SubscriptionStartedAt = entity.SubscriptionStartedAt
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
