package repositories

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
	"golang.org/x/sync/errgroup"
)

// profitRowScan is the shape every profitability query returns.
type profitRowScan struct {
	ID        string
	Name      string
	Subtitle  string
	ColorHex  string
	Revenue   float64
	Profit    float64
	Count     int
	Grams     float64
	Minutes   float64
	Discount  float64
	SalesEver int
}

func (p profitRowScan) toRow() entities.ProfitRow {
	row := entities.ProfitRow{
		ID:       p.ID,
		Name:     p.Name,
		Subtitle: p.Subtitle,
		ColorHex: p.ColorHex,
		Revenue:  int64(p.Revenue),
		Profit:   int64(p.Profit),
		Count:    p.Count,
		Grams:    p.Grams,
		Hours:    p.Minutes / 60,
		Repeat:   p.SalesEver > 1,
	}
	if p.Revenue != 0 {
		row.Margin = p.Profit / p.Revenue * 100
	}
	if p.Minutes > 0 {
		row.ProfitPerHour = int64(p.Profit / (p.Minutes / 60))
	}
	if gross := p.Revenue + p.Discount; p.Discount > 0 && gross > 0 {
		row.DiscountRate = p.Discount / gross * 100
	}
	return row
}

// salesCTE selects the period's sales with net revenue and profit, plus each
// item's share of the budget cost and each filament's share of the item grams.
// Allocation: budget → items by item_total_cost (equal split when every item
// costs zero), item → filaments by grams. Items without filaments (services)
// have no material, so material/filament totals can be below the sales total.
const salesCTE = `WITH sales AS (
	SELECT b.id, b.customer_id, b.machine_preset_id, b.cost_preset_id, b.discount_amount,
	       ` + netRevenueExpr + ` AS net, ` + profitExpr + ` AS profit
	FROM budgets b
	WHERE b.organization_id = ? AND b.deleted_at IS NULL AND b.status IN ` + saleStatuses + ` %s
), items AS (
	SELECT bi.id, bi.budget_id,
	       COALESCE(bi.item_total_cost::float / NULLIF(SUM(bi.item_total_cost) OVER (PARTITION BY bi.budget_id), 0),
	                1.0 / COUNT(*) OVER (PARTITION BY bi.budget_id)) AS share
	FROM budget_items bi JOIN sales s ON s.id = bi.budget_id
), fils AS (
	SELECT bif.budget_item_id, bif.filament_id, bif.quantity::float AS grams,
	       COALESCE(bif.quantity::float / NULLIF(SUM(bif.quantity) OVER (PARTITION BY bif.budget_item_id), 0), 0) AS share
	FROM budget_item_filaments bif JOIN items i ON i.id = bif.budget_item_id
)`

func (r *DashboardRepositoryImpl) profitQuery(organizationID string, start, end time.Time, body string, limit int) ([]entities.ProfitRow, error) {
	window := ""
	params := []any{organizationID}
	if !isAllPeriod(start) {
		window = "AND b.approved_at >= ? AND b.approved_at < ?"
		params = append(params, start, end)
	}
	sql := sprintfSafe(salesCTE, window) + "\n" + body + " LIMIT ?"
	params = append(params, limit)
	var scans []profitRowScan
	if err := r.db.Raw(sql, params...).Scan(&scans).Error; err != nil {
		return nil, err
	}
	rows := make([]entities.ProfitRow, 0, len(scans))
	for _, s := range scans {
		rows = append(rows, s.toRow())
	}
	return rows, nil
}

