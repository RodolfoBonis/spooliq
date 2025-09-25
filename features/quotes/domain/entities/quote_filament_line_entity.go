package entities

import (
	"time"

	filament_entities "github.com/RodolfoBonis/spooliq/features/filaments/domain/entities"
	"github.com/jinzhu/gorm"
)

// QuoteFilamentLine representa uma linha de filamento em um orçamento
// @Description Linha de filamento em um orçamento de impressão 3D
type QuoteFilamentLine struct {
	ID               uint                        `gorm:"primary_key;auto_increment" json:"id"`
	QuoteID          uint                        `gorm:"not null;index" json:"quote_id" validate:"required"`
	FilamentID       uint                        `gorm:"not null;index" json:"filament_id" validate:"required"`
	Filament         *filament_entities.Filament `gorm:"foreignkey:FilamentID" json:"filament,omitempty"`
	WeightGrams      float64                     `gorm:"type:decimal(10,2);not null" json:"weight_grams" validate:"required,min=0"` // em gramas
	LengthMeters     *float64                    `gorm:"type:decimal(10,2)" json:"length_meters,omitempty"`                         // comprimento estimado em metros
	PrintTimeSeconds int                         `gorm:"type:integer;default:0" json:"print_time_seconds"`                          // tempo de impressão em segundos
	Cost             float64                     `gorm:"type:decimal(10,2);default:0" json:"cost"`                                  // custo calculado
	Notes            string                      `gorm:"type:text" json:"notes"`
	// Snapshot dos dados do filamento (para preservar histórico)
	FilamentSnapshotName          string    `gorm:"type:varchar(255)" json:"filament_snapshot_name"`
	FilamentSnapshotBrand         string    `gorm:"type:varchar(255)" json:"filament_snapshot_brand"`
	FilamentSnapshotMaterial      string    `gorm:"type:varchar(100)" json:"filament_snapshot_material"`
	FilamentSnapshotColor         string    `gorm:"type:varchar(100)" json:"filament_snapshot_color"`
	FilamentSnapshotColorHex      string    `gorm:"type:varchar(7)" json:"filament_snapshot_color_hex"`
	FilamentSnapshotPricePerKg    float64   `gorm:"type:decimal(10,2)" json:"filament_snapshot_price_per_kg"`
	FilamentSnapshotPricePerMeter *float64  `gorm:"type:decimal(10,4)" json:"filament_snapshot_price_per_meter,omitempty"`
	FilamentSnapshotURL           string    `gorm:"type:text" json:"filament_snapshot_url"`
	CreatedAt                     time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt                     time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName especifica o nome da tabela para o GORM
func (QuoteFilamentLine) TableName() string {
	return "quote_filament_lines"
}

// BeforeCreate é um hook do GORM executado antes de criar uma linha de filamento
func (qfl *QuoteFilamentLine) BeforeCreate(scope *gorm.Scope) error {
	qfl.CreatedAt = time.Now()
	qfl.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate é um hook do GORM executado antes de atualizar uma linha de filamento
func (qfl *QuoteFilamentLine) BeforeUpdate(scope *gorm.Scope) error {
	qfl.UpdatedAt = time.Now()
	return nil
}
