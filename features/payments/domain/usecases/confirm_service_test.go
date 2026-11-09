package usecases_test

import (
	"context"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/crypto"
	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/usecases"
)

func newConfirmService(repo domainRepos.PaymentRepository, client *fakeClient, box *crypto.SecretBox) *usecases.ConfirmService {
	svc := usecases.NewService(repo, client, box, testConfig(), nil, nil, nil, logger.NewLogger("test"))
	return usecases.NewConfirmService(svc)
}

func seedPendingForConfirm(repo *fakeRepo, box *crypto.SecretBox, pixPct float64, total int64, status entities.PaymentRecordStatus) (*entities.BudgetChargeContext, *entities.BudgetPayment) {
	repo.seedIntegration("org1", box, pixPct, 1, nil)
	budget := repo.seedBudget("org1", total, 0, nil, "approved")
	p := repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budget.ID, Kind: entities.KindFull,
		OrderNSU: "spq_confirm", AmountCents: total, Status: status,
	})
	return budget, p
}

func TestConfirm_Paid(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	budget, p := seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentPending)
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{
		Paid: true, PaidAmount: 2000, CaptureMethod: "pix", Installments: 1,
		TransactionNSU: "tx1", ReceiptURL: "https://receipt.infinitepay.io/x",
	}}

	cs := newConfirmService(repo, client, box)
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if res.State != entities.ConfirmPaidState || !res.First {
		t.Fatalf("expected first paid, got state=%s first=%v", res.State, res.First)
	}
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_confirm")
	if got.Status != entities.PaymentPaid || got.PaidAmountCents != 2000 {
		t.Fatalf("payment not marked paid: %+v", got)
	}
	fresh, _ := repo.GetBudgetChargeContext(context.Background(), budget.ID, "org1")
	if fresh.AmountPaidCents != 2000 {
		t.Fatalf("budget aggregate not updated: %d", fresh.AmountPaidCents)
	}
	_ = p
}

func TestConfirm_NotPaid(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	_, _ = seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentPending)
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Paid: false, PaidAmount: 0}}

	cs := newConfirmService(repo, client, box)
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if res.State != entities.ConfirmProcessingState {
		t.Fatalf("state = %s want processing", res.State)
	}
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_confirm")
	if got.Status != entities.PaymentPending {
		t.Fatalf("payment should stay pending, got %s", got.Status)
	}
}

func TestConfirm_ValueDivergenceBelowPixFloor(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	// pix 10% => floor = 1800 - 1 = 1799. A R$ 10,00 payment is a divergence.
	_, _ = seedPendingForConfirm(repo, box, 10, 2000, entities.PaymentPending)
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Paid: true, PaidAmount: 1000}}

	cs := newConfirmService(repo, client, box)
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if res.State != entities.ConfirmProcessingState {
		t.Fatalf("divergence should NOT settle: state=%s", res.State)
	}
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_confirm")
	if got.Status != entities.PaymentPending {
		t.Fatalf("payment should remain pending on divergence, got %s", got.Status)
	}
}

func TestConfirm_PixDiscountPaidIsAccepted(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	// pix 10% => the Pix payer pays 1800, which is >= floor (1799): accepted.
	_, _ = seedPendingForConfirm(repo, box, 10, 2000, entities.PaymentPending)
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Paid: true, PaidAmount: 1800, CaptureMethod: "pix"}}

	cs := newConfirmService(repo, client, box)
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if res.State != entities.ConfirmPaidState {
		t.Fatalf("pix-discounted payment should settle: state=%s", res.State)
	}
}

func TestConfirm_Idempotent(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	_, _ = seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentPending)
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Paid: true, PaidAmount: 2000}}
	cs := newConfirmService(repo, client, box)

	first, _ := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if !first.First {
		t.Fatal("first confirmation should report First=true")
	}
	callsAfterFirst := client.checkCalls

	second, _ := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if second.First {
		t.Fatal("second confirmation must NOT be first")
	}
	if second.State != entities.ConfirmPaidState {
		t.Fatalf("second state = %s want paid", second.State)
	}
	// An already-paid payment short-circuits before calling payment_check again.
	if client.checkCalls != callsAfterFirst {
		t.Fatalf("payment_check should not be called again on a paid payment")
	}
}

func TestConfirm_PaidAfterCancel(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	_, _ = seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentCanceled)
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Paid: true, PaidAmount: 2000}}

	cs := newConfirmService(repo, client, box)
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if res.State != entities.ConfirmPaidState || !res.First {
		t.Fatalf("expected first paid, got state=%s first=%v", res.State, res.First)
	}
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_confirm")
	if got.Status != entities.PaymentPaid || got.StatusReason != entities.ReasonPaidAfterCancel {
		t.Fatalf("expected paid_after_cancel, got status=%s reason=%s", got.Status, got.StatusReason)
	}
}

func TestConfirm_RefundNotAutoApplied(t *testing.T) {
	// SpoolIQ never flips a paid payment to refunded on its own: a payment_check that
	// no longer reports paid leaves the paid record intact (refunds are manual).
	box := testBox()
	repo := newFakeRepo()
	_, _ = seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentPaid)
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Paid: false}}

	cs := newConfirmService(repo, client, box)
	res, _ := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if res.State != entities.ConfirmPaidState {
		t.Fatalf("a paid payment stays paid, state=%s", res.State)
	}
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_confirm")
	if got.Status != entities.PaymentPaid {
		t.Fatalf("payment must remain paid, got %s", got.Status)
	}
}

func TestConfirm_ProviderUnavailableStaysPending(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	_, _ = seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentPending)
	client := &fakeClient{checkErr: coreServices.ErrIPUnavailable}

	cs := newConfirmService(repo, client, box)
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceJob)
	if err != nil {
		t.Fatalf("provider-unavailable should be a soft no-op, got err=%v", err)
	}
	if res.State != entities.ConfirmProcessingState {
		t.Fatalf("state = %s want processing", res.State)
	}
}
