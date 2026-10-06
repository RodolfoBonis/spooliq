package repositories

import (
	"errors"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/brand/data/models"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type brandRepository struct {
	db *gorm.DB
}

// NewBrandRepository creates a new instance of the brand repository.
func NewBrandRepository(db *gorm.DB) repositories.BrandRepository {
	return &brandRepository{
		db: db,
	}
}

func (b *brandRepository) FindByID(id uuid.UUID, organizationID string) (*entities.BrandEntity, error) {
	var brand models.BrandModel
	err := b.db.Model(models.BrandModel{}).
		Where("id = ? AND organization_id = ?", id, organizationID).
		First(&brand).Error
	if err != nil {
		return nil, err
	}

	entity := brand.ToEntity()

	return &entity, nil
}

// FindAll returns a page of brands scoped to the organization, filtered by an
// optional case-insensitive name search and ordered by a pre-validated clause.
func (b *brandRepository) FindAll(organizationID, search, order string, limit, offset int) ([]entities.BrandEntity, int64, error) {
	query := b.db.Model(&models.BrandModel{}).
		Where("organization_id = ?", organizationID)

	if search != "" {
		query = query.Where(`name ILIKE ? ESCAPE '\'`, "%"+helpers.EscapeLike(search)+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if order == "" {
		order = "name asc"
	}

	var brandsData []models.BrandModel
	if err := query.
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&brandsData).Error; err != nil {
		return nil, 0, err
	}

	brands := make([]entities.BrandEntity, 0, len(brandsData))
	for _, brand := range brandsData {
		brands = append(brands, brand.ToEntity())
	}

	return brands, total, nil
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
	return b.db.Delete(&models.BrandModel{}, "id = ?", id).Error
}

func (b *brandRepository) Update(entity *entities.BrandEntity) error {
	brand := models.BrandModel{}

	brand.FromEntity(entity)

	return b.db.Model(brand).Where("id = ?", brand.ID).Updates(brand).Error
}

func (b *brandRepository) Exists(name string, organizationID string) (bool, error) {
	var count int64
	err := b.db.Model(models.BrandModel{}).
		Where("name = ? AND organization_id = ?", name, organizationID).
		Count(&count).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return count > 0, err
}
