package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/quotes/data/mappers"
	"github.com/RodolfoBonis/spooliq/features/quotes/data/models"
	"github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/quotes/domain/repositories"
	"github.com/jinzhu/gorm"
)

type quoteRepositoryImpl struct {
	db *gorm.DB
}

func NewQuoteRepository(db *gorm.DB) repositories.QuoteRepository {
	return &quoteRepositoryImpl{
		db: db,
	}
}

func (r *quoteRepositoryImpl) Create(ctx context.Context, quote *entities.Quote) error {
	// Convert entity to model
	model := mappers.EntityToModel(quote)

	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create quote
	if err := tx.Create(model).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update entity with generated ID
	quote.ID = model.ID

	// Update filament lines with quote ID and their IDs
	for i := range quote.FilamentLines {
		quote.FilamentLines[i].QuoteID = model.ID
		quote.FilamentLines[i].ID = model.FilamentLines[i].ID
	}

	// Update profiles with quote ID and their IDs
	if quote.MachineProfile != nil && model.MachineProfile != nil {
		quote.MachineProfile.QuoteID = model.ID
		quote.MachineProfile.ID = model.MachineProfile.ID
	}
	if quote.EnergyProfile != nil && model.EnergyProfile != nil {
		quote.EnergyProfile.QuoteID = model.ID
		quote.EnergyProfile.ID = model.EnergyProfile.ID
	}
	if quote.CostProfile != nil && model.CostProfile != nil {
		quote.CostProfile.QuoteID = model.ID
		quote.CostProfile.ID = model.CostProfile.ID
	}
	if quote.MarginProfile != nil && model.MarginProfile != nil {
		quote.MarginProfile.QuoteID = model.ID
		quote.MarginProfile.ID = model.MarginProfile.ID
	}

	return tx.Commit().Error
}

func (r *quoteRepositoryImpl) GetByID(ctx context.Context, id uint, userID string) (*entities.Quote, error) {
	var model models.QuoteModel

	err := r.db.Preload("FilamentLines").
		Preload("MachineProfile").
		Preload("EnergyProfile").
		Preload("CostProfile").
		Preload("MarginProfile").
		Where("id = ? AND owner_user_id = ?", id, userID).
		First(&model).Error

	if err != nil {
		return nil, err
	}

	// Convert model to entity
	return mappers.ModelToEntity(&model), nil
}

func (r *quoteRepositoryImpl) GetByUser(ctx context.Context, userID string) ([]*entities.Quote, error) {
	var models []*models.QuoteModel

	err := r.db.Preload("FilamentLines").
		Preload("MachineProfile").
		Preload("EnergyProfile").
		Preload("CostProfile").
		Preload("MarginProfile").
		Where("owner_user_id = ?", userID).
		Find(&models).Error

	if err != nil {
		return nil, err
	}

	// Convert models to entities
	return mappers.ModelsToEntities(models), nil
}

func (r *quoteRepositoryImpl) Update(ctx context.Context, quote *entities.Quote, userID string) error {
	// Convert entity to model
	model := mappers.EntityToModel(quote)

	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if user owns the quote
	var existingModel models.QuoteModel
	if err := tx.Where("id = ? AND owner_user_id = ?", model.ID, userID).First(&existingModel).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update quote
	if err := tx.Model(&existingModel).Updates(model).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete existing filament lines and create new ones
	if err := tx.Where("quote_id = ?", model.ID).Delete(&models.QuoteFilamentLineModel{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, line := range model.FilamentLines {
		line.QuoteID = model.ID
		line.ID = 0 // Reset ID to create new
		if err := tx.Create(&line).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// Update or create profiles
	if model.MachineProfile != nil {
		model.MachineProfile.QuoteID = model.ID
		if err := tx.Where("quote_id = ?", model.ID).Assign(model.MachineProfile).FirstOrCreate(&models.MachineProfileModel{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if model.EnergyProfile != nil {
		model.EnergyProfile.QuoteID = model.ID
		if err := tx.Where("quote_id = ?", model.ID).Assign(model.EnergyProfile).FirstOrCreate(&models.EnergyProfileModel{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if model.CostProfile != nil {
		model.CostProfile.QuoteID = model.ID
		if err := tx.Where("quote_id = ?", model.ID).Assign(model.CostProfile).FirstOrCreate(&models.CostProfileModel{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if model.MarginProfile != nil {
		model.MarginProfile.QuoteID = model.ID
		if err := tx.Where("quote_id = ?", model.ID).Assign(model.MarginProfile).FirstOrCreate(&models.MarginProfileModel{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (r *quoteRepositoryImpl) Delete(ctx context.Context, id uint, userID string) error {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if user owns the quote
	var model models.QuoteModel
	if err := tx.Where("id = ? AND owner_user_id = ?", id, userID).First(&model).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Soft delete related records
	if err := tx.Where("quote_id = ?", id).Delete(&models.QuoteFilamentLineModel{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("quote_id = ?", id).Delete(&models.MachineProfileModel{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("quote_id = ?", id).Delete(&models.EnergyProfileModel{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("quote_id = ?", id).Delete(&models.CostProfileModel{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("quote_id = ?", id).Delete(&models.MarginProfileModel{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Soft delete quote
	if err := tx.Delete(&model).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *quoteRepositoryImpl) GetWithFilamentLines(ctx context.Context, id uint, userID string) (*entities.Quote, error) {
	// This is the same as GetByID since we always preload filament lines
	return r.GetByID(ctx, id, userID)
}