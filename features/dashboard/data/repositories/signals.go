package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
	"golang.org/x/sync/errgroup"
)

const (
	staleDraftAfter      = 7 * 24 * time.Hour
	inactiveCustomerDays = 60
	stockLookbackDays    = 30
	signalListLimit      = 3
)

// countStaleDrafts counts drafts not touched for staleDraftAfter
// (uses idx_budgets_org_status_updated).
func (r *DashboardRepositoryImpl) countStaleDrafts(organizationID string, now time.Time) *sql.Row {
	return r.db.Raw(`SELECT COUNT(*) FROM budgets
		WHERE organization_id = ? AND status = 'draft' AND updated_at < ? AND deleted_at IS NULL`,
		organizationID, now.Add(-staleDraftAfter)).Row()
}

// GetInsightSignals loads the aggregates the insight rules need beyond the
// other dashboard endpoints. Waste is measured on sales approved in [start, end).
func (r *DashboardRepositoryImpl) GetInsightSignals(organizationID string, start, end, now time.Time) (*entities.InsightSignals, error) {
	s := &entities.InsightSignals{}
	g := new(errgroup.Group)

	g.Go(func() error { return r.countStaleDrafts(organizationID, now).Scan(&s.StaleDrafts) })

	// Repeat customers (2+ sales ever) with no budget created recently.
	g.Go(func() error {
		type row struct {
			ID    string
			Name  string
			Total int
		}
		var rows []row
		err := r.db.Raw(`WITH repeat AS (
				SELECT b.customer_id FROM budgets b
				WHERE b.organization_id = ? AND b.deleted_at IS NULL AND b.status IN `+saleStatuses+`
				GROUP BY b.customer_id HAVING COUNT(*) >= 2
			)
			SELECT c.id::text AS id, c.name, COUNT(*) OVER () AS total
			FROM customers c JOIN repeat ON repeat.customer_id = c.id
			WHERE c.deleted_at IS NULL AND NOT EXISTS (
				SELECT 1 FROM budgets b WHERE b.customer_id = c.id AND b.deleted_at IS NULL AND b.created_at >= ?)
			ORDER BY (SELECT MAX(b.created_at) FROM budgets b WHERE b.customer_id = c.id) DESC
			LIMIT ?`,
			organizationID, now.AddDate(0, 0, -inactiveCustomerDays), signalListLimit).Scan(&rows).Error
		if err != nil {
			return fmt.Errorf("failed to query inactive customers: %w", err)
		}
		for _, row := range rows {
			s.InactiveRepeatCustomers = append(s.InactiveRepeatCustomers, entities.NamedRef{ID: row.ID, Name: row.Name})
			s.InactiveRepeatCustomersTotal = row.Total
		}
		return nil
	})

	g.Go(func() error {
		query := `SELECT COALESCE(SUM(b.waste_cost), 0), COALESCE(SUM(b.filament_cost), 0)
			FROM budgets b WHERE b.organization_id = ? AND b.deleted_at IS NULL AND b.status IN ` + saleStatuses
		params := []any{organizationID}
		query, params = saleWindow(query, params, start, end)
		return r.db.Raw(query, params...).Row().Scan(&s.WasteCost, &s.FilamentCost)
	})

	// Stock-tracked filaments whose balance does not cover 30 days of sales.
	g.Go(func() error {
		err := r.db.Raw(`SELECT f.id::text AS id,
				TRIM(f.name || ' ' || COALESCE(f.color, '')) AS name,
				f.stock_grams, used.grams AS consumed_grams30
			FROM filaments f
			JOIN (
				SELECT bif.filament_id, SUM(bif.quantity) AS grams
				FROM budget_item_filaments bif
				JOIN budget_items bi ON bi.id = bif.budget_item_id
				JOIN budgets b ON b.id = bi.budget_id
				WHERE b.organization_id = ? AND b.deleted_at IS NULL AND b.status IN `+saleStatuses+`
				  AND b.approved_at >= ?
				GROUP BY bif.filament_id
			) used ON used.filament_id = f.id
			WHERE f.organization_id = ? AND f.deleted_at IS NULL AND f.track_stock = true
			  AND f.stock_grams < used.grams
			ORDER BY (f.stock_grams - used.grams) ASC
			LIMIT ?`,
			organizationID, now.AddDate(0, 0, -stockLookbackDays), organizationID, signalListLimit).
			Scan(&s.StockShortfall).Error
		if err != nil {
			return fmt.Errorf("failed to query stock shortfall: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return s, nil
}
