package repositories

import (
	"context"
	"fmt"

	"github.com/RodolfoBonis/spooliq/features/company/data/models"
	"github.com/RodolfoBonis/spooliq/features/company/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"gorm.io/gorm"
)

type brandingRepositoryImpl struct {
	db *gorm.DB
}

// NewBrandingRepository creates a new instance of BrandingRepository
func NewBrandingRepository(db *gorm.DB) repositories.BrandingRepository {
	return &brandingRepositoryImpl{db: db}
}

// FindByOrganizationID retrieves branding configuration by organization ID
func (r *brandingRepositoryImpl) FindByOrganizationID(ctx context.Context, orgID string) (*entities.CompanyBrandingEntity, error) {
	var model models.CompanyBrandingModel
	err := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("branding not found for organization %s", orgID)
		}
		return nil, fmt.Errorf("failed to fetch branding: %w", err)
	}

	return modelToEntity(&model), nil
}

// Create creates a new branding configuration
func (r *brandingRepositoryImpl) Create(ctx context.Context, branding *entities.CompanyBrandingEntity) error {
	model := entityToModel(branding)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create branding: %w", err)
	}

	branding.ID = model.ID
	branding.CreatedAt = model.CreatedAt
	branding.UpdatedAt = model.UpdatedAt

	return nil
}

// Update updates an existing branding configuration
func (r *brandingRepositoryImpl) Update(ctx context.Context, branding *entities.CompanyBrandingEntity) error {
	model := entityToModel(branding)
	if err := r.db.WithContext(ctx).Model(&models.CompanyBrandingModel{}).
		Where("organization_id = ?", branding.OrganizationID).
		Updates(model).Error; err != nil {
		return fmt.Errorf("failed to update branding: %w", err)
	}

	return nil
}

// GetTemplates returns all available branding templates
func (r *brandingRepositoryImpl) GetTemplates() []entities.BrandingTemplate {
	return entities.DefaultTemplates
}

// modelToEntity converts a database model to a domain entity
func modelToEntity(model *models.CompanyBrandingModel) *entities.CompanyBrandingEntity {
	return &entities.CompanyBrandingEntity{
		ID:                 model.ID,
		OrganizationID:     model.OrganizationID,
		TemplateName:       model.TemplateName,
		HeaderBgColor:      model.HeaderBgColor,
		HeaderTextColor:    model.HeaderTextColor,
		PrimaryColor:       model.PrimaryColor,
		PrimaryTextColor:   model.PrimaryTextColor,
		SecondaryColor:     model.SecondaryColor,
		SecondaryTextColor: model.SecondaryTextColor,
		TitleColor:         model.TitleColor,
		BodyTextColor:      model.BodyTextColor,
		AccentColor:        model.AccentColor,
		BorderColor:        model.BorderColor,
		BackgroundColor:    model.BackgroundColor,
		TableHeaderBgColor: model.TableHeaderBgColor,
		TableRowAltBgColor: model.TableRowAltBgColor,
		CreatedAt:          model.CreatedAt,
		UpdatedAt:          model.UpdatedAt,
	}
}

// entityToModel converts a domain entity to a database model
func entityToModel(entity *entities.CompanyBrandingEntity) *models.CompanyBrandingModel {
	return &models.CompanyBrandingModel{
		ID:                 entity.ID,
		OrganizationID:     entity.OrganizationID,
		TemplateName:       entity.TemplateName,
		HeaderBgColor:      entity.HeaderBgColor,
		HeaderTextColor:    entity.HeaderTextColor,
		PrimaryColor:       entity.PrimaryColor,
		PrimaryTextColor:   entity.PrimaryTextColor,
		SecondaryColor:     entity.SecondaryColor,
		SecondaryTextColor: entity.SecondaryTextColor,
		TitleColor:         entity.TitleColor,
		BodyTextColor:      entity.BodyTextColor,
		AccentColor:        entity.AccentColor,
		BorderColor:        entity.BorderColor,
		BackgroundColor:    entity.BackgroundColor,
		TableHeaderBgColor: entity.TableHeaderBgColor,
		TableRowAltBgColor: entity.TableRowAltBgColor,
		CreatedAt:          entity.CreatedAt,
		UpdatedAt:          entity.UpdatedAt,
	}
}
