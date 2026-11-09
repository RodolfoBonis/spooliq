package usecases_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	budgetEntities "github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/usecases"
	"github.com/gin-gonic/gin"
)

func TestBudgetPayments_ListExposesStatusReason(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	repo.seedIntegration("org1", box, 0, 1, nil)
	budget := repo.seedBudget("org1", 2000, 2000, nil, "approved")
	repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budget.ID, Kind: entities.KindFull,
		OrderNSU: "spq_pac", AmountCents: 2000, PaidAmountCents: 2000,
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
		t.Fatalf("status = %d want 200; body=%s", w.Code, w.Body.String())
	}
	var resp entities.BudgetPaymentsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(resp.Payments))
	}
	if resp.Payments[0].StatusReason != entities.ReasonPaidAfterCancel {
		t.Fatalf("status_reason = %q want %q", resp.Payments[0].StatusReason, entities.ReasonPaidAfterCancel)
	}
}

func TestPublicConfirmResponse_FlattensViewWithConfirmation(t *testing.T) {
	// The confirm response must be the full public view with a top-level
	// `confirmation` field (the embedded view's fields are promoted).
	resp := usecases.PublicConfirmResponse{
		PublicBudgetView: &budgetEntities.PublicBudgetView{Name: "Vaso", Status: "approved"},
		Confirmation:     entities.ConfirmPaidState,
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["confirmation"] != "paid" {
		t.Fatalf("confirmation = %v want paid", m["confirmation"])
	}
	if m["name"] != "Vaso" {
		t.Fatalf("view field `name` not promoted to top level: %v", m["name"])
	}
	if _, ok := m["status"]; !ok {
		t.Fatalf("view field `status` not present at top level")
	}
}
