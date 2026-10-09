package repositories

import (
	"testing"
	"time"

	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
	"github.com/stretchr/testify/assert"
)

func TestNewMonthWindow(t *testing.T) {
	loc := time.FixedZone("BRT", -3*60*60)
	// 2026-10-01 01:00 UTC is still 2026-09-30 22:00 in São Paulo.
	w := newMonthWindow(time.Date(2026, 10, 1, 1, 0, 0, 0, time.UTC), loc)
	assert.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, loc), w.Start)
	assert.Equal(t, 30, w.TotalDays)
	assert.Equal(t, 0, w.DaysLeft)

	w = newMonthWindow(time.Date(2026, 10, 1, 6, 0, 0, 0, loc), loc)
	assert.Equal(t, 1.0, w.ElapsedDays, "elapsed is floored at one day")
	assert.Equal(t, 30, w.DaysLeft)
}

func TestBuildGoal(t *testing.T) {
	w := monthWindow{TotalDays: 30, ElapsedDays: 10, DaysLeft: 20}

	g := buildGoal(entities.GoalProfit, 30000, 120000, w)
	assert.True(t, g.Configured)
	assert.Equal(t, "Lucro", g.Name)
	assert.InDelta(t, 25, g.Progress, 0.001)
	assert.InDelta(t, 90000, g.Projected, 0.001)
	assert.InDelta(t, 75, g.ProjectedProgress, 0.001)
	assert.InDelta(t, 4500, g.RequiredPerDay, 0.001) // (120000-30000)/20

	g = buildGoal(entities.GoalApprovalRate, 55, 60, w)
	assert.Equal(t, 55.0, g.Projected, "ratios are not extrapolated")
	assert.Zero(t, g.RequiredPerDay)

	g = buildGoal(entities.GoalBudgets, 12, 0, w)
	assert.False(t, g.Configured)
	assert.Zero(t, g.Progress)
	assert.InDelta(t, 36, g.Projected, 0.001)

	g = buildGoal(entities.GoalRevenue, 200, 100, w)
	assert.Equal(t, 100.0, g.Progress, "progress is capped")
	assert.Zero(t, g.RequiredPerDay)
}
