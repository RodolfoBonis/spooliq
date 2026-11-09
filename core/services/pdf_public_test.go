package services

import (
	"context"
	"strings"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	budgetEntities "github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
)

// TestCustomerDetailLines_PublicViewOmitsPII proves the public (customer-facing)
// PDF renders the customer NAME only: email, phone and CPF/CNPJ are never emitted.
// This asserts at the data-building level because the PDF stream is compressed.
func TestCustomerDetailLines_PublicViewOmitsPII(t *testing.T) {
	email := "secret@example.com"
	phone := "+55 11 99999-9999"
	doc := "123.456.789-00"
	customer := &budgetEntities.CustomerInfo{
		Name:     "Cliente Teste",
		Email:    &email,
		Phone:    &phone,
		Document: &doc,
	}

	// Public view: no contact lines at all.
	publicLines := customerDetailLines(customer, true)
	if len(publicLines) != 0 {
		t.Fatalf("public view must not emit contact lines, got %v", publicLines)
	}
	joined := strings.Join(publicLines, "\n")
	for _, pii := range []string{email, phone, doc} {
		if strings.Contains(joined, pii) {
			t.Errorf("public customer block leaked PII %q", pii)
		}
	}

	// Org view: contact lines are present.
	orgLines := customerDetailLines(customer, false)
	orgJoined := strings.Join(orgLines, "\n")
	for _, pii := range []string{email, phone, doc} {
		if !strings.Contains(orgJoined, pii) {
			t.Errorf("org customer block should contain %q, got %v", pii, orgLines)
		}
	}
}

// TestGenerateBudgetPDF_PublicViewRenders is a smoke test that the public PDF still
// builds (name-only customer block) and produces a non-empty document.
func TestGenerateBudgetPDF_PublicViewRenders(t *testing.T) {
	svc := NewPDFService(nil, logger.NewLogger("test"))

	email := "secret@example.com"
	doc := "123.456.789-00"
	budget := &budgetEntities.BudgetEntity{Name: "Pedido", TotalCost: 10000}
	data := BudgetPDFData{
		Budget:   budget,
		Customer: &budgetEntities.CustomerInfo{Name: "Cliente", Email: &email, Document: &doc},
		Company:  &budgetEntities.CompanyInfo{Name: "ACME 3D"},
		Items: []budgetEntities.BudgetItemResponse{
			{ProductName: "Item", ProductQuantity: 1, SaleUnitPrice: 10000, SaleTotal: 10000},
		},
		PublicView: true,
	}

	out, err := svc.GenerateBudgetPDF(context.Background(), data)
	if err != nil {
		t.Fatalf("generate public pdf: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected a non-empty PDF")
	}
}
