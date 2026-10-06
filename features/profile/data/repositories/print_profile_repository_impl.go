// Package repositories contains the GORM implementation of the print profile repository.
package repositories

import (
	"time"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/profile/data/models"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PrintProfileRepositoryImpl implements repositories.PrintProfileRepository.
type PrintProfileRepositoryImpl struct {
	db *gorm.DB
}

// NewPrintProfileRepository creates a new PrintProfileRepositoryImpl.
func NewPrintProfileRepository(db *gorm.DB) repositories.PrintProfileRepository {
	return &PrintProfileRepositoryImpl{db: db}
}

// clearOtherDefaults unsets is_default on every other live profile of the same
// organization, excluding exceptID. Run inside the mutating transaction so the
// "single default per organization" invariant — also enforced by a partial
// unique index — is never transiently violated.
func clearOtherDefaults(db *gorm.DB, organizationID string, exceptID uuid.UUID) error {
	return db.Model(&models.PrintProfileModel{}).
		Where("organization_id = ? AND id <> ? AND is_default = ?", organizationID, exceptID, true).
		Update("is_default", false).Error
}

// Create inserts a new profile. It clears sibling defaults when the new profile
// is the default; when the organization has no live default profile yet, the
// new one is promoted to default so budgets always have a profile to fall back to.
func (r *PrintProfileRepositoryImpl) Create(profile *entities.ProfileEntity) error {
	return translateDefaultErr(r.db.Transaction(func(tx *gorm.DB) error {
		if err := lockDefaults(tx, profile.OrganizationID); err != nil {
			return err
		}
		if profile.IsDefault {
			if err := clearOtherDefaults(tx, profile.OrganizationID, profile.ID); err != nil {
				return err
			}
		} else {
			var count int64
			if err := tx.Model(&models.PrintProfileModel{}).
				Where("organization_id = ? AND is_default = ?", profile.OrganizationID, true).
				Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				profile.IsDefault = true
			}
		}
		model := &models.PrintProfileModel{}
		model.FromEntity(profile)
		return tx.Create(model).Error
	}))
}

// GetByID retrieves a profile scoped to the organization (soft-deleted excluded).
func (r *PrintProfileRepositoryImpl) GetByID(id uuid.UUID, organizationID string) (*entities.ProfileEntity, error) {
	var model models.PrintProfileModel
	if err := r.db.
		Where("id = ? AND organization_id = ?", id, organizationID).
		First(&model).Error; err != nil {
		return nil, err
	}
	entity := model.ToEntity()
	return &entity, nil
}

// List retrieves all profiles for an organization, newest first.
func (r *PrintProfileRepositoryImpl) List(organizationID string) ([]*entities.ProfileEntity, error) {
	var profileModels []models.PrintProfileModel
	if err := r.db.
		Where("organization_id = ?", organizationID).
		Order("created_at DESC").
		Find(&profileModels).Error; err != nil {
		return nil, err
	}

	result := make([]*entities.ProfileEntity, 0, len(profileModels))
	for i := range profileModels {
		entity := profileModels[i].ToEntity()
		result = append(result, &entity)
	}
	return result, nil
}

// ListPage returns a page of profiles for an organization applying free-text
// search (on name), sorting and pagination, plus the total count of matches.
func (r *PrintProfileRepositoryImpl) ListPage(organizationID string, q helpers.ListQuery) ([]*entities.ProfileEntity, int64, error) {
	build := func() *gorm.DB {
		query := r.db.Model(&models.PrintProfileModel{}).Where("organization_id = ?", organizationID)
		if q.Search != "" {
			query = query.Where("name ILIKE ?", "%"+q.Search+"%")
		}
		return query
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := build()
	if order := q.OrderClause(); order != "" {
		query = query.Order(order)
	} else {
		query = query.Order("created_at DESC")
	}

	var profileModels []models.PrintProfileModel
	if err := query.Offset(q.Offset()).Limit(q.Limit()).Find(&profileModels).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*entities.ProfileEntity, 0, len(profileModels))
	for i := range profileModels {
		entity := profileModels[i].ToEntity()
		result = append(result, &entity)
	}
	return result, total, nil
}

// Update updates a profile, scoped to its organization, clearing a sibling
// default first when the profile is being set as default. Returns
// gorm.ErrRecordNotFound when no row matches the id+organization pair.
func (r *PrintProfileRepositoryImpl) Update(profile *entities.ProfileEntity) error {
	return translateDefaultErr(r.db.Transaction(func(tx *gorm.DB) error {
		if profile.IsDefault {
			if err := lockDefaults(tx, profile.OrganizationID); err != nil {
				return err
			}
			if err := clearOtherDefaults(tx, profile.OrganizationID, profile.ID); err != nil {
				return err
			}
		}
		model := &models.PrintProfileModel{}
		model.FromEntity(profile)
		result := tx.Model(&models.PrintProfileModel{}).
			Where("id = ? AND organization_id = ?", model.ID, model.OrganizationID).
			Select("name", "description", "machine_preset_id", "energy_preset_id", "cost_preset_id", "is_default", "updated_at").
			Updates(model)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	}))
}

// Delete soft deletes a profile, scoped to its organization.
func (r *PrintProfileRepositoryImpl) Delete(id uuid.UUID, organizationID string) error {
	result := r.db.
		Where("id = ? AND organization_id = ?", id, organizationID).
		Delete(&models.PrintProfileModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SetDefault marks the profile as the single default for its organization.
func (r *PrintProfileRepositoryImpl) SetDefault(id uuid.UUID, organizationID string) (*entities.ProfileEntity, error) {
	var result *entities.ProfileEntity
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var model models.PrintProfileModel
		if err := tx.Where("id = ? AND organization_id = ?", id, organizationID).First(&model).Error; err != nil {
			return err
		}
		if err := lockDefaults(tx, organizationID); err != nil {
			return err
		}
		if err := clearOtherDefaults(tx, organizationID, id); err != nil {
			return err
		}
		now := time.Now()
		if err := tx.Model(&models.PrintProfileModel{}).
			Where("id = ? AND organization_id = ?", id, organizationID).
			Updates(map[string]interface{}{"is_default": true, "updated_at": now}).Error; err != nil {
			return err
		}
		model.IsDefault = true
		model.UpdatedAt = now
		entity := model.ToEntity()
		result = &entity
		return nil
	})
	if err != nil {
		return nil, translateDefaultErr(err)
	}
	return result, nil
}

// Duplicate copies a profile within the same organization. The copy is never a
// default and takes the given name.
func (r *PrintProfileRepositoryImpl) Duplicate(id uuid.UUID, organizationID string, newName string) (*entities.ProfileEntity, error) {
	var result *entities.ProfileEntity
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var base models.PrintProfileModel
		if err := tx.Where("id = ? AND organization_id = ?", id, organizationID).First(&base).Error; err != nil {
			return err
		}
		now := time.Now()
		base.ID = uuid.New()
		base.Name = newName
		base.IsDefault = false
		base.CreatedAt = now
		base.UpdatedAt = now
		base.DeletedAt = gorm.DeletedAt{}
		base.Organization = nil
		if err := tx.Create(&base).Error; err != nil {
			return err
		}
		entity := base.ToEntity()
		result = &entity
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
