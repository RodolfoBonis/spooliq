package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/users/data/models"
	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepositoryImpl implements the UserRepository interface
type UserRepositoryImpl struct {
	db *gorm.DB
}

// NewUserRepository creates a new instance of UserRepositoryImpl
func NewUserRepository(db *gorm.DB) domainRepositories.UserRepository {
	return &UserRepositoryImpl{db: db}
}

// FindAll retrieves all users for a given organization
func (r *UserRepositoryImpl) FindAll(ctx context.Context, organizationID string, q helpers.ListQuery) ([]*entities.UserEntity, int64, error) {
	build := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&models.UserModel{}).Where("organization_id = ?", organizationID)
		if q.Search != "" {
			like := "%" + q.Search + "%"
			query = query.Where("name ILIKE ? OR email ILIKE ?", like, like)
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

	var userModels []models.UserModel
	if err := paginate(query, q).Find(&userModels).Error; err != nil {
		return nil, 0, err
	}

	users := make([]*entities.UserEntity, len(userModels))
	for i, model := range userModels {
		users[i] = model.ToEntity()
	}

	return users, total, nil
}

// FindByID retrieves a user by ID and organization
func (r *UserRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID, organizationID string) (*entities.UserEntity, error) {
	var userModel models.UserModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, organizationID).
		First(&userModel).Error

	if err != nil {
		return nil, err
	}

	return userModel.ToEntity(), nil
}

// FindByEmail retrieves a user by email
func (r *UserRepositoryImpl) FindByEmail(ctx context.Context, email string) (*entities.UserEntity, error) {
	var userModel models.UserModel
	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&userModel).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return userModel.ToEntity(), nil
}

// FindByKeycloakUserID retrieves a user by Keycloak user ID
func (r *UserRepositoryImpl) FindByKeycloakUserID(ctx context.Context, keycloakUserID string) (*entities.UserEntity, error) {
	var userModel models.UserModel
	err := r.db.WithContext(ctx).
		Where("keycloak_user_id = ?", keycloakUserID).
		First(&userModel).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return userModel.ToEntity(), nil
}

// FindOwner retrieves the owner user for a given organization
func (r *UserRepositoryImpl) FindOwner(ctx context.Context, organizationID string) (*entities.UserEntity, error) {
	var userModel models.UserModel
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND user_type = ?", organizationID, "owner").
		First(&userModel).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return userModel.ToEntity(), nil
}

// Create creates a new user
func (r *UserRepositoryImpl) Create(ctx context.Context, user *entities.UserEntity) error {
	var userModel models.UserModel
	userModel.FromEntity(user)

	return r.db.WithContext(ctx).Create(&userModel).Error
}

// Update updates an existing user
func (r *UserRepositoryImpl) Update(ctx context.Context, id uuid.UUID, organizationID string, user *entities.UserEntity) error {
	return r.db.WithContext(ctx).
		Model(&models.UserModel{}).
		Where("id = ? AND organization_id = ?", id, organizationID).
		Updates(map[string]interface{}{
			"name":      user.Name,
			"is_active": user.IsActive,
		}).Error
}

// Delete soft deletes a user
func (r *UserRepositoryImpl) Delete(ctx context.Context, id uuid.UUID, organizationID string) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, organizationID).
		Delete(&models.UserModel{}).Error
}

// paginate applies the page's offset and limit to a query. A non-positive limit
// (e.g. a zero-value ListQuery used in tests) means "no limit" instead of gorm's
// literal LIMIT 0, which would return no rows.
func paginate(db *gorm.DB, q helpers.ListQuery) *gorm.DB {
	db = db.Offset(q.Offset())
	if q.Limit() > 0 {
		db = db.Limit(q.Limit())
	}
	return db
}
