package repositories

import (
	"math"

	"github.com/RodolfoBonis/spooliq/features/activity/data/models"
	"github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/activity/domain/repositories"
	"gorm.io/gorm"
)

type activityRepository struct {
	db *gorm.DB
}

// NewActivityRepository creates a new instance of the activity repository.
func NewActivityRepository(db *gorm.DB) repositories.ActivityRepository {
	return &activityRepository{
		db: db,
	}
}

func (r *activityRepository) Create(activity *entities.ActivityEntity) error {
	model := models.ActivityModel{}
	model.FromEntity(activity)

	if err := r.db.Create(&model).Error; err != nil {
		return err
	}

	*activity = model.ToEntity()
	return nil
}

func (r *activityRepository) FindByOrganization(filter *entities.ActivityFilter) (*entities.PaginatedActivities, error) {
	var total int64

	query := r.db.Model(&models.ActivityModel{}).Where("organization_id = ?", filter.OrganizationID)

	if filter.EntityType != nil {
		query = query.Where("entity_type = ?", string(*filter.EntityType))
	}
	if filter.Action != nil {
		query = query.Where("action = ?", string(*filter.Action))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	var activityModels []models.ActivityModel

	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(filter.PageSize).
		Find(&activityModels).Error; err != nil {
		return nil, err
	}

	activities := make([]entities.ActivityEntity, 0, len(activityModels))
	for _, m := range activityModels {
		activities = append(activities, m.ToEntity())
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PageSize)))

	return &entities.PaginatedActivities{
		Activities: activities,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *activityRepository) FindRecentByOrganization(organizationID string, limit int) ([]entities.ActivityEntity, error) {
	var activityModels []models.ActivityModel

	if err := r.db.Where("organization_id = ?", organizationID).
		Order("created_at DESC").
		Limit(limit).
		Find(&activityModels).Error; err != nil {
		return nil, err
	}

	activities := make([]entities.ActivityEntity, 0, len(activityModels))
	for _, m := range activityModels {
		activities = append(activities, m.ToEntity())
	}

	return activities, nil
}
