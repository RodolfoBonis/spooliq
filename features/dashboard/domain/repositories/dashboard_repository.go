package repositories

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
)

// DashboardRepository defines the interface for dashboard data access.
type DashboardRepository interface {
	GetOverview(organizationID string, start, end, prevStart, prevEnd time.Time) (*entities.OverviewResponse, error)
	GetRevenueTrend(organizationID string, start, end time.Time, truncate string) (*entities.RevenueTrendResponse, error)
	GetConversionFunnel(organizationID string, start, end time.Time) (*entities.ConversionFunnelResponse, error)
	GetTopCustomers(organizationID string, start, end time.Time, limit int) (*entities.TopCustomersResponse, error)
	GetOperationalInsights(organizationID string, start, end, prevStart, prevEnd time.Time) (*entities.OperationalInsightsResponse, error)
	GetTopFilaments(organizationID string, start, end time.Time, limit int) (*entities.TopFilamentsResponse, error)
	GetTopMaterials(organizationID string, start, end time.Time, limit int) (*entities.TopMaterialsResponse, error)
	// GetGoalsAlerts returns the current month's goals (user targets, progress,
	// month-end projection) and operational alerts.
	GetGoalsAlerts(organizationID string, now time.Time) (*entities.GoalsAlertsResponse, error)
	// GetGoalTargets returns the organization's configured monthly targets.
	GetGoalTargets(organizationID string) ([]entities.GoalTarget, error)
	// SaveGoalTargets upserts targets; a zero target removes the goal.
	SaveGoalTargets(organizationID, userID string, goals []entities.GoalTarget) error
	// GetInsightSignals loads the extra aggregates used by the insight rules.
	GetInsightSignals(organizationID string, start, end, now time.Time) (*entities.InsightSignals, error)
	// GetProfitability breaks the period's profit down by material, filament,
	// customer, machine and cost preset (sales dated by approval).
	GetProfitability(organizationID string, start, end time.Time, limit int) (*entities.ProfitabilityResponse, error)
	// GetResponseTimes reports how long customers take to decide on sent budgets.
	GetResponseTimes(organizationID string, start, end time.Time) (*entities.ResponseTimesResponse, error)
	// GetLowStockFilaments returns tracked filaments at or below their alert threshold,
	// ordered by how far below (stock - threshold) ascending, capped at limit.
	GetLowStockFilaments(organizationID string, limit int) ([]entities.LowStockFilament, error)
}
