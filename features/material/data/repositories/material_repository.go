package repositories

import (
	"errors"

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

func (m *materialRepository) FindAll(organizationID string) ([]entities.MaterialEntity, error) {
	var materialsData []models.MaterialModel
	err := m.db.Where("organization_id = ?", organizationID).Order("name ASC").Find(&materialsData).Error
	if err != nil {
		return nil, err
	}

	materials := make([]entities.MaterialEntity, 0, len(materialsData))

	for _, material := range materialsData {
		materials = append(materials, material.ToEntity())
	}

	return materials, nil
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
	return m.db.Model(models.MaterialModel{}).Delete("id = ?", id).Error
}

func (m *materialRepository) Update(entity *entities.MaterialEntity) error {
	material := models.MaterialModel{}

	material.FromEntity(entity)

	return m.db.Model(material).Where("id = ?", material.ID).Updates(material).Error
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
