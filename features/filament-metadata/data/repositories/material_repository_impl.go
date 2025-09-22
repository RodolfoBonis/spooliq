package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/repositories"
	"github.com/jinzhu/gorm"
)

type materialRepositoryImpl struct {
	db *gorm.DB
}

// NewMaterialRepository creates a new instance of material repository
func NewMaterialRepository(db *gorm.DB) repositories.MaterialRepository {
	return &materialRepositoryImpl{
		db: db,
	}
}

func (r *materialRepositoryImpl) Create(ctx context.Context, material *entities.FilamentMaterial) error {
	return r.db.Create(material).Error
}

func (r *materialRepositoryImpl) GetByID(ctx context.Context, id uint) (*entities.FilamentMaterial, error) {
	var material entities.FilamentMaterial
	err := r.db.Where("id = ?", id).First(&material).Error
	if err != nil {
		return nil, err
	}
	return &material, nil
}

func (r *materialRepositoryImpl) GetByName(ctx context.Context, name string) (*entities.FilamentMaterial, error) {
	var material entities.FilamentMaterial
	err := r.db.Where("name = ?", name).First(&material).Error
	if err != nil {
		return nil, err
	}
	return &material, nil
}

func (r *materialRepositoryImpl) GetAll(ctx context.Context, activeOnly bool) ([]*entities.FilamentMaterial, error) {
	var materials []*entities.FilamentMaterial
	query := r.db.Model(&entities.FilamentMaterial{})

	if activeOnly {
		query = query.Where("active = ?", true)
	}

	err := query.Order("name ASC").Find(&materials).Error
	return materials, err
}

func (r *materialRepositoryImpl) Update(ctx context.Context, material *entities.FilamentMaterial) error {
	return r.db.Model(material).Where("id = ?", material.ID).Updates(material).Error
}

func (r *materialRepositoryImpl) Delete(ctx context.Context, id uint) error {
	// Soft delete by setting active to false
	return r.db.Model(&entities.FilamentMaterial{}).Where("id = ?", id).Update("active", false).Error
}

func (r *materialRepositoryImpl) Exists(ctx context.Context, name string) (bool, error) {
	var count int
	err := r.db.Model(&entities.FilamentMaterial{}).Where("name = ?", name).Count(&count).Error
	return count > 0, err
}