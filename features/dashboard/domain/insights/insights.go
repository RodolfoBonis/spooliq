// Package insights turns dashboard aggregates into ranked, actionable insights
// using deterministic rules (no ML). Every rule is a pure function so it can be
// tested with plain fixtures.
package insights

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
)

// Limit caps how many insights are returned.
const Limit = 8

// Thresholds used by the rules. Kept together so they are easy to tune.
const (
	marginDropPoints       = 3.0  // margin fell by more than this (p.p.)
	lowMarginRatio         = 0.7  // material margin below 70% of the average
	relevantRevenueShare   = 0.10 // a row matters when it is 10%+ of revenue
	highDiscountRate       = 15.0 // customer average discount above 15%
	customerRevenueShare   = 0.05
	idleMachineRatio       = 0.6 // machine profit/hour below 60% of the best
	minMachineHours        = 5.0
	slowApprovalHours      = 72.0 // median send→approval above 3 days
	highExpirationRate     = 20.0
	highRejectionRate      = 30.0
	minDecisions           = 5
	highWasteRate          = 8.0 // waste above 8% of filament cost
	goalOffPaceRatio       = 0.9 // projection below 90% of the target
	profitGrowthHighlight  = 10.0
	minSalesForComparisons = 3
)

// Input aggregates everything the rules read. Nil sections are skipped, so a
// failing source degrades the insights instead of failing the request.
type Input struct {
	Overview      *entities.OverviewResponse
	CostNow       *entities.CostBreakdown
	CostPrev      *entities.CostBreakdown
	Profitability *entities.ProfitabilityResponse
	ResponseTimes *entities.ResponseTimesResponse
	Signals       *entities.InsightSignals
	Goals         []entities.Goal
}

type rule func(Input) []entities.Insight

var rules = []rule{
	goalsOffPace,
	marginDrop,
	lowMarginMaterial,
	highDiscountCustomer,
	idleMachine,
	expiringSoon,
	slowResponse,
	highExpiration,
	highRejection,
	stockShortfall,
	highWaste,
	staleDrafts,
	inactiveCustomers,
	profitGrowth,
}

// Generate runs every rule and returns the most relevant insights: by
// severity, then by money at stake, capped at Limit.
func Generate(in Input) []entities.Insight {
	out := make([]entities.Insight, 0, Limit)
	for _, r := range rules {
		out = append(out, r(in)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if ri, rj := out[i].Severity.Rank(), out[j].Severity.Rank(); ri != rj {
			return ri < rj
		}
		return out[i].Impact > out[j].Impact
	})
	if len(out) > Limit {
		out = out[:Limit]
	}
	return out
}

func goalsOffPace(in Input) []entities.Insight {
	var out []entities.Insight
	for _, g := range in.Goals {
		if !g.Configured || g.Progress >= 100 {
			continue
		}
		switch {
		case g.Metric.Cumulative() && g.DaysLeft > 0 && g.ProjectedProgress < goalOffPaceRatio*100:
			gap := g.Target - g.Projected
			out = append(out, entities.Insight{
				Kind:     "goal_off_pace",
				Severity: entities.SeverityWarning,
				Title:    fmt.Sprintf("Meta de %s fora do ritmo", strings.ToLower(g.Name)),
				Detail: fmt.Sprintf("No ritmo atual o mês fecha em %s da meta (%s de %s). Faltam %d %s: é preciso %s por dia.",
					pct(g.ProjectedProgress), goalValue(g, g.Projected), goalValue(g, g.Target),
					g.DaysLeft, plural(g.DaysLeft, "dia", "dias"), goalValue(g, g.RequiredPerDay)),
				Metric: pct(g.Progress),
				Impact: goalImpact(g, gap),
				Action: &entities.InsightAction{Label: "Ver metas", Target: "goals"},
			})
		case !g.Metric.Cumulative() && g.Current > 0 && g.Current < g.Target*goalOffPaceRatio:
			out = append(out, entities.Insight{
				Kind:     "goal_off_pace",
				Severity: entities.SeverityInfo,
				Title:    "Aprovação abaixo da meta",
				Detail:   fmt.Sprintf("A taxa de aprovação do mês está em %s, abaixo da meta de %s.", pct(g.Current), pct(g.Target)),
				Metric:   pct(g.Current),
				Action:   &entities.InsightAction{Label: "Ver metas", Target: "goals"},
			})
		}
	}
	return out
}

