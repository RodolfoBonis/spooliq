package dashboard

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/dashboard/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSaveGoals_UpsertsAndReturnsTargets(t *testing.T) {
	repo := mocks.NewMockDashboardRepository()
	h := NewDashboardHandler(repo, mocks.NewMockActivityService(), noopLogger{})
	goals := []entities.GoalTarget{{Metric: entities.GoalProfit, Target: 500000}}
	repo.On("SaveGoalTargets", testOrganizationID, "user-1", goals).Return(nil)
	repo.On("GetGoalTargets", testOrganizationID).Return(goals, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodPut, "/dashboard/goals")
	body, _ := json.Marshal(entities.SaveGoalTargetsRequest{Goals: goals})
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	setOrganizationID(c, testOrganizationID)
	c.Set("user_id", "user-1")

	h.SaveGoals(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp entities.GoalTargetsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, goals, resp.Goals)
	repo.AssertExpectations(t)
}

func TestSaveGoals_RejectsInvalidInput(t *testing.T) {
	for name, payload := range map[string]string{
		"unknown metric":    `{"goals":[{"metric":"vibes","target":1}]}`,
		"approval over 100": `{"goals":[{"metric":"approval_rate","target":120}]}`,
		"negative target":   `{"goals":[{"metric":"profit","target":-1}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			repo := mocks.NewMockDashboardRepository()
			h := NewDashboardHandler(repo, mocks.NewMockActivityService(), noopLogger{})
			w, c := setupTestContext(setupTestRouter(), http.MethodPut, "/dashboard/goals")
			c.Request.Body = io.NopCloser(bytes.NewReader([]byte(payload)))
			c.Request.Header.Set("Content-Type", "application/json")
			setOrganizationID(c, testOrganizationID)

			h.SaveGoals(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			repo.AssertNotCalled(t, "SaveGoalTargets", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestGetInsights_DegradesWhenASourceFails(t *testing.T) {
	repo := mocks.NewMockDashboardRepository()
	h := NewDashboardHandler(repo, mocks.NewMockActivityService(), noopLogger{})
	any := mock.Anything
	repo.On("GetOverview", testOrganizationID, any, any, any, any).Return(nil, errors.New("boom"))
	repo.On("GetOperationalInsights", testOrganizationID, any, any, any, any).Return(&entities.OperationalInsightsResponse{}, nil)
	repo.On("GetProfitability", testOrganizationID, any, any, any).Return(&entities.ProfitabilityResponse{}, nil)
	repo.On("GetResponseTimes", testOrganizationID, any, any).Return(&entities.ResponseTimesResponse{}, nil)
	repo.On("GetInsightSignals", testOrganizationID, any, any, any).Return(&entities.InsightSignals{StaleDrafts: 2}, nil)
	repo.On("GetGoalsAlerts", testOrganizationID, any).Return(&entities.GoalsAlertsResponse{}, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/insights?period=30d")
	setOrganizationID(c, testOrganizationID)

	h.GetInsights(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp entities.InsightsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Insights, 1)
	assert.Equal(t, "stale_drafts", resp.Insights[0].Kind)
	assert.Equal(t, "30d", resp.Period)
	assert.True(t, resp.Partial)
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"), "partial results must not be cached")
}

func TestGetInsights_RecoversFromAPanickingSource(t *testing.T) {
	repo := mocks.NewMockDashboardRepository()
	h := NewDashboardHandler(repo, mocks.NewMockActivityService(), noopLogger{})
	any := mock.Anything
	repo.On("GetOverview", testOrganizationID, any, any, any, any).Return(&entities.OverviewResponse{}, nil)
	repo.On("GetOperationalInsights", testOrganizationID, any, any, any, any).Return(&entities.OperationalInsightsResponse{}, nil)
	repo.On("GetProfitability", testOrganizationID, any, any, any).Run(func(mock.Arguments) { panic("scan exploded") })
	repo.On("GetResponseTimes", testOrganizationID, any, any).Return(&entities.ResponseTimesResponse{}, nil)
	repo.On("GetInsightSignals", testOrganizationID, any, any, any).Return(&entities.InsightSignals{}, nil)
	repo.On("GetGoalsAlerts", testOrganizationID, any).Return(&entities.GoalsAlertsResponse{}, nil)

	w, c := setupTestContext(setupTestRouter(), http.MethodGet, "/dashboard/insights")
	setOrganizationID(c, testOrganizationID)

	require.NotPanics(t, func() { h.GetInsights(c) })
	require.Equal(t, http.StatusOK, w.Code)
	var resp entities.InsightsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Partial)
}
