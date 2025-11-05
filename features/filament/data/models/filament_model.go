package models

import (
	"encoding/json"
	"time"

	brandModels "github.com/RodolfoBonis/spooliq/features/brand/data/models"
	companyModels "github.com/RodolfoBonis/spooliq/features/company/data/models"
	"github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	materialModels "github.com/RodolfoBonis/spooliq/features/material/data/models"
	userModels "github.com/RodolfoBonis/spooliq/features/users/data/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FilamentModel represents the filament data model for GORM
type FilamentModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;index:idx_filament_org;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"organization_id"`
	Name           string    `gorm:"type:varchar(255);not null" json:"name"`
	Description    string    `gorm:"type:text" json:"description"`

	// Foreign keys with relationships
	BrandID    uuid.UUID `gorm:"type:uuid;not null;index" json:"brand_id"`
	MaterialID uuid.UUID `gorm:"type:uuid;not null;index" json:"material_id"`

	// Legacy color fields
	Color    string `gorm:"type:varchar(100);not null" json:"color"`
	ColorHex string `gorm:"type:varchar(7)" json:"color_hex"`

	// Advanced color system
	ColorType    string `gorm:"type:varchar(20)" json:"color_type"`
	ColorData    string `gorm:"type:text" json:"color_data"`
	ColorPreview string `gorm:"type:text" json:"color_preview"`

	// Physical properties
	Diameter   float64  `gorm:"type:numeric;not null" json:"diameter"`
	Weight     *float64 `gorm:"type:numeric" json:"weight"`
	PricePerKg float64  `gorm:"type:numeric;not null" json:"price_per_kg"`

	// Additional properties
	URL string `gorm:"type:text" json:"url"`

	// Ownership and access control
	OwnerUserID *string `gorm:"type:varchar(255);index" json:"owner_user_id"`
	IsActive    bool    `gorm:"" json:"is_active"`

	// GORM v2 Relationships - BelongsTo
	Brand        *brandModels.BrandModel       `gorm:"foreignKey:BrandID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"brand,omitempty"`
	Material     *materialModels.MaterialModel `gorm:"foreignKey:MaterialID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"material,omitempty"`
	User         *userModels.UserModel         `gorm:"foreignKey:OwnerUserID;references:KeycloakUserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"user,omitempty"`
	Organization *companyModels.CompanyModel   `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"organization,omitempty"`

	// Technical specifications
	PrintTemperature *int `gorm:"type:integer" json:"print_temperature"`
	BedTemperature   *int `gorm:"type:integer" json:"bed_temperature"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// TableName specifies the table name for GORM
func (FilamentModel) TableName() string {
	return "filaments"
}

// BeforeCreate is a GORM hook executed before creating a filament
func (f *FilamentModel) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}

// ToEntity converts the GORM model to domain entity
func (f *FilamentModel) ToEntity() *entities.FilamentEntity {
	return &entities.FilamentEntity{
		ID:               f.ID,
		OrganizationID:   f.OrganizationID,
		Name:             f.Name,
		Description:      f.Description,
		BrandID:          f.BrandID,
		MaterialID:       f.MaterialID,
		Color:            f.Color,
		ColorHex:         f.ColorHex,
		ColorType:        entities.ColorType(f.ColorType),
		ColorData:        json.RawMessage(f.ColorData),
		ColorPreview:     f.ColorPreview,
		Diameter:         f.Diameter,
		Weight:           f.Weight,
		PricePerKg:       f.PricePerKg,
		URL:              f.URL,
		OwnerUserID:      f.OwnerUserID,
		IsActive:         f.IsActive,
		PrintTemperature: f.PrintTemperature,
		BedTemperature:   f.BedTemperature,
		CreatedAt:        f.CreatedAt,
		UpdatedAt:        f.UpdatedAt,
		DeletedAt:        getDeletedAt(f.DeletedAt),
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
func (f *FilamentModel) FromEntity(entity *entities.FilamentEntity) {
	f.ID = entity.ID
	f.OrganizationID = entity.OrganizationID
	f.Name = entity.Name
	f.Description = entity.Description
	f.BrandID = entity.BrandID
	f.MaterialID = entity.MaterialID
	f.Color = entity.Color
	f.ColorHex = entity.ColorHex
	f.ColorType = string(entity.ColorType)
	f.ColorData = string(entity.ColorData)
	f.ColorPreview = entity.ColorPreview
	f.Diameter = entity.Diameter
	f.Weight = entity.Weight
	f.PricePerKg = entity.PricePerKg
	f.URL = entity.URL
	f.OwnerUserID = entity.OwnerUserID
	f.IsActive = entity.IsActive
	f.PrintTemperature = entity.PrintTemperature
	f.BedTemperature = entity.BedTemperature
	f.CreatedAt = entity.CreatedAt
	f.UpdatedAt = entity.UpdatedAt
	if entity.DeletedAt != nil {
		f.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}
}

// BrandInfo represents brand information for JOIN queries
type BrandInfo struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// MaterialInfo represents material information for JOIN queries
type MaterialInfo struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	TempTable    float32   `json:"temp_table"`
	TempExtruder float32   `json:"temp_extruder"`
}

// FilamentWithRelations represents a filament with its related brand and material
type FilamentWithRelations struct {
	FilamentModel
	Brand    *BrandInfo    `gorm:"embedded;embeddedPrefix:brand_" json:"brand"`
	Material *MaterialInfo `gorm:"embedded;embeddedPrefix:material_" json:"material"`
}
