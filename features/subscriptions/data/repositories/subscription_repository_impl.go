package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/data/models"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubscriptionRepositoryImpl implements the SubscriptionRepository interface
type SubscriptionRepositoryImpl struct {
	db *gorm.DB
}

// NewSubscriptionRepository creates a new instance of SubscriptionRepositoryImpl
func NewSubscriptionRepository(db *gorm.DB) domainRepositories.SubscriptionRepository {
	return &SubscriptionRepositoryImpl{db: db}
}

// FindAll retrieves all subscription payment records for a given organization
func (r *SubscriptionRepositoryImpl) FindAll(ctx context.Context, organizationID uuid.UUID, limit, offset int) ([]*entities.SubscriptionEntity, error) {
	var subscriptionModels []models.SubscriptionModel
	err := r.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&subscriptionModels).Error

	if err != nil {
		return nil, err
	}

	subscriptions := make([]*entities.SubscriptionEntity, len(subscriptionModels))
	for i, model := range subscriptionModels {
		subscriptions[i] = model.ToEntity()
	}

	return subscriptions, nil
}

// FindByID retrieves a subscription payment record by ID and organization
func (r *SubscriptionRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID, organizationID uuid.UUID) (*entities.SubscriptionEntity, error) {
	var subscriptionModel models.SubscriptionModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, organizationID).
		First(&subscriptionModel).Error

	if err != nil {
		return nil, err
	}

	return subscriptionModel.ToEntity(), nil
}

// FindByAsaasPaymentID retrieves a subscription payment record by Asaas payment ID
func (r *SubscriptionRepositoryImpl) FindByAsaasPaymentID(ctx context.Context, asaasPaymentID string) (*entities.SubscriptionEntity, error) {
	var subscriptionModel models.SubscriptionModel
	err := r.db.WithContext(ctx).
		Where("asaas_payment_id = ?", asaasPaymentID).
		First(&subscriptionModel).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return subscriptionModel.ToEntity(), nil
}

// Create creates a new subscription payment record
func (r *SubscriptionRepositoryImpl) Create(ctx context.Context, subscription *entities.SubscriptionEntity) error {
	var subscriptionModel models.SubscriptionModel
	subscriptionModel.FromEntity(subscription)

	return r.db.WithContext(ctx).Create(&subscriptionModel).Error
}

// Update updates an existing subscription payment record
func (r *SubscriptionRepositoryImpl) Update(ctx context.Context, id uuid.UUID, organizationID uuid.UUID, subscription *entities.SubscriptionEntity) error {
	return r.db.WithContext(ctx).
		Model(&models.SubscriptionModel{}).
		Where("id = ? AND organization_id = ?", id, organizationID).
		Updates(map[string]interface{}{
			"status":       subscription.Status,
			"amount":       subscription.Amount,
			"due_date":     subscription.DueDate,
			"payment_date": subscription.PaymentDate,
			"invoice_url":  subscription.InvoiceURL,
		}).Error
}

// UpdateByEntity updates an existing subscription payment record using the entity's ID
func (r *SubscriptionRepositoryImpl) UpdateByEntity(ctx context.Context, subscription *entities.SubscriptionEntity) error {
	var subscriptionModel models.SubscriptionModel
	subscriptionModel.FromEntity(subscription)

	return r.db.WithContext(ctx).
		Model(&models.SubscriptionModel{}).
		Where("id = ?", subscription.ID).
		Updates(map[string]interface{}{
			"status":           subscription.Status,
			"amount":           subscription.Amount,
			"due_date":         subscription.DueDate,
			"payment_date":     subscription.PaymentDate,
			"invoice_url":      subscription.InvoiceURL,
			"asaas_payment_id": subscription.AsaasPaymentID,
			"asaas_invoice_id": subscription.AsaasInvoiceID,
		}).Error
}

// Delete soft deletes a subscription payment record
func (r *SubscriptionRepositoryImpl) Delete(ctx context.Context, id uuid.UUID, organizationID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, organizationID).
		Delete(&models.SubscriptionModel{}).Error
}

// CountByOrganizationID counts total subscription payment records for an organization
func (r *SubscriptionRepositoryImpl) CountByOrganizationID(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.SubscriptionModel{}).
		Where("organization_id = ?", organizationID).
		Count(&count).Error

	return count, err
}
