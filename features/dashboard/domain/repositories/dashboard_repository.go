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
	GetGoalsAlerts(organizationID string) (*entities.GoalsAlertsResponse, error)
}
