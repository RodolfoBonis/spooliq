package dashboard

import (
	"net/http"
	"strconv"

	coreerrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/repositories"
	"github.com/gin-gonic/gin"
)

// maxDashboardLimit caps the number of rows any dashboard widget (recent-activity
// and the top-* endpoints) may request, protecting the database from unbounded
// scans driven by the client-supplied limit.
const maxDashboardLimit = 50

// Handler provides HTTP handlers for dashboard endpoints.
type Handler struct {
	repo            repositories.DashboardRepository
	activityService activityUc.IActivityService
}

// NewDashboardHandler creates a new dashboard handler.
func NewDashboardHandler(repo repositories.DashboardRepository, activityService activityUc.IActivityService) *Handler {
	return &Handler{
		repo:            repo,
		activityService: activityService,
	}
}

// requireOrganizationID extracts organization_id from context, writing a 400 and
// returning ok=false when it is missing.
func requireOrganizationID(c *gin.Context) (string, bool) {
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		coreerrors.Respond(c, coreerrors.BadRequest("organization_id_missing", "Organização não encontrada no contexto"))
		return "", false
	}
	return organizationID, true
}

// parseWidgetLimit reads the "limit" query param, defaulting to def and clamping
// the result to [1, maxDashboardLimit]. Widgets are not paginated lists, so they
// use this single-value cap instead of the standard pagination envelope.
func parseWidgetLimit(c *gin.Context, def int) int {
	limit := def
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > maxDashboardLimit {
		limit = maxDashboardLimit
	}
	if limit < 1 {
		limit = 1
	}
	return limit
}

// GetOverview godoc
// @Summary Get dashboard overview metrics
// @Description Returns key metrics including revenue, budgets, approval rate, profit margin, and new customers with period-over-period changes
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param period query string false "Period filter" Enums(7d, 30d, 3m, 6m, 1y, all) default(30d)
// @Success 200 {object} entities.OverviewResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Security BearerAuth
// @Router /dashboard/overview [get]
func (h *Handler) GetOverview(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	period := entities.ParsePeriod(c.Query("period"))
	start, end, prevStart, prevEnd := period.ToTimeRange()

	resp, err := h.repo.GetOverview(organizationID, start, end, prevStart, prevEnd)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	resp.Period = string(period)
	c.JSON(http.StatusOK, resp)
}

// GetRevenueTrend godoc
// @Summary Get revenue trend data
// @Description Returns revenue, cost, and profit data points over time for charting purposes
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param period query string false "Period filter" Enums(7d, 30d, 3m, 6m, 1y, all) default(30d)
// @Success 200 {object} entities.RevenueTrendResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Security BearerAuth
// @Router /dashboard/revenue-trend [get]
func (h *Handler) GetRevenueTrend(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	period := entities.ParsePeriod(c.Query("period"))
	start, end, _, _ := period.ToTimeRange()

	resp, err := h.repo.GetRevenueTrend(organizationID, start, end, period.TruncateFunc())
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	resp.Period = string(period)
	c.JSON(http.StatusOK, resp)
}

// GetConversionFunnel godoc
// @Summary Get budget conversion funnel
// @Description Returns the budget status funnel showing conversion rates between stages (draft, sent, approved, rejected, completed)
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param period query string false "Period filter" Enums(7d, 30d, 3m, 6m, 1y, all) default(30d)
// @Success 200 {object} entities.ConversionFunnelResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Security BearerAuth
// @Router /dashboard/conversion-funnel [get]
func (h *Handler) GetConversionFunnel(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	period := entities.ParsePeriod(c.Query("period"))
	start, end, _, _ := period.ToTimeRange()

	resp, err := h.repo.GetConversionFunnel(organizationID, start, end)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	resp.Period = string(period)
	c.JSON(http.StatusOK, resp)
}

