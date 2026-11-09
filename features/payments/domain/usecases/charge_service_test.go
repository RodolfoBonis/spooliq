package usecases_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/crypto"
	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/usecases"
	"github.com/google/uuid"
)

func newChargeService(repo domainRepos.PaymentRepository, client *fakeClient, box *crypto.SecretBox) *usecases.ChargeService {
	svc := usecases.NewService(repo, client, box, testConfig(), nil, nil, nil, logger.NewLogger("test"))
	return usecases.NewChargeService(svc)
}

func futureTime() *time.Time {
	t := time.Now().Add(24 * time.Hour)
	return &t
}

func TestCharge_ReusesEquivalentPendingLink(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", box, 0, 1, nil)
	budget := repo.seedBudget("org1", 2000, 0, nil, "approved") // no deposit -> full 2000
	repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budget.ID, Kind: entities.KindFull,
		OrderNSU: "spq_existing", AmountCents: 2000, Status: entities.PaymentPending,
		CheckoutURL: "https://checkout.infinitepay.io/existing", LinkExpiresAt: futureTime(),
	})
	client := &fakeClient{}

	cs := newChargeService(repo, client, box)
	resp, err := cs.CreateOrReuseLink(context.Background(), budget, integ, nil)
	if err != nil {
		t.Fatalf("CreateOrReuseLink: %v", err)
	}
	if resp.CheckoutURL != "https://checkout.infinitepay.io/existing" {
		t.Fatalf("expected reused URL, got %q", resp.CheckoutURL)
	}
	if client.createCalls != 0 {
		t.Fatalf("provider should not be called on reuse, calls=%d", client.createCalls)
	}
}

func TestCharge_SupersedesOnAmountChange(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", box, 0, 1, nil)
	budget := repo.seedBudget("org1", 2000, 0, nil, "approved")
	old := repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budget.ID, Kind: entities.KindFull,
		OrderNSU: "spq_old", AmountCents: 1000, Status: entities.PaymentPending,
		CheckoutURL: "https://checkout.infinitepay.io/old", LinkExpiresAt: futureTime(),
	})
	client := &fakeClient{}

	cs := newChargeService(repo, client, box)
	resp, err := cs.CreateOrReuseLink(context.Background(), budget, integ, nil)
	if err != nil {
		t.Fatalf("CreateOrReuseLink: %v", err)
	}
	if client.createCalls != 1 {
		t.Fatalf("provider should be called once, calls=%d", client.createCalls)
	}
	if resp.Amount != 2000 {
		t.Fatalf("amount = %d want 2000", resp.Amount)
	}
	// Old pending must be superseded.
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_old")
	_ = old
	if got.Status != entities.PaymentCanceled || got.StatusReason != entities.ReasonSuperseded {
		t.Fatalf("old link not superseded: status=%s reason=%s", got.Status, got.StatusReason)
	}
}

func TestCharge_RetriesWithSameOrderNSU(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", box, 0, 1, nil)
	budget := repo.seedBudget("org1", 2000, 0, nil, "approved")
	// A previous API failure left a pending row with no checkout_url.
	repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budget.ID, Kind: entities.KindFull,
		OrderNSU: "spq_retry", AmountCents: 2000, Status: entities.PaymentPending,
		CheckoutURL: "", LinkExpiresAt: futureTime(),
	})
	client := &fakeClient{}

	cs := newChargeService(repo, client, box)
	_, err := cs.CreateOrReuseLink(context.Background(), budget, integ, nil)
	if err != nil {
		t.Fatalf("CreateOrReuseLink: %v", err)
	}
	if client.createCalls != 1 {
		t.Fatalf("provider should be called once, calls=%d", client.createCalls)
	}
	if client.lastCreateOrder != "spq_retry" {
		t.Fatalf("retry must reuse the same order_nsu, got %q", client.lastCreateOrder)
	}
}

func TestCharge_KindMismatch(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", box, 0, 1, nil)
	budget := repo.seedBudget("org1", 2000, 0, nil, "approved") // next is "full"
	client := &fakeClient{}

	cs := newChargeService(repo, client, box)
	deposit := entities.KindDeposit
	_, err := cs.CreateOrReuseLink(context.Background(), budget, integ, &deposit)
	if err != usecases.ErrChargeKindMismatch {
		t.Fatalf("err = %v want ErrChargeKindMismatch", err)
	}
}

func TestCharge_NotPayableStatus(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", box, 0, 1, nil)
	budget := repo.seedBudget("org1", 2000, 0, nil, "sent") // not payable
	client := &fakeClient{}

	cs := newChargeService(repo, client, box)
	_, err := cs.CreateOrReuseLink(context.Background(), budget, integ, nil)
	if err != usecases.ErrChargeNotAvailable {
		t.Fatalf("err = %v want ErrChargeNotAvailable", err)
	}
}

