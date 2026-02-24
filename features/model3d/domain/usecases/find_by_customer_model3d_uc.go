package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FindByCustomer handles listing all 3D models for a specific customer.
// @Summary Get 3D Models by Customer
// Schemes
// @Description List all 3D models associated with a specific customer
// @Tags 3D Models
// @Accept json
// @Produce json
// @Param customer_id path string true "Customer ID" format(uuid)
// @Success 200 {object} entities.FindAllModel3DResponse "Successfully retrieved 3D models"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /models3d/by-customer/{customer_id} [get]
// @Security Bearer
func (uc *Model3DUseCase) FindByCustomer(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}

	customerIDParam := c.Param("customer_id")
	customerID, err := uuid.Parse(customerIDParam)
	if err != nil {
		uc.logger.Error(ctx, "Invalid customer ID", map[string]interface{}{
			"customer_id": customerIDParam,
			"error":       err.Error(),
		})
		appError := coreErrors.UsecaseError("Invalid customer ID format")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	models, err := uc.repository.FindByCustomerID(customerID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to list 3D models by customer", map[string]interface{}{
			"customer_id": customerID,
			"error":       err.Error(),
		})
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	c.JSON(http.StatusOK, models)
}
