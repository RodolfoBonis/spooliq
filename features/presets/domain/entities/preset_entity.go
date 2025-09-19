package entities

import (
	"encoding/json"
	"time"

	"github.com/jinzhu/gorm"
)

// Preset representa um preset de configuração (tarifas, máquinas, etc.)
// @Description Preset de configuração do sistema
// @Example {"id": 1, "key": "energy_maceio_al_2025", "data": {"base_tariff": 0.804, "flag_surcharge": 0, "location": "Maceió-AL", "year": 2025}}
type Preset struct {
	ID        uint       `gorm:"primary_key;auto_increment" json:"id"`
	Key       string     `gorm:"type:varchar(255);unique_index;not null" json:"key" validate:"required,min=1,max=255"`
	Data      string     `gorm:"type:text;not null" json:"data" validate:"required"` // JSON blob
	CreatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName especifica o nome da tabela para o GORM
func (Preset) TableName() string {
	return "presets"
}

// BeforeCreate é um hook do GORM executado antes de criar um preset
func (p *Preset) BeforeCreate(scope *gorm.Scope) error {
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate é um hook do GORM executado antes de atualizar um preset
func (p *Preset) BeforeUpdate(scope *gorm.Scope) error {
	p.UpdatedAt = time.Now()
	return nil
}

// GetDataAsMap retorna os dados do preset como um map
func (p *Preset) GetDataAsMap() (map[string]interface{}, error) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(p.Data), &data)
	return data, err
}

// SetDataFromMap define os dados do preset a partir de um map
func (p *Preset) SetDataFromMap(data map[string]interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	p.Data = string(jsonData)
	return nil
}

// UnmarshalDataTo deserializa os dados para uma estrutura específica
func (p *Preset) UnmarshalDataTo(target interface{}) error {
	return json.Unmarshal([]byte(p.Data), target)
}

// MarshalDataFrom serializa dados de uma estrutura específica
func (p *Preset) MarshalDataFrom(source interface{}) error {
	jsonData, err := json.Marshal(source)
	if err != nil {
		return err
	}
	p.Data = string(jsonData)
	return nil
}

// Estruturas específicas para diferentes tipos de presets

// EnergyPreset representa um preset de tarifa energética
type EnergyPreset struct {
	BaseTariff    float64 `json:"base_tariff" validate:"required,min=0"`
	FlagSurcharge float64 `json:"flag_surcharge" validate:"min=0"`
	Location      string  `json:"location" validate:"required"`
	Year          int     `json:"year" validate:"required,min=2020"`
	Description   string  `json:"description,omitempty"`
}

// MachinePreset representa um preset de máquina/impressora
type MachinePreset struct {
	Name        string  `json:"name" validate:"required"`
	Brand       string  `json:"brand" validate:"required"`
	Model       string  `json:"model" validate:"required"`
	Watt        float64 `json:"watt" validate:"required,min=0"`
	IdleFactor  float64 `json:"idle_factor" validate:"min=0,max=1"`
	Description string  `json:"description,omitempty"`
	URL         string  `json:"url,omitempty"`
}
