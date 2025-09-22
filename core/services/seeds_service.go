package services

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/logger"
	presetsEntities "github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
)

// RunSeeds executa todos os seeds iniciais do sistema
func RunSeeds(logger logger.Logger) {
	ctx := context.Background()

	logger.Info(ctx, "Iniciando seeds do sistema...")

	// Seeds de presets
	seedEnergyPresets(logger)
	seedMachinePresets(logger)

	// Seeds de filamentos removidos - usar filament_metadata

	logger.Info(ctx, "Seeds concluídos com sucesso!")
}

// seedEnergyPresets cria presets de energia/tarifa
func seedEnergyPresets(logger logger.Logger) {
	ctx := context.Background()

	energyPresets := []struct {
		Key  string
		Data presetsEntities.EnergyPreset
	}{
		{
			Key: "energy_maceio_al_2025",
			Data: presetsEntities.EnergyPreset{
				BaseTariff:    0.804,
				FlagSurcharge: 0,
				Location:      "Maceió-AL",
				Year:          2025,
				Description:   "Tarifa energética padrão para Maceió, Alagoas em 2025",
			},
		},
		{
			Key: "energy_sao_paulo_sp_2025",
			Data: presetsEntities.EnergyPreset{
				BaseTariff:    0.892,
				FlagSurcharge: 0,
				Location:      "São Paulo-SP",
				Year:          2025,
				Description:   "Tarifa energética padrão para São Paulo, SP em 2025",
			},
		},
		{
			Key: "energy_rio_de_janeiro_rj_2025",
			Data: presetsEntities.EnergyPreset{
				BaseTariff:    0.756,
				FlagSurcharge: 0,
				Location:      "Rio de Janeiro-RJ",
				Year:          2025,
				Description:   "Tarifa energética padrão para Rio de Janeiro, RJ em 2025",
			},
		},
	}

	for _, presetData := range energyPresets {
		var existingPreset presetsEntities.Preset
		result := Connector.Where("key = ?", presetData.Key).First(&existingPreset)

		if result.RecordNotFound() {
			preset := presetsEntities.Preset{
				Key: presetData.Key,
			}

			err := preset.MarshalDataFrom(presetData.Data)
			if err != nil {
				logger.Error(ctx, "Erro ao serializar preset de energia", map[string]interface{}{
					"key":   presetData.Key,
					"error": err.Error(),
				})
				continue
			}

			if err := Connector.Create(&preset).Error; err != nil {
				logger.Error(ctx, "Erro ao criar preset de energia", map[string]interface{}{
					"key":   presetData.Key,
					"error": err.Error(),
				})
			} else {
				logger.Info(ctx, "Preset de energia criado", map[string]interface{}{
					"key":      presetData.Key,
					"location": presetData.Data.Location,
				})
			}
		}
	}
}

// seedMachinePresets cria presets de máquinas/impressoras
func seedMachinePresets(logger logger.Logger) {
	ctx := context.Background()

	machinePresets := []struct {
		Key  string
		Data presetsEntities.MachinePreset
	}{
		{
			Key: "machine_bambulab_a1_combo",
			Data: presetsEntities.MachinePreset{
				Name:        "BambuLab A1 Combo",
				Brand:       "BambuLab",
				Model:       "A1 Combo",
				Watt:        95,
				IdleFactor:  0,
				Description: "Impressora 3D BambuLab A1 Combo - consumo 95W",
				URL:         "https://bambulab.com/en/a1",
			},
		},
		{
			Key: "machine_ender3_v2",
			Data: presetsEntities.MachinePreset{
				Name:        "Creality Ender 3 V2",
				Brand:       "Creality",
				Model:       "Ender 3 V2",
				Watt:        270,
				IdleFactor:  0.1,
				Description: "Impressora 3D Creality Ender 3 V2 - consumo 270W",
				URL:         "https://www.creality.com/products/ender-3-v2-3d-printer",
			},
		},
		{
			Key: "machine_prusa_mk4",
			Data: presetsEntities.MachinePreset{
				Name:        "Prusa i3 MK4",
				Brand:       "Prusa Research",
				Model:       "i3 MK4",
				Watt:        120,
				IdleFactor:  0.05,
				Description: "Impressora 3D Prusa i3 MK4 - consumo 120W",
				URL:         "https://www.prusa3d.com/product/original-prusa-i3-mk4-3d-printer/",
			},
		},
	}

	for _, presetData := range machinePresets {
		var existingPreset presetsEntities.Preset
		result := Connector.Where("key = ?", presetData.Key).First(&existingPreset)

		if result.RecordNotFound() {
			preset := presetsEntities.Preset{
				Key: presetData.Key,
			}

			err := preset.MarshalDataFrom(presetData.Data)
			if err != nil {
				logger.Error(ctx, "Erro ao serializar preset de máquina", map[string]interface{}{
					"key":   presetData.Key,
					"error": err.Error(),
				})
				continue
			}

			if err := Connector.Create(&preset).Error; err != nil {
				logger.Error(ctx, "Erro ao criar preset de máquina", map[string]interface{}{
					"key":   presetData.Key,
					"error": err.Error(),
				})
			} else {
				logger.Info(ctx, "Preset de máquina criado", map[string]interface{}{
					"key":  presetData.Key,
					"name": presetData.Data.Name,
				})
			}
		}
	}
}
