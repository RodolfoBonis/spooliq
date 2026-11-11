package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	coreerrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/insights"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

// GetGoals godoc
// @Summary Get monthly goal targets
// @Description Returns the organization's user-defined monthly targets (revenue and profit in cents, budgets as a count, approval_rate as a percentage).
// @Tags Dashboard
// @Produce json
// @Success 200 {object} entities.GoalTargetsResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Security BearerAuth
// @Router /dashboard/goals [get]
func (h *Handler) GetGoals(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}
	goals, err := h.repo.GetGoalTargets(organizationID)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, entities.GoalTargetsResponse{Goals: goals})
}

// SaveGoals godoc
// @Summary Save monthly goal targets
// @Description Upserts monthly targets per metric (revenue, profit, budgets, approval_rate). A target of 0 removes that goal. Owner/OrgAdmin only.
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param request body entities.SaveGoalTargetsRequest true "Targets"
// @Success 200 {object} entities.GoalTargetsResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Security BearerAuth
// @Router /dashboard/goals [put]
func (h *Handler) SaveGoals(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}
	var req entities.SaveGoalTargetsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreerrors.Respond(c, err)
		return
	}
	for _, goal := range req.Goals {
		if !goal.Metric.Valid() {
			coreerrors.Respond(c, coreerrors.BadRequest("invalid_goal_metric", "Métrica de meta inválida: "+string(goal.Metric)))
			return
		}
		if goal.Metric == entities.GoalApprovalRate && goal.Target > 100 {
			coreerrors.Respond(c, coreerrors.BadRequest("invalid_goal_target", "A meta de aprovação deve estar entre 0 e 100%"))
			return
		}
	}
	if err := h.repo.SaveGoalTargets(organizationID, helpers.GetUserID(c), req.Goals); err != nil {
		coreerrors.Respond(c, err)
		return
	}
	goals, err := h.repo.GetGoalTargets(organizationID)
	if err != nil {
		// Saved already: answer 200 (so the dashboard cache is invalidated) with
		// what was just written instead of failing the request.
		h.logger.Error(c.Request.Context(), "failed to reload dashboard goals after save", logger.Fields{
			"organization_id": organizationID, "error": err.Error(),
		})
		goals = make([]entities.GoalTarget, 0, len(req.Goals))
		for _, g := range req.Goals {
			if g.Target > 0 {
				goals = append(goals, g)
			}
		}
	}
	c.JSON(http.StatusOK, entities.GoalTargetsResponse{Goals: goals})
}

// GetInsights godoc
// @Summary Get actionable insights
// @Description Returns up to 8 rule-based insights (pt-BR) for the period, ranked by severity and money at stake: margin drops, low-margin materials, high-discount customers, low profit/hour machines, slow responses, expiring budgets, expirations, rejections, stock vs demand, waste, stale drafts, inactive repeat customers and goals off pace.
// @Tags Dashboard
// @Produce json
// @Param period query string false "Period filter" Enums(7d, 30d, 3m, 6m, 1y, all) default(30d)
// @Success 200 {object} entities.InsightsResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Security BearerAuth
// @Router /dashboard/insights [get]
func (h *Handler) GetInsights(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}
	period := entities.ParsePeriod(c.Query("period"))
	start, end, prevStart, prevEnd := period.ToTimeRange()
	now := time.Now()

	in, failed := h.collectInsightInput(c.Request.Context(), organizationID, start, end, prevStart, prevEnd, now)
	if failed {
		// Do not let a transient failure stick in the cache for the whole TTL.
		c.Header("Cache-Control", "no-store")
	}
	c.JSON(http.StatusOK, entities.InsightsResponse{
		Insights: insights.Generate(in),
		Period:   string(period),
		Partial:  failed,
	})
}

// collectInsightInput loads every source in parallel. A failing (or panicking)
// source is logged and left nil so the remaining rules still run; failed
// reports whether any source was lost.
func (h *Handler) collectInsightInput(ctx context.Context, organizationID string, start, end, prevStart, prevEnd, now time.Time) (in insights.Input, failed bool) {
	var failures atomic.Int32
	g := new(errgroup.Group)
	load := func(source string, fn func() error) {
		g.Go(func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic: %v", r)
				}
				if err != nil {
					failures.Add(1)
					h.logger.Error(ctx, "dashboard insights source failed", logger.Fields{
						"source": source, "organization_id": organizationID, "error": err.Error(),
					})
				}
				err = nil // degrade instead of failing the request
			}()
			return fn()
		})
	}

	load("overview", func() (err error) {
		in.Overview, err = h.repo.GetOverview(organizationID, start, end, prevStart, prevEnd)
		return
	})
	load("costs", func() error {
		ops, err := h.repo.GetOperationalInsights(organizationID, start, end, prevStart, prevEnd)
		if err != nil {
			return err
		}
		in.CostNow = &ops.CostBreakdown
		return nil
	})
	if !prevStart.IsZero() {
		load("previous_costs", func() error {
			ops, err := h.repo.GetOperationalInsights(organizationID, prevStart, prevEnd, time.Time{}, time.Time{})
			if err != nil {
				return err
			}
			in.CostPrev = &ops.CostBreakdown
			return nil
		})
	}
	load("profitability", func() (err error) {
		in.Profitability, err = h.repo.GetProfitability(organizationID, start, end, maxDashboardLimit)
		return
	})
	load("response_times", func() (err error) {
		in.ResponseTimes, err = h.repo.GetResponseTimes(organizationID, start, end)
		return
	})
	load("signals", func() (err error) {
		in.Signals, err = h.repo.GetInsightSignals(organizationID, start, end, now)
		return
	})
	load("goals", func() error {
		goals, err := h.repo.GetGoalsAlerts(organizationID, now)
		if err != nil {
			return err
		}
		in.Goals = goals.Goals
		return nil
	})
	_ = g.Wait()
	return in, failures.Load() > 0
}
