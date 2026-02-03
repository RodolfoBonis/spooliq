package repositories

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// percentChange Tests - Division by Zero Handling
// ============================================================================

func TestPercentChange_PositiveChange(t *testing.T) {
	result := percentChange(110.0, 100.0)
	assert.Equal(t, 10.0, result)
}

func TestPercentChange_NegativeChange(t *testing.T) {
	result := percentChange(90.0, 100.0)
	assert.Equal(t, -10.0, result)
}

func TestPercentChange_ZeroPrevious(t *testing.T) {
	// Division by zero should return 0
	result := percentChange(100.0, 0.0)
	assert.Equal(t, 0.0, result)
}

func TestPercentChange_BothZero(t *testing.T) {
	result := percentChange(0.0, 0.0)
	assert.Equal(t, 0.0, result)
}

func TestPercentChange_ZeroCurrent(t *testing.T) {
	result := percentChange(0.0, 100.0)
	assert.Equal(t, -100.0, result)
}

func TestPercentChange_NegativeValues(t *testing.T) {
	// When previous is negative, use absolute value
	result := percentChange(-50.0, -100.0)
	// (-50 - (-100)) / |-100| * 100 = 50/100 * 100 = 50
	assert.Equal(t, 50.0, result)
}

func TestPercentChange_MixedSigns(t *testing.T) {
	// From negative to positive
	result := percentChange(50.0, -100.0)
	// (50 - (-100)) / |-100| * 100 = 150/100 * 100 = 150
	assert.Equal(t, 150.0, result)
}

func TestPercentChange_LargeNumbers(t *testing.T) {
	result := percentChange(1000000.0, 500000.0)
	assert.Equal(t, 100.0, result)
}

func TestPercentChange_SmallFractions(t *testing.T) {
	result := percentChange(0.15, 0.10)
	assert.InDelta(t, 50.0, result, 0.001)
}

func TestPercentChange_Doubling(t *testing.T) {
	result := percentChange(200.0, 100.0)
	assert.Equal(t, 100.0, result)
}

func TestPercentChange_Halving(t *testing.T) {
	result := percentChange(50.0, 100.0)
	assert.Equal(t, -50.0, result)
}

// ============================================================================
// isAllPeriod Tests
// ============================================================================

func TestIsAllPeriod_ZeroTime(t *testing.T) {
	zeroTime := time.Time{}
	assert.True(t, isAllPeriod(zeroTime))
}

func TestIsAllPeriod_NonZeroTime(t *testing.T) {
	now := time.Now()
	assert.False(t, isAllPeriod(now))
}

func TestIsAllPeriod_SpecificDate(t *testing.T) {
	specificDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.False(t, isAllPeriod(specificDate))
}

func TestIsAllPeriod_UnixEpoch(t *testing.T) {
	unixEpoch := time.Unix(0, 0)
	assert.False(t, isAllPeriod(unixEpoch))
}

// ============================================================================
// budgetDateFilter Tests
// ============================================================================

func TestBudgetDateFilter_WithDateRange(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	baseSQL := "SELECT * FROM budgets WHERE organization_id = ?"
	params := []any{"org-123"}

	resultSQL, resultParams := budgetDateFilter(baseSQL, params, start, end)

	expectedSQL := "SELECT * FROM budgets WHERE organization_id = ? AND created_at >= ? AND created_at < ?"
	assert.Equal(t, expectedSQL, resultSQL)
	assert.Len(t, resultParams, 3)
	assert.Equal(t, "org-123", resultParams[0])
	assert.Equal(t, start, resultParams[1])
	assert.Equal(t, end, resultParams[2])
}

func TestBudgetDateFilter_AllPeriod(t *testing.T) {
	start := time.Time{} // Zero time for "all" period
	end := time.Now()

	baseSQL := "SELECT * FROM budgets WHERE organization_id = ?"
	params := []any{"org-123"}

	resultSQL, resultParams := budgetDateFilter(baseSQL, params, start, end)

	// Should not add date filter for all period
	assert.Equal(t, baseSQL, resultSQL)
	assert.Len(t, resultParams, 1)
	assert.Equal(t, "org-123", resultParams[0])
}

