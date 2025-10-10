package repositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/RodolfoBonis/spooliq/features/customer/data/models"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type customerRepositoryImpl struct {
	db *gorm.DB
}

// NewCustomerRepository creates a new instance of CustomerRepository
func NewCustomerRepository(db *gorm.DB) repositories.CustomerRepository {
	return &customerRepositoryImpl{db: db}
}

func (r *customerRepositoryImpl) Create(ctx context.Context, customer *entities.CustomerEntity) error {
	model := &models.CustomerModel{}
	model.FromEntity(customer)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create customer: %w", err)
	}
	return nil
}

func (r *customerRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID, organizationID string) (*entities.CustomerEntity, error) {
	model := &models.CustomerModel{}

	if err := r.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, organizationID).
		First(model).Error; err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	return model.ToEntity(), nil
}

func (r *customerRepositoryImpl) Update(ctx context.Context, customer *entities.CustomerEntity) error {
	model := &models.CustomerModel{}
	model.FromEntity(customer)

	// Use Updates instead of Save to avoid issues with zero values
	if err := r.db.WithContext(ctx).
		Model(&models.CustomerModel{}).
		Where("id = ?", customer.ID).
		Updates(model).Error; err != nil {
		return fmt.Errorf("failed to update customer: %w", err)
	}

	return nil
}

func (r *customerRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.CustomerModel{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}
	return nil
}

func (r *customerRepositoryImpl) FindAll(ctx context.Context, organizationID string, limit, offset int) ([]*entities.CustomerEntity, int, error) {
	var customers []*models.CustomerModel
	var total int64

	query := r.db.WithContext(ctx).
		Model(&models.CustomerModel{}).
		Where("organization_id = ?", organizationID)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count customers: %w", err)
	}

	// Get paginated results
	if err := query.
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&customers).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find customers: %w", err)
	}

	// Convert to entities
	entities := make([]*entities.CustomerEntity, len(customers))
	for i, model := range customers {
		entities[i] = model.ToEntity()
	}

	return entities, int(total), nil
}

func (r *customerRepositoryImpl) SearchCustomers(ctx context.Context, organizationID string, filters map[string]interface{}, limit, offset int) ([]*entities.CustomerEntity, int, error) {
	var customers []*models.CustomerModel
	var total int64

	query := r.db.WithContext(ctx).
		Model(&models.CustomerModel{}).
		Where("organization_id = ?", organizationID)

	// Apply filters
	if name, ok := filters["name"].(string); ok && name != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%")
	}
	if email, ok := filters["email"].(string); ok && email != "" {
		query = query.Where("LOWER(email) LIKE ?", "%"+strings.ToLower(email)+"%")
	}
	if phone, ok := filters["phone"].(string); ok && phone != "" {
		query = query.Where("phone LIKE ?", "%"+phone+"%")
	}
	if document, ok := filters["document"].(string); ok && document != "" {
		query = query.Where("document LIKE ?", "%"+document+"%")
	}
	if city, ok := filters["city"].(string); ok && city != "" {
		query = query.Where("LOWER(city) LIKE ?", "%"+strings.ToLower(city)+"%")
	}
	if state, ok := filters["state"].(string); ok && state != "" {
		query = query.Where("LOWER(state) LIKE ?", "%"+strings.ToLower(state)+"%")
	}
	if isActive, ok := filters["is_active"].(bool); ok {
		query = query.Where("is_active = ?", isActive)
	}
	if id, ok := filters["id"].(uuid.UUID); ok && id != uuid.Nil {
		query = query.Where("id = ?", id)
	}

	// Apply sorting
	sortBy := "created_at"
	sortDir := "DESC"
	if sort, ok := filters["sort_by"].(string); ok && sort != "" {
		sortBy = sort
	}
	if dir, ok := filters["sort_dir"].(string); ok && strings.ToUpper(dir) == "ASC" {
		sortDir = "ASC"
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count customers: %w", err)
	}

	// Get paginated results
	if err := query.
		Limit(limit).
		Offset(offset).
		Order(fmt.Sprintf("%s %s", sortBy, sortDir)).
		Find(&customers).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search customers: %w", err)
	}

	// Convert to entities
	entities := make([]*entities.CustomerEntity, len(customers))
	for i, model := range customers {
		entities[i] = model.ToEntity()
	}

	return entities, int(total), nil
}

func (r *customerRepositoryImpl) ExistsByEmail(ctx context.Context, email string, organizationID string, excludeID *uuid.UUID) (bool, error) {
	var count int64

	query := r.db.WithContext(ctx).
		Model(&models.CustomerModel{}).
		Where("email = ? AND organization_id = ?", email, organizationID)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check customer existence: %w", err)
	}

	return count > 0, nil
}

func (r *customerRepositoryImpl) CountBudgetsByCustomer(ctx context.Context, customerID uuid.UUID) (int64, error) {
	var count int64

	// Query budgets table to count budgets for this customer
	if err := r.db.WithContext(ctx).
		Table("budgets").
		Where("customer_id = ?", customerID).
		Count(&count).Error; err != nil {
		// If table doesn't exist yet, return 0
		if strings.Contains(err.Error(), "does not exist") {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to count budgets: %w", err)
	}

	return count, nil
}
