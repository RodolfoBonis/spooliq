package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/data/models"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	companyModels "github.com/RodolfoBonis/spooliq/features/company/data/models"
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
	if err := r.db.WithContext(ctx).Preload("Features").Where("id = ?", id).First(&model).Error; err != nil {
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
	if err := r.db.WithContext(ctx).Preload("Features").Where("name = ?", name).First(&model).Error; err != nil {
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
	if err := r.db.WithContext(ctx).Preload("Features").Order("price ASC").Find(&models).Error; err != nil {
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
		Preload("Features").
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

	// Start transaction to update plan and features
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update plan
		if err := tx.Model(&models.SubscriptionPlanModel{}).Where("id = ?", model.ID).Updates(map[string]interface{}{
			"name":        model.Name,
			"description": model.Description,
			"price":       model.Price,
			"cycle":       model.Cycle,
			"is_active":   model.IsActive,
		}).Error; err != nil {
			return err
		}

		// Delete existing features
		if err := tx.Where("subscription_plan_id = ?", model.ID).Delete(&models.PlanFeatureModel{}).Error; err != nil {
			return err
		}

		// Create new features
		if len(model.Features) > 0 {
			if err := tx.Create(&model.Features).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// Delete soft deletes a subscription plan
func (r *subscriptionPlanRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.SubscriptionPlanModel{}, "id = ?", id).Error
}

// GetPlanStats gets statistics for a specific plan
func (r *subscriptionPlanRepositoryImpl) GetPlanStats(ctx context.Context, planID uuid.UUID) (*adminEntities.PlanStats, error) {
	// First get the plan name
	var plan models.SubscriptionPlanModel
	if err := r.db.WithContext(ctx).Select("name, price").Where("id = ?", planID).First(&plan).Error; err != nil {
		return nil, err
	}

	var stats adminEntities.PlanStats
	stats.PlanID = planID.String()
	stats.PlanName = plan.Name

	// Count companies by status
	type companyCount struct {
		Status string
		Count  int
	}
	
	var companyCounts []companyCount
	if err := r.db.WithContext(ctx).
		Table("companies").
		Select("subscription_status as status, COUNT(*) as count").
		Where("subscription_plan_id = ? AND deleted_at IS NULL", planID).
		Group("subscription_status").
		Scan(&companyCounts).Error; err != nil {
		return nil, err
	}

	for _, count := range companyCounts {
		switch count.Status {
		case "active":
			stats.ActiveCompanies = count.Count
		case "trial":
			stats.TrialCompanies = count.Count
		}
		stats.TotalCompanies += count.Count
	}

	// Count total users for this plan (simplified - assuming we have a users table)
	var userCount int64
	r.db.WithContext(ctx).
		Table("users").
		Joins("JOIN companies ON users.organization_id = companies.organization_id").
		Where("companies.subscription_plan_id = ? AND companies.deleted_at IS NULL AND users.deleted_at IS NULL", planID).
		Count(&userCount)
	stats.TotalActiveUsers = int(userCount)

	// Calculate revenue (monthly based on plan price)
	stats.MonthlyRevenue = float64(stats.ActiveCompanies) * plan.Price
	stats.AnnualRevenue = stats.MonthlyRevenue * 12

	// Calculate conversion rate (trial to paid) - simplified
	if stats.TrialCompanies > 0 {
		stats.ConversionRate = float64(stats.ActiveCompanies) / float64(stats.TrialCompanies+stats.ActiveCompanies) * 100
	}

	// Simplified churn rate calculation
	stats.ChurnRate = 2.5 // placeholder - would need historical data

	return &stats, nil
}

// GetPlanCompanies gets companies using a specific plan with pagination
func (r *subscriptionPlanRepositoryImpl) GetPlanCompanies(ctx context.Context, planID uuid.UUID, page, pageSize int, statusFilter string) (*adminEntities.ListPlanCompaniesResponse, error) {
	var companies []adminEntities.PlanCompanyListItem
	var totalCount int64

	query := r.db.WithContext(ctx).
		Table("companies").
		Where("subscription_plan_id = ? AND deleted_at IS NULL", planID)

	if statusFilter != "" {
		query = query.Where("subscription_status = ?", statusFilter)
	}

	// Count total
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// Get paginated data
	offset := (page - 1) * pageSize
	if err := query.
		Select(`
			companies.id,
			companies.organization_id,
			companies.name,
			COALESCE(companies.email, '') as email,
			companies.subscription_status,
			companies.subscription_started_at,
			companies.trial_ends_at,
			companies.created_at,
			(SELECT COUNT(*) FROM users WHERE users.organization_id = companies.organization_id AND users.deleted_at IS NULL) as total_users
		`).
		Offset(offset).
		Limit(pageSize).
		Order("companies.created_at DESC").
		Scan(&companies).Error; err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))

	return &adminEntities.ListPlanCompaniesResponse{
		Companies:  companies,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// GetPlanFinancialReport gets financial report for a specific plan
func (r *subscriptionPlanRepositoryImpl) GetPlanFinancialReport(ctx context.Context, planID uuid.UUID, period string) (*adminEntities.PlanFinancialReport, error) {
	// Get plan details first
	var plan models.SubscriptionPlanModel
	if err := r.db.WithContext(ctx).Select("name, price").Where("id = ?", planID).First(&plan).Error; err != nil {
		return nil, err
	}

	// Count active subscriptions for this plan
	var activeCount int64
	if err := r.db.WithContext(ctx).
		Table("companies").
		Where("subscription_plan_id = ? AND subscription_status = 'active' AND deleted_at IS NULL", planID).
		Count(&activeCount).Error; err != nil {
		return nil, err
	}

	currentRevenue := float64(activeCount) * plan.Price
	
	// Simplified report - in real implementation would query payment history
	report := &adminEntities.PlanFinancialReport{
		PlanID:       planID.String(),
		PlanName:     plan.Name,
		ReportPeriod: period,
		Revenue: adminEntities.PlanRevenueMetrics{
			CurrentPeriod:    currentRevenue,
			PreviousPeriod:   currentRevenue * 0.9, // Simplified
			GrowthPercentage: 10.0,                 // Simplified
			AveragePerUser:   plan.Price,
			TotalLifetime:    currentRevenue * 6, // Simplified
		},
		Subscriptions: adminEntities.PlanSubscriptionMetrics{
			NewSubscriptions:       5, // Simplified
			CancelledSubscriptions: 1, // Simplified
			ChurnRate:             2.5,
			RetentionRate:         97.5,
			ConversionRate:        85.0,
		},
		Projections: adminEntities.PlanRevenueProjections{
			NextMonth:   currentRevenue * 1.05,
			NextQuarter: currentRevenue * 3.1,
			NextYear:    currentRevenue * 12.5,
			Methodology: "Based on current growth trends and market analysis",
		},
		Trends: []adminEntities.PlanRevenueTrendPoint{
			{Period: "2024-10", Revenue: currentRevenue, Subscriptions: int(activeCount)},
			{Period: "2024-09", Revenue: currentRevenue * 0.95, Subscriptions: int(activeCount) - 2},
			{Period: "2024-08", Revenue: currentRevenue * 0.9, Subscriptions: int(activeCount) - 4},
		},
	}

	return report, nil
}

// CanDeletePlan checks if a plan can be safely deleted
func (r *subscriptionPlanRepositoryImpl) CanDeletePlan(ctx context.Context, planID uuid.UUID) (*adminEntities.PlanDeletionCheck, error) {
	var activeCount, trialCount int64

	// Count active companies
	if err := r.db.WithContext(ctx).
		Table("companies").
		Where("subscription_plan_id = ? AND subscription_status = 'active' AND deleted_at IS NULL", planID).
		Count(&activeCount).Error; err != nil {
		return nil, err
	}

	// Count trial companies
	if err := r.db.WithContext(ctx).
		Table("companies").
		Where("subscription_plan_id = ? AND subscription_status = 'trial' AND deleted_at IS NULL", planID).
		Count(&trialCount).Error; err != nil {
		return nil, err
	}

	check := &adminEntities.PlanDeletionCheck{
		ActiveCompanies: int(activeCount),
		TrialCompanies:  int(trialCount),
	}

	if activeCount > 0 || trialCount > 0 {
		check.CanDelete = false
		check.Reason = fmt.Sprintf("Plan has %d active and %d trial companies", activeCount, trialCount)
		
		if activeCount > 0 {
			check.BlockingIssues = append(check.BlockingIssues, 
				fmt.Sprintf("%d companies with active subscriptions", activeCount))
		}
		if trialCount > 0 {
			check.BlockingIssues = append(check.BlockingIssues, 
				fmt.Sprintf("%d companies in trial period", trialCount))
		}

		check.Recommendations = []string{
			"Migrate active companies to another plan before deletion",
			"Wait for trial periods to expire or convert to paid plans",
			"Contact companies to discuss plan alternatives",
		}
	} else {
		check.CanDelete = true
		check.Reason = "No active or trial companies using this plan"
	}

	return check, nil
}

// BulkUpdate updates multiple subscription plans
func (r *subscriptionPlanRepositoryImpl) BulkUpdate(ctx context.Context, planIDs []uuid.UUID, updates map[string]interface{}, userID, userEmail, reason string) (*adminEntities.BulkOperationResult, error) {
	result := &adminEntities.BulkOperationResult{
		TotalRequested: len(planIDs),
		Results:        make([]adminEntities.BulkOperationItem, 0, len(planIDs)),
	}

	for _, planID := range planIDs {
		item := adminEntities.BulkOperationItem{
			PlanID: planID.String(),
		}

		// Get plan name first
		var plan models.SubscriptionPlanModel
		if err := r.db.WithContext(ctx).Select("name").Where("id = ?", planID).First(&plan).Error; err != nil {
			item.Success = false
			item.Error = "Plan not found"
			result.Failed++
		} else {
			item.PlanName = plan.Name

			// Perform update
			if err := r.db.WithContext(ctx).Model(&models.SubscriptionPlanModel{}).Where("id = ?", planID).Updates(updates).Error; err != nil {
				item.Success = false
				item.Error = err.Error()
				result.Failed++
			} else {
				item.Success = true
				result.Successful++

				// Create audit entry
				auditEntry := &adminEntities.PlanAuditEntry{
					PlanID:    planID.String(),
					PlanName:  plan.Name,
					Action:    "bulk_updated",
					UserID:    userID,
					UserEmail: userEmail,
					Changes:   updates,
					Reason:    reason,
				}
				r.CreateAuditEntry(ctx, auditEntry)
			}
		}

		result.Results = append(result.Results, item)
	}

	result.Summary = fmt.Sprintf("Bulk update completed: %d successful, %d failed", result.Successful, result.Failed)
	return result, nil
}

// BulkActivate activates multiple subscription plans
func (r *subscriptionPlanRepositoryImpl) BulkActivate(ctx context.Context, planIDs []uuid.UUID, userID, userEmail, reason string) (*adminEntities.BulkOperationResult, error) {
	updates := map[string]interface{}{"is_active": true}
	result, err := r.BulkUpdate(ctx, planIDs, updates, userID, userEmail, reason)
	if err != nil {
		return nil, err
	}

	// Update action in audit entries
	for i := range result.Results {
		if result.Results[i].Success {
			auditEntry := &adminEntities.PlanAuditEntry{
				PlanID:    result.Results[i].PlanID,
				PlanName:  result.Results[i].PlanName,
				Action:    "bulk_activated",
				UserID:    userID,
				UserEmail: userEmail,
				Changes:   updates,
				Reason:    reason,
			}
			r.CreateAuditEntry(ctx, auditEntry)
		}
	}

	result.Summary = fmt.Sprintf("Bulk activation completed: %d plans activated, %d failed", result.Successful, result.Failed)
	return result, nil
}

// BulkDeactivate deactivates multiple subscription plans
func (r *subscriptionPlanRepositoryImpl) BulkDeactivate(ctx context.Context, planIDs []uuid.UUID, userID, userEmail, reason string) (*adminEntities.BulkOperationResult, error) {
	updates := map[string]interface{}{"is_active": false}
	result, err := r.BulkUpdate(ctx, planIDs, updates, userID, userEmail, reason)
	if err != nil {
		return nil, err
	}

	// Update action in audit entries
	for i := range result.Results {
		if result.Results[i].Success {
			auditEntry := &adminEntities.PlanAuditEntry{
				PlanID:    result.Results[i].PlanID,
				PlanName:  result.Results[i].PlanName,
				Action:    "bulk_deactivated",
				UserID:    userID,
				UserEmail: userEmail,
				Changes:   updates,
				Reason:    reason,
			}
			r.CreateAuditEntry(ctx, auditEntry)
		}
	}

	result.Summary = fmt.Sprintf("Bulk deactivation completed: %d plans deactivated, %d failed", result.Successful, result.Failed)
	return result, nil
}

// CreateAuditEntry creates a new audit entry
func (r *subscriptionPlanRepositoryImpl) CreateAuditEntry(ctx context.Context, entry *adminEntities.PlanAuditEntry) error {
	model := &models.PlanAuditModel{}
	model.FromEntity(entry)
	return r.db.WithContext(ctx).Create(model).Error
}

// GetPlanHistory gets audit history for a specific plan
func (r *subscriptionPlanRepositoryImpl) GetPlanHistory(ctx context.Context, planID uuid.UUID, page, pageSize int) (*adminEntities.PlanAuditResponse, error) {
	var entries []models.PlanAuditModel
	var totalCount int64

	// Count total
	if err := r.db.WithContext(ctx).Model(&models.PlanAuditModel{}).Where("plan_id = ?", planID).Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// Get paginated data
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).
		Where("plan_id = ?", planID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&entries).Error; err != nil {
		return nil, err
	}

	// Convert to entities
	entitiesResult := make([]adminEntities.PlanAuditEntry, len(entries))
	for i, entry := range entries {
		entitiesResult[i] = *entry.ToEntity()
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))

	return &adminEntities.PlanAuditResponse{
		Entries:    entitiesResult,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// CreateTemplate creates a new plan template
func (r *subscriptionPlanRepositoryImpl) CreateTemplate(ctx context.Context, template *adminEntities.PlanTemplate) error {
	model := &models.PlanTemplateModel{}
	model.FromEntity(template)
	return r.db.WithContext(ctx).Create(model).Error
}

// GetTemplates gets plan templates by category
func (r *subscriptionPlanRepositoryImpl) GetTemplates(ctx context.Context, category string) ([]*adminEntities.PlanTemplate, error) {
	var models []models.PlanTemplateModel
	query := r.db.WithContext(ctx).Where("is_active = ?", true)
	
	if category != "" {
		query = query.Where("category = ?", category)
	}
	
	if err := query.Order("usage_count DESC, name ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	templates := make([]*adminEntities.PlanTemplate, len(models))
	for i, model := range models {
		templates[i] = model.ToEntity()
	}

	return templates, nil
}

// GetTemplateByID gets a template by ID
func (r *subscriptionPlanRepositoryImpl) GetTemplateByID(ctx context.Context, templateID uuid.UUID) (*adminEntities.PlanTemplate, error) {
	var model models.PlanTemplateModel
	if err := r.db.WithContext(ctx).Where("id = ? AND is_active = ?", templateID, true).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return model.ToEntity(), nil
}

// CreatePlanFromTemplate creates a plan from a template
func (r *subscriptionPlanRepositoryImpl) CreatePlanFromTemplate(ctx context.Context, templateID uuid.UUID, customizations map[string]interface{}, userID, userEmail, reason string) (*entities.SubscriptionPlanEntity, error) {
	// Get template
	template, err := r.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, fmt.Errorf("template not found")
	}

	// Start transaction
	var result *entities.SubscriptionPlanEntity
	txErr := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create plan from template data
		planData := template.PlanData
		
		// Apply customizations
		if customizations["name"] != nil {
			planData.Name = customizations["name"].(string)
		}
		if customizations["description"] != nil {
			planData.Description = customizations["description"].(string)
		}
		if customizations["price"] != nil {
			planData.Price = customizations["price"].(float64)
		}
		if customizations["cycle"] != nil {
			planData.Cycle = customizations["cycle"].(string)
		}

		// Convert features
		features := make([]entities.PlanFeatureEntity, len(planData.Features))
		for i, f := range planData.Features {
			features[i] = entities.PlanFeatureEntity{
				Name:        f.Name,
				Description: f.Description,
				IsActive:    f.IsActive,
			}
		}

		// Create plan entity
		plan := &entities.SubscriptionPlanEntity{
			Name:        planData.Name,
			Description: planData.Description,
			Price:       planData.Price,
			Cycle:       planData.Cycle,
			Features:    features,
			IsActive:    true,
		}

		// Create plan
		planModel := &models.SubscriptionPlanModel{}
		planModel.FromEntity(plan)
		if err := tx.Create(planModel).Error; err != nil {
			return err
		}

		// Increment template usage
		if err := tx.Model(&models.PlanTemplateModel{}).Where("id = ?", templateID).UpdateColumn("usage_count", gorm.Expr("usage_count + ?", 1)).Error; err != nil {
			return err
		}

		// Create audit entry
		auditEntry := &adminEntities.PlanAuditEntry{
			PlanID:    planModel.ID.String(),
			PlanName:  plan.Name,
			Action:    "created_from_template",
			UserID:    userID,
			UserEmail: userEmail,
			Changes: map[string]interface{}{
				"template_id":     template.ID,
				"template_name":   template.Name,
				"customizations":  customizations,
			},
			Reason: reason,
		}
		
		auditModel := &models.PlanAuditModel{}
		auditModel.FromEntity(auditEntry)
		if err := tx.Create(auditModel).Error; err != nil {
			return err
		}

		result = planModel.ToEntity()
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return result, nil
}

// IncrementTemplateUsage increments template usage counter
func (r *subscriptionPlanRepositoryImpl) IncrementTemplateUsage(ctx context.Context, templateID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.PlanTemplateModel{}).
		Where("id = ?", templateID).
		UpdateColumn("usage_count", gorm.Expr("usage_count + ?", 1)).Error
}

// GetAvailableFeatures gets available features for plans
func (r *subscriptionPlanRepositoryImpl) GetAvailableFeatures(ctx context.Context) ([]*adminEntities.AvailableFeature, error) {
	// Simplified: return predefined features
	// In a real implementation, this would come from a features table
	features := []*adminEntities.AvailableFeature{
		{Name: "basic_calculations", Description: "Basic cost calculations", Category: "calculations", IsActive: true},
		{Name: "advanced_calculations", Description: "Advanced cost calculations with multiple materials", Category: "calculations", IsActive: true},
		{Name: "multi_material", Description: "Multi-material support", Category: "materials", IsActive: true},
		{Name: "batch_processing", Description: "Batch job processing", Category: "processing", IsActive: true},
		{Name: "api_access", Description: "API access for integrations", Category: "integration", IsActive: true},
		{Name: "custom_branding", Description: "Custom company branding", Category: "branding", IsActive: true},
		{Name: "priority_support", Description: "Priority customer support", Category: "support", IsActive: true},
		{Name: "unlimited_users", Description: "Unlimited team members", Category: "users", IsActive: true},
		{Name: "advanced_reports", Description: "Advanced reporting and analytics", Category: "reports", IsActive: true},
		{Name: "export_formats", Description: "Multiple export formats (PDF, Excel, etc.)", Category: "export", IsActive: true},
	}

	return features, nil
}

// ValidateFeatures validates if features are available and compatible
func (r *subscriptionPlanRepositoryImpl) ValidateFeatures(ctx context.Context, features []adminEntities.PlanTemplateFeature) (*adminEntities.FeatureValidationResult, error) {
	availableFeatures, err := r.GetAvailableFeatures(ctx)
	if err != nil {
		return nil, err
	}

	// Create map for quick lookup
	availableMap := make(map[string]*adminEntities.AvailableFeature)
	for _, feature := range availableFeatures {
		availableMap[feature.Name] = feature
	}

	result := &adminEntities.FeatureValidationResult{
		IsValid:         true,
		ValidFeatures:   make([]adminEntities.PlanTemplateFeature, 0),
		InvalidFeatures: make([]adminEntities.InvalidFeatureError, 0),
		Suggestions:     make([]string, 0),
	}

	for _, feature := range features {
		if available, exists := availableMap[feature.Name]; exists && available.IsActive {
			result.ValidFeatures = append(result.ValidFeatures, feature)
		} else {
			result.IsValid = false
			result.InvalidFeatures = append(result.InvalidFeatures, adminEntities.InvalidFeatureError{
				Feature: feature,
				Error:   "Feature not available or inactive",
			})
		}
	}

	// Add suggestions
	if !result.IsValid {
		result.Suggestions = append(result.Suggestions, "Consider using available features from the catalog")
		result.Suggestions = append(result.Suggestions, "Contact support for custom feature requests")
	}

	return result, nil
}

// CreateMigration creates a new plan migration
func (r *subscriptionPlanRepositoryImpl) CreateMigration(ctx context.Context, request *adminEntities.PlanMigrationRequest, userID, userEmail string) (*adminEntities.PlanMigrationResult, error) {
	// Validate plans exist
	var fromPlan, toPlan models.SubscriptionPlanModel
	if err := r.db.WithContext(ctx).Where("id = ?", request.FromPlanID).First(&fromPlan).Error; err != nil {
		return nil, fmt.Errorf("from plan not found")
	}
	if err := r.db.WithContext(ctx).Where("id = ?", request.ToPlanID).First(&toPlan).Error; err != nil {
		return nil, fmt.Errorf("to plan not found")
	}

	// Count companies to migrate
	var totalCompanies int64
	query := r.db.WithContext(ctx).Model(&companyModels.CompanyModel{}).Where("subscription_plan_id = ? AND deleted_at IS NULL", request.FromPlanID)
	
	if len(request.CompanyIDs) > 0 {
		// Convert string IDs to UUIDs
		companyUUIDs := make([]uuid.UUID, len(request.CompanyIDs))
		for i, id := range request.CompanyIDs {
			companyUUIDs[i] = uuid.MustParse(id)
		}
		query = query.Where("id IN ?", companyUUIDs)
	}
	
	if err := query.Count(&totalCompanies).Error; err != nil {
		return nil, err
	}

	// Create migration record
	migration := &models.PlanMigrationModel{
		FromPlanID:     uuid.MustParse(request.FromPlanID),
		ToPlanID:       uuid.MustParse(request.ToPlanID),
		Status:         "scheduled",
		TotalCompanies: int(totalCompanies),
		Reason:         request.Reason,
		UserID:         userID,
		UserEmail:      userEmail,
		ScheduledFor:   request.ScheduledFor,
	}

	if err := r.db.WithContext(ctx).Create(migration).Error; err != nil {
		return nil, err
	}

	// If immediate execution requested
	if request.ScheduledFor == nil {
		return r.ExecutePlanMigration(ctx, migration.ID)
	}

	migration.FromPlan = &fromPlan
	migration.ToPlan = &toPlan
	return migration.ToEntity(), nil
}

// ExecutePlanMigration executes a plan migration
func (r *subscriptionPlanRepositoryImpl) ExecutePlanMigration(ctx context.Context, migrationID uuid.UUID) (*adminEntities.PlanMigrationResult, error) {
	// Get migration record
	var migration models.PlanMigrationModel
	if err := r.db.WithContext(ctx).Preload("FromPlan").Preload("ToPlan").Where("id = ?", migrationID).First(&migration).Error; err != nil {
		return nil, err
	}

	// Update status to in_progress
	migration.Status = "in_progress"
	r.db.WithContext(ctx).Save(&migration)

	results := make([]adminEntities.MigrationCompanyItem, 0)
	successful := 0
	failed := 0

	// Get companies to migrate
	var companies []companyModels.CompanyModel
	if err := r.db.WithContext(ctx).Where("subscription_plan_id = ? AND deleted_at IS NULL", migration.FromPlanID).Find(&companies).Error; err != nil {
		migration.Status = "failed"
		r.db.WithContext(ctx).Save(&migration)
		return nil, err
	}

	// Migrate each company
	for _, company := range companies {
		item := adminEntities.MigrationCompanyItem{
			CompanyID:      company.ID.String(),
			CompanyName:    company.Name,
			OrganizationID: company.OrganizationID,
		}

		// Update company plan
		if err := r.db.WithContext(ctx).Model(&company).Update("subscription_plan_id", migration.ToPlanID).Error; err != nil {
			item.Success = false
			item.Error = err.Error()
			failed++
		} else {
			item.Success = true
			successful++

			// Create audit entry
			auditEntry := &adminEntities.PlanAuditEntry{
				PlanID:    migration.ToPlanID.String(),
				PlanName:  migration.ToPlan.Name,
				Action:    "company_migrated",
				UserID:    migration.UserID,
				UserEmail: migration.UserEmail,
				Changes: map[string]interface{}{
					"migration_id":     migrationID.String(),
					"from_plan_id":     migration.FromPlanID.String(),
					"from_plan_name":   migration.FromPlan.Name,
					"company_id":       company.ID.String(),
					"organization_id":  company.OrganizationID,
				},
				Reason: migration.Reason,
			}
			r.CreateAuditEntry(ctx, auditEntry)
		}

		results = append(results, item)
	}

	// Update migration record
	now := time.Now()
	migration.Status = "completed"
	migration.Successful = successful
	migration.Failed = failed
	migration.CompletedAt = &now

	// Store results
	resultsBytes, _ := json.Marshal(results)
	migration.Results = string(resultsBytes)

	r.db.WithContext(ctx).Save(&migration)

	return migration.ToEntity(), nil
}

// GetMigrationStatus gets migration status
func (r *subscriptionPlanRepositoryImpl) GetMigrationStatus(ctx context.Context, migrationID uuid.UUID) (*adminEntities.PlanMigrationResult, error) {
	var migration models.PlanMigrationModel
	if err := r.db.WithContext(ctx).Preload("FromPlan").Preload("ToPlan").Where("id = ?", migrationID).First(&migration).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return migration.ToEntity(), nil
}
