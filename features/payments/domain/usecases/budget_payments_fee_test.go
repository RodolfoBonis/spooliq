package usecases_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/usecases"
	"github.com/gin-gonic/gin"
)

// TestBudgetPayments_EstimatesFeeFromCurrentTable asserts a payment confirmed before a
// fee table existed (no snapshot) gets its fee/net estimated from the CURRENT table on
// read, and the totals (fees / net_received) exclude paid_after_cancel.
func TestBudgetPayments_EstimatesFeeFromCurrentTable(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", box, 0, 1, nil)
	integ.FeeRates = &entities.FeeRates{Credit1x: 4.2, CreditInstallments: map[string]float64{"4": 7.9}}

	budget := repo.seedBudget("org1", 21712, 21712, nil, "approved")
	// Paid, counted: snapshot is nil (confirmed before the table) -> estimated now.
	repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budget.ID, Kind: entities.KindFull,
		OrderNSU: "spq_old", AmountCents: 21712, PaidAmountCents: 21712,
		Installments: 4, CaptureMethod: "credit_card", Status: entities.PaymentPaid,
	})
	// Paid-after-cancel: must be excluded from totals but still show a per-line estimate.
	repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budget.ID, Kind: entities.KindBalance,
		OrderNSU: "spq_pac", AmountCents: 1000, PaidAmountCents: 1000,
		Installments: 1, CaptureMethod: "credit_card",
		Status: entities.PaymentPaid, StatusReason: entities.ReasonPaidAfterCancel,
	})

	svc := usecases.NewService(repo, &fakeClient{}, box, testConfig(), nil, nil, nil, logger.NewLogger("test"))
	uc := usecases.NewBudgetPaymentsUseCase(svc, usecases.NewChargeService(svc))

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/budgets/x/payments", nil)
	c.Params = gin.Params{{Key: "id", Value: budget.ID.String()}}
	c.Set("organization_id", "org1")
	c.Set("user_roles", []string{"Owner"})

	uc.List(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", w.Code, w.Body.String())
	}
	var resp entities.BudgetPaymentsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Totals: only the counted 4x payment -> fee 1715, net 21712-1715 = 19997.
	if resp.Fees != 1715 {
		t.Errorf("fees = %d, want 1715", resp.Fees)
	}
	if resp.NetReceived != 19997 {
		t.Errorf("net_received = %d, want 19997", resp.NetReceived)
	}

	// Per-line estimate present on the counted payment.
	var counted *entities.PaymentLineResponse
	for i := range resp.Payments {
		if resp.Payments[i].Kind == entities.KindFull {
			counted = &resp.Payments[i]
		}
	}
	if counted == nil || counted.FeeAmount == nil || *counted.FeeAmount != 1715 {
		t.Fatalf("counted line fee estimate missing/wrong: %+v", counted)
	}
	if counted.NetAmount == nil || *counted.NetAmount != 19997 {
		t.Errorf("counted net = %v, want 19997", counted.NetAmount)
	}
	if counted.FeeRatePercent == nil || *counted.FeeRatePercent != 7.9 {
		t.Errorf("counted rate = %v, want 7.9", counted.FeeRatePercent)
	}
}

// TestBudgetPayments_SnapshotWinsOverTable asserts a stored snapshot is used verbatim
// even when the current table would estimate a different value.
func TestBudgetPayments_SnapshotWinsOverTable(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", box, 0, 1, nil)
	integ.FeeRates = &entities.FeeRates{Credit1x: 10} // current table would say 10%

	budget := repo.seedBudget("org1", 10000, 10000, nil, "approved")
	snapRate := 4.2
	snapCents := int64(420)
	repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budget.ID, Kind: entities.KindFull,
		OrderNSU: "spq_snap", AmountCents: 10000, PaidAmountCents: 10000,
		Installments: 1, CaptureMethod: "credit_card", Status: entities.PaymentPaid,
		FeeRatePercent: &snapRate, FeeCents: &snapCents,
	})

	svc := usecases.NewService(repo, &fakeClient{}, box, testConfig(), nil, nil, nil, logger.NewLogger("test"))
	uc := usecases.NewBudgetPaymentsUseCase(svc, usecases.NewChargeService(svc))

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/budgets/x/payments", nil)
	c.Params = gin.Params{{Key: "id", Value: budget.ID.String()}}
	c.Set("organization_id", "org1")
	c.Set("user_roles", []string{"Owner"})

	uc.List(c)
	var resp entities.BudgetPaymentsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Fees != 420 {
		t.Errorf("fees = %d, want 420 (snapshot wins)", resp.Fees)
	}
}