// GetRecentActivity godoc
// @Summary Get recent activity
// @Description Returns the most recent activity items for the organization including budget changes, customer actions, and filament updates
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param limit query int false "Number of items to return (max 50)" default(20) minimum(1) maximum(50)
// @Success 200 {object} entities.RecentActivityResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Security BearerAuth
// @Router /dashboard/recent-activity [get]
func (h *Handler) GetRecentActivity(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	limit := parseWidgetLimit(c, 20)

	activities, err := h.activityService.FindRecentByOrganization(organizationID, limit)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	items := make([]entities.RecentActivityItem, 0, len(activities))
	for _, a := range activities {
		items = append(items, entities.RecentActivityItem{
			ID:          a.ID,
			UserID:      a.UserID,
			Action:      string(a.Action),
			EntityType:  string(a.EntityType),
			EntityID:    a.EntityID,
			EntityName:  a.EntityName,
			Description: a.Description,
			Metadata:    a.Metadata,
			CreatedAt:   a.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, entities.RecentActivityResponse{
		Activities: items,
		Total:      len(items),
	})
}

// GetTopCustomers godoc
// @Summary Get top customers by revenue
// @Description Returns the top customers ranked by total revenue for the selected period, including budget count and average ticket
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param period query string false "Period filter" Enums(7d, 30d, 3m, 6m, 1y, all) default(30d)
// @Param limit query int false "Number of customers to return (max 50)" default(5) minimum(1) maximum(50)
// @Success 200 {object} entities.TopCustomersResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Security BearerAuth
// @Router /dashboard/top-customers [get]
func (h *Handler) GetTopCustomers(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	period := entities.ParsePeriod(c.Query("period"))
	start, end, _, _ := period.ToTimeRange()

	limit := parseWidgetLimit(c, 5)

	resp, err := h.repo.GetTopCustomers(organizationID, start, end, limit)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	resp.Period = string(period)
	c.JSON(http.StatusOK, resp)
}

// GetOperationalInsights godoc
// @Summary Get operational insights and metrics
// @Description Returns operational metrics including average ticket, profit margin, print time, rejection rate, and cost breakdown by category
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param period query string false "Period filter" Enums(7d, 30d, 3m, 6m, 1y, all) default(30d)
// @Success 200 {object} entities.OperationalInsightsResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Security BearerAuth
// @Router /dashboard/operational-insights [get]
func (h *Handler) GetOperationalInsights(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	period := entities.ParsePeriod(c.Query("period"))
	start, end, prevStart, prevEnd := period.ToTimeRange()

	resp, err := h.repo.GetOperationalInsights(organizationID, start, end, prevStart, prevEnd)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	resp.Period = string(period)
	c.JSON(http.StatusOK, resp)
}

// GetTopFilaments godoc
// @Summary Get top filaments by usage
// @Description Returns the most used filaments ranked by total grams consumed, including brand, material, and color information
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param period query string false "Period filter" Enums(7d, 30d, 3m, 6m, 1y, all) default(30d)
// @Param limit query int false "Number of filaments to return (max 50)" default(5) minimum(1) maximum(50)
// @Success 200 {object} entities.TopFilamentsResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Security BearerAuth
// @Router /dashboard/top-filaments [get]
func (h *Handler) GetTopFilaments(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	period := entities.ParsePeriod(c.Query("period"))
	start, end, _, _ := period.ToTimeRange()

	limit := parseWidgetLimit(c, 5)

	resp, err := h.repo.GetTopFilaments(organizationID, start, end, limit)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	resp.Period = string(period)
	c.JSON(http.StatusOK, resp)
}

// GetTopMaterials godoc
// @Summary Get top materials by usage
// @Description Returns the most used materials ranked by total grams consumed across all filaments
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param period query string false "Period filter" Enums(7d, 30d, 3m, 6m, 1y, all) default(30d)
// @Param limit query int false "Number of materials to return (max 50)" default(5) minimum(1) maximum(50)
// @Success 200 {object} entities.TopMaterialsResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Security BearerAuth
// @Router /dashboard/top-materials [get]
func (h *Handler) GetTopMaterials(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	period := entities.ParsePeriod(c.Query("period"))
	start, end, _, _ := period.ToTimeRange()

	limit := parseWidgetLimit(c, 5)

	resp, err := h.repo.GetTopMaterials(organizationID, start, end, limit)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	resp.Period = string(period)
	c.JSON(http.StatusOK, resp)
}

// GetGoalsAlerts godoc
// @Summary Get goals progress and alerts
// @Description Returns monthly goals progress (revenue, budgets, profit margin) and active alerts for low stock, expiring budgets, and pending approvals
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {object} entities.GoalsAlertsResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Security BearerAuth
// @Router /dashboard/goals-alerts [get]
func (h *Handler) GetGoalsAlerts(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	resp, err := h.repo.GetGoalsAlerts(organizationID)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// lowStockLimit caps the low-stock widget at the 20 most depleted filaments.
const lowStockLimit = 20

// GetLowStock godoc
// @Summary Get low-stock filaments
// @Description Returns tracked filaments at or below their alert threshold, ordered by shortfall (stock - threshold) ascending, limited to 20.
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {object} entities.LowStockResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Security BearerAuth
// @Router /dashboard/low-stock [get]
func (h *Handler) GetLowStock(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	rows, err := h.repo.GetLowStockFilaments(organizationID, lowStockLimit)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}
	if rows == nil {
		rows = []entities.LowStockFilament{}
	}

	c.JSON(http.StatusOK, entities.LowStockResponse{Data: rows})
}
