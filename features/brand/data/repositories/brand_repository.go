package repositories

import (
	"errors"

	"github.com/RodolfoBonis/spooliq/features/brand/data/models"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/repositories"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
)

type brandRepository struct {
	db *gorm.DB
}

func NewBrandRepository(db *gorm.DB) repositories.BrandRepository {
	return &brandRepository{
		db: db,
	}
}

func (b *brandRepository) FindByID(id uuid.UUID) (*entities.BrandEntity, error) {
	var brand entities.BrandEntity
	err := b.db.Where("id = ?", id).First(&brand).Error
	if err != nil {
		return nil, err
	}
	return &brand, nil
}

func (b *brandRepository) FindAll() ([]entities.BrandEntity, error) {
	var brandsData []models.BrandModel
	err := b.db.Order("name ASC").Find(&brandsData).Error
	if err != nil {
		return nil, err
	}

	brands := make([]entities.BrandEntity, 0, len(brandsData))

	for _, brand := range brandsData {
		brands = append(brands, brand.ToEntity())
	}

	return brands, nil
}

func (b *brandRepository) Create(entity *entities.BrandEntity) error {
	brand := models.BrandModel{}

	brand.FromEntity(entity)

	if err := b.db.Create(&brand).Error; err != nil {
		return err
	}

	*entity = brand.ToEntity()

	return nil
}

func (b *brandRepository) Delete(id uuid.UUID) error {
	return b.db.Model(models.BrandModel{}).Delete("id = ?", id).Error
}

func (b *brandRepository) Update(entity *entities.BrandEntity) error {
	brand := models.BrandModel{}

	brand.FromEntity(entity)

	return b.db.Model(brand).Where("id = ?", brand.ID).Updates(brand).Error
}

func (b *brandRepository) Exists(name string) (bool, error) {
	var count int
	err := b.db.Model(models.BrandModel{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return false, nil
		}
		return false, err
	}
	return count > 0, err
}
