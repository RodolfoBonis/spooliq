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
	TotalRevenue    int64
	RevenueCount    int64
	TotalBudgets    int
	ApprovedCount   int
	RejectedCount   int
	AvgProfitMargin float64
	NewCustomers    int
}

func (r *DashboardRepositoryImpl) queryOverviewMetrics(organizationID string, start, end time.Time) (*overviewMetrics, error) {
	m := &overviewMetrics{}

	// Single query with conditional aggregation replaces 4 separate budget queries.
	// Uses idx_budgets_org_created (org + date without status filter on full table scan).
	{
		sql := `SELECT
			COUNT(*) AS total_budgets,
			COALESCE(SUM(CASE WHEN status IN ('approved','printing','completed') THEN total_cost ELSE 0 END), 0) AS total_revenue,
			COALESCE(SUM(CASE WHEN status IN ('approved','printing','completed') THEN 1 ELSE 0 END), 0) AS revenue_count,
			COALESCE(SUM(CASE WHEN status = 'approved' THEN 1 ELSE 0 END), 0) AS approved_count,
			COALESCE(SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END), 0) AS rejected_count,
			COALESCE(AVG(CASE WHEN status IN ('approved','printing','completed') THEN profit_amount * 100.0 / NULLIF(total_cost, 0) END), 0) AS avg_profit_margin
		FROM budgets
		WHERE organization_id = ? AND deleted_at IS NULL`
		params := []any{organizationID}
		sql, params = budgetDateFilter(sql, params, start, end)

		if err := r.db.Raw(sql, params...).Row().Scan(
			&m.TotalBudgets, &m.TotalRevenue, &m.RevenueCount,
			&m.ApprovedCount, &m.RejectedCount, &m.AvgProfitMargin,
		); err != nil {
			return nil, err
		}
	}

	// New customers (separate table, cannot merge with budgets query)
	{
		sql := "SELECT COUNT(*) FROM customers WHERE organization_id = ? AND deleted_at IS NULL"
		params := []any{organizationID}
		if !isAllPeriod(start) {
			sql += " AND created_at >= ? AND created_at < ?"
			params = append(params, start, end)
		}
		if err := r.db.Raw(sql, params...).Row().Scan(&m.NewCustomers); err != nil {
			return nil, err
		}
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
			previous = &overviewMetrics{}
			return nil
		}
		var err error
		previous, err = r.queryOverviewMetrics(organizationID, prevStart, prevEnd)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var avgTicket int64
	if current.RevenueCount > 0 {
		avgTicket = current.TotalRevenue / current.RevenueCount
	}
	var prevAvgTicket float64
	if previous.RevenueCount > 0 {
		prevAvgTicket = float64(previous.TotalRevenue) / float64(previous.RevenueCount)
	}

	var approvalRate float64
	decided := current.ApprovedCount + current.RejectedCount
	if decided > 0 {
		approvalRate = float64(current.ApprovedCount) / float64(decided) * 100
	}
	var prevApprovalRate float64
	prevDecided := previous.ApprovedCount + previous.RejectedCount
	if prevDecided > 0 {
		prevApprovalRate = float64(previous.ApprovedCount) / float64(prevDecided) * 100
	}

	// Budgets by status
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
		budgetsByStatus = append(budgetsByStatus, entities.BudgetStatusCount{
			Status: row.Status,
			Count:  row.Count,
		})
	}

	return &entities.OverviewResponse{
		TotalRevenue:       current.TotalRevenue,
		RevenueChange:      percentChange(float64(current.TotalRevenue), float64(previous.TotalRevenue)),
		TotalBudgets:       current.TotalBudgets,
		BudgetsChange:      percentChange(float64(current.TotalBudgets), float64(previous.TotalBudgets)),
		AvgTicket:          avgTicket,
		AvgTicketChange:    percentChange(float64(avgTicket), prevAvgTicket),
		ApprovalRate:       approvalRate,
		ApprovalRateChange: percentChange(approvalRate, prevApprovalRate),
		AvgProfitMargin:    current.AvgProfitMargin,
		ProfitMarginChange: percentChange(current.AvgProfitMargin, previous.AvgProfitMargin),
		NewCustomers:       current.NewCustomers,
		NewCustomersChange: percentChange(float64(current.NewCustomers), float64(previous.NewCustomers)),
		BudgetsByStatus:    budgetsByStatus,
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

	sql := `SELECT to_char(date_trunc(?, created_at), ?) AS date,
	               COALESCE(SUM(total_cost), 0) AS revenue,
	               COALESCE(SUM(total_cost - profit_amount), 0) AS cost,
	               COALESCE(SUM(profit_amount), 0) AS profit,
	               COUNT(*) AS budget_count
	        FROM budgets
	        WHERE organization_id = ?
	          AND status IN ('approved','printing','completed')
	          AND deleted_at IS NULL`
	params := []any{truncate, dateFormat, organizationID}

	if !isAllPeriod(start) {
		sql += " AND created_at >= ? AND created_at < ?"
		params = append(params, start, end)
	}

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

type opsMetrics struct {
	AvgTicket       int64
	RevenueCount    int64
	AvgProfitMargin float64
	TotalPrintMins  float64
	RejectedCount   int
	DecidedCount    int
	FilamentCost    int64
	WasteCost       int64
	EnergyCost      int64
	SetupCost       int64
	LaborCost       int64
	OverheadCost    int64
}

func (r *DashboardRepositoryImpl) queryOpsMetrics(organizationID string, start, end time.Time) (*opsMetrics, error) {
	m := &opsMetrics{}

	// Single query merging revenue stats, cost breakdown, and rejection rate
	// using conditional aggregation on the full budgets table.
	// Uses idx_budgets_org_created for the base scan.
	{
		sql := `SELECT
			COALESCE(SUM(CASE WHEN status IN ('approved','printing','completed') THEN total_cost ELSE 0 END), 0) AS total_revenue,
			COALESCE(SUM(CASE WHEN status IN ('approved','printing','completed') THEN 1 ELSE 0 END), 0) AS revenue_count,
			COALESCE(AVG(CASE WHEN status IN ('approved','printing','completed') THEN profit_amount * 100.0 / NULLIF(total_cost, 0) END), 0) AS avg_profit_margin,
			COALESCE(SUM(CASE WHEN status IN ('approved','printing','completed') THEN filament_cost ELSE 0 END), 0) AS filament_cost,
			COALESCE(SUM(CASE WHEN status IN ('approved','printing','completed') THEN waste_cost ELSE 0 END), 0) AS waste_cost,
			COALESCE(SUM(CASE WHEN status IN ('approved','printing','completed') THEN energy_cost ELSE 0 END), 0) AS energy_cost,
			COALESCE(SUM(CASE WHEN status IN ('approved','printing','completed') THEN setup_cost ELSE 0 END), 0) AS setup_cost,
			COALESCE(SUM(CASE WHEN status IN ('approved','printing','completed') THEN labor_cost ELSE 0 END), 0) AS labor_cost,
			COALESCE(SUM(CASE WHEN status IN ('approved','printing','completed') THEN overhead_cost ELSE 0 END), 0) AS overhead_cost,
			COALESCE(SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END), 0) AS rejected_count,
			COALESCE(SUM(CASE WHEN status IN ('approved','rejected') THEN 1 ELSE 0 END), 0) AS decided_count
		FROM budgets
		WHERE organization_id = ? AND deleted_at IS NULL`
		params := []any{organizationID}
		sql, params = budgetDateFilter(sql, params, start, end)

		var totalRevenue int64
		if err := r.db.Raw(sql, params...).Row().Scan(
			&totalRevenue, &m.RevenueCount, &m.AvgProfitMargin,
			&m.FilamentCost, &m.WasteCost, &m.EnergyCost,
			&m.SetupCost, &m.LaborCost, &m.OverheadCost,
			&m.RejectedCount, &m.DecidedCount,
		); err != nil {
			return nil, err
		}
		if m.RevenueCount > 0 {
			m.AvgTicket = totalRevenue / m.RevenueCount
		}
	}

	// Print time from budget_items (requires JOIN, kept as separate query)
	{
		sql := `SELECT COALESCE(SUM(bi.print_time_hours * 60 + bi.print_time_minutes), 0)
		        FROM budget_items bi
		        JOIN budgets b ON b.id = bi.budget_id AND b.deleted_at IS NULL
		        WHERE b.organization_id = ?
		          AND b.status IN ('approved','printing','completed')`
		params := []any{organizationID}
		if !isAllPeriod(start) {
			sql += " AND b.created_at >= ? AND b.created_at < ?"
			params = append(params, start, end)
		}
		if err := r.db.Raw(sql, params...).Row().Scan(&m.TotalPrintMins); err != nil {
			return nil, err
		}
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
			previous = &opsMetrics{}
			return nil
		}
		var err error
		previous, err = r.queryOpsMetrics(organizationID, prevStart, prevEnd)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var prevAvgTicket float64
	if previous.RevenueCount > 0 {
		prevAvgTicket = float64(previous.AvgTicket)
	}

	currentPrintHours := current.TotalPrintMins / 60.0
	prevPrintHours := previous.TotalPrintMins / 60.0

	var rejectionRate float64
	if current.DecidedCount > 0 {
		rejectionRate = float64(current.RejectedCount) / float64(current.DecidedCount) * 100
	}
	var prevRejectionRate float64
	if previous.DecidedCount > 0 {
		prevRejectionRate = float64(previous.RejectedCount) / float64(previous.DecidedCount) * 100
	}

	// Cost breakdown percentages
	grandTotal := float64(current.FilamentCost + current.WasteCost + current.EnergyCost + current.SetupCost + current.LaborCost + current.OverheadCost)
	var breakdown entities.CostBreakdown
	if grandTotal > 0 {
		breakdown = entities.CostBreakdown{
			FilamentPct: float64(current.FilamentCost) / grandTotal * 100,
			WastePct:    float64(current.WasteCost) / grandTotal * 100,
			EnergyPct:   float64(current.EnergyCost) / grandTotal * 100,
			SetupPct:    float64(current.SetupCost) / grandTotal * 100,
			LaborPct:    float64(current.LaborCost) / grandTotal * 100,
			OverheadPct: float64(current.OverheadCost) / grandTotal * 100,
		}
	}

	return &entities.OperationalInsightsResponse{
		AvgTicket:           current.AvgTicket,
		AvgTicketChange:     percentChange(float64(current.AvgTicket), prevAvgTicket),
		AvgProfitMargin:     current.AvgProfitMargin,
		ProfitMarginChange:  percentChange(current.AvgProfitMargin, previous.AvgProfitMargin),
		TotalPrintTimeHours: currentPrintHours,
		PrintTimeChange:     percentChange(currentPrintHours, prevPrintHours),
		RejectionRate:       rejectionRate,
		RejectionRateChange: percentChange(rejectionRate, prevRejectionRate),
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
