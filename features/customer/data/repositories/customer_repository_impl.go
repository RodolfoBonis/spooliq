package repositories

import (
	"context"
	"fmt"
	"strings"

	budgetModels "github.com/RodolfoBonis/spooliq/features/budget/data/models"
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
		return nil, err
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

// FindAll returns a page of customers for the organization.
func (r *customerRepositoryImpl) FindAll(ctx context.Context, organizationID, search, order string, limit, offset int) ([]*entities.CustomerEntity, int64, error) {
	return r.list(ctx, organizationID, nil, search, order, limit, offset)
}

// SearchCustomers returns a page of customers matching the structured filters
// and the free-text search.
func (r *customerRepositoryImpl) SearchCustomers(ctx context.Context, organizationID string, filters map[string]interface{}, search, order string, limit, offset int) ([]*entities.CustomerEntity, int64, error) {
	return r.list(ctx, organizationID, filters, search, order, limit, offset)
}

// list is the shared query builder behind FindAll and SearchCustomers. All
// queries are organization-scoped. search is a broad case-insensitive term
// matched against name/email/phone/document; filters add column-specific
// clauses. order is trusted (whitelisted by the caller) and defaults to
// newest-first.
func (r *customerRepositoryImpl) list(ctx context.Context, organizationID string, filters map[string]interface{}, search, order string, limit, offset int) ([]*entities.CustomerEntity, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&models.CustomerModel{}).
		Where("organization_id = ?", organizationID)

	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(phone) LIKE ? OR LOWER(document) LIKE ?",
			like, like, like, like,
		)
	}

	query = applyCustomerFilters(query, filters)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count customers: %w", err)
	}

	if order == "" {
		order = "created_at desc"
	}

	var customers []*models.CustomerModel
	if err := query.
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&customers).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find customers: %w", err)
	}

	result := make([]*entities.CustomerEntity, len(customers))
	for i, model := range customers {
		result[i] = model.ToEntity()
	}

	return result, total, nil
}

// applyCustomerFilters adds the column-specific WHERE clauses for the supported
// structured filters.
func applyCustomerFilters(query *gorm.DB, filters map[string]interface{}) *gorm.DB {
	if filters == nil {
		return query
	}
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
	return query
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

	if err := r.db.WithContext(ctx).
		Table("budgets").
		Where("customer_id = ?", customerID).
		Where("budgets.deleted_at IS NULL").
		Count(&count).Error; err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to count budgets: %w", err)
	}

	return count, nil
}

func (r *customerRepositoryImpl) GetCustomerBudgets(ctx context.Context, customerID uuid.UUID) ([]entities.BudgetSummary, error) {
	var budgetModels []budgetModels.BudgetModel

	if err := r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Limit(10).
		Find(&budgetModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get customer budgets: %w", err)
	}

	// Converte para BudgetSummary
	summaries := make([]entities.BudgetSummary, len(budgetModels))
	for i, budget := range budgetModels {
		summaries[i] = entities.BudgetSummary{
			ID:        budget.ID,
			Name:      budget.Name,
			Status:    budget.Status,
			TotalCost: budget.TotalCost,
			CreatedAt: budget.CreatedAt,
		}
	}

	return summaries, nil
}

func (r *customerRepositoryImpl) SumBudgetTotalsByCustomerAndStatus(ctx context.Context, customerID uuid.UUID, statuses []string) (int64, error) {
	var totalSum int64

	if err := r.db.WithContext(ctx).
		Table("budgets").
		Select("COALESCE(SUM(total_cost), 0)").
		Where("customer_id = ?", customerID).
		Where("status IN ?", statuses).
		Where("budgets.deleted_at IS NULL").
		Scan(&totalSum).Error; err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to sum budget totals: %w", err)
	}

	return totalSum, nil
}

// GetBudgetStatsByCustomers computes the budget count and status-filtered total
// for many customers in a single grouped query, avoiding the per-row N+1 the
// list endpoints used to perform.
func (r *customerRepositoryImpl) GetBudgetStatsByCustomers(ctx context.Context, customerIDs []uuid.UUID, totalStatuses []string) (map[uuid.UUID]entities.CustomerBudgetStats, error) {
	result := make(map[uuid.UUID]entities.CustomerBudgetStats)

	ids := make([]uuid.UUID, 0, len(customerIDs))
	seen := make(map[uuid.UUID]struct{}, len(customerIDs))
	for _, id := range customerIDs {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return result, nil
	}

	var rows []struct {
		CustomerID uuid.UUID `gorm:"column:customer_id"`
		Count      int64     `gorm:"column:count"`
		Total      int64     `gorm:"column:total"`
	}

	if err := r.db.WithContext(ctx).
		Table("budgets").
		Select(
			"customer_id, COUNT(*) AS count, COALESCE(SUM(CASE WHEN status IN (?) THEN total_cost ELSE 0 END), 0) AS total",
			totalStatuses,
		).
		Where("customer_id IN ?", ids).
		Where("budgets.deleted_at IS NULL").
		Group("customer_id").
		Scan(&rows).Error; err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return result, nil
		}
		return nil, fmt.Errorf("failed to aggregate budget stats: %w", err)
	}

	for _, row := range rows {
		result[row.CustomerID] = entities.CustomerBudgetStats{
			Count: row.Count,
			Total: row.Total,
		}
	}

	return result, nil
}
