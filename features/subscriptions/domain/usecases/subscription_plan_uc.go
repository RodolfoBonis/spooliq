package usecases

import (
	"net/http"

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

	// Create plan
	plan := &entities.SubscriptionPlanEntity{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Cycle:       req.Cycle,
		Features:    req.Features,
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
	if req.Features != nil {
		plan.Features = *req.Features
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
