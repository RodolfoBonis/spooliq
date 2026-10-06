package services

import (
	"context"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	budgetEntities "github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
)

// TestGenerateBudgetPDF_UsesSaleValues is a smoke test proving the PDF renders from
// the precomputed per-item SALE values (SaleTotal / SaleUnitPrice) carried on the
// item responses — the single source of truth shared with the API — and produces a
// non-empty document. It does not touch the CDN (no logo URL).
func TestGenerateBudgetPDF_UsesSaleValues(t *testing.T) {
	svc := NewPDFService(nil, logger.NewLogger("test"))

	name := "ACME 3D"
	budget := &budgetEntities.BudgetEntity{
		Name:      "Pedido de teste",
		TotalCost: 13200,
	}
	customer := &budgetEntities.CustomerInfo{Name: "Cliente Teste"}
	company := &budgetEntities.CompanyInfo{Name: name}

	items := []budgetEntities.BudgetItemResponse{
		{
			ProductName:     "Cubo",
			ProductQuantity: 2,
			ItemTotalCost:   10000,
			UnitPrice:       5000,
			SaleTotal:       13200, // cost + markup
			SaleUnitPrice:   6600,
		},
	}

	data := BudgetPDFData{
		Budget:   budget,
		Customer: customer,
		Items:    items,
		Company:  company,
	}

	pdf, err := svc.GenerateBudgetPDF(context.Background(), data)
	if err != nil {
		t.Fatalf("GenerateBudgetPDF returned error: %v", err)
	}
	if len(pdf) == 0 {
		t.Fatal("expected a non-empty PDF")
	}
}
