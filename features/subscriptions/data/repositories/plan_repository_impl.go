package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/data/models"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlanRepositoryImpl implements the PlanRepository interface
type PlanRepositoryImpl struct {
	db *gorm.DB
}

// NewPlanRepository creates a new instance of PlanRepositoryImpl
func NewPlanRepository(db *gorm.DB) domainRepositories.PlanRepository {
	return &PlanRepositoryImpl{db: db}
}

// FindAll retrieves all plans
func (r *PlanRepositoryImpl) FindAll(ctx context.Context, activeOnly bool) ([]*entities.PlanEntity, error) {
	var planModels []models.PlanModel
	query := r.db.WithContext(ctx).Order("sort_order ASC")

	if activeOnly {
		query = query.Where("active = ?", true)
	}

	err := query.Find(&planModels).Error
	if err != nil {
		return nil, err
	}

	plans := make([]*entities.PlanEntity, len(planModels))
	for i, model := range planModels {
		// Load features for this plan
		var featureModels []models.PlanFeatureModel
		err := r.db.WithContext(ctx).
			Where("plan_id = ?", model.ID).
			Order("sort_order ASC").
			Find(&featureModels).Error
		if err != nil {
			return nil, err
		}

		// Convert features to entities
		features := make([]entities.PlanFeature, len(featureModels))
		for j, fm := range featureModels {
			features[j] = *fm.ToEntity()
		}

		plans[i] = model.ToEntity(features)
	}

	return plans, nil
}

// FindByID retrieves a plan by ID
func (r *PlanRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entities.PlanEntity, error) {
	var planModel models.PlanModel
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&planModel).Error

	if err != nil {
		return nil, err
	}

	// Load features for this plan
	var featureModels []models.PlanFeatureModel
	err = r.db.WithContext(ctx).
		Where("plan_id = ?", planModel.ID).
		Order("sort_order ASC").
		Find(&featureModels).Error
	if err != nil {
		return nil, err
	}

	// Convert features to entities
	features := make([]entities.PlanFeature, len(featureModels))
	for i, fm := range featureModels {
		features[i] = *fm.ToEntity()
	}

	return planModel.ToEntity(features), nil
}

// FindBySlug retrieves a plan by slug
func (r *PlanRepositoryImpl) FindBySlug(ctx context.Context, slug string) (*entities.PlanEntity, error) {
	var planModel models.PlanModel
	err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		First(&planModel).Error

	if err != nil {
		return nil, err
	}

	// Load features for this plan
	var featureModels []models.PlanFeatureModel
	err = r.db.WithContext(ctx).
		Where("plan_id = ?", planModel.ID).
		Order("sort_order ASC").
		Find(&featureModels).Error
	if err != nil {
		return nil, err
	}

	// Convert features to entities
	features := make([]entities.PlanFeature, len(featureModels))
	for i, fm := range featureModels {
		features[i] = *fm.ToEntity()
	}

	return planModel.ToEntity(features), nil
}

// Create creates a new plan
func (r *PlanRepositoryImpl) Create(ctx context.Context, plan *entities.PlanEntity) error {
	var planModel models.PlanModel
	planModel.FromEntity(plan)

	// Create plan with features in a transaction
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&planModel).Error; err != nil {
			return err
		}

		// Create features if provided
		if len(plan.Features) > 0 {
			features := make([]models.PlanFeatureModel, len(plan.Features))
			for i, feature := range plan.Features {
				features[i].FromEntity(&feature)
				features[i].PlanID = planModel.ID
			}
			if err := tx.Create(&features).Error; err != nil {
				return err
			}
		}

		// Update plan ID in entity
		plan.ID = planModel.ID
		return nil
	})
}

// Update updates an existing plan
func (r *PlanRepositoryImpl) Update(ctx context.Context, plan *entities.PlanEntity) error {
	var planModel models.PlanModel
	planModel.FromEntity(plan)

	return r.db.WithContext(ctx).
		Model(&models.PlanModel{}).
		Where("id = ?", plan.ID).
		Updates(map[string]interface{}{
			"name":        plan.Name,
			"slug":        plan.Slug,
			"description": plan.Description,
			"price":       plan.Price,
			"currency":    plan.Currency,
			"interval":    plan.Interval,
			"active":      plan.Active,
			"popular":     plan.Popular,
			"recommended": plan.Recommended,
			"sort_order":  plan.SortOrder,
		}).Error
}

// Delete soft deletes a plan
func (r *PlanRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.PlanModel{}).Error
}

// AddFeature adds a feature to a plan
func (r *PlanRepositoryImpl) AddFeature(ctx context.Context, feature *entities.PlanFeature) error {
	var featureModel models.PlanFeatureModel
	featureModel.FromEntity(feature)

	if err := r.db.WithContext(ctx).Create(&featureModel).Error; err != nil {
		return err
	}

	feature.ID = featureModel.ID
	return nil
}

// UpdateFeature updates a plan feature
func (r *PlanRepositoryImpl) UpdateFeature(ctx context.Context, feature *entities.PlanFeature) error {
	var featureModel models.PlanFeatureModel
	featureModel.FromEntity(feature)

	return r.db.WithContext(ctx).
		Model(&models.PlanFeatureModel{}).
		Where("id = ?", feature.ID).
		Updates(map[string]interface{}{
			"name":        feature.Name,
			"key":         feature.Key,
			"description": feature.Description,
			"value":       feature.Value,
			"value_type":  feature.ValueType,
			"available":   feature.Available,
			"sort_order":  feature.SortOrder,
		}).Error
}

// DeleteFeature deletes a plan feature
func (r *PlanRepositoryImpl) DeleteFeature(ctx context.Context, featureID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", featureID).
		Delete(&models.PlanFeatureModel{}).Error
}
