package usecases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/crypto"
	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/usecases"
	"github.com/google/uuid"
)

func newConfirmService(repo domainRepos.PaymentRepository, client *fakeClient, box *crypto.SecretBox) *usecases.ConfirmService {
	svc := usecases.NewService(repo, client, box, testConfig(), nil, nil, nil, logger.NewLogger("test"))
	return usecases.NewConfirmService(svc)
}

func seedPendingForConfirm(repo *fakeRepo, box *crypto.SecretBox, pixPct int, total int64, status entities.PaymentRecordStatus) (*entities.BudgetChargeContext, *entities.BudgetPayment) {
	repo.seedIntegration("org1", box, pixPct, 1, nil)
	budget := repo.seedBudget("org1", total, 0, nil, "approved")
	p := repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budget.ID, Kind: entities.KindFull,
		OrderNSU: "spq_confirm", AmountCents: total, Status: status,
		PixDiscountPercent: pixPct,
	})
	return budget, p
}

func TestConfirm_Paid(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	budget, p := seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentPending)
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{
		Success: true, Paid: true, PaidAmount: 2000, CaptureMethod: "pix", Installments: 1,
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
	if got.ReceiptURL == nil || *got.ReceiptURL != "https://receipt.infinitepay.io/x" {
		t.Fatalf("receipt from payment_check should be preferred, got %v", got.ReceiptURL)
	}
	_ = p
}

func TestConfirm_MissingBudgetRecordsPaidAfterCancel(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	repo.seedIntegration("org1", box, 0, 1, nil)
	// A payment whose budget is NOT present (deleted).
	repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: uuid.New(), Kind: entities.KindFull,
		OrderNSU: "spq_missing", AmountCents: 2000, Status: entities.PaymentPending,
	})
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: true, Paid: true, PaidAmount: 2000}}

	cs := newConfirmService(repo, client, box)
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_missing"}, "", entities.SourceWebhook)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if res.State != entities.ConfirmPaidState || !res.First {
		t.Fatalf("missing-budget payment must still be recorded paid, got state=%s first=%v", res.State, res.First)
	}
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_missing")
	if got.Status != entities.PaymentPaid || got.StatusReason != entities.ReasonPaidAfterCancel {
		t.Fatalf("missing budget => paid_after_cancel, got status=%s reason=%s", got.Status, got.StatusReason)
	}
}

func TestConfirm_SpoofedRedirectDoesNotBlockLegitWebhook(t *testing.T) {
	// 🔴1: a redirect with spoofed identifiers must never flag the row for review, so a
	// later legit webhook still settles it. The request slug/txn are ignored (stored win).
	box := testBox()
	repo := newFakeRepo()
	_, p := seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentPending)
	stored := "slug-ours"
	p.Slug = &stored
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: false, Paid: false}}
	cs := newConfirmService(repo, client, box)

	// Spoofed redirect: payment not yet paid at the gateway.
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm", Slug: "slug-evil", TransactionNSU: "txn-evil"}, "", entities.SourceRedirect)
	if err != nil {
		t.Fatalf("Confirm(redirect): %v", err)
	}
	if res.State != entities.ConfirmProcessingState {
		t.Fatalf("spoofed redirect must not settle, state=%s", res.State)
	}
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_confirm")
	if got.NeedsReview {
		t.Fatal("redirect must NEVER flag needs_review (would block legit settlement)")
	}
	if got.TransactionNSU != nil {
		t.Fatal("redirect must not persist the spoofed transaction_nsu")
	}

	// Legit webhook: gateway now reports paid.
	client.checkResp = &coreServices.InfinitePayPaymentCheckResponse{Success: true, Paid: true, PaidAmount: 2000}
	res, err = cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if err != nil {
		t.Fatalf("Confirm(webhook): %v", err)
	}
	if res.State != entities.ConfirmPaidState || !res.First {
		t.Fatalf("legit webhook must settle, state=%s first=%v", res.State, res.First)
	}
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
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: true, Paid: true, PaidAmount: 1000}}

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
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: true, Paid: true, PaidAmount: 1800, CaptureMethod: "pix"}}

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
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: true, Paid: true, PaidAmount: 2000}}
	cs := newConfirmService(repo, client, box)

	first, _ := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if !first.First {
		t.Fatal("first confirmation should report First=true")
	}
	second, _ := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if second.First {
		t.Fatal("second confirmation must NOT be first")
	}
	if second.State != entities.ConfirmPaidState {
		t.Fatalf("second state = %s want paid", second.State)
	}
}

