package repositories

import (
	"math"

	"github.com/RodolfoBonis/spooliq/features/model3d/data/models"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type model3dRepository struct {
	db *gorm.DB
}

// NewModel3DRepository creates a new instance of the model3d repository.
func NewModel3DRepository(db *gorm.DB) repositories.Model3DRepository {
	return &model3dRepository{db: db}
}

func (r *model3dRepository) Create(entity *entities.Model3DEntity) error {
	model := models.Model3DModel{}
	model.FromEntity(entity)

	if err := r.db.Create(&model).Error; err != nil {
		return err
	}

	*entity = *model.ToEntity()
	return nil
}

func (r *model3dRepository) Update(entity *entities.Model3DEntity) error {
	model := models.Model3DModel{}
	model.FromEntity(entity)

	return r.db.Model(&model).Where("id = ? AND organization_id = ?", model.ID, model.OrganizationID).
		Updates(map[string]interface{}{
			"name":        model.Name,
			"description": model.Description,
			"customer_id": model.CustomerID,
			"tags":        model.Tags,
			"notes":       model.Notes,
		}).Error
}

func (r *model3dRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.Model3DModel{}).Error
}

func (r *model3dRepository) FindByID(id uuid.UUID, organizationID string) (*entities.Model3DEntity, error) {
	var model models.Model3DModel
	err := r.db.Where("id = ? AND organization_id = ?", id, organizationID).
		First(&model).Error
	if err != nil {
		return nil, err
	}
	return model.ToEntity(), nil
}

func (r *model3dRepository) FindAll(organizationID string, filters repositories.Model3DFilters) (*entities.FindAllModel3DResponse, error) {
	query := r.db.Model(&models.Model3DModel{}).Where("organization_id = ?", organizationID)

	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		query = query.Where("(name ILIKE ? OR tags ILIKE ?)", searchPattern, searchPattern)
	}

	if filters.CustomerID != nil {
		query = query.Where("customer_id = ?", *filters.CustomerID)
	}

	if filters.Format != "" {
		query = query.Where("file_format = ?", filters.Format)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	page := filters.Page
	if page < 1 {
		page = 1
	}
	pageSize := filters.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	var modelsData []models.Model3DModel
	err := query.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&modelsData).Error
	if err != nil {
		return nil, err
	}

	result := make([]*entities.Model3DEntity, 0, len(modelsData))
	for _, m := range modelsData {
		result = append(result, m.ToEntity())
	}

	return &entities.FindAllModel3DResponse{
		Data:       result,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *model3dRepository) FindByCustomerID(customerID uuid.UUID, organizationID string) ([]*entities.Model3DEntity, error) {
	var modelsData []models.Model3DModel
	err := r.db.Where("customer_id = ? AND organization_id = ?", customerID, organizationID).
		Order("created_at DESC").
		Find(&modelsData).Error
	if err != nil {
		return nil, err
	}

	result := make([]*entities.Model3DEntity, 0, len(modelsData))
	for _, m := range modelsData {
		result = append(result, m.ToEntity())
	}
	return result, nil
}

func (r *model3dRepository) FindByHash(hash string, organizationID string) (*entities.Model3DEntity, error) {
	var model models.Model3DModel
	err := r.db.Where("file_hash = ? AND organization_id = ?", hash, organizationID).
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToEntity(), nil
}
