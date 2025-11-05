package usecases

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/logger"
	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PlanAdvancedUseCase handles advanced subscription plan operations
type PlanAdvancedUseCase struct {
	planRepo repositories.SubscriptionPlanRepository
	logger   logger.Logger
}

// NewPlanAdvancedUseCase creates a new instance of PlanAdvancedUseCase
func NewPlanAdvancedUseCase(
	planRepo repositories.SubscriptionPlanRepository,
	logger logger.Logger,
) *PlanAdvancedUseCase {
	return &PlanAdvancedUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

// GetPlanTemplates gets plan templates (admin only)
// @Summary Get plan templates
// @Description Get list of available plan templates (admin only)
// @Tags admin-plans
// @Produce json
// @Param category query string false "Filter by category"
// @Success 200 {array} adminEntities.PlanTemplate "Plan templates"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/templates [get]
func (uc *PlanAdvancedUseCase) GetPlanTemplates(c *gin.Context) {
	ctx := c.Request.Context()
	category := c.Query("category")

	templates, err := uc.planRepo.GetTemplates(ctx, category)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get plan templates", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get plan templates"})
		return
	}

	c.JSON(http.StatusOK, templates)
}

// CreatePlanFromTemplate creates a plan from template (admin only)
// @Summary Create plan from template
// @Description Create a new subscription plan from an existing template (admin only)
// @Tags admin-plans
// @Accept json
// @Produce json
// @Param request body adminEntities.CreateFromTemplateRequest true "Create from template request"
// @Success 201 {object} entities.SubscriptionPlanResponse "Created plan"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 404 {object} map[string]string "Template not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/from-template [post]
func (uc *PlanAdvancedUseCase) CreatePlanFromTemplate(c *gin.Context) {
	ctx := c.Request.Context()

	var req adminEntities.CreateFromTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get user info
	userID := helpers.GetUserID(c)
	userEmail := helpers.GetUserEmail(c)

	templateID, err := uuid.Parse(req.TemplateID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	plan, err := uc.planRepo.CreatePlanFromTemplate(ctx, templateID, req.Customizations, userID, userEmail, req.Reason)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create plan from template", map[string]interface{}{
			"error":       err.Error(),
			"template_id": req.TemplateID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plan from template"})
		return
	}

	response := toPlanResponse(plan)
	c.JSON(http.StatusCreated, response)
}

// GetAvailableFeatures gets available features for plans (admin only)
// @Summary Get available features
// @Description Get list of available features that can be added to plans (admin only)
// @Tags admin-features
// @Produce json
// @Success 200 {array} adminEntities.AvailableFeature "Available features"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/features/available [get]
func (uc *PlanAdvancedUseCase) GetAvailableFeatures(c *gin.Context) {
	ctx := c.Request.Context()

	features, err := uc.planRepo.GetAvailableFeatures(ctx)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get available features", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get available features"})
		return
	}

	c.JSON(http.StatusOK, features)
}

// ValidateFeatures validates if features are available and compatible (admin only)
// @Summary Validate features
// @Description Validate if the provided features are available and compatible (admin only)
// @Tags admin-features
// @Accept json
// @Produce json
// @Param request body adminEntities.FeatureValidationRequest true "Feature validation request"
// @Success 200 {object} adminEntities.FeatureValidationResult "Validation result"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/features/validate [post]
func (uc *PlanAdvancedUseCase) ValidateFeatures(c *gin.Context) {
	ctx := c.Request.Context()

	var req adminEntities.FeatureValidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	result, err := uc.planRepo.ValidateFeatures(ctx, req.Features)
	if err != nil {
		uc.logger.Error(ctx, "Failed to validate features", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate features"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreatePlanMigration creates a new plan migration (admin only)
// @Summary Create plan migration
// @Description Create a migration to move companies from one plan to another (admin only)
// @Tags admin-plans
// @Accept json
// @Produce json
// @Param request body adminEntities.PlanMigrationRequest true "Plan migration request"
// @Success 201 {object} adminEntities.PlanMigrationResult "Migration result"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/migrate [post]
func (uc *PlanAdvancedUseCase) CreatePlanMigration(c *gin.Context) {
	ctx := c.Request.Context()

	var req adminEntities.PlanMigrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get user info
	userID := helpers.GetUserID(c)
	userEmail := helpers.GetUserEmail(c)

	result, err := uc.planRepo.CreateMigration(ctx, &req, userID, userEmail)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create plan migration", map[string]interface{}{
			"error":        err.Error(),
			"from_plan_id": req.FromPlanID,
			"to_plan_id":   req.ToPlanID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plan migration"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetMigrationStatus gets migration status (admin only)
// @Summary Get migration status
// @Description Get the status of a plan migration (admin only)
// @Tags admin-plans
// @Produce json
// @Param migration_id path string true "Migration ID"
// @Success 200 {object} adminEntities.PlanMigrationResult "Migration status"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Migration not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/migrations/{migration_id} [get]
func (uc *PlanAdvancedUseCase) GetMigrationStatus(c *gin.Context) {
	ctx := c.Request.Context()
	migrationIDStr := c.Param("migration_id")

	migrationID, err := uuid.Parse(migrationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid migration ID"})
		return
	}

	result, err := uc.planRepo.GetMigrationStatus(ctx, migrationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get migration status", map[string]interface{}{
			"error":        err.Error(),
			"migration_id": migrationIDStr,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get migration status"})
		return
	}

	if result == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Migration not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutePlanMigration executes a scheduled plan migration (admin only)
// @Summary Execute plan migration
// @Description Execute a scheduled plan migration immediately (admin only)
// @Tags admin-plans
// @Param migration_id path string true "Migration ID"
// @Success 200 {object} adminEntities.PlanMigrationResult "Migration result"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Migration not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/migrations/{migration_id}/execute [post]
func (uc *PlanAdvancedUseCase) ExecutePlanMigration(c *gin.Context) {
	ctx := c.Request.Context()
	migrationIDStr := c.Param("migration_id")

	migrationID, err := uuid.Parse(migrationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid migration ID"})
		return
	}

	result, err := uc.planRepo.ExecutePlanMigration(ctx, migrationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to execute plan migration", map[string]interface{}{
			"error":        err.Error(),
			"migration_id": migrationIDStr,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute plan migration"})
		return
	}

	c.JSON(http.StatusOK, result)
}
