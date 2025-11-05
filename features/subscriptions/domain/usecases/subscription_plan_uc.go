package usecases

import (
	"net/http"
	"strconv"

	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SubscriptionPlanUseCase handles subscription plan operations
type SubscriptionPlanUseCase struct {
	planRepo repositories.SubscriptionPlanRepository
	logger   logger.Logger
}

// NewSubscriptionPlanUseCase creates a new instance of SubscriptionPlanUseCase
func NewSubscriptionPlanUseCase(
	planRepo repositories.SubscriptionPlanRepository,
	logger logger.Logger,
) *SubscriptionPlanUseCase {
	return &SubscriptionPlanUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

// CreatePlan creates a new subscription plan (admin only)
// @Summary Create subscription plan
// @Description Create a new subscription plan (admin only)
// @Tags admin-plans
// @Accept json
// @Produce json
// @Param request body entities.SubscriptionPlanCreateRequest true "Plan data"
// @Success 201 {object} entities.SubscriptionPlanResponse "Plan created"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 409 {object} map[string]string "Plan with this name already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/plans [post]
func (uc *SubscriptionPlanUseCase) CreatePlan(c *gin.Context) {
	ctx := c.Request.Context()

	var req entities.SubscriptionPlanCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Error(ctx, "Invalid plan request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Check if plan with this name already exists
	existing, err := uc.planRepo.FindByName(ctx, req.Name)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check existing plan", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing plan"})
		return
	}

	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Plan with this name already exists"})
		return
	}

	// Convert feature requests to entities
	features := make([]entities.PlanFeatureEntity, len(req.Features))
	for i, f := range req.Features {
		features[i] = entities.PlanFeatureEntity{
			Name:        f.Name,
			Description: f.Description,
			IsActive:    f.IsActive,
		}
	}

	// Create plan
	plan := &entities.SubscriptionPlanEntity{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Cycle:       req.Cycle,
		Features:    features,
		IsActive:    true,
	}

	if err := uc.planRepo.Create(ctx, plan); err != nil {
		uc.logger.Error(ctx, "Failed to create plan", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plan"})
		return
	}

	uc.logger.Info(ctx, "Subscription plan created", map[string]interface{}{
		"plan_id": plan.ID,
		"name":    plan.Name,
	})

	c.JSON(http.StatusCreated, toPlanResponse(plan))
}

// UpdatePlan updates a subscription plan (admin only)
// @Summary Update subscription plan
// @Description Update a subscription plan (admin only)
// @Tags admin-plans
// @Accept json
// @Produce json
// @Param id path string true "Plan ID"
// @Param request body entities.SubscriptionPlanUpdateRequest true "Plan data"
// @Success 200 {object} entities.SubscriptionPlanResponse "Plan updated"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 404 {object} map[string]string "Plan not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/plans/{id} [put]
func (uc *SubscriptionPlanUseCase) UpdatePlan(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}

	var req entities.SubscriptionPlanUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Error(ctx, "Invalid plan update request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Find plan
	plan, err := uc.planRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find plan", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find plan"})
		return
	}

	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	// Update fields
	if req.Name != nil {
		plan.Name = *req.Name
	}
	if req.Description != nil {
		plan.Description = *req.Description
	}
	if req.Price != nil {
		plan.Price = *req.Price
	}
	if req.Cycle != nil {
		plan.Cycle = *req.Cycle
	}
	if len(req.Features) > 0 {
		// Convert feature requests to entities
		features := make([]entities.PlanFeatureEntity, len(req.Features))
		for i, f := range req.Features {
			features[i] = entities.PlanFeatureEntity{
				Name:        f.Name,
				Description: f.Description,
				IsActive:    f.IsActive,
			}
		}
		plan.Features = features
	}
	if req.IsActive != nil {
		plan.IsActive = *req.IsActive
	}

	if err := uc.planRepo.Update(ctx, plan); err != nil {
		uc.logger.Error(ctx, "Failed to update plan", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update plan"})
		return
	}

	uc.logger.Info(ctx, "Subscription plan updated", map[string]interface{}{
		"plan_id": plan.ID,
	})

	c.JSON(http.StatusOK, toPlanResponse(plan))
}

