package repositories

import (
	"context"
	"errors"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/data/models"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// subscriptionPlanRepositoryImpl implements SubscriptionPlanRepository
type subscriptionPlanRepositoryImpl struct {
	db *gorm.DB
}

// NewSubscriptionPlanRepository creates a new instance of SubscriptionPlanRepository
func NewSubscriptionPlanRepository(db *gorm.DB) repositories.SubscriptionPlanRepository {
	return &subscriptionPlanRepositoryImpl{db: db}
}

// Create creates a new subscription plan
func (r *subscriptionPlanRepositoryImpl) Create(ctx context.Context, plan *entities.SubscriptionPlanEntity) error {
	model := &models.SubscriptionPlanModel{}
	model.FromEntity(plan)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}

	*plan = *model.ToEntity()
	return nil
}

// FindByID finds a subscription plan by ID
func (r *subscriptionPlanRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entities.SubscriptionPlanEntity, error) {
	var model models.SubscriptionPlanModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return model.ToEntity(), nil
}

// FindByName finds a subscription plan by name
func (r *subscriptionPlanRepositoryImpl) FindByName(ctx context.Context, name string) (*entities.SubscriptionPlanEntity, error) {
	var model models.SubscriptionPlanModel
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return model.ToEntity(), nil
}

// FindAll finds all subscription plans
func (r *subscriptionPlanRepositoryImpl) FindAll(ctx context.Context) ([]*entities.SubscriptionPlanEntity, error) {
	var models []models.SubscriptionPlanModel
	if err := r.db.WithContext(ctx).Order("price ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	entities := make([]*entities.SubscriptionPlanEntity, len(models))
	for i, model := range models {
		entities[i] = model.ToEntity()
	}

	return entities, nil
}

// FindAllActive finds all active subscription plans
func (r *subscriptionPlanRepositoryImpl) FindAllActive(ctx context.Context) ([]*entities.SubscriptionPlanEntity, error) {
	var models []models.SubscriptionPlanModel
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("price ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}

	entities := make([]*entities.SubscriptionPlanEntity, len(models))
	for i, model := range models {
		entities[i] = model.ToEntity()
	}

	return entities, nil
}

// Update updates a subscription plan
func (r *subscriptionPlanRepositoryImpl) Update(ctx context.Context, plan *entities.SubscriptionPlanEntity) error {
	model := &models.SubscriptionPlanModel{}
	model.FromEntity(plan)

	return r.db.WithContext(ctx).Save(model).Error
}

// Delete soft deletes a subscription plan
func (r *subscriptionPlanRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.SubscriptionPlanModel{}, "id = ?", id).Error
}