func TestConfirm_PaidAfterCancel(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	_, p := seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentCanceled)
	p.StatusReason = entities.ReasonBudgetCancelled // refund-worthy cancel
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: true, Paid: true, PaidAmount: 2000}}

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
	if client.checkCalls == 0 {
		t.Fatal("payment_check must run even for a paid row (refund detection)")
	}
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_confirm")
	if got.Status != entities.PaymentPaid {
		t.Fatalf("payment must remain paid, got %s", got.Status)
	}
}

func TestConfirm_OrderMismatchRejected(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	_, _ = seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentPending)
	// payment_check echoes a DIFFERENT order_nsu: reject, do not settle.
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: true, Paid: true, PaidAmount: 2000, OrderNSU: "spq_someone_else"}}

	cs := newConfirmService(repo, client, box)
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if res.State != entities.ConfirmProcessingState {
		t.Fatalf("order mismatch must not settle, state=%s", res.State)
	}
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_confirm")
	if got.Status != entities.PaymentPending {
		t.Fatalf("payment must remain pending on order mismatch, got %s", got.Status)
	}
}

func TestConfirm_ProviderUnavailableStaysPending(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	_, _ = seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentPending)
	client := &fakeClient{checkErr: coreServices.ErrIPUnavailable}

	cs := newConfirmService(repo, client, box)
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceJob)
	if !errors.Is(err, usecases.ErrConfirmRetry) {
		t.Fatalf("provider-unavailable must be retryable, got err=%v", err)
	}
	if res.State != entities.ConfirmProcessingState {
		t.Fatalf("state = %s want processing", res.State)
	}
}

func TestConfirm_ExpiredLinkSettlesNormally(t *testing.T) {
	// A link canceled with reason "expired" that is paid settles as a NORMAL payment
	// (InfinitePay enforces valid_until), not paid_after_cancel.
	box := testBox()
	repo := newFakeRepo()
	budget, p := seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentCanceled)
	p.StatusReason = entities.ReasonExpired
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: true, Paid: true, PaidAmount: 2000}}

	cs := newConfirmService(repo, client, box)
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if res.State != entities.ConfirmPaidState || !res.First {
		t.Fatalf("expected first paid, got state=%s first=%v", res.State, res.First)
	}
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_confirm")
	if got.Status != entities.PaymentPaid || got.StatusReason != "" {
		t.Fatalf("expired link must settle as normal paid (reason cleared), got status=%s reason=%q", got.Status, got.StatusReason)
	}
	fresh, _ := repo.GetBudgetChargeContext(context.Background(), budget.ID, "org1")
	if fresh.AmountPaidCents != 2000 {
		t.Fatalf("settled expired payment must count toward amount_paid, got %d", fresh.AmountPaidCents)
	}
}

func TestConfirm_RedirectOnPaidRowDoesNothing(t *testing.T) {
	// N3: a redirect confirm on an already-paid row performs no check and no alert.
	box := testBox()
	repo := newFakeRepo()
	_, _ = seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentPaid)
	client := &fakeClient{checkErr: coreServices.ErrIPUnavailable} // would error if called

	cs := newConfirmService(repo, client, box)
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm", TransactionNSU: "garbage"}, "", entities.SourceRedirect)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if res.State != entities.ConfirmPaidState {
		t.Fatalf("state = %s want paid", res.State)
	}
	if client.checkCalls != 0 {
		t.Fatal("redirect on a paid row must NOT call payment_check")
	}
}

func TestConfirm_RedirectDoesNotPersistHintsBeforeVerification(t *testing.T) {
	// N3: from the redirect source, a garbage transaction_nsu is never persisted on a
	// pending row before verification.
	box := testBox()
	repo := newFakeRepo()
	_, _ = seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentPending)
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: false, Paid: false}}

	cs := newConfirmService(repo, client, box)
	_, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm", TransactionNSU: "garbage-txn"}, "", entities.SourceRedirect)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_confirm")
	if got.TransactionNSU != nil {
		t.Fatalf("redirect must not persist transaction_nsu before verification, got %v", *got.TransactionNSU)
	}
}

// confirmLockRaceRepo simulates a cancel committing between the initial read and the
// locked re-read: GetPaymentByOrderNSU returns the pending row, but the locked read
// returns it as canceled with a refund-worthy reason.
type confirmLockRaceRepo struct {
	*fakeRepo
}

func (r *confirmLockRaceRepo) GetPaymentByOrderNSUForUpdate(ctx context.Context, org, orderNSU string) (*entities.BudgetPayment, error) {
	p, err := r.fakeRepo.GetPaymentByOrderNSU(ctx, org, orderNSU)
	if err != nil || p == nil {
		return p, err
	}
	p.Status = entities.PaymentCanceled
	p.StatusReason = entities.ReasonBudgetCancelled
	return p, nil
}

