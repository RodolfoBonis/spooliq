package repositories

import (
	"context"

	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
)

// SubscriptionPlanRepository defines the interface for subscription plan data operations
type SubscriptionPlanRepository interface {
	// Create creates a new subscription plan
	Create(ctx context.Context, plan *entities.SubscriptionPlanEntity) error

	// FindByID finds a subscription plan by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entities.SubscriptionPlanEntity, error)

	// FindByName finds a subscription plan by name
	FindByName(ctx context.Context, name string) (*entities.SubscriptionPlanEntity, error)

	// FindAll finds all subscription plans
	FindAll(ctx context.Context) ([]*entities.SubscriptionPlanEntity, error)

	// FindAllActive finds all active subscription plans
	FindAllActive(ctx context.Context) ([]*entities.SubscriptionPlanEntity, error)

	// Update updates a subscription plan
	Update(ctx context.Context, plan *entities.SubscriptionPlanEntity) error

	// Delete soft deletes a subscription plan
	Delete(ctx context.Context, id uuid.UUID) error

	// GetPlanStats gets statistics for a specific plan
	GetPlanStats(ctx context.Context, planID uuid.UUID) (*adminEntities.PlanStats, error)

	// GetPlanCompanies gets companies using a specific plan with pagination
	GetPlanCompanies(ctx context.Context, planID uuid.UUID, page, pageSize int, statusFilter string) (*adminEntities.ListPlanCompaniesResponse, error)

	// GetPlanFinancialReport gets financial report for a specific plan
	GetPlanFinancialReport(ctx context.Context, planID uuid.UUID, period string) (*adminEntities.PlanFinancialReport, error)

	// CanDeletePlan checks if a plan can be safely deleted
	CanDeletePlan(ctx context.Context, planID uuid.UUID) (*adminEntities.PlanDeletionCheck, error)

	// Bulk Operations
	BulkUpdate(ctx context.Context, planIDs []uuid.UUID, updates map[string]interface{}, userID, userEmail, reason string) (*adminEntities.BulkOperationResult, error)
	BulkActivate(ctx context.Context, planIDs []uuid.UUID, userID, userEmail, reason string) (*adminEntities.BulkOperationResult, error)
	BulkDeactivate(ctx context.Context, planIDs []uuid.UUID, userID, userEmail, reason string) (*adminEntities.BulkOperationResult, error)

	// Audit Operations
	CreateAuditEntry(ctx context.Context, entry *adminEntities.PlanAuditEntry) error
	GetPlanHistory(ctx context.Context, planID uuid.UUID, page, pageSize int) (*adminEntities.PlanAuditResponse, error)

	// Template Operations
	CreateTemplate(ctx context.Context, template *adminEntities.PlanTemplate) error
	GetTemplates(ctx context.Context, category string) ([]*adminEntities.PlanTemplate, error)
	GetTemplateByID(ctx context.Context, templateID uuid.UUID) (*adminEntities.PlanTemplate, error)
	CreatePlanFromTemplate(ctx context.Context, templateID uuid.UUID, customizations map[string]interface{}, userID, userEmail, reason string) (*entities.SubscriptionPlanEntity, error)
	IncrementTemplateUsage(ctx context.Context, templateID uuid.UUID) error

	// Feature Validation
	GetAvailableFeatures(ctx context.Context) ([]*adminEntities.AvailableFeature, error)
	ValidateFeatures(ctx context.Context, features []adminEntities.PlanTemplateFeature) (*adminEntities.FeatureValidationResult, error)

	// Plan Migration
	CreateMigration(ctx context.Context, request *adminEntities.PlanMigrationRequest, userID, userEmail string) (*adminEntities.PlanMigrationResult, error)
	ExecutePlanMigration(ctx context.Context, migrationID uuid.UUID) (*adminEntities.PlanMigrationResult, error)
	GetMigrationStatus(ctx context.Context, migrationID uuid.UUID) (*adminEntities.PlanMigrationResult, error)
}