func goalValue(g entities.Goal, v float64) string {
	switch g.Unit {
	case "cents":
		return brl(int64(v))
	case "percent":
		return pct(v)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}

func goalImpact(g entities.Goal, gap float64) int64 {
	if g.Unit == "cents" {
		return int64(gap)
	}
	return 0
}

func marginDrop(in Input) []entities.Insight {
	o := in.Overview
	if o == nil || o.SalesCount < minSalesForComparisons || o.ProfitMarginPointsChange > -marginDropPoints {
		return nil
	}
	detail := fmt.Sprintf("A margem caiu %s p.p. em relação ao período anterior.", num(-o.ProfitMarginPointsChange))
	if driver, delta := biggestCostIncrease(in.CostNow, in.CostPrev); driver != "" {
		detail += fmt.Sprintf(" O maior aumento foi em %s (+%s p.p. do custo).", driver, num(delta))
	}
	detail += " Revise os presets de custo ou os preços."
	return []entities.Insight{{
		Kind:     "margin_drop",
		Severity: entities.SeverityWarning,
		Title:    "Margem caindo",
		Detail:   detail,
		Metric:   pct(o.ProfitMargin),
		Impact:   int64(-o.ProfitMarginPointsChange / 100 * float64(o.NetRevenue)),
		Action:   &entities.InsightAction{Label: "Revisar custos", Target: "costs"},
	}}
}

// biggestCostIncrease returns the cost component whose share grew the most.
func biggestCostIncrease(now, prev *entities.CostBreakdown) (string, float64) {
	if now == nil || prev == nil {
		return "", 0
	}
	parts := []struct {
		name      string
		now, prev float64
	}{
		{"filamento", now.FilamentPct, prev.FilamentPct},
		{"desperdício", now.WastePct, prev.WastePct},
		{"energia", now.EnergyPct, prev.EnergyPct},
		{"máquina", now.MachinePct, prev.MachinePct},
		{"setup", now.SetupPct, prev.SetupPct},
		{"mão de obra", now.LaborPct, prev.LaborPct},
		{"pós-processamento", now.PostProcessingPct, prev.PostProcessingPct},
		{"embalagem", now.PackagingPct, prev.PackagingPct},
		{"controle de qualidade", now.QualityControlPct, prev.QualityControlPct},
		{"falhas", now.FailurePct, prev.FailurePct},
		{"overhead", now.OverheadPct, prev.OverheadPct},
	}
	best, delta := "", 1.0 // ignore noise below 1 p.p.
	for _, p := range parts {
		if d := p.now - p.prev; d > delta {
			best, delta = p.name, d
		}
	}
	if best == "" {
		return "", 0
	}
	return best, delta
}

func totalRevenue(rows []entities.ProfitRow) int64 {
	var total int64
	for _, r := range rows {
		total += r.Revenue
	}
	return total
}

func lowMarginMaterial(in Input) []entities.Insight {
	p := in.Profitability
	if p == nil || len(p.ByMaterial) < 2 || p.AverageMargin <= 0 {
		return nil
	}
	total := totalRevenue(p.ByMaterial)
	var out []entities.Insight
	for _, m := range p.ByMaterial {
		if float64(m.Revenue) < relevantRevenueShare*float64(total) || m.Margin >= lowMarginRatio*p.AverageMargin {
			continue
		}
		out = append(out, entities.Insight{
			Kind:     "low_margin_material",
			Severity: entities.SeverityWarning,
			Title:    fmt.Sprintf("%s rende pouco", m.Name),
			Detail: fmt.Sprintf("A margem com %s é %s, contra %s na média. Revise o preço do filamento ou o markup dos orçamentos com esse material.",
				m.Name, pct(m.Margin), pct(p.AverageMargin)),
			Metric: pct(m.Margin),
			Impact: int64((p.AverageMargin - m.Margin) / 100 * float64(m.Revenue)),
			Action: &entities.InsightAction{Label: "Ver materiais", Target: "materials", ID: m.ID},
		})
	}
	return out
}

func highDiscountCustomer(in Input) []entities.Insight {
	p := in.Profitability
	if p == nil {
		return nil
	}
	total := totalRevenue(p.ByCustomer)
	var out []entities.Insight
	for _, c := range p.ByCustomer {
		if c.DiscountRate <= highDiscountRate || float64(c.Revenue) < customerRevenueShare*float64(total) {
			continue
		}
		out = append(out, entities.Insight{
			Kind:     "high_discount_customer",
			Severity: entities.SeverityInfo,
			Title:    fmt.Sprintf("Desconto alto para %s", c.Name),
			Detail: fmt.Sprintf("%s recebeu em média %s de desconto (margem de %s). Avalie reduzir o desconto nos próximos orçamentos.",
				c.Name, pct(c.DiscountRate), pct(c.Margin)),
			Metric: pct(c.DiscountRate),
			Impact: int64(c.DiscountRate / 100 * float64(c.Revenue)),
			Action: &entities.InsightAction{Label: "Ver cliente", Target: "customer", ID: c.ID},
		})
	}
	return out
}

func idleMachine(in Input) []entities.Insight {
	p := in.Profitability
	if p == nil || len(p.ByMachine) < 2 {
		return nil
	}
	best := p.ByMachine[0]
	for _, m := range p.ByMachine {
		if m.ProfitPerHour > best.ProfitPerHour {
			best = m
		}
	}
	if best.ProfitPerHour <= 0 {
		return nil
	}
	var out []entities.Insight
	for _, m := range p.ByMachine {
		if m.ID == best.ID || m.Hours < minMachineHours || float64(m.ProfitPerHour) >= idleMachineRatio*float64(best.ProfitPerHour) {
			continue
		}
		out = append(out, entities.Insight{
			Kind:     "low_profit_machine",
			Severity: entities.SeverityInfo,
			Title:    fmt.Sprintf("%s lucra pouco por hora", m.Name),
			Detail: fmt.Sprintf("%s rende %s/h, contra %s/h na %s. Direcione os trabalhos mais rentáveis para a melhor máquina ou revise o custo/hora.",
				m.Name, brl(m.ProfitPerHour), brl(best.ProfitPerHour), best.Name),
			Metric: brl(m.ProfitPerHour) + "/h",
			Impact: int64(float64(best.ProfitPerHour-m.ProfitPerHour) * m.Hours),
			Action: &entities.InsightAction{Label: "Ver máquinas", Target: "machines", ID: m.ID},
		})
	}
	return out
}

func expiringSoon(in Input) []entities.Insight {
	r := in.ResponseTimes
	if r == nil || len(r.ExpiringSoon) == 0 {
		return nil
	}
	var total int64
	names := make([]string, 0, 3)
	for i, b := range r.ExpiringSoon {
		total += b.Total
		if i < 3 {
			names = append(names, b.Customer)
		}
	}
	n := len(r.ExpiringSoon)
	return []entities.Insight{{
		Kind:     "expiring_soon",
		Severity: entities.SeverityWarning,
		Title:    fmt.Sprintf("%d %s vencendo em até 2 dias", n, plural(n, "orçamento", "orçamentos")),
		Detail:   fmt.Sprintf("Faça um follow-up com %s antes que expirem.", joinNames(names, n)),
		Metric:   brl(total),
		Impact:   total,
		Action:   &entities.InsightAction{Label: "Ver enviados", Target: "budgets", Filter: "sent"},
	}}
}

func slowResponse(in Input) []entities.Insight {
	r := in.ResponseTimes
	if r == nil || r.Approved < minSalesForComparisons || r.ApprovalMedianHours <= slowApprovalHours {
		return nil
	}
	return []entities.Insight{{
		Kind:     "slow_response",
		Severity: entities.SeverityInfo,
		Title:    "Clientes demoram para aprovar",
		Detail: fmt.Sprintf("Metade das aprovações leva mais de %s. Um lembrete 1–2 dias após o envio costuma acelerar a decisão.",
			days(r.ApprovalMedianHours)),
		Metric: days(r.ApprovalMedianHours),
		Action: &entities.InsightAction{Label: "Ver enviados", Target: "budgets", Filter: "sent"},
	}}
}

func decided(r *entities.ResponseTimesResponse) int { return r.Approved + r.Rejected + r.Expired }

func highExpiration(in Input) []entities.Insight {
	r := in.ResponseTimes
	if r == nil || decided(r) < minDecisions || r.ExpirationRate <= highExpirationRate {
		return nil
	}
	return []entities.Insight{{
		Kind:     "high_expiration",
		Severity: entities.SeverityWarning,
		Title:    "Muitos orçamentos expirando",
		Detail: fmt.Sprintf("%s dos orçamentos decididos expiraram sem resposta. Aumente a validade ou faça follow-up antes do vencimento.",
			pct(r.ExpirationRate)),
		Metric: pct(r.ExpirationRate),
		Action: &entities.InsightAction{Label: "Ver expirados", Target: "budgets", Filter: "expired"},
	}}
}

func highRejection(in Input) []entities.Insight {
	r := in.ResponseTimes
	if r == nil || decided(r) < minDecisions || r.RejectionRate <= highRejectionRate {
		return nil
	}
	detail := fmt.Sprintf("%s dos orçamentos foram recusados.", pct(r.RejectionRate))
	if reason := topReason(r.RecentRejections); reason != "" {
		detail += fmt.Sprintf(" Motivo mais citado: \"%s\".", reason)
	}
	detail += " Compare preços e prazos com a concorrência."
	return []entities.Insight{{
		Kind:     "high_rejection",
		Severity: entities.SeverityWarning,
		Title:    "Recusas acima do normal",
		Detail:   detail,
		Metric:   pct(r.RejectionRate),
		Action:   &entities.InsightAction{Label: "Ver recusados", Target: "budgets", Filter: "rejected"},
	}}
}

// topReason returns the most frequent rejection reason (case-insensitive),
// falling back to the most recent one when all differ.
func topReason(list []entities.RejectionReason) string {
	counts := map[string]int{}
	first := map[string]string{}
	best, bestCount := "", 0
	for _, r := range list {
		reason := strings.TrimSpace(r.Reason)
		if reason == "" {
			continue
		}
		key := strings.ToLower(reason)
		if _, ok := first[key]; !ok {
			first[key] = reason
		}
		counts[key]++
		if counts[key] > bestCount {
			best, bestCount = key, counts[key]
		}
	}
	return first[best]
}

func stockShortfall(in Input) []entities.Insight {
	s := in.Signals
	if s == nil {
		return nil
	}
	out := make([]entities.Insight, 0, len(s.StockShortfall))
	for _, f := range s.StockShortfall {
		out = append(out, entities.Insight{
			Kind:     "stock_shortfall",
			Severity: entities.SeverityWarning,
			Title:    fmt.Sprintf("Estoque de %s não cobre a demanda", f.Name),
			Detail: fmt.Sprintf("Restam %d g, mas as vendas dos últimos 30 dias consumiram %.0f g. Reponha antes de faltar.",
				f.StockGrams, f.ConsumedGrams30),
			Metric: fmt.Sprintf("%d g", f.StockGrams),
			Action: &entities.InsightAction{Label: "Ver filamento", Target: "filament", ID: f.ID},
		})
	}
	return out
}

func highWaste(in Input) []entities.Insight {
	s := in.Signals
	if s == nil || s.FilamentCost <= 0 {
		return nil
	}
	rate := float64(s.WasteCost) / float64(s.FilamentCost) * 100
	if rate <= highWasteRate {
		return nil
	}
	return []entities.Insight{{
		Kind:     "high_waste",
		Severity: entities.SeverityInfo,
		Title:    "Desperdício alto",
		Detail: fmt.Sprintf("O desperdício equivale a %s do custo de filamento (%s no período). Revise purga, suportes e o fator de desperdício dos presets.",
			pct(rate), brl(s.WasteCost)),
		Metric: pct(rate),
		Impact: s.WasteCost,
		Action: &entities.InsightAction{Label: "Ver presets de custo", Target: "costs"},
	}}
}

func staleDrafts(in Input) []entities.Insight {
	s := in.Signals
	if s == nil || s.StaleDrafts == 0 {
		return nil
	}
	n := s.StaleDrafts
	return []entities.Insight{{
		Kind:     "stale_drafts",
		Severity: entities.SeverityInfo,
		Title:    fmt.Sprintf("%d %s parados", n, plural(n, "rascunho", "rascunhos")),
		Detail:   "Rascunhos sem atualização há mais de 7 dias: envie ao cliente ou descarte.",
		Metric:   fmt.Sprintf("%d", n),
		Action:   &entities.InsightAction{Label: "Ver rascunhos", Target: "budgets", Filter: "draft"},
	}}
}

func inactiveCustomers(in Input) []entities.Insight {
	s := in.Signals
	if s == nil || s.InactiveRepeatCustomersTotal == 0 {
		return nil
	}
	names := make([]string, 0, len(s.InactiveRepeatCustomers))
	for _, c := range s.InactiveRepeatCustomers {
		names = append(names, c.Name)
	}
	n := s.InactiveRepeatCustomersTotal
	action := &entities.InsightAction{Label: "Ver clientes", Target: "customers"}
	if n == 1 && len(s.InactiveRepeatCustomers) == 1 {
		action = &entities.InsightAction{Label: "Ver cliente", Target: "customer", ID: s.InactiveRepeatCustomers[0].ID}
	}
	return []entities.Insight{{
		Kind:     "inactive_customers",
		Severity: entities.SeverityInfo,
		Title:    fmt.Sprintf("%d %s sumiram", n, plural(n, "cliente recorrente", "clientes recorrentes")),
		Detail: fmt.Sprintf("%s %s mais de uma vez e não %s orçamento há 60+ dias. Vale um contato para reativar.",
			joinNames(names, n), plural(n, "comprou", "compraram"), plural(n, "pede", "pedem")),
		Metric: fmt.Sprintf("%d", n),
		Action: action,
	}}
}

// joinNames lists names in pt-BR ("Ana, Beto e Caio"), adding "e outros" when
// total exceeds the names shown.
func joinNames(names []string, total int) string {
	if total > len(names) {
		return strings.Join(names, ", ") + " e outros"
	}
	if len(names) <= 1 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " e " + names[len(names)-1]
}

func profitGrowth(in Input) []entities.Insight {
	o := in.Overview
	if o == nil || o.SalesCount < minSalesForComparisons || o.Profit <= 0 || o.ProfitChange < profitGrowthHighlight {
		return nil
	}
	return []entities.Insight{{
		Kind:     "profit_growth",
		Severity: entities.SeverityPositive,
		Title:    "Lucro em alta",
		Detail:   fmt.Sprintf("O lucro cresceu %s em relação ao período anterior, somando %s.", pct(o.ProfitChange), brl(o.Profit)),
		Metric:   "+" + pct(o.ProfitChange),
	}}
}
