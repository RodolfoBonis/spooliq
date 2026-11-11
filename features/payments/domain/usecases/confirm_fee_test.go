package usecases_test

import (
	"context"
	"testing"

	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
)

// TestConfirm_SnapshotsFee asserts the InfinitePay fee is computed from the org's fee
// table at confirmation and persisted (fee_rate_percent / fee_cents) on the payment.
func TestConfirm_SnapshotsFee(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	budget, _ := seedPendingForConfirm(repo, box, 0, 21712, entities.PaymentPending)
	// The org has a fee table: credit in 4x costs 7.9%.
	repo.integrations["org1"].FeeRates = &entities.FeeRates{
		Credit1x:           4.2,
		CreditInstallments: map[string]float64{"4": 7.9},
	}
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{
		Success: true, Paid: true, PaidAmount: 21712, CaptureMethod: "credit_card", Installments: 4,
	}}

	cs := newConfirmService(repo, client, box)
	if _, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook); err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_confirm")
	if got.FeeRatePercent == nil || *got.FeeRatePercent != 7.9 {
		t.Fatalf("fee_rate_percent = %v, want 7.9", got.FeeRatePercent)
	}
	if got.FeeCents == nil || *got.FeeCents != 1715 {
		t.Fatalf("fee_cents = %v, want 1715", got.FeeCents)
	}
	_ = budget
}

// TestConfirm_NoFeeTableLeavesSnapshotNull asserts that when no fee table is configured
// the snapshot columns stay nil (the GET later estimates from the current table).
func TestConfirm_NoFeeTableLeavesSnapshotNull(t *testing.T) {
	box := testBox()
	repo := newFakeRepo()
	seedPendingForConfirm(repo, box, 0, 2000, entities.PaymentPending)
	// No FeeRates on the integration.
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{
		Success: true, Paid: true, PaidAmount: 2000, CaptureMethod: "pix", Installments: 1,
	}}

	cs := newConfirmService(repo, client, box)
	if _, err := cs.Confirm(context.Background(), "org1", entities.WebhookPayload{OrderNSU: "spq_confirm"}, "", entities.SourceWebhook); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	got, _ := repo.GetPaymentByOrderNSU(context.Background(), "org1", "spq_confirm")
	if got.FeeRatePercent != nil || got.FeeCents != nil {
		t.Fatalf("snapshot should be nil without a fee table, got %v / %v", got.FeeRatePercent, got.FeeCents)
	}
}
