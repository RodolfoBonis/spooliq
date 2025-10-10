package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/company/data/models"
	"github.com/RodolfoBonis/spooliq/features/company/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type companyRepositoryImpl struct {
	db *gorm.DB
}

// NewCompanyRepository creates a new instance of CompanyRepository
func NewCompanyRepository(db *gorm.DB) repositories.CompanyRepository {
	return &companyRepositoryImpl{db: db}
}

// Create creates a new company in the database
func (r *companyRepositoryImpl) Create(ctx context.Context, company *entities.CompanyEntity) error {
	model := &models.CompanyModel{}
	model.FromEntity(company)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}

	*company = *model.ToEntity()
	return nil
}

// FindByOrganizationID finds a company by organization ID
func (r *companyRepositoryImpl) FindByOrganizationID(ctx context.Context, organizationID string) (*entities.CompanyEntity, error) {
	var model models.CompanyModel
	if err := r.db.WithContext(ctx).Where("organization_id = ?", organizationID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrCompanyNotFound
		}
		return nil, err
	}

	return model.ToEntity(), nil
}

// FindByID finds a company by ID
func (r *companyRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entities.CompanyEntity, error) {
	var model models.CompanyModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrCompanyNotFound
		}
		return nil, err
	}

	return model.ToEntity(), nil
}

// Update updates an existing company in the database
func (r *companyRepositoryImpl) Update(ctx context.Context, company *entities.CompanyEntity) error {
	model := &models.CompanyModel{}
	model.FromEntity(company)

	result := r.db.WithContext(ctx).Model(&models.CompanyModel{}).
		Where("id = ?", company.ID).
		Updates(model)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return entities.ErrCompanyNotFound
	}

	return nil
}

// Delete soft deletes a company from the database
func (r *companyRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.CompanyModel{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return entities.ErrCompanyNotFound
	}

	return nil
}

// ExistsByOrganizationID checks if a company exists for the given organization ID
func (r *companyRepositoryImpl) ExistsByOrganizationID(ctx context.Context, organizationID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.CompanyModel{}).
		Where("organization_id = ?", organizationID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
