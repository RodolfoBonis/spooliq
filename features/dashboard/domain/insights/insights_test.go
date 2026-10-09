package insights

import (
	"testing"

	"github.com/RodolfoBonis/spooliq/features/dashboard/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func kinds(list []entities.Insight) []string {
	out := make([]string, 0, len(list))
	for _, i := range list {
		out = append(out, i.Kind)
	}
	return out
}

func TestFormatters(t *testing.T) {
	assert.Equal(t, "R$ 1.234,56", brl(123456))
	assert.Equal(t, "R$ 0,05", brl(5))
	assert.Equal(t, "-R$ 1.000.000,00", brl(-100000000))
	assert.Equal(t, "12,3%", pct(12.34))
	assert.Equal(t, "3 dias", days(80))
	assert.Equal(t, "1 dia", days(5))
	assert.Equal(t, "Ana, Beto e Caio", joinNames([]string{"Ana", "Beto", "Caio"}, 3))
	assert.Equal(t, "Ana, Beto e outros", joinNames([]string{"Ana", "Beto"}, 5))
	assert.Equal(t, "Ana", joinNames([]string{"Ana"}, 1))
}

func TestRules(t *testing.T) {
	cases := []struct {
		name string
		in   Input
		want []string
	}{
		{name: "empty input yields nothing", in: Input{}, want: []string{}},
		{
			name: "margin drop cites the biggest cost increase",
			in: Input{
				Overview: &entities.OverviewResponse{SalesCount: 5, ProfitMargin: 20, ProfitMarginPointsChange: -6, NetRevenue: 100000},
				CostNow:  &entities.CostBreakdown{FilamentPct: 40, EnergyPct: 15},
				CostPrev: &entities.CostBreakdown{FilamentPct: 41, EnergyPct: 8},
			},
			want: []string{"margin_drop"},
		},
		{
			name: "small margin drop or few sales is noise",
			in:   Input{Overview: &entities.OverviewResponse{SalesCount: 2, ProfitMarginPointsChange: -10}},
			want: []string{},
		},
		{
			name: "low margin material needs relevant revenue",
			in: Input{Profitability: &entities.ProfitabilityResponse{
				AverageMargin: 30,
				ByMaterial: []entities.ProfitRow{
					{ID: "pla", Name: "PLA", Revenue: 80000, Margin: 35},
					{ID: "petg", Name: "PETG", Revenue: 15000, Margin: 15},
					{ID: "tpu", Name: "TPU", Revenue: 5000, Margin: 5}, // 5% of revenue: ignored
				},
			}},
			want: []string{"low_margin_material"},
		},
		{
			name: "high discount customer",
			in: Input{Profitability: &entities.ProfitabilityResponse{ByCustomer: []entities.ProfitRow{
				{ID: "a", Name: "Ana", Revenue: 50000, DiscountRate: 20},
				{ID: "b", Name: "Beto", Revenue: 50000, DiscountRate: 5},
			}}},
			want: []string{"high_discount_customer"},
		},
		{
			name: "idle machine compared with the best one",
			in: Input{Profitability: &entities.ProfitabilityResponse{ByMachine: []entities.ProfitRow{
				{ID: "a1", Name: "A1", ProfitPerHour: 3000, Hours: 40},
				{ID: "x1", Name: "X1", ProfitPerHour: 1000, Hours: 20},
				{ID: "mini", Name: "Mini", ProfitPerHour: 500, Hours: 2}, // too few hours
			}}},
			want: []string{"low_profit_machine"},
		},
		{
			name: "response time rules",
			in: Input{ResponseTimes: &entities.ResponseTimesResponse{
				Approved: 5, Rejected: 4, Expired: 3, ApprovalMedianHours: 100,
				ExpirationRate: 25, RejectionRate: 33.3,
				RecentRejections: []entities.RejectionReason{{Reason: "Muito caro"}, {Reason: "prazo"}, {Reason: "muito caro"}},
				ExpiringSoon:     []entities.ExpiringBudget{{Customer: "Ana", Total: 10000}},
			}},
			want: []string{"expiring_soon", "high_expiration", "high_rejection", "slow_response"},
		},
		{
			name: "signals",
			in: Input{Signals: &entities.InsightSignals{
				StaleDrafts:                  3,
				InactiveRepeatCustomers:      []entities.NamedRef{{ID: "c1", Name: "Ana"}},
				InactiveRepeatCustomersTotal: 1,
				WasteCost:                    1000, FilamentCost: 10000,
				StockShortfall: []entities.StockShortfall{{ID: "f1", Name: "PLA Preto", StockGrams: 200, ConsumedGrams30: 900}},
			}},
			want: []string{"stock_shortfall", "high_waste", "stale_drafts", "inactive_customers"},
		},
		{
			name: "goal off pace only when projection misses",
			in: Input{Goals: []entities.Goal{
				{Metric: entities.GoalProfit, Name: "Lucro", Configured: true, Target: 100000, Current: 20000, Progress: 20, Projected: 60000, ProjectedProgress: 60, RequiredPerDay: 8000, DaysLeft: 10, Unit: "cents"},
				{Metric: entities.GoalRevenue, Name: "Receita líquida", Configured: true, Target: 100000, Current: 50000, Progress: 50, Projected: 95000, ProjectedProgress: 95, DaysLeft: 10, Unit: "cents"},
				{Metric: entities.GoalBudgets, Name: "Vendas"}, // not configured
			}},
			want: []string{"goal_off_pace"},
		},
		{
			name: "profit growth is highlighted last",
			in:   Input{Overview: &entities.OverviewResponse{SalesCount: 4, Profit: 50000, ProfitChange: 25}},
			want: []string{"profit_growth"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, kinds(Generate(tc.in)))
		})
	}
}

