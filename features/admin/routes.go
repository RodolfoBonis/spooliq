package admin

import (
	"net/http"
	"strconv"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/roles"
	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/admin/domain/usecases"
	subscriptionUsecases "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/usecases"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for admin operations
type Handler struct {
	listCompaniesUC          *usecases.ListCompaniesUseCase
	getCompanyDetailsUC      *usecases.GetCompanyDetailsUseCase
	updateCompanyStatusUC    *usecases.UpdateCompanyStatusUseCase
	listSubscriptionsUC      *usecases.ListSubscriptionsUseCase
	getSubscriptionDetailsUC *usecases.GetSubscriptionDetailsUseCase
	getPaymentHistoryUC      *usecases.GetPaymentHistoryUseCase
	getStatsUC               *usecases.GetStatsUseCase
	subscriptionPlanUC       *subscriptionUsecases.SubscriptionPlanUseCase
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	listCompaniesUC *usecases.ListCompaniesUseCase,
	getCompanyDetailsUC *usecases.GetCompanyDetailsUseCase,
	updateCompanyStatusUC *usecases.UpdateCompanyStatusUseCase,
	listSubscriptionsUC *usecases.ListSubscriptionsUseCase,
	getSubscriptionDetailsUC *usecases.GetSubscriptionDetailsUseCase,
	getPaymentHistoryUC *usecases.GetPaymentHistoryUseCase,
	getStatsUC *usecases.GetStatsUseCase,
	subscriptionPlanUC *subscriptionUsecases.SubscriptionPlanUseCase,
) *Handler {
	return &Handler{
		listCompaniesUC:          listCompaniesUC,
		getCompanyDetailsUC:      getCompanyDetailsUC,
		updateCompanyStatusUC:    updateCompanyStatusUC,
		listSubscriptionsUC:      listSubscriptionsUC,
		getSubscriptionDetailsUC: getSubscriptionDetailsUC,
		getPaymentHistoryUC:      getPaymentHistoryUC,
		getStatsUC:               getStatsUC,
		subscriptionPlanUC:       subscriptionPlanUC,
	}
}

// SetupRoutes configures admin-related HTTP routes (all require PlatformAdmin role)
func SetupRoutes(route *gin.RouterGroup, handler *Handler, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	admin := route.Group("/admin")
	{
		// Company Management (PlatformAdmin only)
		companies := admin.Group("/companies")
		{
			companies.GET("", protectFactory(handler.ListCompanies, roles.PlatformAdminRole))
			companies.GET("/:organization_id", protectFactory(handler.GetCompanyDetails, roles.PlatformAdminRole))
			companies.PATCH("/:organization_id/status", protectFactory(handler.UpdateCompanyStatus, roles.PlatformAdminRole))
		}

		// Billing & Subscription Management (PlatformAdmin only)
		subscriptions := admin.Group("/subscriptions")
		{
			subscriptions.GET("", protectFactory(handler.ListSubscriptions, roles.PlatformAdminRole))
			subscriptions.GET("/:organization_id", protectFactory(handler.GetSubscriptionDetails, roles.PlatformAdminRole))
			subscriptions.GET("/:organization_id/payments", protectFactory(handler.GetPaymentHistory, roles.PlatformAdminRole))
		}

		// Subscription Plan Management (PlatformAdmin only)
		plansGroup := admin.Group("/subscription-plans")
		{
			plansGroup.POST("", protectFactory(handler.subscriptionPlanUC.CreatePlan, roles.PlatformAdminRole))
			plansGroup.GET("", protectFactory(handler.subscriptionPlanUC.ListAllPlans, roles.PlatformAdminRole))
			plansGroup.PUT("/:id", protectFactory(handler.subscriptionPlanUC.UpdatePlan, roles.PlatformAdminRole))
			plansGroup.DELETE("/:id", protectFactory(handler.subscriptionPlanUC.DeletePlan, roles.PlatformAdminRole))
		}

		// Platform Stats (PlatformAdmin only)
		admin.GET("/stats", protectFactory(handler.GetStats, roles.PlatformAdminRole))
	}
}

// ListCompanies handles listing all companies
// @Summary List all companies
// @Description Lists all companies in the system with pagination (PlatformAdmin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Param status query string false "Filter by subscription status"
// @Success 200 {object} adminEntities.ListCompaniesResponse "List of companies"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/admin/companies [get]
func (h *Handler) ListCompanies(c *gin.Context) {
	ctx := c.Request.Context()

	// Get user roles from context
	userRoles := helpers.GetUserRoles(c)

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	statusFilter := c.Query("status")

	// Execute use case
	response, err := h.listCompaniesUC.Execute(ctx, userRoles, page, pageSize, statusFilter)
	if err != nil {
		if appError, ok := err.(*errors.AppError); ok {
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		appError := errors.InternalServerError("Failed to list companies")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetCompanyDetails handles getting company details
// @Summary Get company details
// @Description Gets detailed information about a specific company (PlatformAdmin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param organization_id path string true "Organization ID (UUID)"
// @Success 200 {object} adminEntities.CompanyDetailsResponse "Company details"
// @Failure 400 {object} map[string]string "Invalid organization ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Company not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/admin/companies/{organization_id} [get]
func (h *Handler) GetCompanyDetails(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse organization ID from URL
	organizationIDStr := c.Param("organization_id")
	organizationID, err := uuid.Parse(organizationIDStr)
	if err != nil {
		appError := errors.BadRequestError("Invalid organization ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get user roles from context
	userRoles := helpers.GetUserRoles(c)

	// Execute use case
	response, err := h.getCompanyDetailsUC.Execute(ctx, userRoles, organizationID)
	if err != nil {
		if appError, ok := err.(*errors.AppError); ok {
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		appError := errors.InternalServerError("Failed to get company details")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateCompanyStatus handles updating company subscription status
// @Summary Update company subscription status
// @Description Manually updates a company's subscription status (PlatformAdmin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param organization_id path string true "Organization ID (UUID)"
// @Param request body adminEntities.UpdateStatusRequest true "Status update request"
// @Success 200 {object} adminEntities.CompanyDetailsResponse "Updated company details"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Company not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/admin/companies/{organization_id}/status [patch]
func (h *Handler) UpdateCompanyStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse organization ID from URL
	organizationIDStr := c.Param("organization_id")
	organizationID, err := uuid.Parse(organizationIDStr)
	if err != nil {
		appError := errors.BadRequestError("Invalid organization ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get user roles from context
	userRoles := helpers.GetUserRoles(c)

	// Parse request body
	var req adminEntities.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := errors.BadRequestError("Invalid request body")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Execute use case
	response, err := h.updateCompanyStatusUC.Execute(ctx, userRoles, organizationID, &req)
	if err != nil {
		if appError, ok := err.(*errors.AppError); ok {
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		appError := errors.InternalServerError("Failed to update company status")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ListSubscriptions handles listing all subscriptions
// @Summary List all subscriptions
// @Description Lists all subscriptions in the system with pagination (PlatformAdmin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Param status query string false "Filter by subscription status"
// @Success 200 {object} adminEntities.ListSubscriptionsResponse "List of subscriptions"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/admin/subscriptions [get]
func (h *Handler) ListSubscriptions(c *gin.Context) {
	ctx := c.Request.Context()

	// Get user roles from context
	userRoles := helpers.GetUserRoles(c)

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	statusFilter := c.Query("status")

	// Execute use case
	response, err := h.listSubscriptionsUC.Execute(ctx, userRoles, page, pageSize, statusFilter)
	if err != nil {
		if appError, ok := err.(*errors.AppError); ok {
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		appError := errors.InternalServerError("Failed to list subscriptions")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetSubscriptionDetails handles getting subscription details
// @Summary Get subscription details
// @Description Gets detailed information about a company's subscription (PlatformAdmin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param organization_id path string true "Organization ID (UUID)"
// @Success 200 {object} adminEntities.SubscriptionDetailsResponse "Subscription details"
// @Failure 400 {object} map[string]string "Invalid organization ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Company not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/admin/subscriptions/{organization_id} [get]
func (h *Handler) GetSubscriptionDetails(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse organization ID from URL
	organizationIDStr := c.Param("organization_id")
	organizationID, err := uuid.Parse(organizationIDStr)
	if err != nil {
		appError := errors.BadRequestError("Invalid organization ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get user roles from context
	userRoles := helpers.GetUserRoles(c)

	// Execute use case
	response, err := h.getSubscriptionDetailsUC.Execute(ctx, userRoles, organizationID)
	if err != nil {
		if appError, ok := err.(*errors.AppError); ok {
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		appError := errors.InternalServerError("Failed to get subscription details")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetPaymentHistory handles getting payment history
// @Summary Get payment history
// @Description Gets payment history for a specific company (PlatformAdmin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param organization_id path string true "Organization ID (UUID)"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Success 200 {object} adminEntities.PaymentHistoryResponse "Payment history"
// @Failure 400 {object} map[string]string "Invalid organization ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Company not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/admin/subscriptions/{organization_id}/payments [get]
func (h *Handler) GetPaymentHistory(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse organization ID from URL
	organizationIDStr := c.Param("organization_id")
	organizationID, err := uuid.Parse(organizationIDStr)
	if err != nil {
		appError := errors.BadRequestError("Invalid organization ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get user roles from context
	userRoles := helpers.GetUserRoles(c)

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// Execute use case
	response, err := h.getPaymentHistoryUC.Execute(ctx, userRoles, organizationID, page, pageSize)
	if err != nil {
		if appError, ok := err.(*errors.AppError); ok {
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		appError := errors.InternalServerError("Failed to get payment history")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	c.JSON(http.StatusOK, response)
}

