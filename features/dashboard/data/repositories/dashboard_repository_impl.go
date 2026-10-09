package repositories

import (
	"fmt"
	"math"
	"time"

	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// DashboardRepositoryImpl implements DashboardRepository.
type DashboardRepositoryImpl struct {
	db *gorm.DB
}

// NewDashboardRepository creates a new dashboard repository.
func NewDashboardRepository(db *gorm.DB) *DashboardRepositoryImpl {
	return &DashboardRepositoryImpl{db: db}
}

// percentChange computes ((current - previous) / previous) * 100, returning 0 when previous is zero.
func percentChange(current, previous float64) float64 {
	if previous == 0 {
		return 0
	}
	return (current - previous) / math.Abs(previous) * 100
}

// isAllPeriod returns true when the start time is the zero value, meaning "all time".
func isAllPeriod(start time.Time) bool {
	return start.IsZero()
}

// budgetDateFilter appends "AND created_at >= ? AND created_at < ?" when applicable.
func budgetDateFilter(base string, params []any, start, end time.Time) (string, []any) {
	if !isAllPeriod(start) {
		base += " AND created_at >= ? AND created_at < ?"
		params = append(params, start, end)
	}
	return base, params
}

// ──────────────────────────────────────────────
// GetOverview
// ──────────────────────────────────────────────

type overviewMetrics struct {
	TotalBudgets int
	NewCustomers int
	Sales        *salesMetrics
	Decisions    *decisionMetrics
}

// queryOverviewMetrics gathers one period: volume by creation date, sales and
// profit by approval date, and decisions by history date.
func (r *DashboardRepositoryImpl) queryOverviewMetrics(organizationID string, start, end time.Time) (*overviewMetrics, error) {
	m := &overviewMetrics{}
	g := new(errgroup.Group)
	g.Go(func() error {
		sql := "SELECT COUNT(*) FROM budgets WHERE organization_id = ? AND deleted_at IS NULL"
		params := []any{organizationID}
		sql, params = budgetDateFilter(sql, params, start, end)
		return r.db.Raw(sql, params...).Row().Scan(&m.TotalBudgets)
	})
	g.Go(func() error {
		sql := "SELECT COUNT(*) FROM customers WHERE organization_id = ? AND deleted_at IS NULL"
		params := []any{organizationID}
		if !isAllPeriod(start) {
			sql += " AND created_at >= ? AND created_at < ?"
			params = append(params, start, end)
		}
		return r.db.Raw(sql, params...).Row().Scan(&m.NewCustomers)
	})
	g.Go(func() error {
		var err error
		m.Sales, err = r.querySales(organizationID, start, end)
		return err
	})
	g.Go(func() error {
		var err error
		m.Decisions, err = r.queryDecisions(organizationID, start, end)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return m, nil
}

// GetOverview returns dashboard overview metrics.
func (r *DashboardRepositoryImpl) GetOverview(organizationID string, start, end, prevStart, prevEnd time.Time) (*entities.OverviewResponse, error) {
	var current, previous *overviewMetrics

	g := new(errgroup.Group)
	g.Go(func() error {
		var err error
		current, err = r.queryOverviewMetrics(organizationID, start, end)
		return err
	})
	g.Go(func() error {
		// Only query previous period when a date range is provided.
		if isAllPeriod(start) {
			previous = &overviewMetrics{Sales: &salesMetrics{}, Decisions: &decisionMetrics{}}
			return nil
		}
		var err error
		previous, err = r.queryOverviewMetrics(organizationID, prevStart, prevEnd)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	cur, prev := current.Sales, previous.Sales
	avgTicket := func(s *salesMetrics) int64 {
		if s.SalesCount == 0 {
			return 0
		}
		return s.GrossRevenue / s.SalesCount
	}

	// Budgets by status (volume, by creation date)
	type statusRow struct {
		Status string
		Count  int
	}
	var statusRows []statusRow
	{
		sql := "SELECT status, COUNT(*) as count FROM budgets WHERE organization_id = ? AND deleted_at IS NULL"
		params := []any{organizationID}
		sql, params = budgetDateFilter(sql, params, start, end)
		sql += " GROUP BY status"
		if err := r.db.Raw(sql, params...).Scan(&statusRows).Error; err != nil {
			return nil, err
		}
	}
	budgetsByStatus := make([]entities.BudgetStatusCount, 0, len(statusRows))
	for _, row := range statusRows {
		budgetsByStatus = append(budgetsByStatus, entities.BudgetStatusCount{Status: row.Status, Count: row.Count})
	}

	approval, prevApproval := current.Decisions.approvalRate(), previous.Decisions.approvalRate()
	margin, prevMargin := cur.margin(), prev.margin()

	return &entities.OverviewResponse{
		TotalRevenue:       cur.GrossRevenue,
		RevenueChange:      percentChange(float64(cur.GrossRevenue), float64(prev.GrossRevenue)),
		TotalBudgets:       current.TotalBudgets,
		BudgetsChange:      percentChange(float64(current.TotalBudgets), float64(previous.TotalBudgets)),
		AvgTicket:          avgTicket(cur),
		AvgTicketChange:    percentChange(float64(avgTicket(cur)), float64(avgTicket(prev))),
		ApprovalRate:       approval,
		ApprovalRateChange: percentChange(approval, prevApproval),
		AvgProfitMargin:    margin,
		ProfitMarginChange: percentChange(margin, prevMargin),
		NewCustomers:       current.NewCustomers,
		NewCustomersChange: percentChange(float64(current.NewCustomers), float64(previous.NewCustomers)),
		BudgetsByStatus:    budgetsByStatus,

		NetRevenue:               cur.NetRevenue,
		NetRevenueChange:         percentChange(float64(cur.NetRevenue), float64(prev.NetRevenue)),
		Profit:                   cur.Profit,
		ProfitChange:             percentChange(float64(cur.Profit), float64(prev.Profit)),
		ProfitRealized:           cur.ProfitRealized,
		ProfitForecast:           cur.Profit - cur.ProfitRealized,
		ProductionCost:           cur.NetRevenue - cur.Profit,
		ProfitMargin:             margin,
		ProfitMarginPointsChange: margin - prevMargin,
		ProfitPerPrintHour:       cur.profitPerHour(),
		ProfitPerPrintHourChange: percentChange(float64(cur.profitPerHour()), float64(prev.profitPerHour())),
		PrintHours:               cur.printHours(),
		SalesCount:               int(cur.SalesCount),
		ApprovalRatePointsChange: approval - prevApproval,
	}, nil
}

// ──────────────────────────────────────────────
// GetRevenueTrend
// ──────────────────────────────────────────────

// GetRevenueTrend returns revenue trend data points.
func (r *DashboardRepositoryImpl) GetRevenueTrend(organizationID string, start, end time.Time, truncate string) (*entities.RevenueTrendResponse, error) {
	dateFormat := "YYYY-MM-DD"
	if truncate == "month" {
		dateFormat = "YYYY-MM"
	}

	// Sales by approval date, bucketed in Brazil's time zone.
	sql := `SELECT to_char(date_trunc(?, b.approved_at AT TIME ZONE 'America/Sao_Paulo'), ?) AS date,
	               COALESCE(SUM(` + netRevenueExpr + `), 0) AS revenue,
	               COALESCE(SUM(` + netRevenueExpr + ` - ` + profitExpr + `), 0) AS cost,
	               COALESCE(SUM(` + profitExpr + `), 0) AS profit,
	               COUNT(*) AS budget_count
	        FROM budgets b
	        WHERE b.organization_id = ?
	          AND b.status IN ` + saleStatuses + `
	          AND b.approved_at IS NOT NULL
	          AND b.deleted_at IS NULL`
	params := []any{truncate, dateFormat, organizationID}
	sql, params = saleWindow(sql, params, start, end)

	sql += " GROUP BY date ORDER BY date"

	type trendRow struct {
		Date        string
		Revenue     int64
		Cost        int64
		Profit      int64
		BudgetCount int `gorm:"column:budget_count"`
	}
	var rows []trendRow
	if err := r.db.Raw(sql, params...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	points := make([]entities.RevenueTrendPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, entities.RevenueTrendPoint{
			Date:        row.Date,
			Revenue:     row.Revenue,
			Cost:        row.Cost,
			Profit:      row.Profit,
			BudgetCount: row.BudgetCount,
		})
	}

	return &entities.RevenueTrendResponse{
		Points: points,
	}, nil
}

// ──────────────────────────────────────────────
// GetConversionFunnel
// ──────────────────────────────────────────────

// GetConversionFunnel returns conversion funnel data with step-to-step conversion rates.
// It uses budget_status_history to count how many budgets reached each status,
// then derives the rate as: reached(next) / reached(current) * 100.
func (r *DashboardRepositoryImpl) GetConversionFunnel(organizationID string, start, end time.Time) (*entities.ConversionFunnelResponse, error) {
	// 1. Total budgets = budgets that reached "draft" (all budgets start as draft).
	totalSQL := "SELECT COUNT(*) FROM budgets WHERE organization_id = ? AND deleted_at IS NULL"
	totalParams := []any{organizationID}
	totalSQL, totalParams = budgetDateFilter(totalSQL, totalParams, start, end)

	var totalBudgets int
	if err := r.db.Raw(totalSQL, totalParams...).Row().Scan(&totalBudgets); err != nil {
		return nil, err
	}

	// 2. Count distinct budgets that reached each status via history.
	//    Filter by budget created_at (not transition created_at) for consistency.
	historySQL := `SELECT bsh.new_status, COUNT(DISTINCT bsh.budget_id) AS reached
		FROM budget_status_history bsh
		JOIN budgets b ON b.id = bsh.budget_id AND b.deleted_at IS NULL
		WHERE bsh.organization_id = ?`
	historyParams := []any{organizationID}
	if !isAllPeriod(start) {
		historySQL += " AND b.created_at >= ? AND b.created_at < ?"
		historyParams = append(historyParams, start, end)
	}
	historySQL += " GROUP BY bsh.new_status"

	type reachedRow struct {
		NewStatus string `gorm:"column:new_status"`
		Reached   int    `gorm:"column:reached"`
	}
	var rows []reachedRow
	if err := r.db.Raw(historySQL, historyParams...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	// 3. Build reached map. Draft = totalBudgets (every budget starts there).
	reached := map[string]int{"draft": totalBudgets}
	for _, row := range rows {
		reached[row.NewStatus] = row.Reached
	}

	// 4. Compute step-to-step conversion rates.
	//    Main funnel: draft → sent → approved → printing → completed
	//    Rejected is a branch off "sent".
	funnelOrder := []string{"draft", "sent", "approved", "printing", "completed", "rejected"}
	steps := make([]entities.FunnelStep, 0, len(funnelOrder))

	for i, status := range funnelOrder {
		count := reached[status]
		var rate float64

		switch {
		case status == "draft":
			rate = 100
		case status == "rejected":
			if reached["sent"] > 0 {
				rate = float64(count) / float64(reached["sent"]) * 100
			}
		default:
			prevStatus := funnelOrder[i-1]
			if reached[prevStatus] > 0 {
				rate = float64(count) / float64(reached[prevStatus]) * 100
			}
		}

		steps = append(steps, entities.FunnelStep{
			Status:         status,
			Count:          count,
			ConversionRate: rate,
		})
	}

	var overallConversion float64
	if totalBudgets > 0 {
		overallConversion = float64(reached["completed"]) / float64(totalBudgets) * 100
	}

	return &entities.ConversionFunnelResponse{
		Steps:             steps,
		TotalBudgets:      totalBudgets,
		OverallConversion: overallConversion,
	}, nil
}

// ──────────────────────────────────────────────
// GetTopCustomers
// ──────────────────────────────────────────────

// GetTopCustomers returns top customers by revenue.
func (r *DashboardRepositoryImpl) GetTopCustomers(organizationID string, start, end time.Time, limit int) (*entities.TopCustomersResponse, error) {
	sql := `SELECT c.id, c.name, c.email,
	               COALESCE(SUM(b.total_cost), 0) AS total_revenue,
	               COUNT(b.id) AS budget_count,
	               CAST(CASE WHEN COUNT(b.id) > 0 THEN COALESCE(SUM(b.total_cost), 0) / COUNT(b.id) ELSE 0 END AS BIGINT) AS avg_ticket
	        FROM customers c
	        JOIN budgets b ON b.customer_id = c.id AND b.deleted_at IS NULL
	        WHERE c.organization_id = ?
	          AND c.deleted_at IS NULL
	          AND b.status IN ('approved','printing','completed')`
	params := []any{organizationID}

	if !isAllPeriod(start) {
		sql += " AND b.created_at >= ? AND b.created_at < ?"
		params = append(params, start, end)
	}

	sql += " GROUP BY c.id, c.name, c.email ORDER BY total_revenue DESC LIMIT ?"
	params = append(params, limit)

	var rows []entities.TopCustomer
	if err := r.db.Raw(sql, params...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	if rows == nil {
		rows = []entities.TopCustomer{}
	}

	return &entities.TopCustomersResponse{
		Customers: rows,
	}, nil
}

// ──────────────────────────────────────────────
// GetOperationalInsights
// ──────────────────────────────────────────────

// costTotals sums the sales' cost components (cents) for the breakdown.
type costTotals struct {
	Filament, Waste, Energy, Machine, Setup, Labor               int64
	PostProcessing, Packaging, QualityControl, Failure, Overhead int64
}

func (c costTotals) sum() int64 {
	return c.Filament + c.Waste + c.Energy + c.Machine + c.Setup + c.Labor +
		c.PostProcessing + c.Packaging + c.QualityControl + c.Failure + c.Overhead
}

type opsMetrics struct {
	Sales     *salesMetrics
	Decisions *decisionMetrics
	Costs     costTotals
}

func (r *DashboardRepositoryImpl) queryOpsMetrics(organizationID string, start, end time.Time) (*opsMetrics, error) {
	m := &opsMetrics{}
	g := new(errgroup.Group)
	g.Go(func() error {
		var err error
		m.Sales, err = r.querySales(organizationID, start, end)
		return err
	})
	g.Go(func() error {
		var err error
		m.Decisions, err = r.queryDecisions(organizationID, start, end)
		return err
	})
	g.Go(func() error {
		sql := `SELECT
			COALESCE(SUM(b.filament_cost), 0), COALESCE(SUM(b.waste_cost), 0),
			COALESCE(SUM(b.energy_cost), 0), COALESCE(SUM(b.machine_cost), 0),
			COALESCE(SUM(b.setup_cost), 0), COALESCE(SUM(b.labor_cost), 0),
			COALESCE(SUM(b.post_processing_cost), 0), COALESCE(SUM(b.packaging_cost), 0),
			COALESCE(SUM(b.quality_control_cost), 0), COALESCE(SUM(b.failure_cost), 0),
			COALESCE(SUM(b.overhead_cost), 0)
		FROM budgets b
		WHERE b.organization_id = ? AND b.deleted_at IS NULL AND b.status IN ` + saleStatuses
		params := []any{organizationID}
		sql, params = saleWindow(sql, params, start, end)
		c := &m.Costs
		return r.db.Raw(sql, params...).Row().Scan(
			&c.Filament, &c.Waste, &c.Energy, &c.Machine, &c.Setup, &c.Labor,
			&c.PostProcessing, &c.Packaging, &c.QualityControl, &c.Failure, &c.Overhead,
		)
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return m, nil
}

// GetOperationalInsights returns operational metrics.
func (r *DashboardRepositoryImpl) GetOperationalInsights(organizationID string, start, end, prevStart, prevEnd time.Time) (*entities.OperationalInsightsResponse, error) {
	var current, previous *opsMetrics

	g := new(errgroup.Group)
	g.Go(func() error {
		var err error
		current, err = r.queryOpsMetrics(organizationID, start, end)
		return err
	})
	g.Go(func() error {
		if isAllPeriod(start) {
			previous = &opsMetrics{Sales: &salesMetrics{}, Decisions: &decisionMetrics{}}
			return nil
		}
		var err error
		previous, err = r.queryOpsMetrics(organizationID, prevStart, prevEnd)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	ticket := func(s *salesMetrics) int64 {
		if s.SalesCount == 0 {
			return 0
		}
		return s.GrossRevenue / s.SalesCount
	}
	cur, prev := current.Sales, previous.Sales
	rejection := current.Decisions.rate(current.Decisions.Rejected)
	prevRejection := previous.Decisions.rate(previous.Decisions.Rejected)

	var breakdown entities.CostBreakdown
	if total := float64(current.Costs.sum()); total > 0 {
		pct := func(v int64) float64 { return float64(v) / total * 100 }
		c := current.Costs
		breakdown = entities.CostBreakdown{
			FilamentPct:       pct(c.Filament),
			WastePct:          pct(c.Waste),
			EnergyPct:         pct(c.Energy),
			SetupPct:          pct(c.Setup),
			LaborPct:          pct(c.Labor),
			OverheadPct:       pct(c.Overhead),
			MachinePct:        pct(c.Machine),
			PostProcessingPct: pct(c.PostProcessing),
			PackagingPct:      pct(c.Packaging),
			QualityControlPct: pct(c.QualityControl),
			FailurePct:        pct(c.Failure),
		}
	}

	return &entities.OperationalInsightsResponse{
		AvgTicket:           ticket(cur),
		AvgTicketChange:     percentChange(float64(ticket(cur)), float64(ticket(prev))),
		AvgProfitMargin:     cur.margin(),
		ProfitMarginChange:  percentChange(cur.margin(), prev.margin()),
		TotalPrintTimeHours: cur.printHours(),
		PrintTimeChange:     percentChange(cur.printHours(), prev.printHours()),
		RejectionRate:       rejection,
		RejectionRateChange: percentChange(rejection, prevRejection),
		CostBreakdown:       breakdown,
	}, nil
}

// ──────────────────────────────────────────────
// GetTopFilaments
// ──────────────────────────────────────────────

// GetTopFilaments returns top filaments by usage.
func (r *DashboardRepositoryImpl) GetTopFilaments(organizationID string, start, end time.Time, limit int) (*entities.TopFilamentsResponse, error) {
	sql := `SELECT f.id, f.name, COALESCE(br.name, '') AS brand_name, COALESCE(m.name, '') AS material_name, COALESCE(f.color_hex, '') AS color_hex,
	               COALESCE(SUM(bif.quantity), 0) AS total_grams,
	               COUNT(DISTINCT bif.budget_item_id) AS usage_count
	        FROM budget_item_filaments bif
	        JOIN budget_items bi ON bi.id = bif.budget_item_id
	        JOIN budgets b ON b.id = bi.budget_id AND b.deleted_at IS NULL
	        JOIN filaments f ON f.id = bif.filament_id AND f.deleted_at IS NULL
	        LEFT JOIN brands br ON br.id = f.brand_id AND br.deleted_at IS NULL
	        LEFT JOIN materials m ON m.id = f.material_id AND m.deleted_at IS NULL
	        WHERE b.organization_id = ?
	          AND b.status IN ('approved','printing','completed')`
	params := []any{organizationID}

	if !isAllPeriod(start) {
		sql += " AND b.created_at >= ? AND b.created_at < ?"
		params = append(params, start, end)
	}

	sql += " GROUP BY f.id, f.name, br.name, m.name, f.color_hex ORDER BY total_grams DESC LIMIT ?"
	params = append(params, limit)

	var rows []entities.TopFilament
	if err := r.db.Raw(sql, params...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	if rows == nil {
		rows = []entities.TopFilament{}
	}

	return &entities.TopFilamentsResponse{
		Filaments: rows,
	}, nil
}

// ──────────────────────────────────────────────
// GetTopMaterials
// ──────────────────────────────────────────────

// GetTopMaterials returns top materials by usage.
func (r *DashboardRepositoryImpl) GetTopMaterials(organizationID string, start, end time.Time, limit int) (*entities.TopMaterialsResponse, error) {
	sql := `SELECT m.id, m.name,
	               COALESCE(SUM(bif.quantity), 0) AS total_grams,
	               COUNT(DISTINCT bif.filament_id) AS usage_count
	        FROM budget_item_filaments bif
	        JOIN budget_items bi ON bi.id = bif.budget_item_id
	        JOIN budgets b ON b.id = bi.budget_id AND b.deleted_at IS NULL
	        JOIN filaments f ON f.id = bif.filament_id AND f.deleted_at IS NULL
	        JOIN materials m ON m.id = f.material_id AND m.deleted_at IS NULL
	        WHERE b.organization_id = ?
	          AND b.status IN ('approved','printing','completed')`
	params := []any{organizationID}

	if !isAllPeriod(start) {
		sql += " AND b.created_at >= ? AND b.created_at < ?"
		params = append(params, start, end)
	}

	sql += " GROUP BY m.id, m.name ORDER BY total_grams DESC LIMIT ?"
	params = append(params, limit)

	var rows []entities.TopMaterial
	if err := r.db.Raw(sql, params...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	if rows == nil {
		rows = []entities.TopMaterial{}
	}

	return &entities.TopMaterialsResponse{
		Materials: rows,
	}, nil
}

// ──────────────────────────────────────────────
// GetGoalsAlerts
// ──────────────────────────────────────────────

// goalPeriodMetrics holds aggregated metrics for a single period used by GetGoalsAlerts.
type goalPeriodMetrics struct {
	Revenue     int64
	BudgetCount int
	Approved    int
	Decided     int
}

// GetGoalsAlerts returns goals progress and active alerts.
func (r *DashboardRepositoryImpl) GetGoalsAlerts(organizationID string) (*entities.GoalsAlertsResponse, error) {
	now := time.Now().UTC()
	currentStart := now.AddDate(0, 0, -30)
	prevStart := now.AddDate(0, 0, -60)

	var current, previous goalPeriodMetrics
	var staleDrafts, inactiveCustomers int

	g := new(errgroup.Group)

	// Query 1: Both periods in a single query using period bucketing.
	// Replaces 4 sequential queries (prev revenue, prev approval, cur revenue, cur approval).
	g.Go(func() error {
		type periodRow struct {
			Period      string `gorm:"column:period"`
			Revenue     int64  `gorm:"column:revenue"`
			BudgetCount int    `gorm:"column:budget_count"`
			Approved    int    `gorm:"column:approved"`
			Decided     int    `gorm:"column:decided"`
		}

		sql := `SELECT
			CASE WHEN created_at >= ? THEN 'current' ELSE 'previous' END AS period,
			COALESCE(SUM(CASE WHEN status IN ('approved','printing','completed') THEN total_cost ELSE 0 END), 0) AS revenue,
			COALESCE(SUM(CASE WHEN status IN ('approved','printing','completed') THEN 1 ELSE 0 END), 0) AS budget_count,
			COALESCE(SUM(CASE WHEN status = 'approved' THEN 1 ELSE 0 END), 0) AS approved,
			COALESCE(SUM(CASE WHEN status IN ('approved','rejected') THEN 1 ELSE 0 END), 0) AS decided
		FROM budgets
		WHERE organization_id = ?
		  AND deleted_at IS NULL
		  AND created_at >= ? AND created_at < ?
		GROUP BY period`

		var rows []periodRow
		if err := r.db.Raw(sql, currentStart, organizationID, prevStart, now).Scan(&rows).Error; err != nil {
			return err
		}

		for _, row := range rows {
			switch row.Period {
			case "current":
				current = goalPeriodMetrics{Revenue: row.Revenue, BudgetCount: row.BudgetCount, Approved: row.Approved, Decided: row.Decided}
			case "previous":
				previous = goalPeriodMetrics{Revenue: row.Revenue, BudgetCount: row.BudgetCount, Approved: row.Approved, Decided: row.Decided}
			}
		}
		return nil
	})

	// Query 2: Stale drafts (uses idx_budgets_org_status_updated)
	g.Go(func() error {
		sql := `SELECT COUNT(*)
		        FROM budgets
		        WHERE organization_id = ?
		          AND status = 'draft'
		          AND updated_at < ?
		          AND deleted_at IS NULL`
		staleThreshold := now.AddDate(0, 0, -7)
		return r.db.Raw(sql, organizationID, staleThreshold).Row().Scan(&staleDrafts)
	})

	// Query 3: Inactive customers
	g.Go(func() error {
		sql := `SELECT COUNT(*)
		        FROM customers c
		        WHERE c.organization_id = ?
		          AND c.is_active = true
		          AND c.deleted_at IS NULL
		          AND NOT EXISTS (
		            SELECT 1 FROM budgets b
		            WHERE b.customer_id = c.id
		              AND b.deleted_at IS NULL
		              AND b.created_at >= ?
		          )`
		return r.db.Raw(sql, organizationID, currentStart).Row().Scan(&inactiveCustomers)
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Build goals
	goals := make([]entities.Goal, 0, 3)

	revenueTarget := float64(previous.Revenue) * 1.1
	var revProgress float64
	if revenueTarget > 0 {
		revProgress = float64(current.Revenue) / revenueTarget * 100
	}
	goals = append(goals, entities.Goal{
		Name:     "Monthly Revenue",
		Current:  float64(current.Revenue),
		Target:   revenueTarget,
		Progress: math.Min(revProgress, 100),
		Unit:     "cents",
	})

	budgetTarget := float64(previous.BudgetCount) * 1.1
	var budgetProgress float64
	if budgetTarget > 0 {
		budgetProgress = float64(current.BudgetCount) / budgetTarget * 100
	}
	goals = append(goals, entities.Goal{
		Name:     "Monthly Budgets",
		Current:  float64(current.BudgetCount),
		Target:   budgetTarget,
		Progress: math.Min(budgetProgress, 100),
		Unit:     "count",
	})

	var prevConvRate float64
	if previous.Decided > 0 {
		prevConvRate = float64(previous.Approved) / float64(previous.Decided) * 100
	}
	convTarget := prevConvRate * 1.1
	var curConvRate float64
	if current.Decided > 0 {
		curConvRate = float64(current.Approved) / float64(current.Decided) * 100
	}
	var convProgress float64
	if convTarget > 0 {
		convProgress = curConvRate / convTarget * 100
	}
	goals = append(goals, entities.Goal{
		Name:     "Conversion Rate",
		Current:  curConvRate,
		Target:   convTarget,
		Progress: math.Min(convProgress, 100),
		Unit:     "percent",
	})

	// Build alerts
	alerts := make([]entities.Alert, 0, 3)

	if staleDrafts > 0 {
		alerts = append(alerts, entities.Alert{
			Type:       "stale_drafts",
			Severity:   "warning",
			Message:    fmt.Sprintf("%d budget(s) in draft status with no updates in 7+ days", staleDrafts),
			Count:      staleDrafts,
			EntityType: "budget",
		})
	}

	if current.Decided > 0 {
		rejRate := float64(current.Decided-current.Approved) / float64(current.Decided) * 100
		if rejRate > 30 {
			alerts = append(alerts, entities.Alert{
				Type:       "high_rejection",
				Severity:   "danger",
				Message:    fmt.Sprintf("Rejection rate is %.1f%% in the last 30 days", rejRate),
				Count:      current.Decided - current.Approved,
				EntityType: "budget",
			})
		}
	}

	if inactiveCustomers > 0 {
		alerts = append(alerts, entities.Alert{
			Type:       "inactive_customers",
			Severity:   "info",
			Message:    fmt.Sprintf("%d active customer(s) with no budgets in the last 30 days", inactiveCustomers),
			Count:      inactiveCustomers,
			EntityType: "customer",
		})
	}

	return &entities.GoalsAlertsResponse{
		Goals:  goals,
		Alerts: alerts,
	}, nil
}

// GetLowStockFilaments returns tracked filaments at or below their alert threshold,
// ordered by the shortfall (stock_grams - threshold) ascending so the most depleted
// come first, capped at limit. It joins brands/materials for display names.
func (r *DashboardRepositoryImpl) GetLowStockFilaments(organizationID string, limit int) ([]entities.LowStockFilament, error) {
	var rows []entities.LowStockFilament
	err := r.db.
		Table("filaments AS f").
		Select("f.id AS id, f.name AS name, f.color AS color, f.color_hex AS color_hex, b.name AS brand_name, m.name AS material_name, f.stock_grams AS stock_grams, f.low_stock_threshold_grams AS low_stock_threshold_grams").
		Joins("LEFT JOIN brands b ON b.id = f.brand_id").
		Joins("LEFT JOIN materials m ON m.id = f.material_id").
		Where("f.organization_id = ? AND f.deleted_at IS NULL AND f.track_stock = ? AND f.low_stock_threshold_grams IS NOT NULL AND f.stock_grams <= f.low_stock_threshold_grams", organizationID, true).
		Order("(f.stock_grams - f.low_stock_threshold_grams) ASC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query low stock filaments: %w", err)
	}
	return rows, nil
}