// ListAllPlans lists all subscription plans (admin only)
// @Summary List all subscription plans
// @Description List all subscription plans including inactive (admin only)
// @Tags admin-plans
// @Produce json
// @Success 200 {array} entities.SubscriptionPlanResponse "Plans list"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/plans [get]
func (uc *SubscriptionPlanUseCase) ListAllPlans(c *gin.Context) {
	ctx := c.Request.Context()

	plans, err := uc.planRepo.FindAll(ctx)
	if err != nil {
		uc.logger.Error(ctx, "Failed to list plans", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list plans"})
		return
	}

	response := make([]entities.SubscriptionPlanResponse, len(plans))
	for i, plan := range plans {
		response[i] = *toPlanResponse(plan)
	}

	c.JSON(http.StatusOK, response)
}

// ListActivePlans lists active subscription plans (public)
// @Summary List active subscription plans
// @Description List all active subscription plans available for subscription
// @Tags plans
// @Produce json
// @Success 200 {array} entities.SubscriptionPlanResponse "Plans list"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/plans [get]
func (uc *SubscriptionPlanUseCase) ListActivePlans(c *gin.Context) {
	ctx := c.Request.Context()

	plans, err := uc.planRepo.FindAllActive(ctx)
	if err != nil {
		uc.logger.Error(ctx, "Failed to list active plans", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list plans"})
		return
	}

	response := make([]entities.SubscriptionPlanResponse, len(plans))
	for i, plan := range plans {
		response[i] = *toPlanResponse(plan)
	}

	c.JSON(http.StatusOK, response)
}

// GetPlanByID gets a specific subscription plan by ID (admin only)
// @Summary Get subscription plan by ID
// @Description Get a specific subscription plan by ID (admin only)
// @Tags admin-plans
// @Produce json
// @Param id path string true "Plan ID"
// @Success 200 {object} entities.SubscriptionPlanResponse "Plan details"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Plan not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/{id} [get]
func (uc *SubscriptionPlanUseCase) GetPlanByID(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}

	plan, err := uc.planRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find plan", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find plan"})
		return
	}

	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	c.JSON(http.StatusOK, toPlanResponse(plan))
}

// DeletePlan soft deletes a subscription plan (admin only)
// @Summary Delete subscription plan
// @Description Soft delete a subscription plan (admin only)
// @Tags admin-plans
// @Param id path string true "Plan ID"
// @Success 200 {object} map[string]string "Plan deleted"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Plan not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/plans/{id} [delete]
func (uc *SubscriptionPlanUseCase) DeletePlan(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}

	// Verify plan exists
	plan, err := uc.planRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find plan", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find plan"})
		return
	}

	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	// Delete
	if err := uc.planRepo.Delete(ctx, id); err != nil {
		uc.logger.Error(ctx, "Failed to delete plan", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete plan"})
		return
	}

	uc.logger.Info(ctx, "Subscription plan deleted", map[string]interface{}{
		"plan_id": id,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Plan deleted successfully"})
}

// GetPlanStats gets statistics for a specific subscription plan (admin only)
// @Summary Get subscription plan statistics
// @Description Get detailed statistics for a specific subscription plan (admin only)
// @Tags admin-plans
// @Produce json
// @Param id path string true "Plan ID"
// @Success 200 {object} entities.PlanStats "Plan statistics"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Plan not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/{id}/stats [get]
func (uc *SubscriptionPlanUseCase) GetPlanStats(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}

	// Verify plan exists first
	plan, err := uc.planRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find plan", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find plan"})
		return
	}

	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	stats, err := uc.planRepo.GetPlanStats(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get plan stats", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get plan statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetPlanCompanies gets companies using a specific subscription plan (admin only)
// @Summary Get companies using a subscription plan
// @Description Get paginated list of companies using a specific subscription plan (admin only)
// @Tags admin-plans
// @Produce json
// @Param id path string true "Plan ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Param status query string false "Filter by subscription status"
// @Success 200 {object} entities.ListPlanCompaniesResponse "Companies using the plan"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Plan not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/{id}/companies [get]
func (uc *SubscriptionPlanUseCase) GetPlanCompanies(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}

	// Parse query parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	statusFilter := c.Query("status")

	// Verify plan exists first
	plan, err := uc.planRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find plan", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find plan"})
		return
	}

	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	companies, err := uc.planRepo.GetPlanCompanies(ctx, id, page, pageSize, statusFilter)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get plan companies", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get companies"})
		return
	}

	c.JSON(http.StatusOK, companies)
}

// GetPlanFinancialReport gets financial report for a specific subscription plan (admin only)
// @Summary Get subscription plan financial report
// @Description Get detailed financial report for a specific subscription plan (admin only)
// @Tags admin-plans
// @Produce json
// @Param id path string true "Plan ID"
// @Param period query string false "Report period" default("monthly") Enums(monthly, quarterly, yearly)
// @Success 200 {object} entities.PlanFinancialReport "Plan financial report"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Plan not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/{id}/financial-report [get]
func (uc *SubscriptionPlanUseCase) GetPlanFinancialReport(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}

	period := c.DefaultQuery("period", "monthly")
	if period != "monthly" && period != "quarterly" && period != "yearly" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid period. Must be 'monthly', 'quarterly', or 'yearly'"})
		return
	}

	// Verify plan exists first
	plan, err := uc.planRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find plan", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find plan"})
		return
	}

	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	report, err := uc.planRepo.GetPlanFinancialReport(ctx, id, period)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get plan financial report", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get financial report"})
		return
	}

	c.JSON(http.StatusOK, report)
}

// CanDeletePlan checks if a subscription plan can be safely deleted (admin only)
// @Summary Check if subscription plan can be deleted
// @Description Check if a subscription plan can be safely deleted without affecting active subscriptions (admin only)
// @Tags admin-plans
// @Produce json
// @Param id path string true "Plan ID"
// @Success 200 {object} entities.PlanDeletionCheck "Deletion validation result"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Plan not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/{id}/can-delete [get]
func (uc *SubscriptionPlanUseCase) CanDeletePlan(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}

	// Verify plan exists first
	plan, err := uc.planRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find plan", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find plan"})
		return
	}

	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	check, err := uc.planRepo.CanDeletePlan(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check plan deletion", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check deletion eligibility"})
		return
	}

	c.JSON(http.StatusOK, check)
}

