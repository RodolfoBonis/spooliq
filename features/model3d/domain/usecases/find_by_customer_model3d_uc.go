package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FindByCustomer lists all 3D models for a specific customer as a FLAT array. The
// web app relies on this un-paginated shape, so it is intentionally not wrapped in
// a Page envelope. It remains organization-scoped.
// @Summary Get 3D Models by Customer
// @Description List all 3D models associated with a specific customer (flat array).
// @Tags models3d
// @Accept json
// @Produce json
// @Param customer_id path string true "Customer ID" format(uuid)
// @Success 200 {array} entities.Model3DEntity "3D models for the customer"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /models3d/by-customer/{customer_id} [get]
// @Security BearerAuth
func (uc *Model3DUseCase) FindByCustomer(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, errOrganizationRequired())
		return
	}

	customerID, err := uuid.Parse(c.Param("customer_id"))
	if err != nil {
		coreErrors.Respond(c, errInvalidCustomerID())
		return
	}

	models, err := uc.repository.FindByCustomerID(ctx, customerID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to list 3D models by customer", map[string]interface{}{"customer_id": customerID, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, models)
}
