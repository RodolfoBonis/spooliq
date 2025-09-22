package migrations

import (
	metadataEntities "github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/entities"
	filamentsEntities "github.com/RodolfoBonis/spooliq/features/filaments/domain/entities"
	presetsEntities "github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
	quotesEntities "github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
	"github.com/jinzhu/gorm"
)

// GetAllMigrations returns all available migrations in the correct order
func GetAllMigrations() []Migration {
	return []Migration{
		Migration001InitialSchema,
		Migration002AddFilamentDiameterWeight,
		Migration003AddColorHexField,
		Migration004CreateFilamentMetadataTables,
		// Add new migrations here in version order
	}
}

// Migration001InitialSchema creates the initial database schema
var Migration001InitialSchema = Migration{
	Version: "001",
	Name:    "Initial Schema",
	Up: func(db *gorm.DB) error {
		// Create Filaments table if it doesn't exist
		if !db.HasTable(&filamentsEntities.Filament{}) {
			if err := db.CreateTable(&filamentsEntities.Filament{}).Error; err != nil {
				return err
			}
		}

		// Create Presets table if it doesn't exist
		if !db.HasTable(&presetsEntities.Preset{}) {
			if err := db.CreateTable(&presetsEntities.Preset{}).Error; err != nil {
				return err
			}
		}

		// Create Machine Profiles table if it doesn't exist
		if !db.HasTable(&quotesEntities.MachineProfile{}) {
			if err := db.CreateTable(&quotesEntities.MachineProfile{}).Error; err != nil {
				return err
			}
		}

		// Create Energy Profiles table if it doesn't exist
		if !db.HasTable(&quotesEntities.EnergyProfile{}) {
			if err := db.CreateTable(&quotesEntities.EnergyProfile{}).Error; err != nil {
				return err
			}
		}

		// Create Cost Profiles table if it doesn't exist
		if !db.HasTable(&quotesEntities.CostProfile{}) {
			if err := db.CreateTable(&quotesEntities.CostProfile{}).Error; err != nil {
				return err
			}
		}

		// Create Margin Profiles table if it doesn't exist
		if !db.HasTable(&quotesEntities.MarginProfile{}) {
			if err := db.CreateTable(&quotesEntities.MarginProfile{}).Error; err != nil {
				return err
			}
		}

		// Create Quotes table if it doesn't exist, or migrate existing structure
		if !db.HasTable(&quotesEntities.Quote{}) {
			if err := db.CreateTable(&quotesEntities.Quote{}).Error; err != nil {
				return err
			}
		} else {
			// Auto-migrate to add any missing columns to existing table
			if err := db.AutoMigrate(&quotesEntities.Quote{}).Error; err != nil {
				return err
			}
		}

		// Create Quote Filament Lines table if it doesn't exist, or migrate existing structure
		if !db.HasTable(&quotesEntities.QuoteFilamentLine{}) {
			if err := db.CreateTable(&quotesEntities.QuoteFilamentLine{}).Error; err != nil {
				return err
			}
		} else {
			// Auto-migrate to add any missing columns to existing table
			if err := db.AutoMigrate(&quotesEntities.QuoteFilamentLine{}).Error; err != nil {
				return err
			}
		}

		return nil
	},
	Down: func(db *gorm.DB) error {
		// Drop tables in reverse order (to handle foreign key constraints)
		if err := db.DropTableIfExists(&quotesEntities.QuoteFilamentLine{}).Error; err != nil {
			return err
		}

		if err := db.DropTableIfExists(&quotesEntities.Quote{}).Error; err != nil {
			return err
		}

		if err := db.DropTableIfExists(&quotesEntities.MarginProfile{}).Error; err != nil {
			return err
		}

		if err := db.DropTableIfExists(&quotesEntities.CostProfile{}).Error; err != nil {
			return err
		}

		if err := db.DropTableIfExists(&quotesEntities.EnergyProfile{}).Error; err != nil {
			return err
		}

		if err := db.DropTableIfExists(&quotesEntities.MachineProfile{}).Error; err != nil {
			return err
		}

		if err := db.DropTableIfExists(&presetsEntities.Preset{}).Error; err != nil {
			return err
		}

		if err := db.DropTableIfExists(&filamentsEntities.Filament{}).Error; err != nil {
			return err
		}

		return nil
	},
}

