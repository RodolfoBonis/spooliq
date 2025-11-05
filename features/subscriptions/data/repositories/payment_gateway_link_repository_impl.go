package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/data/models"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"gorm.io/gorm"
)

// PaymentGatewayLinkRepositoryImpl implements PaymentGatewayLinkRepository
type PaymentGatewayLinkRepositoryImpl struct {
	db *gorm.DB
}

// NewPaymentGatewayLinkRepository creates a new PaymentGatewayLinkRepository
func NewPaymentGatewayLinkRepository(db *gorm.DB) repositories.PaymentGatewayLinkRepository {
	return &PaymentGatewayLinkRepositoryImpl{db: db}
}

// Create creates a new payment gateway link
func (r *PaymentGatewayLinkRepositoryImpl) Create(ctx context.Context, link *entities.PaymentGatewayLinkEntity) error {
	model := &models.PaymentGatewayLinkModel{}
	model.FromEntity(link)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}

	*link = *model.ToEntity()
	return nil
}

// FindByOrganizationID finds a payment gateway link by organization ID
func (r *PaymentGatewayLinkRepositoryImpl) FindByOrganizationID(ctx context.Context, organizationID string) (*entities.PaymentGatewayLinkEntity, error) {
	var model models.PaymentGatewayLinkModel
	err := r.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return model.ToEntity(), nil
}

// Update updates a payment gateway link
func (r *PaymentGatewayLinkRepositoryImpl) Update(ctx context.Context, link *entities.PaymentGatewayLinkEntity) error {
	model := &models.PaymentGatewayLinkModel{}
	model.FromEntity(link)

	return r.db.WithContext(ctx).
		Where("organization_id = ?", link.OrganizationID).
		Updates(model).Error
}

// Delete soft deletes a payment gateway link
func (r *PaymentGatewayLinkRepositoryImpl) Delete(ctx context.Context, organizationID string) error {
	return r.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Delete(&models.PaymentGatewayLinkModel{}).Error
}
