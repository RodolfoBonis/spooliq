package repositories

import (
	"errors"
	"time"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/supply/data/models"
	"github.com/RodolfoBonis/spooliq/features/supply/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/supply/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type supplyRepository struct {
	db *gorm.DB
}

// NewSupplyRepository creates a new instance of the supply repository.
func NewSupplyRepository(db *gorm.DB) repositories.SupplyRepository {
	return &supplyRepository{
		db: db,
	}
}

func (s *supplyRepository) FindByID(id uuid.UUID, organizationID string) (*entities.SupplyEntity, error) {
	var supply models.SupplyModel
	err := s.db.Model(models.SupplyModel{}).
		Where("id = ? AND organization_id = ?", id, organizationID).
		First(&supply).Error
	if err != nil {
		return nil, err
	}

	entity := supply.ToEntity()

	return &entity, nil
}

// FindAll returns a page of supplies scoped to the organization, filtered by an
// optional case-insensitive name search and an optional is_active flag, ordered by
// a pre-validated clause.
func (s *supplyRepository) FindAll(organizationID, search string, isActive *bool, order string, limit, offset int) ([]entities.SupplyEntity, int64, error) {
	query := s.db.Model(&models.SupplyModel{}).
		Where("organization_id = ?", organizationID)

	if search != "" {
		query = query.Where(`name ILIKE ? ESCAPE '\'`, "%"+helpers.EscapeLike(search)+"%")
	}

	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if order == "" {
		order = "name asc"
	}

	var suppliesData []models.SupplyModel
	if err := query.
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&suppliesData).Error; err != nil {
		return nil, 0, err
	}

	supplies := make([]entities.SupplyEntity, 0, len(suppliesData))
	for _, supply := range suppliesData {
		supplies = append(supplies, supply.ToEntity())
	}

	return supplies, total, nil
}

func (s *supplyRepository) Create(entity *entities.SupplyEntity) error {
	supply := models.SupplyModel{}

	supply.FromEntity(entity)

	if err := s.db.Create(&supply).Error; err != nil {
		return err
	}

	*entity = supply.ToEntity()

	return nil
}

func (s *supplyRepository) Delete(id uuid.UUID) error {
	return s.db.Delete(&models.SupplyModel{}, "id = ?", id).Error
}

// supplyUpdatableColumns are the columns a PUT may overwrite. They are passed to
// Select so GORM persists explicit zero values (e.g. unit_cost: 0, is_active:
// false) instead of skipping them.
var supplyUpdatableColumns = []string{"name", "unit", "unit_cost", "description", "is_active", "updated_at"}

func (s *supplyRepository) Update(entity *entities.SupplyEntity) error {
	supply := models.SupplyModel{}
	supply.FromEntity(entity)
	supply.UpdatedAt = time.Now()

	return s.db.Model(&models.SupplyModel{}).
		Where("id = ? AND organization_id = ?", supply.ID, supply.OrganizationID).
		Select(supplyUpdatableColumns).
		Updates(supply).Error
}

func (s *supplyRepository) Exists(name string, organizationID string) (bool, error) {
	var count int64
	err := s.db.Model(models.SupplyModel{}).
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
