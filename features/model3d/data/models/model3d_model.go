package models

import (
	"time"

	companyModels "github.com/RodolfoBonis/spooliq/features/company/data/models"
	customerModels "github.com/RodolfoBonis/spooliq/features/customer/data/models"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Model3DModel represents the database model for 3D model entities.
type Model3DModel struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string         `gorm:"type:varchar(255);not null;index:idx_model3d_org" json:"organization_id"`
	CustomerID     *uuid.UUID     `gorm:"type:uuid;index:idx_model3d_customer" json:"customer_id,omitempty"`
	Name           string         `gorm:"type:varchar(255);not null" json:"name"`
	Description    string         `gorm:"type:text" json:"description,omitempty"`
	FileName       string         `gorm:"type:varchar(255);not null" json:"file_name"`
	FileURL        string         `gorm:"type:varchar(1024);not null" json:"file_url"`
	FileFormat     string         `gorm:"type:varchar(10);not null" json:"file_format"`
	FileSizeBytes  int64          `gorm:"type:bigint;not null" json:"file_size_bytes"`
	FileHash       string         `gorm:"type:varchar(64);not null;index:idx_model3d_hash" json:"file_hash"`
	ThumbnailURL   *string        `gorm:"type:varchar(1024)" json:"thumbnail_url,omitempty"`
	Notes          *string        `gorm:"type:text" json:"notes,omitempty"`
	Tags           *string        `gorm:"type:text" json:"tags,omitempty"`
	OwnerUserID    string         `gorm:"type:varchar(255);not null" json:"owner_user_id"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// GORM v2 Relationships
	Organization *companyModels.CompanyModel   `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"organization,omitempty"`
	Customer     *customerModels.CustomerModel `gorm:"foreignKey:CustomerID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"customer,omitempty"`
}

// TableName returns the table name for the model3d model.
func (Model3DModel) TableName() string { return "models_3d" }

// BeforeCreate is a GORM hook executed before creating a model3d record.
func (m *Model3DModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// ToEntity converts the Model3DModel to a Model3DEntity.
func (m *Model3DModel) ToEntity() *entities.Model3DEntity {
	var deletedAt *time.Time
	if m.DeletedAt.Valid {
		deletedAt = &m.DeletedAt.Time
	}

	return &entities.Model3DEntity{
		ID:             m.ID,
		OrganizationID: m.OrganizationID,
		CustomerID:     m.CustomerID,
		Name:           m.Name,
		Description:    m.Description,
		FileName:       m.FileName,
		FileURL:        m.FileURL,
		FileFormat:     m.FileFormat,
		FileSizeBytes:  m.FileSizeBytes,
		FileHash:       m.FileHash,
		ThumbnailURL:   m.ThumbnailURL,
		Notes:          m.Notes,
		Tags:           m.Tags,
		OwnerUserID:    m.OwnerUserID,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		DeletedAt:      deletedAt,
	}
}

// FromEntity populates the Model3DModel from a Model3DEntity.
func (m *Model3DModel) FromEntity(entity *entities.Model3DEntity) {
	if entity.ID != uuid.Nil {
		m.ID = entity.ID
	}
	m.OrganizationID = entity.OrganizationID
	m.CustomerID = entity.CustomerID
	m.Name = entity.Name
	m.Description = entity.Description
	m.FileName = entity.FileName
	m.FileURL = entity.FileURL
	m.FileFormat = entity.FileFormat
	m.FileSizeBytes = entity.FileSizeBytes
	m.FileHash = entity.FileHash
	m.ThumbnailURL = entity.ThumbnailURL
	m.Notes = entity.Notes
	m.Tags = entity.Tags
	m.OwnerUserID = entity.OwnerUserID
	m.CreatedAt = entity.CreatedAt
	m.UpdatedAt = entity.UpdatedAt
	if entity.DeletedAt != nil {
		m.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}
}
