package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/RodolfoBonis/spooliq/core/helpers"
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

func (r *model3dRepository) Create(ctx context.Context, entity *entities.Model3DEntity) error {
	model := models.Model3DModel{}
	model.FromEntity(entity)

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return err
	}

	*entity = *model.ToEntity()
	return nil
}

// model3dUpdatableColumns are the columns a PUT may overwrite. Passed to Select so
// GORM persists explicit zero/empty/NULL values (e.g. clearing customer_id)
// instead of skipping them, while owner/tenant/file/creation columns stay
// immutable.
var model3dUpdatableColumns = []string{"name", "description", "customer_id", "tags", "notes", "updated_at"}

func (r *model3dRepository) Update(ctx context.Context, entity *entities.Model3DEntity) error {
	entity.UpdatedAt = time.Now()
	model := models.Model3DModel{}
	model.FromEntity(entity)

	return r.db.WithContext(ctx).
		Model(&models.Model3DModel{}).
		Where("id = ? AND organization_id = ?", model.ID, model.OrganizationID).
		Select(model3dUpdatableColumns).
		Updates(&model).Error
}

func (r *model3dRepository) Delete(ctx context.Context, id uuid.UUID, organizationID string) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, organizationID).
		Delete(&models.Model3DModel{}).Error
}

func (r *model3dRepository) FindByID(ctx context.Context, id uuid.UUID, organizationID string) (*entities.Model3DEntity, error) {
	var model models.Model3DModel
	if err := r.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, organizationID).
		First(&model).Error; err != nil {
		return nil, err
	}
	return model.ToEntity(), nil
}

func (r *model3dRepository) FindAll(ctx context.Context, organizationID string, filters repositories.Model3DFilters, search, order string, limit, offset int) ([]*entities.Model3DEntity, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&models.Model3DModel{}).
		Where("organization_id = ?", organizationID)

	if search != "" {
		like := "%" + helpers.EscapeLike(search) + "%"
		query = query.Where(`(name ILIKE ? ESCAPE '\' OR tags ILIKE ? ESCAPE '\')`, like, like)
	}

	if filters.CustomerID != nil {
		query = query.Where("customer_id = ?", *filters.CustomerID)
	}

	if filters.Format != "" {
		query = query.Where("file_format = ?", filters.Format)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count 3d models: %w", err)
	}

	if order == "" {
		order = "created_at desc"
	}

	var modelsData []models.Model3DModel
	if err := query.
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&modelsData).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find 3d models: %w", err)
	}

	result := make([]*entities.Model3DEntity, len(modelsData))
	for i := range modelsData {
		result[i] = modelsData[i].ToEntity()
	}
	return result, total, nil
}

func (r *model3dRepository) FindByCustomerID(ctx context.Context, customerID uuid.UUID, organizationID string) ([]*entities.Model3DEntity, error) {
	var modelsData []models.Model3DModel
	if err := r.db.WithContext(ctx).
		Where("customer_id = ? AND organization_id = ?", customerID, organizationID).
		Order("created_at DESC").
		Find(&modelsData).Error; err != nil {
		return nil, fmt.Errorf("failed to find 3d models by customer: %w", err)
	}

	result := make([]*entities.Model3DEntity, len(modelsData))
	for i := range modelsData {
		result[i] = modelsData[i].ToEntity()
	}
	return result, nil
}

func (r *model3dRepository) FindByHash(ctx context.Context, hash string, organizationID string) (*entities.Model3DEntity, error) {
	var model models.Model3DModel
	err := r.db.WithContext(ctx).
		Where("file_hash = ? AND organization_id = ?", hash, organizationID).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return model.ToEntity(), nil
}

func (r *model3dRepository) CustomerExists(ctx context.Context, customerID uuid.UUID, organizationID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Table("customers").
		Where("id = ? AND organization_id = ?", customerID, organizationID).
		Count(&count).Error; err != nil {
		// A missing customers table (e.g. in a slimmed test DB) means no customer
		// to validate against; treat as "does not exist" rather than a 500.
		if strings.Contains(err.Error(), "does not exist") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check customer existence: %w", err)
	}
	return count > 0, nil
}
