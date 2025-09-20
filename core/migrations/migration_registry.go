package migrations

import (
	filamentsEntities "github.com/RodolfoBonis/spooliq/features/filaments/domain/entities"
	presetsEntities "github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
	quotesEntities "github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
	"github.com/jinzhu/gorm"
)

// GetAllMigrations returns all available migrations in the correct order
func GetAllMigrations() []Migration {
	return []Migration{
		Migration001InitialSchema,
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