// Migration002AddFilamentDiameterWeight adds diameter and weight columns to filaments table
var Migration002AddFilamentDiameterWeight = Migration{
	Version: "002",
	Name:    "Add Filament Diameter and Weight Fields",
	Up: func(db *gorm.DB) error {
		// Add diameter column as nullable first
		if err := db.Exec("ALTER TABLE filaments ADD COLUMN IF NOT EXISTS diameter DECIMAL(3,2)").Error; err != nil {
			return err
		}

		// Add weight column as nullable
		if err := db.Exec("ALTER TABLE filaments ADD COLUMN IF NOT EXISTS weight DECIMAL(8,2)").Error; err != nil {
			return err
		}

		// Set default diameter for existing records
		if err := db.Exec("UPDATE filaments SET diameter = 1.75 WHERE diameter IS NULL").Error; err != nil {
			return err
		}

		// Now make diameter NOT NULL
		if err := db.Exec("ALTER TABLE filaments ALTER COLUMN diameter SET NOT NULL").Error; err != nil {
			return err
		}

		return nil
	},
	Down: func(db *gorm.DB) error {
		// Remove weight column using raw SQL
		if err := db.Exec("ALTER TABLE filaments DROP COLUMN IF EXISTS weight").Error; err != nil {
			return err
		}

		// Remove diameter column using raw SQL
		if err := db.Exec("ALTER TABLE filaments DROP COLUMN IF EXISTS diameter").Error; err != nil {
			return err
		}

		return nil
	},
}

// Migration003AddColorHexField adds the missing color_hex column to filaments table
var Migration003AddColorHexField = Migration{
	Version: "003",
	Name:    "Add ColorHex Field to Filaments",
	Up: func(db *gorm.DB) error {
		// Add color_hex column to filaments table
		if err := db.Exec("ALTER TABLE filaments ADD COLUMN IF NOT EXISTS color_hex VARCHAR(7)").Error; err != nil {
			return err
		}

		// Also ensure price_per_meter column exists (it might be missing too)
		if err := db.Exec("ALTER TABLE filaments ADD COLUMN IF NOT EXISTS price_per_meter DECIMAL(10,4)").Error; err != nil {
			return err
		}

		// Ensure URL column exists
		if err := db.Exec("ALTER TABLE filaments ADD COLUMN IF NOT EXISTS url TEXT").Error; err != nil {
			return err
		}

		return nil
	},
	Down: func(db *gorm.DB) error {
		// Remove color_hex column from filaments table
		if err := db.Exec("ALTER TABLE filaments DROP COLUMN IF EXISTS color_hex").Error; err != nil {
			return err
		}

		// Note: Not removing price_per_meter and url as they might have been there before

		return nil
	},
}

// Migration004CreateFilamentMetadataTables creates filament brands and materials tables
var Migration004CreateFilamentMetadataTables = Migration{
	Version: "004",
	Name:    "Create Filament Metadata Tables",
	Up: func(db *gorm.DB) error {
		// Create FilamentBrand table if it doesn't exist
		if !db.HasTable(&metadataEntities.FilamentBrand{}) {
			if err := db.CreateTable(&metadataEntities.FilamentBrand{}).Error; err != nil {
				return err
			}
		}

		// Create FilamentMaterial table if it doesn't exist
		if !db.HasTable(&metadataEntities.FilamentMaterial{}) {
			if err := db.CreateTable(&metadataEntities.FilamentMaterial{}).Error; err != nil {
				return err
			}
		}

		// Insert default brands from existing data
		brands := []string{"SUNLU", "Creality", "Polymaker", "Prusament", "eSUN", "PETG", "Overture", "ANYCUBIC", "Hatchbox", "Amazon Basics"}
		for _, brandName := range brands {
			var count int
			db.Model(&metadataEntities.FilamentBrand{}).Where("name = ?", brandName).Count(&count)
			if count == 0 {
				brand := &metadataEntities.FilamentBrand{
					Name:   brandName,
					Active: true,
				}
				if err := db.Create(brand).Error; err != nil {
					return err
				}
			}
		}

		// Insert default materials from existing data
		materials := []string{"PLA", "ABS", "PETG", "TPU", "WOOD", "ASA", "PC", "NYLON", "PVA", "HIPS"}
		for _, materialName := range materials {
			var count int
			db.Model(&metadataEntities.FilamentMaterial{}).Where("name = ?", materialName).Count(&count)
			if count == 0 {
				material := &metadataEntities.FilamentMaterial{
					Name:   materialName,
					Active: true,
				}
				if err := db.Create(material).Error; err != nil {
					return err
				}
			}
		}

		return nil
	},
	Down: func(db *gorm.DB) error {
		// Drop tables
		if err := db.DropTableIfExists(&metadataEntities.FilamentMaterial{}).Error; err != nil {
			return err
		}

		if err := db.DropTableIfExists(&metadataEntities.FilamentBrand{}).Error; err != nil {
			return err
		}

		return nil
	},
}
