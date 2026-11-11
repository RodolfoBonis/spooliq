package dashboard

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/dashboard/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const testOrganizationID = "test-org-123"

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func setupTestContext(router *gin.Engine, method, path string) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, nil)
	c.Request = req
	return w, c
}

func setOrganizationID(c *gin.Context, orgID string) {
	c.Set("organization_id", orgID)
}

// ============================================================================
// GetOverview Tests
// ============================================================================

func TestGetOverview_Success(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.OverviewResponse{
		TotalRevenue:       100000,
		RevenueChange:      15.5,
		TotalBudgets:       50,
		BudgetsChange:      10.0,
		AvgTicket:          2000,
		AvgTicketChange:    5.0,
		ApprovalRate:       75.0,
		ApprovalRateChange: 2.0,
		AvgProfitMargin:    25.0,
		ProfitMarginChange: 1.5,
		NewCustomers:       10,
		NewCustomersChange: 20.0,
		BudgetsByStatus: []entities.BudgetStatusCount{
			{Status: "approved", Count: 30},
			{Status: "draft", Count: 20},
		},
	}

	mockRepo.On("GetOverview", testOrganizationID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/overview?period=30d")
	setOrganizationID(c, testOrganizationID)

	handler.GetOverview(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.OverviewResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, int64(100000), response.TotalRevenue)
	assert.Equal(t, "30d", response.Period)
	mockRepo.AssertExpectations(t)
}

func TestGetOverview_MissingOrganizationID(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/overview")
	// Not setting organization ID

	handler.GetOverview(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "organization_id_missing", response["code"])
}

// Top widgets clamp the client-supplied limit to maxDashboardLimit (50).
func TestGetTopCustomers_LimitClampedToMax(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	// The repository must be invoked with 50, never the requested 1000.
	mockRepo.On("GetTopCustomers", testOrganizationID, mock.Anything, mock.Anything, 50).
		Return(&entities.TopCustomersResponse{}, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-customers?limit=1000")
	c.Request.URL.RawQuery = "limit=1000"
	setOrganizationID(c, testOrganizationID)

	handler.GetTopCustomers(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertCalled(t, "GetTopCustomers", testOrganizationID, mock.Anything, mock.Anything, 50)
}

// Top filaments clamp too.
func TestGetTopFilaments_LimitClampedToMax(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	mockRepo.On("GetTopFilaments", testOrganizationID, mock.Anything, mock.Anything, 50).
		Return(&entities.TopFilamentsResponse{}, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-filaments?limit=9999")
	c.Request.URL.RawQuery = "limit=9999"
	setOrganizationID(c, testOrganizationID)

	handler.GetTopFilaments(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertCalled(t, "GetTopFilaments", testOrganizationID, mock.Anything, mock.Anything, 50)
}

func TestGetOverview_RepositoryError(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	mockRepo.On("GetOverview", testOrganizationID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("database error"))

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/overview")
	setOrganizationID(c, testOrganizationID)

	handler.GetOverview(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetOverview_EmptyData(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.OverviewResponse{
		TotalRevenue:       0,
		RevenueChange:      0,
		TotalBudgets:       0,
		BudgetsChange:      0,
		AvgTicket:          0,
		AvgTicketChange:    0,
		ApprovalRate:       0,
		ApprovalRateChange: 0,
		AvgProfitMargin:    0,
		ProfitMarginChange: 0,
		NewCustomers:       0,
		NewCustomersChange: 0,
		BudgetsByStatus:    []entities.BudgetStatusCount{},
	}

	mockRepo.On("GetOverview", testOrganizationID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/overview")
	setOrganizationID(c, testOrganizationID)

	handler.GetOverview(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.OverviewResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), response.TotalRevenue)
	assert.Equal(t, float64(0), response.ApprovalRate)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetRevenueTrend Tests
// ============================================================================

func TestGetRevenueTrend_Success(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.RevenueTrendResponse{
		Points: []entities.RevenueTrendPoint{
			{Date: "2024-01-01", Revenue: 10000, Cost: 7500, Profit: 2500, BudgetCount: 5},
			{Date: "2024-01-02", Revenue: 15000, Cost: 11000, Profit: 4000, BudgetCount: 8},
		},
	}

	mockRepo.On("GetRevenueTrend", testOrganizationID, mock.Anything, mock.Anything, "day").
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/revenue-trend?period=30d")
	setOrganizationID(c, testOrganizationID)

	handler.GetRevenueTrend(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.RevenueTrendResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Points, 2)
	assert.Equal(t, "30d", response.Period)
	mockRepo.AssertExpectations(t)
}

func TestGetRevenueTrend_MissingOrganizationID(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/revenue-trend")

	handler.GetRevenueTrend(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetRevenueTrend_RepositoryError(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	mockRepo.On("GetRevenueTrend", testOrganizationID, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("database error"))

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/revenue-trend")
	setOrganizationID(c, testOrganizationID)

	handler.GetRevenueTrend(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetRevenueTrend_EmptyData(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.RevenueTrendResponse{
		Points: []entities.RevenueTrendPoint{},
	}

	mockRepo.On("GetRevenueTrend", testOrganizationID, mock.Anything, mock.Anything, mock.Anything).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/revenue-trend")
	setOrganizationID(c, testOrganizationID)

	handler.GetRevenueTrend(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.RevenueTrendResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Points, 0)
	mockRepo.AssertExpectations(t)
}

func TestGetRevenueTrend_MonthlyTruncation(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.RevenueTrendResponse{
		Points: []entities.RevenueTrendPoint{
			{Date: "2024-01", Revenue: 100000, Cost: 75000, Profit: 25000, BudgetCount: 50},
		},
	}

	mockRepo.On("GetRevenueTrend", testOrganizationID, mock.Anything, mock.Anything, "month").
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/revenue-trend?period=3m")
	setOrganizationID(c, testOrganizationID)

	handler.GetRevenueTrend(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.RevenueTrendResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "3m", response.Period)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetConversionFunnel Tests
// ============================================================================

func TestGetConversionFunnel_Success(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.ConversionFunnelResponse{
		Steps: []entities.FunnelStep{
			{Status: "draft", Count: 100, ConversionRate: 40.0},
			{Status: "sent", Count: 80, ConversionRate: 32.0},
			{Status: "approved", Count: 60, ConversionRate: 24.0},
			{Status: "completed", Count: 50, ConversionRate: 20.0},
		},
		TotalBudgets:      250,
		OverallConversion: 20.0,
	}

	mockRepo.On("GetConversionFunnel", testOrganizationID, mock.Anything, mock.Anything).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/conversion-funnel")
	setOrganizationID(c, testOrganizationID)

	handler.GetConversionFunnel(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.ConversionFunnelResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Steps, 4)
	assert.Equal(t, 250, response.TotalBudgets)
	mockRepo.AssertExpectations(t)
}

func TestGetConversionFunnel_MissingOrganizationID(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/conversion-funnel")

	handler.GetConversionFunnel(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetConversionFunnel_RepositoryError(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	mockRepo.On("GetConversionFunnel", testOrganizationID, mock.Anything, mock.Anything).
		Return(nil, errors.New("database error"))

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/conversion-funnel")
	setOrganizationID(c, testOrganizationID)

	handler.GetConversionFunnel(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetConversionFunnel_ZeroBudgets(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.ConversionFunnelResponse{
		Steps:             []entities.FunnelStep{},
		TotalBudgets:      0,
		OverallConversion: 0,
	}

	mockRepo.On("GetConversionFunnel", testOrganizationID, mock.Anything, mock.Anything).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/conversion-funnel")
	setOrganizationID(c, testOrganizationID)

	handler.GetConversionFunnel(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.ConversionFunnelResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.TotalBudgets)
	assert.Equal(t, float64(0), response.OverallConversion)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetTopCustomers Tests
// ============================================================================

func TestGetTopCustomers_Success(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.TopCustomersResponse{
		Customers: []entities.TopCustomer{
			{ID: "1", Name: "Customer A", Email: "a@test.com", TotalRevenue: 50000, BudgetCount: 10, AvgTicket: 5000},
			{ID: "2", Name: "Customer B", Email: "b@test.com", TotalRevenue: 30000, BudgetCount: 6, AvgTicket: 5000},
		},
	}

	mockRepo.On("GetTopCustomers", testOrganizationID, mock.Anything, mock.Anything, 5).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-customers")
	setOrganizationID(c, testOrganizationID)

	handler.GetTopCustomers(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.TopCustomersResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Customers, 2)
	mockRepo.AssertExpectations(t)
}

func TestGetTopCustomers_CustomLimit(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.TopCustomersResponse{
		Customers: []entities.TopCustomer{
			{ID: "1", Name: "Customer A", TotalRevenue: 50000},
		},
	}

	mockRepo.On("GetTopCustomers", testOrganizationID, mock.Anything, mock.Anything, 10).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-customers?limit=10")
	c.Request.URL.RawQuery = "limit=10"
	setOrganizationID(c, testOrganizationID)

	handler.GetTopCustomers(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetTopCustomers_MissingOrganizationID(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-customers")

	handler.GetTopCustomers(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetTopCustomers_RepositoryError(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	mockRepo.On("GetTopCustomers", testOrganizationID, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("database error"))

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-customers")
	setOrganizationID(c, testOrganizationID)

	handler.GetTopCustomers(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetTopCustomers_EmptyResult(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.TopCustomersResponse{
		Customers: []entities.TopCustomer{},
	}

	mockRepo.On("GetTopCustomers", testOrganizationID, mock.Anything, mock.Anything, mock.Anything).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-customers")
	setOrganizationID(c, testOrganizationID)

	handler.GetTopCustomers(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.TopCustomersResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Customers, 0)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetOperationalInsights Tests
// ============================================================================

func TestGetOperationalInsights_Success(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.OperationalInsightsResponse{
		AvgTicket:           2500,
		AvgTicketChange:     5.5,
		AvgProfitMargin:     30.0,
		ProfitMarginChange:  2.0,
		TotalPrintTimeHours: 150.5,
		PrintTimeChange:     10.0,
		RejectionRate:       12.0,
		RejectionRateChange: -3.0,
		CostBreakdown: entities.CostBreakdown{
			FilamentPct: 40.0,
			WastePct:    5.0,
			EnergyPct:   10.0,
			SetupPct:    5.0,
			LaborPct:    25.0,
			OverheadPct: 15.0,
		},
	}

	mockRepo.On("GetOperationalInsights", testOrganizationID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/operational-insights")
	setOrganizationID(c, testOrganizationID)

	handler.GetOperationalInsights(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.OperationalInsightsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, int64(2500), response.AvgTicket)
	assert.Equal(t, 40.0, response.CostBreakdown.FilamentPct)
	mockRepo.AssertExpectations(t)
}

func TestGetOperationalInsights_MissingOrganizationID(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/operational-insights")

	handler.GetOperationalInsights(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetOperationalInsights_RepositoryError(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	mockRepo.On("GetOperationalInsights", testOrganizationID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("database error"))

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/operational-insights")
	setOrganizationID(c, testOrganizationID)

	handler.GetOperationalInsights(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetOperationalInsights_ZeroCosts(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.OperationalInsightsResponse{
		AvgTicket:           0,
		AvgTicketChange:     0,
		AvgProfitMargin:     0,
		ProfitMarginChange:  0,
		TotalPrintTimeHours: 0,
		PrintTimeChange:     0,
		RejectionRate:       0,
		RejectionRateChange: 0,
		CostBreakdown: entities.CostBreakdown{
			FilamentPct: 0,
			WastePct:    0,
			EnergyPct:   0,
			SetupPct:    0,
			LaborPct:    0,
			OverheadPct: 0,
		},
	}

	mockRepo.On("GetOperationalInsights", testOrganizationID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/operational-insights")
	setOrganizationID(c, testOrganizationID)

	handler.GetOperationalInsights(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.OperationalInsightsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), response.AvgTicket)
	assert.Equal(t, float64(0), response.CostBreakdown.FilamentPct)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetTopFilaments Tests
// ============================================================================

func TestGetTopFilaments_Success(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.TopFilamentsResponse{
		Filaments: []entities.TopFilament{
			{ID: "1", Name: "PLA White", BrandName: "Brand A", MaterialName: "PLA", ColorHex: "#FFFFFF", TotalGrams: 5000.0, UsageCount: 50},
			{ID: "2", Name: "PETG Black", BrandName: "Brand B", MaterialName: "PETG", ColorHex: "#000000", TotalGrams: 3000.0, UsageCount: 30},
		},
	}

	mockRepo.On("GetTopFilaments", testOrganizationID, mock.Anything, mock.Anything, 5).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-filaments")
	setOrganizationID(c, testOrganizationID)

	handler.GetTopFilaments(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.TopFilamentsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Filaments, 2)
	mockRepo.AssertExpectations(t)
}

func TestGetTopFilaments_CustomLimit(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.TopFilamentsResponse{
		Filaments: []entities.TopFilament{},
	}

	mockRepo.On("GetTopFilaments", testOrganizationID, mock.Anything, mock.Anything, 10).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-filaments?limit=10")
	c.Request.URL.RawQuery = "limit=10"
	setOrganizationID(c, testOrganizationID)

	handler.GetTopFilaments(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetTopFilaments_MissingOrganizationID(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-filaments")

	handler.GetTopFilaments(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetTopFilaments_RepositoryError(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	mockRepo.On("GetTopFilaments", testOrganizationID, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("database error"))

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-filaments")
	setOrganizationID(c, testOrganizationID)

	handler.GetTopFilaments(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetTopFilaments_EmptyResult(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.TopFilamentsResponse{
		Filaments: []entities.TopFilament{},
	}

	mockRepo.On("GetTopFilaments", testOrganizationID, mock.Anything, mock.Anything, mock.Anything).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-filaments")
	setOrganizationID(c, testOrganizationID)

	handler.GetTopFilaments(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.TopFilamentsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Filaments, 0)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetTopMaterials Tests
// ============================================================================

func TestGetTopMaterials_Success(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.TopMaterialsResponse{
		Materials: []entities.TopMaterial{
			{ID: "1", Name: "PLA", TotalGrams: 10000.0, UsageCount: 100},
			{ID: "2", Name: "PETG", TotalGrams: 5000.0, UsageCount: 50},
		},
	}

	mockRepo.On("GetTopMaterials", testOrganizationID, mock.Anything, mock.Anything, 5).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-materials")
	setOrganizationID(c, testOrganizationID)

	handler.GetTopMaterials(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.TopMaterialsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Materials, 2)
	mockRepo.AssertExpectations(t)
}

func TestGetTopMaterials_CustomLimit(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.TopMaterialsResponse{
		Materials: []entities.TopMaterial{},
	}

	mockRepo.On("GetTopMaterials", testOrganizationID, mock.Anything, mock.Anything, 10).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-materials?limit=10")
	c.Request.URL.RawQuery = "limit=10"
	setOrganizationID(c, testOrganizationID)

	handler.GetTopMaterials(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetTopMaterials_MissingOrganizationID(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-materials")

	handler.GetTopMaterials(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetTopMaterials_RepositoryError(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	mockRepo.On("GetTopMaterials", testOrganizationID, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("database error"))

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-materials")
	setOrganizationID(c, testOrganizationID)

	handler.GetTopMaterials(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetTopMaterials_EmptyResult(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.TopMaterialsResponse{
		Materials: []entities.TopMaterial{},
	}

	mockRepo.On("GetTopMaterials", testOrganizationID, mock.Anything, mock.Anything, mock.Anything).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/top-materials")
	setOrganizationID(c, testOrganizationID)

	handler.GetTopMaterials(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.TopMaterialsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Materials, 0)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetGoalsAlerts Tests
// ============================================================================

func TestGetGoalsAlerts_Success(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.GoalsAlertsResponse{
		Goals: []entities.Goal{
			{Name: "Monthly Revenue", Current: 80000, Target: 100000, Progress: 80.0, Unit: "cents"},
			{Name: "Monthly Budgets", Current: 40, Target: 50, Progress: 80.0, Unit: "count"},
		},
		Alerts: []entities.Alert{
			{Type: "stale_drafts", Severity: "warning", Message: "5 budget(s) in draft status", Count: 5, EntityType: "budget"},
		},
	}

	mockRepo.On("GetGoalsAlerts", testOrganizationID, mock.Anything).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/goals-alerts")
	setOrganizationID(c, testOrganizationID)

	handler.GetGoalsAlerts(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.GoalsAlertsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Goals, 2)
	assert.Len(t, response.Alerts, 1)
	mockRepo.AssertExpectations(t)
}

func TestGetGoalsAlerts_MissingOrganizationID(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/goals-alerts")

	handler.GetGoalsAlerts(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetGoalsAlerts_RepositoryError(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	mockRepo.On("GetGoalsAlerts", testOrganizationID, mock.Anything).
		Return(nil, errors.New("database error"))

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/goals-alerts")
	setOrganizationID(c, testOrganizationID)

	handler.GetGoalsAlerts(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetGoalsAlerts_NoAlertsNoGoals(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedResponse := &entities.GoalsAlertsResponse{
		Goals:  []entities.Goal{},
		Alerts: []entities.Alert{},
	}

	mockRepo.On("GetGoalsAlerts", testOrganizationID, mock.Anything).
		Return(expectedResponse, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/goals-alerts")
	setOrganizationID(c, testOrganizationID)

	handler.GetGoalsAlerts(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.GoalsAlertsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Goals, 0)
	assert.Len(t, response.Alerts, 0)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetRecentActivity Tests
// ============================================================================

func TestGetRecentActivity_Success(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	expectedActivities := []activityEntities.ActivityEntity{
		{
			ID:          "act-1",
			UserID:      "user-1",
			Action:      "created",
			EntityType:  "budget",
			EntityID:    "budget-1",
			EntityName:  "Budget A",
			Description: "Created new budget",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "act-2",
			UserID:      "user-1",
			Action:      "updated",
			EntityType:  "customer",
			EntityID:    "customer-1",
			EntityName:  "Customer X",
			Description: "Updated customer info",
			CreatedAt:   time.Now(),
		},
	}

	mockActivitySvc.On("FindRecentByOrganization", testOrganizationID, 20).
		Return(expectedActivities, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/recent-activity")
	setOrganizationID(c, testOrganizationID)

	handler.GetRecentActivity(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.RecentActivityResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Activities, 2)
	assert.Equal(t, 2, response.Total)
	mockActivitySvc.AssertExpectations(t)
}

func TestGetRecentActivity_CustomLimit(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	mockActivitySvc.On("FindRecentByOrganization", testOrganizationID, 10).
		Return([]activityEntities.ActivityEntity{}, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/recent-activity?limit=10")
	c.Request.URL.RawQuery = "limit=10"
	setOrganizationID(c, testOrganizationID)

	handler.GetRecentActivity(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockActivitySvc.AssertExpectations(t)
}

func TestGetRecentActivity_LimitCappedAt50(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	mockActivitySvc.On("FindRecentByOrganization", testOrganizationID, 50).
		Return([]activityEntities.ActivityEntity{}, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/recent-activity?limit=100")
	c.Request.URL.RawQuery = "limit=100"
	setOrganizationID(c, testOrganizationID)

	handler.GetRecentActivity(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockActivitySvc.AssertExpectations(t)
}

func TestGetRecentActivity_MissingOrganizationID(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/recent-activity")

	handler.GetRecentActivity(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetRecentActivity_ServiceError(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	mockActivitySvc.On("FindRecentByOrganization", testOrganizationID, 20).
		Return(nil, errors.New("database error"))

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/recent-activity")
	setOrganizationID(c, testOrganizationID)

	handler.GetRecentActivity(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockActivitySvc.AssertExpectations(t)
}

func TestGetRecentActivity_EmptyResult(t *testing.T) {
	mockRepo := mocks.NewMockDashboardRepository()
	mockActivitySvc := mocks.NewMockActivityService()
	handler := NewDashboardHandler(mockRepo, mockActivitySvc, noopLogger{})

	mockActivitySvc.On("FindRecentByOrganization", testOrganizationID, 20).
		Return([]activityEntities.ActivityEntity{}, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/recent-activity")
	setOrganizationID(c, testOrganizationID)

	handler.GetRecentActivity(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response entities.RecentActivityResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Activities, 0)
	assert.Equal(t, 0, response.Total)
	mockActivitySvc.AssertExpectations(t)
}