func TestCharge_AlreadyPaid(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", box, 0, 1, nil)
	budget := repo.seedBudget("org1", 2000, 2000, nil, "approved")
	client := &fakeClient{}

	cs := newChargeService(repo, client, box)
	_, err := cs.CreateOrReuseLink(context.Background(), budget, integ, nil)
	if err != usecases.ErrAlreadyPaid {
		t.Fatalf("err = %v want ErrAlreadyPaid", err)
	}
}

// raceRepo wraps the fake to simulate a concurrent create winning the pending
// (budget, kind) slot: the in-transaction locked read returns nil (so the service
// tries to create), CreatePayment fails with a unique violation, and the fallback
// (non-locking) read finds the winner's link.
type raceRepo struct {
	*fakeRepo
	winner *entities.BudgetPayment
}

func (r *raceRepo) GetPendingByBudgetKindForUpdate(ctx context.Context, budgetID uuid.UUID, kind entities.ChargeKind) (*entities.BudgetPayment, error) {
	return nil, nil
}

func (r *raceRepo) GetPendingByBudgetKind(ctx context.Context, budgetID uuid.UUID, kind entities.ChargeKind) (*entities.BudgetPayment, error) {
	cp := *r.winner
	return &cp, nil
}

func (r *raceRepo) CreatePayment(ctx context.Context, p *entities.BudgetPayment) error {
	return errUnique
}

func (r *raceRepo) WithTransaction(ctx context.Context, fn func(repo domainRepos.PaymentRepository) error) error {
	return fn(r)
}

func TestCharge_RaceFallsBackToWinnersLink(t *testing.T) {
	box := testBox()
	base := newFakeRepo()
	integ := base.seedIntegration("org1", box, 0, 1, nil)
	budget := base.seedBudget("org1", 2000, 0, nil, "approved")
	repo := &raceRepo{fakeRepo: base, winner: &entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budget.ID, Kind: entities.KindFull,
		OrderNSU: "spq_winner", AmountCents: 2000, Status: entities.PaymentPending,
		CheckoutURL: "https://checkout.infinitepay.io/winner", LinkExpiresAt: futureTime(),
	}}
	client := &fakeClient{}

	cs := newChargeService(repo, client, box)
	resp, err := cs.CreateOrReuseLink(context.Background(), budget, integ, nil)
	if err != nil {
		t.Fatalf("CreateOrReuseLink: %v", err)
	}
	if resp.CheckoutURL != "https://checkout.infinitepay.io/winner" {
		t.Fatalf("expected winner URL, got %q", resp.CheckoutURL)
	}
	if client.createCalls != 0 {
		t.Fatalf("provider should not be called after race fallback, calls=%d", client.createCalls)
	}
}

func TestCharge_PixChangeSupersedes(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	// Integration now offers a 5% Pix discount; the pending link was snapshotted at 0.
	integ := repo.seedIntegration("org1", box, 5, 1, nil)
	budget := repo.seedBudget("org1", 2000, 0, nil, "approved")
	old := repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budget.ID, Kind: entities.KindFull,
		OrderNSU: "spq_pix_old", AmountCents: 2000, PixDiscountPercent: 0, Status: entities.PaymentPending,
		CheckoutURL: "https://checkout.infinitepay.io/old", LinkExpiresAt: futureTime(),
	})
	client := &fakeClient{}

	cs := newChargeService(repo, client, box)
	_, err := cs.CreateOrReuseLink(context.Background(), budget, integ, nil)
	if err != nil {
		t.Fatalf("CreateOrReuseLink: %v", err)
	}
	if client.createCalls != 1 {
		t.Fatalf("a pix-discount change must create a NEW link, calls=%d", client.createCalls)
	}
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_pix_old")
	_ = old
	if got.Status != entities.PaymentCanceled || got.StatusReason != entities.ReasonSuperseded {
		t.Fatalf("old link must be superseded on pix change, status=%s reason=%s", got.Status, got.StatusReason)
	}
}

func TestCharge_SupersedeRaceReturnsNotAvailable(t *testing.T) {
	box := testBox()
	base := newFakeRepo()
	integ := base.seedIntegration("org1", box, 0, 1, nil)
	budget := base.seedBudget("org1", 2000, 0, nil, "approved")
	// A stale pending link (amount differs) that gets paid concurrently: CancelPendingByID
	// affects 0 rows, so the charge aborts for the caller to recompute.
	repo := &supersedeRaceRepo{fakeRepo: base, budgetID: budget.ID}
	client := &fakeClient{}

	cs := newChargeService(repo, client, box)
	_, err := cs.CreateOrReuseLink(context.Background(), budget, integ, nil)
	if err != usecases.ErrChargeNotAvailable {
		t.Fatalf("err = %v want ErrChargeNotAvailable", err)
	}
	if client.createCalls != 0 {
		t.Fatal("no link should be created when the supersede races")
	}
}

