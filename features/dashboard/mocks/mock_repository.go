package mocks

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
	"github.com/stretchr/testify/mock"
)

// MockDashboardRepository is a mock implementation of DashboardRepository
type MockDashboardRepository struct {
	mock.Mock
}

// NewMockDashboardRepository creates a new mock dashboard repository.
func NewMockDashboardRepository() *MockDashboardRepository {
	return &MockDashboardRepository{}
}

// GetOverview mocks the GetOverview method.
func (m *MockDashboardRepository) GetOverview(organizationID string, start, end, prevStart, prevEnd time.Time) (*entities.OverviewResponse, error) {
	args := m.Called(organizationID, start, end, prevStart, prevEnd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.OverviewResponse), args.Error(1)
}

// GetRevenueTrend mocks the GetRevenueTrend method.
func (m *MockDashboardRepository) GetRevenueTrend(organizationID string, start, end time.Time, truncate string) (*entities.RevenueTrendResponse, error) {
	args := m.Called(organizationID, start, end, truncate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.RevenueTrendResponse), args.Error(1)
}

// GetConversionFunnel mocks the GetConversionFunnel method.
func (m *MockDashboardRepository) GetConversionFunnel(organizationID string, start, end time.Time) (*entities.ConversionFunnelResponse, error) {
	args := m.Called(organizationID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.ConversionFunnelResponse), args.Error(1)
}

// GetTopCustomers mocks the GetTopCustomers method.
func (m *MockDashboardRepository) GetTopCustomers(organizationID string, start, end time.Time, limit int) (*entities.TopCustomersResponse, error) {
	args := m.Called(organizationID, start, end, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.TopCustomersResponse), args.Error(1)
}

// GetOperationalInsights mocks the GetOperationalInsights method.
func (m *MockDashboardRepository) GetOperationalInsights(organizationID string, start, end, prevStart, prevEnd time.Time) (*entities.OperationalInsightsResponse, error) {
	args := m.Called(organizationID, start, end, prevStart, prevEnd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.OperationalInsightsResponse), args.Error(1)
}

// GetTopFilaments mocks the GetTopFilaments method.
func (m *MockDashboardRepository) GetTopFilaments(organizationID string, start, end time.Time, limit int) (*entities.TopFilamentsResponse, error) {
	args := m.Called(organizationID, start, end, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.TopFilamentsResponse), args.Error(1)
}

// GetTopMaterials mocks the GetTopMaterials method.
func (m *MockDashboardRepository) GetTopMaterials(organizationID string, start, end time.Time, limit int) (*entities.TopMaterialsResponse, error) {
	args := m.Called(organizationID, start, end, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.TopMaterialsResponse), args.Error(1)
}

// GetGoalsAlerts mocks the GetGoalsAlerts method.
func (m *MockDashboardRepository) GetGoalsAlerts(organizationID string, now time.Time) (*entities.GoalsAlertsResponse, error) {
	args := m.Called(organizationID, now)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.GoalsAlertsResponse), args.Error(1)
}

// GetLowStockFilaments mocks the GetLowStockFilaments method.
func (m *MockDashboardRepository) GetLowStockFilaments(organizationID string, limit int) ([]entities.LowStockFilament, error) {
	args := m.Called(organizationID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entities.LowStockFilament), args.Error(1)
}

// GetProfitability mocks the GetProfitability method.
func (m *MockDashboardRepository) GetProfitability(organizationID string, start, end time.Time, limit int) (*entities.ProfitabilityResponse, error) {
	args := m.Called(organizationID, start, end, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.ProfitabilityResponse), args.Error(1)
}

// GetResponseTimes mocks the GetResponseTimes method.
func (m *MockDashboardRepository) GetResponseTimes(organizationID string, start, end time.Time) (*entities.ResponseTimesResponse, error) {
	args := m.Called(organizationID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.ResponseTimesResponse), args.Error(1)
}

// GetGoalTargets mocks the GetGoalTargets method.
func (m *MockDashboardRepository) GetGoalTargets(organizationID string) ([]entities.GoalTarget, error) {
	args := m.Called(organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entities.GoalTarget), args.Error(1)
}

// SaveGoalTargets mocks the SaveGoalTargets method.
func (m *MockDashboardRepository) SaveGoalTargets(organizationID, userID string, goals []entities.GoalTarget) error {
	return m.Called(organizationID, userID, goals).Error(0)
}

// GetInsightSignals mocks the GetInsightSignals method.
func (m *MockDashboardRepository) GetInsightSignals(organizationID string, start, end, now time.Time) (*entities.InsightSignals, error) {
	args := m.Called(organizationID, start, end, now)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.InsightSignals), args.Error(1)
}
