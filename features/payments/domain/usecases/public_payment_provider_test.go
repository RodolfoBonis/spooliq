package usecases_test

import (
	"context"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	budgetEntities "github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/usecases"
	"github.com/google/uuid"
)

func TestPublicPaymentBlock_ExcludesPaidAfterCancelFromPublicLines(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	repo.seedIntegration("org1", box, 0, 1, nil)
	budgetID := uuid.New()
	repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budgetID, Kind: entities.KindDeposit,
		OrderNSU: "spq_ok", AmountCents: 1000, PaidAmountCents: 1000, Status: entities.PaymentPaid,
	})
	repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budgetID, Kind: entities.KindBalance,
		OrderNSU: "spq_par", AmountCents: 1000, PaidAmountCents: 1000, Status: entities.PaymentPaid,
		StatusReason: entities.ReasonPaidAfterCancel,
	})

	provider := usecases.NewPublicPaymentProvider(usecases.NewService(repo, &fakeClient{}, box, testConfig(), nil, nil, nil, logger.NewLogger("test")))
	budget := &budgetEntities.BudgetEntity{
		ID: budgetID, OrganizationID: "org1", Status: budgetEntities.StatusApproved,
		TotalCost: 2000, AmountPaidCents: 1000,
	}

	raw, err := provider.PublicPaymentBlock(context.Background(), budget)
	if err != nil {
		t.Fatalf("PublicPaymentBlock: %v", err)
	}
	block, ok := raw.(*entities.PublicPaymentBlock)
	if !ok || block == nil {
		t.Fatalf("expected a payment block, got %T", raw)
	}
	if len(block.Payments) != 1 {
		t.Fatalf("public lines must exclude paid_after_cancel, got %d lines", len(block.Payments))
	}
	if block.Payments[0].Kind != entities.KindDeposit {
		t.Fatalf("the remaining public line must be the real payment, got kind=%s", block.Payments[0].Kind)
	}
}
