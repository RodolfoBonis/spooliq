package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/crypto"
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
