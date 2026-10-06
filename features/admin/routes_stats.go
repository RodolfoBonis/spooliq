package admin

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
)

// GetStats handles getting platform analytics
// @Summary Get platform stats
// @Description Gets platform analytics including MRR, subscription counts, and churn rate (PlatformAdmin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.AdminStats "Platform statistics"
// @Failure 401 {object} errors.HTTPError "Unauthorized"
// @Failure 403 {object} errors.HTTPError "Forbidden"
// @Failure 500 {object} errors.HTTPError "Internal server error"
// @Router /admin/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()

	// Get user roles from context
	userRoles := helpers.GetUserRoles(c)

	// Execute use case
	response, err := h.getStatsUC.Execute(ctx, userRoles)
	if err != nil {
		errors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}