// GetProfitability returns the period's profit by material, filament, customer,
// machine and cost preset.
func (r *DashboardRepositoryImpl) GetProfitability(organizationID string, start, end time.Time, limit int) (*entities.ProfitabilityResponse, error) {
	resp := &entities.ProfitabilityResponse{}
	g := new(errgroup.Group)

	g.Go(func() (err error) {
		resp.ByMaterial, err = r.profitQuery(organizationID, start, end, `
		SELECT COALESCE(m.id::text, '') AS id, COALESCE(m.name, 'Sem material') AS name,
		       SUM(s.net * i.share * f.share) AS revenue, SUM(s.profit * i.share * f.share) AS profit,
		       COUNT(DISTINCT f.budget_item_id) AS count, SUM(f.grams) AS grams
		FROM fils f
		JOIN items i ON i.id = f.budget_item_id
		JOIN sales s ON s.id = i.budget_id
		JOIN filaments fl ON fl.id = f.filament_id
		LEFT JOIN materials m ON m.id = fl.material_id
		GROUP BY m.id, m.name
		ORDER BY profit DESC`, limit)
		return err
	})
	g.Go(func() (err error) {
		resp.ByFilament, err = r.profitQuery(organizationID, start, end, `
		SELECT fl.id::text AS id, fl.name AS name,
		       TRIM(BOTH ' ·' FROM CONCAT_WS(' · ', NULLIF(fl.color, ''), m.name)) AS subtitle,
		       COALESCE(fl.color_hex, '') AS color_hex,
		       SUM(s.net * i.share * f.share) AS revenue, SUM(s.profit * i.share * f.share) AS profit,
		       COUNT(DISTINCT f.budget_item_id) AS count, SUM(f.grams) AS grams
		FROM fils f
		JOIN items i ON i.id = f.budget_item_id
		JOIN sales s ON s.id = i.budget_id
		JOIN filaments fl ON fl.id = f.filament_id
		LEFT JOIN materials m ON m.id = fl.material_id
		GROUP BY fl.id, fl.name, fl.color, fl.color_hex, m.name
		ORDER BY profit DESC`, limit)
		return err
	})
	g.Go(func() (err error) {
		resp.ByCustomer, err = r.profitQuery(organizationID, start, end, `
		SELECT c.id::text AS id, c.name AS name, COALESCE(c.email, '') AS subtitle,
		       SUM(s.net) AS revenue, SUM(s.profit) AS profit, COUNT(*) AS count,
		       SUM(s.discount_amount) AS discount,
		       (SELECT COUNT(*) FROM budgets b2
		        WHERE b2.customer_id = c.id AND b2.deleted_at IS NULL
		          AND b2.status IN `+saleStatuses+`) AS sales_ever
		FROM sales s JOIN customers c ON c.id = s.customer_id
		GROUP BY c.id, c.name, c.email
		ORDER BY profit DESC`, limit)
		return err
	})
	g.Go(func() (err error) {
		resp.ByMachine, err = r.profitQuery(organizationID, start, end, `
		SELECT COALESCE(p.id::text, '') AS id, COALESCE(p.name, 'Sem máquina') AS name,
		       SUM(s.net) AS revenue, SUM(s.profit) AS profit, COUNT(*) AS count,
		       SUM((SELECT COALESCE(SUM(bi.print_time_hours * 60 + bi.print_time_minutes), 0)
		            FROM budget_items bi WHERE bi.budget_id = s.id)) AS minutes
		FROM sales s LEFT JOIN presets p ON p.id = s.machine_preset_id
		GROUP BY p.id, p.name
		ORDER BY profit DESC`, limit)
		return err
	})
	g.Go(func() (err error) {
		resp.ByCostPreset, err = r.profitQuery(organizationID, start, end, `
		SELECT COALESCE(p.id::text, '') AS id, COALESCE(p.name, 'Sem preset de custos') AS name,
		       SUM(s.net) AS revenue, SUM(s.profit) AS profit, COUNT(*) AS count
		FROM sales s LEFT JOIN presets p ON p.id = s.cost_preset_id
		GROUP BY p.id, p.name
		ORDER BY profit DESC`, limit)
		return err
	})
	g.Go(func() error {
		sales, err := r.querySales(organizationID, start, end)
		if err != nil {
			return err
		}
		resp.AverageMargin = sales.margin()
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return resp, nil
}