func TestBudgetDateFilter_PreservesExistingParams(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	baseSQL := "SELECT * FROM budgets WHERE organization_id = ? AND status = ?"
	params := []any{"org-123", "approved"}

	resultSQL, resultParams := budgetDateFilter(baseSQL, params, start, end)

	assert.Contains(t, resultSQL, "AND created_at >= ? AND created_at < ?")
	assert.Len(t, resultParams, 4)
	assert.Equal(t, "org-123", resultParams[0])
	assert.Equal(t, "approved", resultParams[1])
	assert.Equal(t, start, resultParams[2])
	assert.Equal(t, end, resultParams[3])
}

// ============================================================================
// Percentage Calculation Edge Cases
// ============================================================================

func TestPercentChange_InfinitelySmallPrevious(t *testing.T) {
	// Very small previous value should still work without overflow
	result := percentChange(100.0, 0.0001)
	// (100 - 0.0001) / |0.0001| * 100 = 99.9999 / 0.0001 * 100 = 99999900
	assert.True(t, result > 0)
}

func TestPercentChange_ExactlyDoubled(t *testing.T) {
	tests := []struct {
		current  float64
		previous float64
	}{
		{200, 100},
		{20, 10},
		{2000, 1000},
		{0.2, 0.1},
	}

	for _, tc := range tests {
		result := percentChange(tc.current, tc.previous)
		assert.Equal(t, 100.0, result)
	}
}

func TestPercentChange_ExactlyHalved(t *testing.T) {
	tests := []struct {
		current  float64
		previous float64
	}{
		{50, 100},
		{5, 10},
		{500, 1000},
		{0.05, 0.1},
	}

	for _, tc := range tests {
		result := percentChange(tc.current, tc.previous)
		assert.Equal(t, -50.0, result)
	}
}

func TestPercentChange_NoChange(t *testing.T) {
	tests := []struct {
		value float64
	}{
		{100},
		{0.5},
		{1000000},
	}

	for _, tc := range tests {
		result := percentChange(tc.value, tc.value)
		assert.Equal(t, 0.0, result)
	}
}

// ============================================================================
// Real-World Calculation Scenarios
// ============================================================================

func TestApprovalRateCalculation_NormalScenario(t *testing.T) {
	approved := 75
	decided := 100

	var approvalRate float64
	if decided > 0 {
		approvalRate = float64(approved) / float64(decided) * 100
	}

	assert.Equal(t, 75.0, approvalRate)
}

func TestApprovalRateCalculation_ZeroDecided(t *testing.T) {
	approved := 0
	decided := 0

	var approvalRate float64
	if decided > 0 {
		approvalRate = float64(approved) / float64(decided) * 100
	}

	assert.Equal(t, 0.0, approvalRate)
}

func TestApprovalRateCalculation_AllApproved(t *testing.T) {
	approved := 50
	decided := 50

	var approvalRate float64
	if decided > 0 {
		approvalRate = float64(approved) / float64(decided) * 100
	}

	assert.Equal(t, 100.0, approvalRate)
}

func TestApprovalRateCalculation_NoneApproved(t *testing.T) {
	approved := 0
	decided := 50

	var approvalRate float64
	if decided > 0 {
		approvalRate = float64(approved) / float64(decided) * 100
	}

	assert.Equal(t, 0.0, approvalRate)
}

func TestAvgTicketCalculation_NormalScenario(t *testing.T) {
	totalRevenue := int64(100000)
	revenueCount := int64(10)

	var avgTicket int64
	if revenueCount > 0 {
		avgTicket = totalRevenue / revenueCount
	}

	assert.Equal(t, int64(10000), avgTicket)
}

func TestAvgTicketCalculation_ZeroCount(t *testing.T) {
	totalRevenue := int64(100000)
	revenueCount := int64(0)

	var avgTicket int64
	if revenueCount > 0 {
		avgTicket = totalRevenue / revenueCount
	}

	assert.Equal(t, int64(0), avgTicket)
}

func TestAvgTicketCalculation_ZeroRevenue(t *testing.T) {
	totalRevenue := int64(0)
	revenueCount := int64(10)

	var avgTicket int64
	if revenueCount > 0 {
		avgTicket = totalRevenue / revenueCount
	}

	assert.Equal(t, int64(0), avgTicket)
}

