package usecases

import (
	"net/http"
	"strconv"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetPaymentHistoryUseCase handles retrieving payment history for a company
type GetPaymentHistoryUseCase struct {
	repository repositories.SubscriptionRepository
	logger     logger.Logger
}

// NewGetPaymentHistoryUseCase creates a new instance of GetPaymentHistoryUseCase
func NewGetPaymentHistoryUseCase(
	repository repositories.SubscriptionRepository,
	logger logger.Logger,
) *GetPaymentHistoryUseCase {
	return &GetPaymentHistoryUseCase{
		repository: repository,
		logger:     logger,
	}
}

// Execute retrieves the payment history for the current organization
func (uc *GetPaymentHistoryUseCase) Execute(c *gin.Context) {
	ctx := c.Request.Context()

	// Get organization ID from context (set by auth middleware)
	organizationIDStr, exists := c.Get("organization_id")
	if !exists {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found"})
		return
	}

	organizationID, err := uuid.Parse(organizationIDStr.(string))
	if err != nil {
		uc.logger.Error(ctx, "Invalid organization ID", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	uc.logger.Info(ctx, "Retrieving payment history", map[string]interface{}{
		"organization_id": organizationID.String(),
		"page":            page,
		"page_size":       pageSize,
	})

	// Get payment history from repository
	payments, err := uc.repository.FindAll(ctx, organizationID, pageSize, offset)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve payment history", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID.String(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve payment history"})
		return
	}

	// Get total count for pagination
	total, err := uc.repository.CountByOrganizationID(ctx, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to count payments", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID.String(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count payments"})
		return
	}

	uc.logger.Info(ctx, "Payment history retrieved successfully", map[string]interface{}{
		"organization_id": organizationID.String(),
		"total_payments":  total,
		"page":            page,
	})

	// Return paginated response
	c.JSON(http.StatusOK, gin.H{
		"payments":  payments,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
