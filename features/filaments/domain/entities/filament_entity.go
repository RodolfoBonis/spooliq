package entities

import (
	"time"

	metadataEntities "github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/entities"
	"github.com/jinzhu/gorm"
)

// Filament representa um filamento para impressão 3D
// @Description Filamento para impressão 3D
// @Example {"id": 1, "name": "PLA Branco", "brand": "SUNLU", "material": "PLA", "color": "Branco", "price_per_kg": 125.0, "url": "https://amazon.com.br/dp/B07PGYHYV8"}
type Filament struct {
	ID   uint   `gorm:"primary_key;auto_increment" json:"id"`
	Name string `gorm:"type:varchar(255);not null" json:"name" validate:"required,min=1,max=255"`

	// Foreign keys para metadados
	BrandID    uint `gorm:"not null;index" json:"brand_id" validate:"required"`
	MaterialID uint `gorm:"not null;index" json:"material_id" validate:"required"`

	// Relacionamentos
	Brand    metadataEntities.FilamentBrand    `gorm:"foreignkey:BrandID" json:"brand"`
	Material metadataEntities.FilamentMaterial `gorm:"foreignkey:MaterialID" json:"material"`

	Color         string     `gorm:"type:varchar(100);not null" json:"color" validate:"required,min=1,max=100"`
	ColorHex      string     `gorm:"type:varchar(7)" json:"color_hex" validate:"omitempty,hexcolor"`
	Diameter      float64    `gorm:"type:decimal(3,2);not null" json:"diameter" validate:"required,min=0,max=10"`
	Weight        *float64   `gorm:"type:decimal(8,2)" json:"weight,omitempty" validate:"omitempty,min=0"`
	PricePerKg    float64    `gorm:"type:decimal(10,2);not null" json:"price_per_kg" validate:"required,min=0"`
	PricePerMeter *float64   `gorm:"type:decimal(10,4)" json:"price_per_meter,omitempty" validate:"omitempty,min=0"`
	URL           string     `gorm:"type:text" json:"url" validate:"omitempty,url"`
	OwnerUserID   *string    `gorm:"type:varchar(255);index" json:"owner_user_id,omitempty"` // null = catálogo global (admin), string = Keycloak User ID
	CreatedAt     time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt     *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName especifica o nome da tabela para o GORM
func (Filament) TableName() string {
	return "filaments"
}

// BeforeCreate é um hook do GORM executado antes de criar um filamento
func (f *Filament) BeforeCreate(scope *gorm.Scope) error {
	f.CreatedAt = time.Now()
	f.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate é um hook do GORM executado antes de atualizar um filamento
func (f *Filament) BeforeUpdate(scope *gorm.Scope) error {
	f.UpdatedAt = time.Now()
	return nil
}

// IsGlobal verifica se o filamento é do catálogo global (sem dono)
func (f *Filament) IsGlobal() bool {
	return f.OwnerUserID == nil
}

// CanUserAccess verifica se um usuário pode acessar este filamento
func (f *Filament) CanUserAccess(userID string, isAdmin bool) bool {
	// Admin pode acessar tudo
	if isAdmin {
		return true
	}
	// Filamentos globais são acessíveis a todos
	if f.IsGlobal() {
		return true
	}
	// Usuário pode acessar seus próprios filamentos
	return f.OwnerUserID != nil && *f.OwnerUserID == userID
}