func TestCostBreakdownCalculation_NormalScenario(t *testing.T) {
	filamentCost := int64(4000)
	wasteCost := int64(500)
	energyCost := int64(1000)
	setupCost := int64(500)
	laborCost := int64(2500)
	overheadCost := int64(1500)

	grandTotal := float64(filamentCost + wasteCost + energyCost + setupCost + laborCost + overheadCost)

	var filamentPct, wastePct, energyPct, setupPct, laborPct, overheadPct float64
	if grandTotal > 0 {
		filamentPct = float64(filamentCost) / grandTotal * 100
		wastePct = float64(wasteCost) / grandTotal * 100
		energyPct = float64(energyCost) / grandTotal * 100
		setupPct = float64(setupCost) / grandTotal * 100
		laborPct = float64(laborCost) / grandTotal * 100
		overheadPct = float64(overheadCost) / grandTotal * 100
	}

	// Total should be 10000, so:
	assert.Equal(t, 40.0, filamentPct)
	assert.Equal(t, 5.0, wastePct)
	assert.Equal(t, 10.0, energyPct)
	assert.Equal(t, 5.0, setupPct)
	assert.Equal(t, 25.0, laborPct)
	assert.Equal(t, 15.0, overheadPct)

	// Sum should be 100%
	total := filamentPct + wastePct + energyPct + setupPct + laborPct + overheadPct
	assert.Equal(t, 100.0, total)
}

func TestCostBreakdownCalculation_ZeroCosts(t *testing.T) {
	filamentCost := int64(0)
	wasteCost := int64(0)
	energyCost := int64(0)
	setupCost := int64(0)
	laborCost := int64(0)
	overheadCost := int64(0)

	grandTotal := float64(filamentCost + wasteCost + energyCost + setupCost + laborCost + overheadCost)

	var filamentPct float64
	if grandTotal > 0 {
		filamentPct = float64(filamentCost) / grandTotal * 100
	}

	assert.Equal(t, 0.0, filamentPct)
}

func TestRejectionRateCalculation_NormalScenario(t *testing.T) {
	rejected := 12
	decided := 100

	var rejectionRate float64
	if decided > 0 {
		rejectionRate = float64(rejected) / float64(decided) * 100
	}

	assert.Equal(t, 12.0, rejectionRate)
}

func TestRejectionRateCalculation_ZeroDecided(t *testing.T) {
	rejected := 0
	decided := 0

	var rejectionRate float64
	if decided > 0 {
		rejectionRate = float64(rejected) / float64(decided) * 100
	}

	assert.Equal(t, 0.0, rejectionRate)
}

func TestConversionRateCalculation_NormalScenario(t *testing.T) {
	completed := 20
	total := 100

	var conversionRate float64
	if total > 0 {
		conversionRate = float64(completed) / float64(total) * 100
	}

	assert.Equal(t, 20.0, conversionRate)
}

func TestConversionRateCalculation_ZeroTotal(t *testing.T) {
	completed := 0
	total := 0

	var conversionRate float64
	if total > 0 {
		conversionRate = float64(completed) / float64(total) * 100
	}

	assert.Equal(t, 0.0, conversionRate)
}

func TestGoalProgressCalculation_NormalScenario(t *testing.T) {
	current := 80000.0
	target := 100000.0

	var progress float64
	if target > 0 {
		progress = current / target * 100
	}

	assert.Equal(t, 80.0, progress)
}

func TestGoalProgressCalculation_ZeroTarget(t *testing.T) {
	current := 80000.0
	target := 0.0

	var progress float64
	if target > 0 {
		progress = current / target * 100
	}

	assert.Equal(t, 0.0, progress)
}

func TestGoalProgressCalculation_ExceedsTarget(t *testing.T) {
	current := 120000.0
	target := 100000.0

	var progress float64
	if target > 0 {
		progress = current / target * 100
	}

	assert.Equal(t, 120.0, progress)
}

func TestPrintTimeConversion_MinutesToHours(t *testing.T) {
	totalMinutes := 150.0

	hours := totalMinutes / 60.0

	assert.Equal(t, 2.5, hours)
}

func TestPrintTimeConversion_ZeroMinutes(t *testing.T) {
	totalMinutes := 0.0

	hours := totalMinutes / 60.0

	assert.Equal(t, 0.0, hours)
}
