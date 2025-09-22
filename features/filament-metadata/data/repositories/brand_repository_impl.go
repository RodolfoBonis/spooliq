package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/repositories"
	"github.com/jinzhu/gorm"
)

type brandRepositoryImpl struct {
	db *gorm.DB
}

// NewBrandRepository creates a new instance of brand repository
func NewBrandRepository(db *gorm.DB) repositories.BrandRepository {
	return &brandRepositoryImpl{
		db: db,
	}
}

func (r *brandRepositoryImpl) Create(ctx context.Context, brand *entities.FilamentBrand) error {
	return r.db.Create(brand).Error
}

func (r *brandRepositoryImpl) GetByID(ctx context.Context, id uint) (*entities.FilamentBrand, error) {
	var brand entities.FilamentBrand
	err := r.db.Where("id = ?", id).First(&brand).Error
	if err != nil {
		return nil, err
	}
	return &brand, nil
}

func (r *brandRepositoryImpl) GetByName(ctx context.Context, name string) (*entities.FilamentBrand, error) {
	var brand entities.FilamentBrand
	err := r.db.Where("name = ?", name).First(&brand).Error
	if err != nil {
		return nil, err
	}
	return &brand, nil
}

func (r *brandRepositoryImpl) GetAll(ctx context.Context, activeOnly bool) ([]*entities.FilamentBrand, error) {
	var brands []*entities.FilamentBrand
	query := r.db.Model(&entities.FilamentBrand{})

	if activeOnly {
		query = query.Where("active = ?", true)
	}

	err := query.Order("name ASC").Find(&brands).Error
	return brands, err
}

func (r *brandRepositoryImpl) Update(ctx context.Context, brand *entities.FilamentBrand) error {
	return r.db.Model(brand).Where("id = ?", brand.ID).Updates(brand).Error
}

func (r *brandRepositoryImpl) Delete(ctx context.Context, id uint) error {
	// Soft delete by setting active to false
	return r.db.Model(&entities.FilamentBrand{}).Where("id = ?", id).Update("active", false).Error
}

func (r *brandRepositoryImpl) Exists(ctx context.Context, name string) (bool, error) {
	var count int
	err := r.db.Model(&entities.FilamentBrand{}).Where("name = ?", name).Count(&count).Error
	return count > 0, err
}