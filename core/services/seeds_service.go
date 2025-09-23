package services

import (
	"context"
	"fmt"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
	presetsEntities "github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
	"github.com/schollz/progressbar/v3"
)

// RunSeeds executa todos os seeds iniciais do sistema
func RunSeeds(logger logger.Logger) {
	ctx := context.Background()

	// Temporarily disable SQL logging
	Connector.LogMode(false)
	defer func() {
		Connector.LogMode(true)
	}()

	// Count total seeds
	totalSeeds := countTotalSeeds()

	if totalSeeds == 0 {
		logger.Info(ctx, "✅ All seeds already exist")
		return
	}

	// Create progress bar
	bar := progressbar.NewOptions(totalSeeds,
		progressbar.OptionSetDescription("🌱 Running seeds..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerHead:    "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(100*time.Millisecond),
	)

	// Seeds de presets
	bar.Describe("🌱 Seeding energy presets...")
	seedEnergyPresetsWithProgress(logger, bar)

	bar.Describe("🌱 Seeding machine presets...")
	seedMachinePresetsWithProgress(logger, bar)

	bar.Describe("✅ Seeds completed successfully!")
	bar.Finish()
	fmt.Println()

	logger.Info(ctx, "Seeds completed successfully!")
}

// countTotalSeeds counts how many seeds need to be inserted
func countTotalSeeds() int {
	count := 0

	// Count energy presets
	energyPresets := getEnergyPresets()
	for _, presetData := range energyPresets {
		var existingPreset presetsEntities.Preset
		result := Connector.Where("key = ?", presetData.Key).First(&existingPreset)
		if result.RecordNotFound() {
			count++
		}
	}

	// Count machine presets
	machinePresets := getMachinePresets()
	for _, presetData := range machinePresets {
		var existingPreset presetsEntities.Preset
		result := Connector.Where("key = ?", presetData.Key).First(&existingPreset)
		if result.RecordNotFound() {
			count++
		}
	}

	return count
}

// getEnergyPresets returns the energy presets data
func getEnergyPresets() []struct {
	Key  string
	Data presetsEntities.EnergyPreset
} {
	return []struct {
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
}

// seedEnergyPresets cria presets de energia/tarifa
func seedEnergyPresets(logger logger.Logger) {
	energyPresets := getEnergyPresets()

	for _, presetData := range energyPresets {
		var existingPreset presetsEntities.Preset
		result := Connector.Where("key = ?", presetData.Key).First(&existingPreset)

		if result.RecordNotFound() {
			preset := presetsEntities.Preset{
				Key: presetData.Key,
			}

			err := preset.MarshalDataFrom(presetData.Data)
			if err != nil {
				continue
			}

			Connector.Create(&preset)
		}
	}
}

// seedEnergyPresetsWithProgress cria presets de energia/tarifa com barra de progresso
func seedEnergyPresetsWithProgress(logger logger.Logger, bar *progressbar.ProgressBar) {
	energyPresets := getEnergyPresets()

	for _, presetData := range energyPresets {
		var existingPreset presetsEntities.Preset
		result := Connector.Where("key = ?", presetData.Key).First(&existingPreset)

		if result.RecordNotFound() {
			preset := presetsEntities.Preset{
				Key: presetData.Key,
			}

			err := preset.MarshalDataFrom(presetData.Data)
			if err != nil {
				continue
			}

			Connector.Create(&preset)
			bar.Add(1)
		}
	}
}

// getMachinePresets returns the machine presets data
func getMachinePresets() []struct {
	Key  string
	Data presetsEntities.MachinePreset
} {
	return []struct {
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
}

// seedMachinePresets cria presets de máquinas/impressoras
func seedMachinePresets(logger logger.Logger) {
	machinePresets := getMachinePresets()

	for _, presetData := range machinePresets {
		var existingPreset presetsEntities.Preset
		result := Connector.Where("key = ?", presetData.Key).First(&existingPreset)

		if result.RecordNotFound() {
			preset := presetsEntities.Preset{
				Key: presetData.Key,
			}

			err := preset.MarshalDataFrom(presetData.Data)
			if err != nil {
				continue
			}

			Connector.Create(&preset)
		}
	}
}

// seedMachinePresetsWithProgress cria presets de máquinas/impressoras com barra de progresso
func seedMachinePresetsWithProgress(logger logger.Logger, bar *progressbar.ProgressBar) {
	machinePresets := getMachinePresets()

	for _, presetData := range machinePresets {
		var existingPreset presetsEntities.Preset
		result := Connector.Where("key = ?", presetData.Key).First(&existingPreset)

		if result.RecordNotFound() {
			preset := presetsEntities.Preset{
				Key: presetData.Key,
			}

			err := preset.MarshalDataFrom(presetData.Data)
			if err != nil {
				continue
			}

			Connector.Create(&preset)
			bar.Add(1)
		}
	}
}
