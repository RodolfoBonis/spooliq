package repositories

import (
	"errors"
	"time"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/material/data/models"
	"github.com/RodolfoBonis/spooliq/features/material/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/material/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type materialRepository struct {
	db *gorm.DB
}

// NewMaterialRepository creates a new instance of the material repository.
func NewMaterialRepository(db *gorm.DB) repositories.MaterialRepository {
	return &materialRepository{
		db: db,
	}
}

func (m *materialRepository) FindByID(id uuid.UUID, organizationID string) (*entities.MaterialEntity, error) {
	var material models.MaterialModel
	err := m.db.Model(models.MaterialModel{}).Where("id = ? AND organization_id = ?", id, organizationID).First(&material).Error
	if err != nil {
		return nil, err
	}

	entity := material.ToEntity()

	return &entity, nil
}

// FindAll returns a page of materials scoped to the organization, filtered by an
// optional case-insensitive name search and ordered by a pre-validated clause.
func (m *materialRepository) FindAll(organizationID, search, order string, limit, offset int) ([]entities.MaterialEntity, int64, error) {
	query := m.db.Model(&models.MaterialModel{}).
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

	var materialsData []models.MaterialModel
	if err := query.
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&materialsData).Error; err != nil {
		return nil, 0, err
	}

	materials := make([]entities.MaterialEntity, 0, len(materialsData))
	for _, material := range materialsData {
		materials = append(materials, material.ToEntity())
	}

	return materials, total, nil
}

func (m *materialRepository) Create(entity *entities.MaterialEntity) error {
	material := models.MaterialModel{}

	material.FromEntity(entity)

	if err := m.db.Create(&material).Error; err != nil {
		return err
	}

	*entity = material.ToEntity()

	return nil
}

func (m *materialRepository) Delete(id uuid.UUID) error {
	return m.db.Delete(&models.MaterialModel{}, "id = ?", id).Error
}

// materialUpdatableColumns are the columns a PUT may overwrite. They are passed
// to Select so GORM persists explicit zero values (e.g. temp_table: 0) instead
// of skipping them.
var materialUpdatableColumns = []string{"name", "description", "temp_table", "temp_extruder", "updated_at"}

func (m *materialRepository) Update(entity *entities.MaterialEntity) error {
	material := models.MaterialModel{}
	material.FromEntity(entity)
	material.UpdatedAt = time.Now()

	return m.db.Model(&models.MaterialModel{}).
		Where("id = ? AND organization_id = ?", material.ID, material.OrganizationID).
		Select(materialUpdatableColumns).
		Updates(material).Error
}

func (m *materialRepository) Exists(name string, organizationID string) (bool, error) {
	var count int64
	err := m.db.Model(models.MaterialModel{}).Where("name = ? AND organization_id = ?", name, organizationID).Count(&count).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return count > 0, err
}
