package repositories

import (
	"context"
	"errors"

	"github.com/RodolfoBonis/spooliq/features/filaments/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/filaments/domain/repositories"
	"github.com/jinzhu/gorm"
)

type filamentRepositoryImpl struct {
	db *gorm.DB
}

// NewFilamentRepository creates a new instance of FilamentRepository with the provided database connection.
func NewFilamentRepository(db *gorm.DB) repositories.FilamentRepository {
	return &filamentRepositoryImpl{db: db}
}

func (r *filamentRepositoryImpl) Create(ctx context.Context, filament *entities.Filament) error {
	return r.db.Create(filament).Error
}

func (r *filamentRepositoryImpl) GetByID(ctx context.Context, id uint, userID *string) (*entities.Filament, error) {
	var filament entities.Filament
	query := r.db.Preload("Brand").Preload("Material")

	if userID != nil {
		query = query.Where("(owner_user_id IS NULL OR owner_user_id = ?)", *userID)
	} else {
		query = query.Where("owner_user_id IS NULL")
	}

	err := query.Where("id = ?", id).First(&filament).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, errors.New("filament not found")
		}
		return nil, err
	}

	return &filament, nil
}

func (r *filamentRepositoryImpl) GetAll(ctx context.Context, userID *string) ([]*entities.Filament, error) {
	var filaments []*entities.Filament
	query := r.db.Preload("Brand").Preload("Material")

	if userID != nil {
		query = query.Where("(owner_user_id IS NULL OR owner_user_id = ?)", *userID)
	} else {
		query = query.Where("owner_user_id IS NULL")
	}

	err := query.Order("created_at DESC").Find(&filaments).Error
	return filaments, err
}

func (r *filamentRepositoryImpl) Update(ctx context.Context, filament *entities.Filament, userID *string) error {
	if userID == nil {
		return errors.New("cannot update filament: user authentication required")
	}

	result := r.db.Model(filament).Where("(owner_user_id IS NULL OR owner_user_id = ?) AND id = ?", *userID, filament.ID).Updates(filament)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("filament not found or access denied")
	}

	return r.db.Where("id = ?", filament.ID).First(filament).Error
}

func (r *filamentRepositoryImpl) Delete(ctx context.Context, id uint, userID *string) error {
	if userID == nil {
		return errors.New("cannot delete filament: user authentication required")
	}

	result := r.db.Where("owner_user_id = ?", *userID).Where("id = ?", id).Delete(&entities.Filament{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("filament not found or access denied")
	}

	return nil
}

func (r *filamentRepositoryImpl) GetByOwner(ctx context.Context, userID string) ([]*entities.Filament, error) {
	var filaments []*entities.Filament
	err := r.db.Preload("Brand").Preload("Material").Where("owner_user_id = ?", userID).Order("created_at DESC").Find(&filaments).Error
	return filaments, err
}

func (r *filamentRepositoryImpl) GetGlobal(ctx context.Context) ([]*entities.Filament, error) {
	var filaments []*entities.Filament
	err := r.db.Preload("Brand").Preload("Material").Where("owner_user_id IS NULL").Order("created_at DESC").Find(&filaments).Error
	return filaments, err
}
