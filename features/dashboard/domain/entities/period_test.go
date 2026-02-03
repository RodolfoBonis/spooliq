package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParsePeriod_ValidPeriods(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Period
	}{
		{name: "7 days", input: "7d", expected: Period7D},
		{name: "30 days", input: "30d", expected: Period30D},
		{name: "3 months", input: "3m", expected: Period3M},
		{name: "6 months", input: "6m", expected: Period6M},
		{name: "1 year", input: "1y", expected: Period1Y},
		{name: "all time", input: "all", expected: PeriodAll},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ParsePeriod(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParsePeriod_InvalidPeriods(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "invalid period", input: "invalid"},
		{name: "uppercase", input: "7D"},
		{name: "with spaces", input: " 7d "},
		{name: "numeric only", input: "7"},
		{name: "unknown format", input: "1w"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ParsePeriod(tc.input)
			assert.Equal(t, Period30D, result, "invalid period should default to 30d")
		})
	}
}

func TestPeriod_ToTimeRange_7Days(t *testing.T) {
	period := Period7D

	start, end, prevStart, prevEnd := period.ToTimeRange()

	// end should be close to now
	now := time.Now()
	assert.WithinDuration(t, now, end, time.Second)

	// start should be 7 days before now
	expectedStart := now.AddDate(0, 0, -7)
	assert.WithinDuration(t, expectedStart, start, time.Second)

	// prevStart should be 14 days before now
	expectedPrevStart := now.AddDate(0, 0, -14)
	assert.WithinDuration(t, expectedPrevStart, prevStart, time.Second)

	// prevEnd should equal start
	assert.Equal(t, start, prevEnd)
}

func TestPeriod_ToTimeRange_30Days(t *testing.T) {
	period := Period30D

	start, end, prevStart, prevEnd := period.ToTimeRange()

	now := time.Now()
	assert.WithinDuration(t, now, end, time.Second)

	expectedStart := now.AddDate(0, 0, -30)
	assert.WithinDuration(t, expectedStart, start, time.Second)

	expectedPrevStart := now.AddDate(0, 0, -60)
	assert.WithinDuration(t, expectedPrevStart, prevStart, time.Second)

	assert.Equal(t, start, prevEnd)
}

func TestPeriod_ToTimeRange_3Months(t *testing.T) {
	period := Period3M

	start, end, prevStart, prevEnd := period.ToTimeRange()

	now := time.Now()
	assert.WithinDuration(t, now, end, time.Second)

	expectedStart := now.AddDate(0, -3, 0)
	assert.WithinDuration(t, expectedStart, start, time.Second)

	expectedPrevStart := now.AddDate(0, -6, 0)
	assert.WithinDuration(t, expectedPrevStart, prevStart, time.Second)

	assert.Equal(t, start, prevEnd)
}

func TestPeriod_ToTimeRange_6Months(t *testing.T) {
	period := Period6M

	start, end, prevStart, prevEnd := period.ToTimeRange()

	now := time.Now()
	assert.WithinDuration(t, now, end, time.Second)

	expectedStart := now.AddDate(0, -6, 0)
	assert.WithinDuration(t, expectedStart, start, time.Second)

	expectedPrevStart := now.AddDate(-1, 0, 0)
	assert.WithinDuration(t, expectedPrevStart, prevStart, time.Second)

	assert.Equal(t, start, prevEnd)
}

func TestPeriod_ToTimeRange_1Year(t *testing.T) {
	period := Period1Y

	start, end, prevStart, prevEnd := period.ToTimeRange()

	now := time.Now()
	assert.WithinDuration(t, now, end, time.Second)

	expectedStart := now.AddDate(-1, 0, 0)
	assert.WithinDuration(t, expectedStart, start, time.Second)

	expectedPrevStart := now.AddDate(-2, 0, 0)
	assert.WithinDuration(t, expectedPrevStart, prevStart, time.Second)

	assert.Equal(t, start, prevEnd)
}

func TestPeriod_ToTimeRange_All(t *testing.T) {
	period := PeriodAll

	start, end, prevStart, prevEnd := period.ToTimeRange()

	now := time.Now()
	assert.WithinDuration(t, now, end, time.Second)

	// For "all" period, start should be zero time
	assert.True(t, start.IsZero())
	assert.True(t, prevStart.IsZero())
	assert.True(t, prevEnd.IsZero())
}

func TestPeriod_TruncateFunc_DailyForShortPeriods(t *testing.T) {
	tests := []struct {
		period   Period
		expected string
	}{
		{Period7D, "day"},
		{Period30D, "day"},
	}

	for _, tc := range tests {
		t.Run(string(tc.period), func(t *testing.T) {
			result := tc.period.TruncateFunc()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPeriod_TruncateFunc_MonthlyForLongPeriods(t *testing.T) {
	tests := []struct {
		period   Period
		expected string
	}{
		{Period3M, "month"},
		{Period6M, "month"},
		{Period1Y, "month"},
		{PeriodAll, "month"},
	}

	for _, tc := range tests {
		t.Run(string(tc.period), func(t *testing.T) {
			result := tc.period.TruncateFunc()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPeriod_Constants(t *testing.T) {
	assert.Equal(t, Period("7d"), Period7D)
	assert.Equal(t, Period("30d"), Period30D)
	assert.Equal(t, Period("3m"), Period3M)
	assert.Equal(t, Period("6m"), Period6M)
	assert.Equal(t, Period("1y"), Period1Y)
	assert.Equal(t, Period("all"), PeriodAll)
}

func TestPeriod_ToTimeRange_PreviousPeriodIsConsecutive(t *testing.T) {
	// Test that previous period ends exactly where current period starts
	periods := []Period{Period7D, Period30D, Period3M, Period6M, Period1Y}

	for _, period := range periods {
		t.Run(string(period), func(t *testing.T) {
			start, _, _, prevEnd := period.ToTimeRange()
			assert.Equal(t, start, prevEnd, "previous period should end exactly where current period starts")
		})
	}
}
