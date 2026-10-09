package repositories

import (
	"fmt"
	"math"
	"time"

	"github.com/RodolfoBonis/spooliq/features/dashboard/data/models"
	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm/clause"
)

// saoPaulo is the business timezone for month boundaries. Brazil has no DST,
// so a fixed UTC-3 is an exact fallback when the zoneinfo DB is missing.
func saoPaulo() *time.Location {
	if loc, err := time.LoadLocation("America/Sao_Paulo"); err == nil {
		return loc
	}
	return time.FixedZone("BRT", -3*60*60)
}

// monthWindow describes the goals month containing now.
type monthWindow struct {
	Start, End  time.Time // [Start, End) in the business timezone
	TotalDays   int
	ElapsedDays float64 // fractional days since Start, at least 1 to avoid wild projections
	DaysLeft    int     // whole days remaining after today
}

func newMonthWindow(now time.Time, loc *time.Location) monthWindow {
	local := now.In(loc)
	start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 1, 0)
	total := end.AddDate(0, 0, -1).Day()
	return monthWindow{
		Start:       start,
		End:         end,
		TotalDays:   total,
		ElapsedDays: math.Max(1, local.Sub(start).Hours()/24),
		DaysLeft:    total - local.Day(),
	}
}

var goalNames = map[entities.GoalMetric]string{
	entities.GoalRevenue:      "Receita líquida",
	entities.GoalProfit:       "Lucro",
	entities.GoalBudgets:      "Vendas",
	entities.GoalApprovalRate: "Taxa de aprovação",
}

var goalUnits = map[entities.GoalMetric]string{
	entities.GoalRevenue:      "cents",
	entities.GoalProfit:       "cents",
	entities.GoalBudgets:      "count",
	entities.GoalApprovalRate: "percent",
}

// buildGoal computes progress and the month-end projection for one metric.
// target <= 0 means the goal is not configured.
func buildGoal(metric entities.GoalMetric, current, target float64, w monthWindow) entities.Goal {
	g := entities.Goal{
		Metric:   metric,
		Name:     goalNames[metric],
		Current:  current,
		Unit:     goalUnits[metric],
		DaysLeft: w.DaysLeft,
	}
	if metric.Cumulative() {
		g.Projected = current / w.ElapsedDays * float64(w.TotalDays)
	} else {
		g.Projected = current
	}
	if target <= 0 {
		return g
	}
	g.Configured = true
	g.Target = target
	g.Progress = math.Min(current/target*100, 100)
	g.ProjectedProgress = g.Projected / target * 100
	if metric.Cumulative() && current < target {
		g.RequiredPerDay = (target - current) / float64(max(w.DaysLeft, 1))
	}
	return g
}

// GetGoalTargets returns the configured monthly targets.
func (r *DashboardRepositoryImpl) GetGoalTargets(organizationID string) ([]entities.GoalTarget, error) {
	var rows []models.DashboardGoalModel
	if err := r.db.Where("organization_id = ?", organizationID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load dashboard goals: %w", err)
	}
	targets := make([]entities.GoalTarget, 0, len(rows))
	for _, row := range rows {
		targets = append(targets, entities.GoalTarget{Metric: entities.GoalMetric(row.Metric), Target: row.Target})
	}
	return targets, nil
}

// SaveGoalTargets upserts the given targets; a zero target deletes the goal.
func (r *DashboardRepositoryImpl) SaveGoalTargets(organizationID, userID string, goals []entities.GoalTarget) error {
	tx := r.db.Begin()
	for _, goal := range goals {
		var err error
		if goal.Target <= 0 {
			err = tx.Where("organization_id = ? AND metric = ?", organizationID, string(goal.Metric)).
				Delete(&models.DashboardGoalModel{}).Error
		} else {
			err = tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "organization_id"}, {Name: "metric"}},
				DoUpdates: clause.AssignmentColumns([]string{"target", "updated_by", "updated_at"}),
			}).Create(&models.DashboardGoalModel{
				OrganizationID: organizationID,
				Metric:         string(goal.Metric),
				Period:         "monthly",
				Target:         goal.Target,
				UpdatedBy:      userID,
			}).Error
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to save dashboard goal %s: %w", goal.Metric, err)
		}
	}
	return tx.Commit().Error
}

// GetGoalsAlerts returns the month's goals (user-defined targets, progress and
// projection) and operational alerts.
func (r *DashboardRepositoryImpl) GetGoalsAlerts(organizationID string, now time.Time) (*entities.GoalsAlertsResponse, error) {
	w := newMonthWindow(now, saoPaulo())

	var (
		sales       *salesMetrics
		decisions   *decisionMetrics
		targets     []entities.GoalTarget
		staleDrafts int
	)
	g := new(errgroup.Group)
	g.Go(func() (err error) { sales, err = r.querySales(organizationID, w.Start, w.End); return })
	g.Go(func() (err error) { decisions, err = r.queryDecisions(organizationID, w.Start, w.End); return })
	g.Go(func() (err error) { targets, err = r.GetGoalTargets(organizationID); return })
	g.Go(func() error { return r.countStaleDrafts(organizationID, now).Scan(&staleDrafts) })
	if err := g.Wait(); err != nil {
		return nil, err
	}

	byMetric := make(map[entities.GoalMetric]float64, len(targets))
	for _, t := range targets {
		byMetric[t.Metric] = t.Target
	}
	current := map[entities.GoalMetric]float64{
		entities.GoalRevenue:      float64(sales.NetRevenue),
		entities.GoalProfit:       float64(sales.Profit),
		entities.GoalBudgets:      float64(sales.SalesCount),
		entities.GoalApprovalRate: decisions.approvalRate(),
	}
	goals := make([]entities.Goal, 0, len(entities.GoalMetrics))
	for _, metric := range entities.GoalMetrics {
		goals = append(goals, buildGoal(metric, current[metric], byMetric[metric], w))
	}

	alerts := make([]entities.Alert, 0, 2)
	if staleDrafts > 0 {
		alerts = append(alerts, entities.Alert{
			Type:       "stale_drafts",
			Severity:   "warning",
			Message:    fmt.Sprintf("%d rascunho(s) sem atualização há mais de 7 dias", staleDrafts),
			Count:      staleDrafts,
			EntityType: "budget",
		})
	}
	if decisions.decided() > 0 && decisions.rate(decisions.Rejected) > 30 {
		alerts = append(alerts, entities.Alert{
			Type:       "high_rejection",
			Severity:   "danger",
			Message:    fmt.Sprintf("Taxa de recusa de %.1f%% neste mês", decisions.rate(decisions.Rejected)),
			Count:      decisions.Rejected,
			EntityType: "budget",
		})
	}

	return &entities.GoalsAlertsResponse{
		Goals:  goals,
		Alerts: alerts,
		Month:  w.Start.Format("2006-01"),
	}, nil
}
