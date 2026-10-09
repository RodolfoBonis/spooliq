package repositories

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
	"golang.org/x/sync/errgroup"
)

// responseDurationsCTE pairs each decision in the window with the budget's first
// "sent" event before it; durations are in hours.
const responseDurationsCTE = `WITH decisions AS (
	SELECT h.budget_id, h.new_status, h.created_at AS decided_at,
	       (SELECT MIN(s.created_at) FROM budget_status_history s
	        WHERE s.budget_id = h.budget_id AND s.new_status = 'sent' AND s.created_at <= h.created_at) AS sent_at
	FROM budget_status_history h
	JOIN budgets b ON b.id = h.budget_id AND b.deleted_at IS NULL
	WHERE h.organization_id = ? AND h.new_status IN ('approved','rejected') %s
), durations AS (
	SELECT new_status, EXTRACT(EPOCH FROM (decided_at - sent_at)) / 3600.0 AS hours
	FROM decisions WHERE sent_at IS NOT NULL
)`

// GetResponseTimes returns how long customers take to decide, expirations and
// recent rejection reasons.
func (r *DashboardRepositoryImpl) GetResponseTimes(organizationID string, start, end time.Time) (*entities.ResponseTimesResponse, error) {
	resp := &entities.ResponseTimesResponse{
		ApprovalBuckets:  []entities.ResponseBucket{},
		RecentRejections: []entities.RejectionReason{},
		ExpiringSoon:     []entities.ExpiringBudget{},
	}
	window := ""
	params := []any{organizationID}
	if !isAllPeriod(start) {
		window = "AND h.created_at >= ? AND h.created_at < ?"
		params = append(params, start, end)
	}
	cte := sprintfSafe(responseDurationsCTE, window)

	g := new(errgroup.Group)
	g.Go(func() error {
		sql := cte + `
		SELECT
			COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY hours) FILTER (WHERE new_status = 'approved'), 0),
			COALESCE(percentile_cont(0.75) WITHIN GROUP (ORDER BY hours) FILTER (WHERE new_status = 'approved'), 0),
			COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY hours) FILTER (WHERE new_status = 'rejected'), 0),
			COUNT(*) FILTER (WHERE new_status = 'approved' AND hours < 24),
			COUNT(*) FILTER (WHERE new_status = 'approved' AND hours >= 24 AND hours < 72),
			COUNT(*) FILTER (WHERE new_status = 'approved' AND hours >= 72 AND hours < 168),
			COUNT(*) FILTER (WHERE new_status = 'approved' AND hours >= 168)
		FROM durations`
		var b1, b2, b3, b4 int
		if err := r.db.Raw(sql, params...).Row().Scan(
			&resp.ApprovalMedianHours, &resp.ApprovalP75Hours, &resp.RejectionMedianHours,
			&b1, &b2, &b3, &b4,
		); err != nil {
			return err
		}
		resp.ApprovalBuckets = []entities.ResponseBucket{
			{Label: "Menos de 1 dia", Count: b1},
			{Label: "1 a 3 dias", Count: b2},
			{Label: "3 a 7 dias", Count: b3},
			{Label: "Mais de 7 dias", Count: b4},
		}
		return nil
	})
	g.Go(func() error {
		d, err := r.queryDecisions(organizationID, start, end)
		if err != nil {
			return err
		}
		resp.Approved, resp.Rejected, resp.Expired = d.Approved, d.Rejected, d.Expired
		resp.ExpirationRate = d.rate(d.Expired)
		resp.RejectionRate = d.rate(d.Rejected)
		return nil
	})
	g.Go(func() error {
		sql := `SELECT b.id::text AS budget_id, b.name AS budget_name, b.quote_number,
		               COALESCE(b.customer_response_name, c.name, '') AS customer,
		               b.rejection_reason AS reason, b.customer_response_at AS at
		        FROM budgets b LEFT JOIN customers c ON c.id = b.customer_id
		        WHERE b.organization_id = ? AND b.deleted_at IS NULL
		          AND b.rejection_reason IS NOT NULL AND b.rejection_reason <> ''
		          AND b.customer_response_at IS NOT NULL`
		p := []any{organizationID}
		if !isAllPeriod(start) {
			sql += " AND b.customer_response_at >= ? AND b.customer_response_at < ?"
			p = append(p, start, end)
		}
		sql += " ORDER BY b.customer_response_at DESC LIMIT 5"
		return r.db.Raw(sql, p...).Scan(&resp.RecentRejections).Error
	})
	g.Go(func() error {
		// Not period-bound: what needs a follow-up right now.
		sql := `SELECT b.id::text AS budget_id, b.name AS budget_name, b.quote_number,
		               COALESCE(c.name, '') AS customer, b.valid_until, b.total_cost AS total
		        FROM budgets b LEFT JOIN customers c ON c.id = b.customer_id
		        WHERE b.organization_id = ? AND b.deleted_at IS NULL AND b.status = 'sent'
		          AND b.valid_until >= now() AND b.valid_until < now() + interval '2 days'
		        ORDER BY b.valid_until LIMIT 10`
		return r.db.Raw(sql, organizationID).Scan(&resp.ExpiringSoon).Error
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return resp, nil
}