type supersedeRaceRepo struct {
	*fakeRepo
	budgetID uuid.UUID
}

func (r *supersedeRaceRepo) GetPendingByBudgetKindForUpdate(ctx context.Context, budgetID uuid.UUID, kind entities.ChargeKind) (*entities.BudgetPayment, error) {
	// A pending link with a DIFFERENT amount (forces supersede).
	exp := futureTime()
	return &entities.BudgetPayment{ID: uuid.New(), OrganizationID: "org1", BudgetID: r.budgetID, Kind: kind, OrderNSU: "spq_stale", AmountCents: 999, Status: entities.PaymentPending, LinkExpiresAt: exp}, nil
}

func (r *supersedeRaceRepo) CancelPendingByID(ctx context.Context, id uuid.UUID, reason string) (int64, error) {
	return 0, nil // lost the race: no row was still pending
}

func (r *supersedeRaceRepo) WithTransaction(ctx context.Context, fn func(repo domainRepos.PaymentRepository) error) error {
	return fn(r)
}

func TestCharge_LeaseReturns409ThenExpires(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", box, 0, 1, nil)
	budget := repo.seedBudget("org1", 2000, 0, nil, "approved")
	fixed := time.Now()
	repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budget.ID, Kind: entities.KindFull,
		OrderNSU: "spq_lease", AmountCents: 2000, PixDiscountPercent: 0, Status: entities.PaymentPending,
		CheckoutURL: "", LinkExpiresAt: futureTime(), LinkCreatingAt: &fixed,
	})
	client := &fakeClient{}
	cs := newChargeService(repo, client, box)

	// Fresh lease held => 409.
	cs.SetNowForTest(func() time.Time { return fixed.Add(5 * time.Second) })
	_, err := cs.CreateOrReuseLink(context.Background(), budget, integ, nil)
	if err != usecases.ErrLinkInProgress {
		t.Fatalf("err = %v want ErrLinkInProgress", err)
	}
	if client.createCalls != 0 {
		t.Fatal("no remote call while the lease is held")
	}

	// After 30s the lease expires and the call proceeds.
	cs.SetNowForTest(func() time.Time { return fixed.Add(31 * time.Second) })
	_, err = cs.CreateOrReuseLink(context.Background(), budget, integ, nil)
	if err != nil {
		t.Fatalf("CreateOrReuseLink after lease: %v", err)
	}
	if client.createCalls != 1 {
		t.Fatalf("remote call should proceed after lease expiry, calls=%d", client.createCalls)
	}
}

// TestCharge_LinkPayloadMatchesInfinitePayContract pins the POST /links body to the
// documented contract: items need "description" (a missing one is rejected with 422)
// and webhook events must come from the documented set.
func TestCharge_LinkPayloadMatchesInfinitePayContract(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", box, 0, 1, nil)
	budget := repo.seedBudget("org1", 2000, 0, nil, "approved")
	client := &fakeClient{}

	cs := newChargeService(repo, client, box)
	if _, err := cs.CreateOrReuseLink(context.Background(), budget, integ, nil); err != nil {
		t.Fatalf("CreateOrReuseLink: %v", err)
	}

	raw, err := json.Marshal(client.lastCreateReq)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var body struct {
		Items   []map[string]interface{} `json:"items"`
		Webhook struct {
			Events []string `json:"events"`
		} `json:"webhook"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(body.Items))
	}
	if d, _ := body.Items[0]["description"].(string); d == "" {
		t.Fatalf("item must carry a non-empty description, got %s", raw)
	}
	if _, ok := body.Items[0]["name"]; ok {
		t.Fatalf("item must not send the undocumented \"name\" field: %s", raw)
	}
	allowed := map[string]bool{"created": true, "paid": true, "settled": true, "canceled": true, "refunded": true, "payment_failed": true}
	if len(body.Webhook.Events) == 0 {
		t.Fatalf("webhook events must be set")
	}
	for _, e := range body.Webhook.Events {
		if !allowed[e] {
			t.Fatalf("webhook event %q is not in the documented set", e)
		}
	}
}

// TestCharge_AcceptsURLFallbackInLinkResponse covers a response that returns the link
// under "url" instead of "checkout_url".
func TestCharge_AcceptsURLFallbackInLinkResponse(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", box, 0, 1, nil)
	budget := repo.seedBudget("org1", 2000, 0, nil, "approved")
	client := &fakeClient{createResp: &coreServices.InfinitePayCreateLinkResponse{URL: "https://checkout.infinitepay.io/fallback"}}

	cs := newChargeService(repo, client, box)
	resp, err := cs.CreateOrReuseLink(context.Background(), budget, integ, nil)
	if err != nil {
		t.Fatalf("CreateOrReuseLink: %v", err)
	}
	if resp.CheckoutURL != "https://checkout.infinitepay.io/fallback" {
		t.Fatalf("expected fallback URL, got %q", resp.CheckoutURL)
	}
}
