package models

import (
	"time"

	filament_entities "github.com/RodolfoBonis/spooliq/features/filaments/domain/entities"
	"github.com/jinzhu/gorm"
)

// QuoteModel representa um orçamento de impressão 3D no banco de dados
type QuoteModel struct {
	ID          uint       `gorm:"primary_key;auto_increment"`
	Title       string     `gorm:"type:varchar(255);not null"`
	Notes       string     `gorm:"type:text"`
	OwnerUserID string     `gorm:"type:varchar(255);not null;index"` // Keycloak User ID
	CreatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	DeletedAt   *time.Time `gorm:"index"`

	// Relacionamentos
	FilamentLines  []QuoteFilamentLineModel `gorm:"foreignkey:QuoteID"`
	MachineProfile *MachineProfileModel     `gorm:"foreignkey:QuoteID"`
	EnergyProfile  *EnergyProfileModel      `gorm:"foreignkey:QuoteID"`
	CostProfile    *CostProfileModel        `gorm:"foreignkey:QuoteID"`
	MarginProfile  *MarginProfileModel      `gorm:"foreignkey:QuoteID"`
}

// TableName especifica o nome da tabela para o GORM
func (QuoteModel) TableName() string {
	return "quotes"
}

// BeforeCreate é um hook do GORM executado antes de criar um orçamento
func (q *QuoteModel) BeforeCreate(scope *gorm.Scope) error {
	q.CreatedAt = time.Now()
	q.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate é um hook do GORM executado antes de atualizar um orçamento
func (q *QuoteModel) BeforeUpdate(scope *gorm.Scope) error {
	q.UpdatedAt = time.Now()
	return nil
}

// QuoteFilamentLineModel representa uma linha de filamento no banco de dados
type QuoteFilamentLineModel struct {
	ID         uint                        `gorm:"primary_key;auto_increment"`
	QuoteID    uint                        `gorm:"not null;index"`
	FilamentID uint                        `gorm:"not null;index"`
	Filament   *filament_entities.Filament `gorm:"foreignkey:FilamentID"`

	// Snapshot dos dados do filamento (para preservar histórico)
	FilamentSnapshotName          string     `gorm:"type:varchar(255);not null"`
	FilamentSnapshotBrand         string     `gorm:"type:varchar(255);not null"`
	FilamentSnapshotMaterial      string     `gorm:"type:varchar(100);not null"`
	FilamentSnapshotColor         string     `gorm:"type:varchar(100);not null"`
	FilamentSnapshotColorHex      string     `gorm:"type:varchar(7)"`
	FilamentSnapshotPricePerKg    float64    `gorm:"type:decimal(10,2);not null"`
	FilamentSnapshotPricePerMeter *float64   `gorm:"type:decimal(10,4)"`
	FilamentSnapshotURL           string     `gorm:"type:text"`
	WeightGrams                   float64    `gorm:"type:decimal(10,3);not null"`
	LengthMeters                  *float64   `gorm:"type:decimal(10,3)"`
	CreatedAt                     time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt                     time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	DeletedAt                     *time.Time `gorm:"index"`
}

// TableName especifica o nome da tabela para o GORM
func (QuoteFilamentLineModel) TableName() string {
	return "quote_filament_lines"
}

// MachineProfileModel representa um perfil de máquina no banco de dados
type MachineProfileModel struct {
	ID          uint       `gorm:"primary_key;auto_increment"`
	QuoteID     uint       `gorm:"not null;index"`
	Name        string     `gorm:"type:varchar(255);not null"`
	Brand       string     `gorm:"type:varchar(255);not null"`
	Model       string     `gorm:"type:varchar(255);not null"`
	Watt        float64    `gorm:"type:decimal(10,2);not null"`
	IdleFactor  float64    `gorm:"type:decimal(5,4);not null;default:0"`
	Description string     `gorm:"type:text"`
	URL         string     `gorm:"type:text"`
	CreatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	DeletedAt   *time.Time `gorm:"index"`
}

// TableName especifica o nome da tabela para o GORM
func (MachineProfileModel) TableName() string {
	return "machine_profiles"
}

// EnergyProfileModel representa um perfil de energia no banco de dados
type EnergyProfileModel struct {
	ID            uint       `gorm:"primary_key;auto_increment"`
	QuoteID       uint       `gorm:"not null;index"`
	Name          string     `gorm:"type:varchar(255);not null"`
	BaseTariff    float64    `gorm:"type:decimal(10,4);not null"`
	FlagSurcharge float64    `gorm:"type:decimal(10,4);not null;default:0"`
	Location      string     `gorm:"type:varchar(255);not null"`
	Year          int        `gorm:"not null"`
	Description   string     `gorm:"type:text"`
	OwnerUserID   *string    `gorm:"type:varchar(255);index"`
	CreatedAt     time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	DeletedAt     *time.Time `gorm:"index"`
}

// TableName especifica o nome da tabela para o GORM
func (EnergyProfileModel) TableName() string {
	return "energy_profiles"
}

// CostProfileModel representa um perfil de custos no banco de dados
type CostProfileModel struct {
	ID             uint       `gorm:"primary_key;auto_increment"`
	QuoteID        uint       `gorm:"not null;index"`
	Name           string     `gorm:"type:varchar(255);not null"`
	WearPercentage float64    `gorm:"type:decimal(5,2);not null;default:0"`
	OverheadAmount float64    `gorm:"type:decimal(10,2);not null;default:0"`
	Description    string     `gorm:"type:text"`
	CreatedAt      time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt      time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	DeletedAt      *time.Time `gorm:"index"`
}

// TableName especifica o nome da tabela para o GORM
func (CostProfileModel) TableName() string {
	return "cost_profiles"
}

// MarginProfileModel representa um perfil de margens no banco de dados
type MarginProfileModel struct {
	ID                  uint       `gorm:"primary_key;auto_increment"`
	QuoteID             uint       `gorm:"not null;index"`
	Name                string     `gorm:"type:varchar(255);not null"`
	PrintingOnlyMargin  float64    `gorm:"type:decimal(8,2);not null;default:0"`
	PrintingPlusMargin  float64    `gorm:"type:decimal(8,2);not null;default:0"`
	FullServiceMargin   float64    `gorm:"type:decimal(8,2);not null;default:0"`
	OperatorRatePerHour float64    `gorm:"type:decimal(10,2);not null;default:0"`
	ModelerRatePerHour  float64    `gorm:"type:decimal(10,2);not null;default:0"`
	Description         string     `gorm:"type:text"`
	CreatedAt           time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt           time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	DeletedAt           *time.Time `gorm:"index"`
}

// TableName especifica o nome da tabela para o GORM
func (MarginProfileModel) TableName() string {
	return "margin_profiles"
}