func TestMarginDropDetail(t *testing.T) {
	got := marginDrop(Input{
		Overview: &entities.OverviewResponse{SalesCount: 5, ProfitMargin: 20, ProfitMarginPointsChange: -6, NetRevenue: 100000},
		CostNow:  &entities.CostBreakdown{EnergyPct: 15},
		CostPrev: &entities.CostBreakdown{EnergyPct: 8},
	})
	require.Len(t, got, 1)
	assert.Contains(t, got[0].Detail, "6,0 p.p.")
	assert.Contains(t, got[0].Detail, "energia (+7,0 p.p.")
	assert.EqualValues(t, 6000, got[0].Impact)
}

func TestTopReasonPrefersMostFrequent(t *testing.T) {
	assert.Equal(t, "Muito caro", topReason([]entities.RejectionReason{{Reason: "Muito caro"}, {Reason: "prazo"}, {Reason: "muito caro"}}))
	assert.Equal(t, "prazo", topReason([]entities.RejectionReason{{Reason: " "}, {Reason: "prazo"}}))
	assert.Equal(t, "", topReason(nil))
}

func TestGenerateRanksBySeverityThenImpactAndCaps(t *testing.T) {
	in := Input{
		Overview: &entities.OverviewResponse{SalesCount: 4, Profit: 50000, ProfitChange: 25},
		Signals: &entities.InsightSignals{
			StaleDrafts: 1,
			WasteCost:   2000, FilamentCost: 10000,
			StockShortfall: []entities.StockShortfall{
				{ID: "1", Name: "a"}, {ID: "2", Name: "b"}, {ID: "3", Name: "c"},
			},
		},
		ResponseTimes: &entities.ResponseTimesResponse{
			ExpiringSoon: []entities.ExpiringBudget{{Customer: "Ana", Total: 90000}},
			Approved:     6, Rejected: 0, Expired: 4, ExpirationRate: 40,
		},
		Profitability: &entities.ProfitabilityResponse{ByCustomer: []entities.ProfitRow{
			{ID: "a", Name: "Ana", Revenue: 50000, DiscountRate: 20},
		}},
	}
	got := Generate(in)
	require.Len(t, got, Limit)
	assert.Equal(t, "expiring_soon", got[0].Kind, "warning with the largest impact first")
	for i := 1; i < len(got); i++ {
		assert.LessOrEqual(t, got[i-1].Severity.Rank(), got[i].Severity.Rank())
	}
	assert.NotContains(t, kinds(got), "profit_growth", "positive insights are dropped first when capped")
}
