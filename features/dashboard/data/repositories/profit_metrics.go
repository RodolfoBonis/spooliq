package repositories

import (
	"strings"
	"time"
)

// Profit definitions (single source of truth for every dashboard endpoint):
//   - a sale is a budget in approved/printing/completed, dated by approved_at;
//   - net revenue = total_cost - tax_amount - shipping_cost (tax and shipping are
//     pass-through, not business revenue);
//   - profit = profit_amount - discount_amount (the markup minus what was given away);
//   - production cost = net revenue - profit (direct costs + overhead);
//   - margin = sum(profit) / sum(net revenue), i.e. weighted by value.
const (
	saleStatuses   = "('approved','printing','completed')"
	netRevenueExpr = "(b.total_cost - b.tax_amount - b.shipping_cost)"
	profitExpr     = "(b.profit_amount - b.discount_amount)"
)

// saleWindow filters budgets aliased "b" by approval date (no filter for "all").
func saleWindow(sql string, params []any, start, end time.Time) (string, []any) {
	if !isAllPeriod(start) {
		sql += " AND b.approved_at >= ? AND b.approved_at < ?"
		params = append(params, start, end)
	}
	return sql, params
}

type salesMetrics struct {
	GrossRevenue   int64
	NetRevenue     int64
	Profit         int64
	ProfitRealized int64
	SalesCount     int64
	PrintMinutes   int64
}

func (m *salesMetrics) margin() float64 {
	if m.NetRevenue == 0 {
		return 0
	}
	return float64(m.Profit) / float64(m.NetRevenue) * 100
}

func (m *salesMetrics) printHours() float64 { return float64(m.PrintMinutes) / 60 }

func (m *salesMetrics) profitPerHour() int64 {
	if m.PrintMinutes == 0 {
		return 0
	}
	return int64(float64(m.Profit) / m.printHours())
}

func (r *DashboardRepositoryImpl) querySales(organizationID string, start, end time.Time) (*salesMetrics, error) {
	m := &salesMetrics{}
	sql := `SELECT
		COALESCE(SUM(b.total_cost), 0),
		COALESCE(SUM(` + netRevenueExpr + `), 0),
		COALESCE(SUM(` + profitExpr + `), 0),
		COALESCE(SUM(CASE WHEN b.status = 'completed' THEN ` + profitExpr + ` ELSE 0 END), 0),
		COUNT(*),
		COALESCE(SUM((SELECT COALESCE(SUM(bi.print_time_hours * 60 + bi.print_time_minutes), 0)
		              FROM budget_items bi WHERE bi.budget_id = b.id)), 0)
	FROM budgets b
	WHERE b.organization_id = ? AND b.deleted_at IS NULL AND b.status IN ` + saleStatuses
	params := []any{organizationID}
	sql, params = saleWindow(sql, params, start, end)
	if err := r.db.Raw(sql, params...).Row().Scan(
		&m.GrossRevenue, &m.NetRevenue, &m.Profit, &m.ProfitRealized, &m.SalesCount, &m.PrintMinutes,
	); err != nil {
		return nil, err
	}
	return m, nil
}

// decisionMetrics counts customer decisions in the window from the status
// history: a budget reaching approved, rejected or expired. Budgets that moved
// on to printing/completed still count as approved (the old rate dropped them).
type decisionMetrics struct {
	Approved int
	Rejected int
	Expired  int
}

func (d *decisionMetrics) decided() int { return d.Approved + d.Rejected + d.Expired }

func (d *decisionMetrics) approvalRate() float64 {
	if d.decided() == 0 {
		return 0
	}
	return float64(d.Approved) / float64(d.decided()) * 100
}

func (d *decisionMetrics) rate(n int) float64 {
	if d.decided() == 0 {
		return 0
	}
	return float64(n) / float64(d.decided()) * 100
}

func (r *DashboardRepositoryImpl) queryDecisions(organizationID string, start, end time.Time) (*decisionMetrics, error) {
	d := &decisionMetrics{}
	// Only the latest decision per budget in the window counts, so a budget
	// approved and later rejected (after reopening) is counted once.
	window := ""
	params := []any{organizationID}
	if !isAllPeriod(start) {
		window = " AND h.created_at >= ? AND h.created_at < ?"
		params = append(params, start, end)
	}
	sql := `SELECT
		COUNT(*) FILTER (WHERE new_status = 'approved'),
		COUNT(*) FILTER (WHERE new_status = 'rejected'),
		COUNT(*) FILTER (WHERE new_status = 'expired')
	FROM (
		SELECT DISTINCT ON (h.budget_id) h.new_status
		FROM budget_status_history h
		JOIN budgets b ON b.id = h.budget_id AND b.deleted_at IS NULL
		WHERE h.organization_id = ? AND h.new_status IN ('approved','rejected','expired')` + window + `
		ORDER BY h.budget_id, h.created_at DESC
	) latest`
	if err := r.db.Raw(sql, params...).Row().Scan(&d.Approved, &d.Rejected, &d.Expired); err != nil {
		return nil, err
	}
	return d, nil
}

// sprintfSafe injects a trusted SQL fragment (never user input) into a template
// with a single %s placeholder.
func sprintfSafe(template, fragment string) string {
	return strings.Replace(template, "%s", fragment, 1)
}
