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

// paymentMethodRepositoryImpl implements PaymentMethodRepository
type paymentMethodRepositoryImpl struct {
	db *gorm.DB
}

// NewPaymentMethodRepository creates a new instance of PaymentMethodRepository
func NewPaymentMethodRepository(db *gorm.DB) repositories.PaymentMethodRepository {
	return &paymentMethodRepositoryImpl{db: db}
}

// Create creates a new payment method
func (r *paymentMethodRepositoryImpl) Create(ctx context.Context, paymentMethod *entities.PaymentMethodEntity) error {
	model := &models.PaymentMethodModel{}
	model.FromEntity(paymentMethod)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}

	*paymentMethod = *model.ToEntity()
	return nil
}

// FindByID finds a payment method by ID
func (r *paymentMethodRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entities.PaymentMethodEntity, error) {
	var model models.PaymentMethodModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return model.ToEntity(), nil
}

// FindByOrganizationID finds all payment methods for an organization
func (r *paymentMethodRepositoryImpl) FindByOrganizationID(ctx context.Context, organizationID string) ([]*entities.PaymentMethodEntity, error) {
	var models []models.PaymentMethodModel
	if err := r.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Order("is_primary DESC, created_at DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}

	entities := make([]*entities.PaymentMethodEntity, len(models))
	for i, model := range models {
		entities[i] = model.ToEntity()
	}

	return entities, nil
}

// FindPrimaryByOrganizationID finds the primary payment method for an organization
func (r *paymentMethodRepositoryImpl) FindPrimaryByOrganizationID(ctx context.Context, organizationID string) (*entities.PaymentMethodEntity, error) {
	var model models.PaymentMethodModel
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND is_primary = ?", organizationID, true).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return model.ToEntity(), nil
}

// Update updates a payment method
func (r *paymentMethodRepositoryImpl) Update(ctx context.Context, paymentMethod *entities.PaymentMethodEntity) error {
	model := &models.PaymentMethodModel{}
	model.FromEntity(paymentMethod)

	return r.db.WithContext(ctx).Save(model).Error
}

// Delete soft deletes a payment method
func (r *paymentMethodRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.PaymentMethodModel{}, "id = ?", id).Error
}

// SetAsPrimary sets a payment method as primary (and unsets others)
func (r *paymentMethodRepositoryImpl) SetAsPrimary(ctx context.Context, organizationID string, paymentMethodID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Unset all primary flags for this organization
		if err := tx.Model(&models.PaymentMethodModel{}).
			Where("organization_id = ?", organizationID).
			Update("is_primary", false).Error; err != nil {
			return err
		}

		// Set the specified payment method as primary
		if err := tx.Model(&models.PaymentMethodModel{}).
			Where("id = ?", paymentMethodID).
			Update("is_primary", true).Error; err != nil {
			return err
		}

		return nil
	})
}