func (r *confirmLockRaceRepo) WithTransaction(ctx context.Context, fn func(repo domainRepos.PaymentRepository) error) error {
	return fn(r)
}

func TestConfirm_CancelBetweenReadAndLockBecomesPaidAfterCancel(t *testing.T) {
	box := testBox()
	base := newFakeRepo()
	budget, _ := seedPendingForConfirm(base, box, 0, 2000, entities.PaymentPending)
	repo := &confirmLockRaceRepo{fakeRepo: base}
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: true, Paid: true, PaidAmount: 2000}}

	svc := usecases.NewService(repo, client, box, testConfig(), nil, nil, nil, logger.NewLogger("test"))
	cs := usecases.NewConfirmService(svc)
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if res.State != entities.ConfirmPaidState || !res.First {
		t.Fatalf("expected first paid, got state=%s first=%v", res.State, res.First)
	}
	got, _ := base.GetPaymentByOrderNSU(context.Background(), "org1", "spq_confirm")
	if got.Status != entities.PaymentPaid || got.StatusReason != entities.ReasonPaidAfterCancel {
		t.Fatalf("cancel-between-read-and-lock must settle as paid_after_cancel, got status=%s reason=%s", got.Status, got.StatusReason)
	}
	fresh, _ := base.GetBudgetChargeContext(context.Background(), budget.ID, "org1")
	if fresh.RefundDueCents != 2000 {
		t.Fatalf("refund_due must be > 0 for paid_after_cancel, got %d", fresh.RefundDueCents)
	}
}

// dbErrRepo returns a generic (non-unique) error from UpdatePaymentFields.
type dbErrRepo struct{ *fakeRepo }

func (r *dbErrRepo) UpdatePaymentFields(_ context.Context, _ uuid.UUID, _ map[string]interface{}) error {
	return errors.New("connection reset")
}
func (r *dbErrRepo) WithTransaction(ctx context.Context, fn func(repo domainRepos.PaymentRepository) error) error {
	return fn(r)
}

func TestConfirm_PreUpdateDBErrorRetries(t *testing.T) {
	box := testBox()
	base := newFakeRepo()
	_, _ = seedPendingForConfirm(base, box, 0, 2000, entities.PaymentPending)
	repo := &dbErrRepo{fakeRepo: base}
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: true, Paid: true, PaidAmount: 2000}}

	svc := usecases.NewService(repo, client, box, testConfig(), nil, nil, nil, logger.NewLogger("test"))
	cs := usecases.NewConfirmService(svc)
	_, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if !errors.Is(err, usecases.ErrConfirmRetry) {
		t.Fatalf("a non-unique DB error in the pre-update must retry, got %v", err)
	}
}

func TestConfirm_SettlingExpiredSupersedesNewerPending(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	budget, p := seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentCanceled)
	p.StatusReason = entities.ReasonExpired
	// A newer pending link for the same (budget, kind).
	newer := repo.seedPayment(&entities.BudgetPayment{
		OrganizationID: "org1", BudgetID: budget.ID, Kind: entities.KindFull,
		OrderNSU: "spq_newer", AmountCents: 2000, Status: entities.PaymentPending,
		CheckoutURL: "https://checkout.infinitepay.io/newer", LinkExpiresAt: futureTime(),
	})
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: true, Paid: true, PaidAmount: 2000}}

	cs := newConfirmService(repo, client, box)
	res, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if err != nil || res.State != entities.ConfirmPaidState {
		t.Fatalf("expected paid, got state=%s err=%v", res.State, err)
	}
	gotNewer, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_newer")
	_ = newer
	if gotNewer.Status != entities.PaymentCanceled || gotNewer.StatusReason != entities.ReasonSuperseded {
		t.Fatalf("newer pending link must be superseded, got status=%s reason=%s", gotNewer.Status, gotNewer.StatusReason)
	}
}

func TestConfirm_RefundNotificationHasDedupeKey(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	_, p := seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentPaid)
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: false, Paid: false}}
	notif := &fakeNotifications{}

	svc := usecases.NewService(repo, client, box, testConfig(), notif, nil, nil, logger.NewLogger("test"))
	cs := usecases.NewConfirmService(svc)
	_, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	sent := notif.all()
	if len(sent) != 1 {
		t.Fatalf("expected 1 refund notification, got %d", len(sent))
	}
	if sent[0].DedupeKey != "refund_review:"+p.ID.String() {
		t.Fatalf("refund notification must carry the dedupe key, got %q", sent[0].DedupeKey)
	}
}