// BulkUpdatePlans handles bulk update of subscription plans (admin only)
// @Summary Bulk update subscription plans
// @Description Update multiple subscription plans at once (admin only)
// @Tags admin-plans
// @Accept json
// @Produce json
// @Param request body adminEntities.BulkUpdateRequest true "Bulk update request"
// @Success 200 {object} adminEntities.BulkOperationResult "Bulk operation result"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/bulk-update [post]
func (uc *SubscriptionPlanUseCase) BulkUpdatePlans(c *gin.Context) {
	ctx := c.Request.Context()

	var req adminEntities.BulkUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get user info
	userID := helpers.GetUserID(c)
	userEmail := helpers.GetUserEmail(c)

	// Convert string IDs to UUIDs
	planIDs := make([]uuid.UUID, len(req.PlanIDs))
	for i, id := range req.PlanIDs {
		planID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID: " + id})
			return
		}
		planIDs[i] = planID
	}

	result, err := uc.planRepo.BulkUpdate(ctx, planIDs, req.Updates, userID, userEmail, req.Reason)
	if err != nil {
		uc.logger.Error(ctx, "Failed to bulk update plans", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to bulk update plans"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// BulkActivatePlans handles bulk activation of subscription plans (admin only)
// @Summary Bulk activate subscription plans
// @Description Activate multiple subscription plans at once (admin only)
// @Tags admin-plans
// @Accept json
// @Produce json
// @Param request body adminEntities.BulkOperationRequest true "Bulk operation request"
// @Success 200 {object} adminEntities.BulkOperationResult "Bulk operation result"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/bulk-activate [put]
func (uc *SubscriptionPlanUseCase) BulkActivatePlans(c *gin.Context) {
	ctx := c.Request.Context()

	var req adminEntities.BulkOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get user info
	userID := helpers.GetUserID(c)
	userEmail := helpers.GetUserEmail(c)

	// Convert string IDs to UUIDs
	planIDs := make([]uuid.UUID, len(req.PlanIDs))
	for i, id := range req.PlanIDs {
		planID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID: " + id})
			return
		}
		planIDs[i] = planID
	}

	result, err := uc.planRepo.BulkActivate(ctx, planIDs, userID, userEmail, req.Reason)
	if err != nil {
		uc.logger.Error(ctx, "Failed to bulk activate plans", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to bulk activate plans"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// BulkDeactivatePlans handles bulk deactivation of subscription plans (admin only)
// @Summary Bulk deactivate subscription plans
// @Description Deactivate multiple subscription plans at once (admin only)
// @Tags admin-plans
// @Accept json
// @Produce json
// @Param request body adminEntities.BulkOperationRequest true "Bulk operation request"
// @Success 200 {object} adminEntities.BulkOperationResult "Bulk operation result"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/bulk-deactivate [put]
func (uc *SubscriptionPlanUseCase) BulkDeactivatePlans(c *gin.Context) {
	ctx := c.Request.Context()

	var req adminEntities.BulkOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get user info
	userID := helpers.GetUserID(c)
	userEmail := helpers.GetUserEmail(c)

	// Convert string IDs to UUIDs
	planIDs := make([]uuid.UUID, len(req.PlanIDs))
	for i, id := range req.PlanIDs {
		planID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID: " + id})
			return
		}
		planIDs[i] = planID
	}

	result, err := uc.planRepo.BulkDeactivate(ctx, planIDs, userID, userEmail, req.Reason)
	if err != nil {
		uc.logger.Error(ctx, "Failed to bulk deactivate plans", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to bulk deactivate plans"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetPlanHistory gets audit history for a subscription plan (admin only)
// @Summary Get subscription plan audit history
// @Description Get detailed audit history for a specific subscription plan (admin only)
// @Tags admin-plans
// @Produce json
// @Param id path string true "Plan ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Success 200 {object} adminEntities.PlanAuditResponse "Plan audit history"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Plan not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/admin/subscription-plans/{id}/history [get]
func (uc *SubscriptionPlanUseCase) GetPlanHistory(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}

	// Parse query parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Verify plan exists first
	plan, err := uc.planRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find plan", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find plan"})
		return
	}

	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	history, err := uc.planRepo.GetPlanHistory(ctx, id, page, pageSize)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get plan history", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get plan history"})
		return
	}

	c.JSON(http.StatusOK, history)
}

// Helper function
func toPlanResponse(plan *entities.SubscriptionPlanEntity) *entities.SubscriptionPlanResponse {
	return &entities.SubscriptionPlanResponse{
		ID:          plan.ID,
		Name:        plan.Name,
		Description: plan.Description,
		Price:       plan.Price,
		Cycle:       plan.Cycle,
		Features:    plan.Features,
		IsActive:    plan.IsActive,
		CreatedAt:   plan.CreatedAt,
		UpdatedAt:   plan.UpdatedAt,
	}
}
