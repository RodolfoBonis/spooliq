package usecases_test

import (
	"context"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/crypto"
	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	notificationEntities "github.com/RodolfoBonis/spooliq/features/notification/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// testKey is a deterministic 32-byte AES key (base64) for the test SecretBox.
func testKey() string {
	return base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
}

func testBox() *crypto.SecretBox {
	b, _ := crypto.NewSecretBox(testKey())
	return b
}

func testConfig() *config.AppConfig {
	return &config.AppConfig{
		InfinitePayEnabled: true,
		InfinitePayBaseURL: "https://api.checkout.infinitepay.io",
		PublicAPIBaseURL:   "https://api.spooliq.test",
		PublicWebBaseURL:   "https://app.spooliq.test",
	}
}

// --- fake InfinitePay client ---

type fakeClient struct {
	mu              sync.Mutex
	createCalls     int
	checkCalls      int
	createResp      *coreServices.InfinitePayCreateLinkResponse
	createErr       error
	checkResp       *coreServices.InfinitePayPaymentCheckResponse
	checkErr        error
	lastCreateOrder string
}

func (f *fakeClient) CreateLink(_ context.Context, _ string, req coreServices.InfinitePayCreateLinkRequest) (*coreServices.InfinitePayCreateLinkResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls++
	f.lastCreateOrder = req.OrderNSU
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResp != nil {
		return f.createResp, nil
	}
	return &coreServices.InfinitePayCreateLinkResponse{CheckoutURL: "https://checkout.infinitepay.io/abc", Slug: "slug-abc"}, nil
}

func (f *fakeClient) PaymentCheck(_ context.Context, _ string, _ coreServices.InfinitePayPaymentCheckRequest) (*coreServices.InfinitePayPaymentCheckResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checkCalls++
	if f.checkErr != nil {
		return nil, f.checkErr
	}
	if f.checkResp != nil {
		return f.checkResp, nil
	}
	return &coreServices.InfinitePayPaymentCheckResponse{}, nil
}

// --- fake repository (in-memory) ---

type fakeRepo struct {
	mu           sync.Mutex
	integrations map[string]*entities.PaymentIntegration
	payments     map[uuid.UUID]*entities.BudgetPayment
	budgets      map[uuid.UUID]*entities.BudgetChargeContext
	tokens       map[string]uuid.UUID
	createErr    error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		integrations: map[string]*entities.PaymentIntegration{},
		payments:     map[uuid.UUID]*entities.BudgetPayment{},
		budgets:      map[uuid.UUID]*entities.BudgetChargeContext{},
		tokens:       map[string]uuid.UUID{},
	}
}

func (r *fakeRepo) WithTransaction(ctx context.Context, fn func(repo domainRepos.PaymentRepository) error) error {
	return fn(r)
}

func (r *fakeRepo) WithReconcileLock(_ context.Context, fn func() error) (bool, error) {
	return true, fn()
}

func (r *fakeRepo) LockBudget(_ context.Context, _ uuid.UUID, _ string) error { return nil }

func (r *fakeRepo) ListAgedOutWebhooks(_ context.Context, _ time.Time, _ int) ([]*entities.WebhookEventRecord, error) {
	return nil, nil
}

func (r *fakeRepo) GetIntegration(_ context.Context, org, _ string) (*entities.PaymentIntegration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.integrations[org], nil
}

func (r *fakeRepo) CreateIntegration(_ context.Context, i *entities.PaymentIntegration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	cp := *i
	r.integrations[i.OrganizationID] = &cp
	return nil
}

func (r *fakeRepo) UpdateIntegrationFields(_ context.Context, id uuid.UUID, fields map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, integ := range r.integrations {
		if integ.ID == id {
			applyIntegrationFields(integ, fields)
		}
	}
	return nil
}

func applyIntegrationFields(integ *entities.PaymentIntegration, fields map[string]interface{}) {
	if v, ok := fields["enabled"].(bool); ok {
		integ.Enabled = v
	}
	if v, ok := fields["handle"].(string); ok {
		integ.Handle = v
	}
	if v, ok := fields["api_key_ciphertext"].(string); ok {
		integ.APIKeyCiphertext = v
	}
	if v, ok := fields["api_key_last4"].(string); ok {
		integ.APIKeyLast4 = v
	}
	if raw, ok := fields["disconnected_at"]; ok {
		if t, ok := raw.(time.Time); ok {
			integ.DisconnectedAt = &t
		} else {
			integ.DisconnectedAt = nil
		}
	}
}

func (r *fakeRepo) CountPendingLinks(_ context.Context, org string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, p := range r.payments {
		if p.OrganizationID == org && p.Status == entities.PaymentPending {
			n++
		}
	}
	return n, nil
}

func (r *fakeRepo) ListDisconnectedBefore(_ context.Context, cutoff time.Time) ([]*entities.PaymentIntegration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entities.PaymentIntegration
	for _, i := range r.integrations {
		if i.DisconnectedAt != nil && !i.DisconnectedAt.After(cutoff) {
			out = append(out, i)
		}
	}
	return out, nil
}

func (r *fakeRepo) DeleteIntegration(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for org, i := range r.integrations {
		if i.ID == id {
			delete(r.integrations, org)
		}
	}
	return nil
}

func (r *fakeRepo) GetPaymentsByBudget(_ context.Context, budgetID uuid.UUID, _ string) ([]*entities.BudgetPayment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entities.BudgetPayment
	for _, p := range r.payments {
		if p.BudgetID == budgetID {
			cp := *p
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *fakeRepo) GetPendingByBudgetKind(_ context.Context, budgetID uuid.UUID, kind entities.ChargeKind) (*entities.BudgetPayment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.payments {
		if p.BudgetID == budgetID && p.Kind == kind && p.Status == entities.PaymentPending {
			cp := *p
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeRepo) GetPendingByBudgetKindForUpdate(ctx context.Context, budgetID uuid.UUID, kind entities.ChargeKind) (*entities.BudgetPayment, error) {
	return r.GetPendingByBudgetKind(ctx, budgetID, kind)
}

func (r *fakeRepo) GetPaymentByOrderNSU(_ context.Context, _ string, orderNSU string) (*entities.BudgetPayment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.payments {
		if p.OrderNSU == orderNSU {
			cp := *p
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeRepo) GetPaymentByOrderNSUForUpdate(ctx context.Context, org, orderNSU string) (*entities.BudgetPayment, error) {
	return r.GetPaymentByOrderNSU(ctx, org, orderNSU)
}

func (r *fakeRepo) CreatePayment(_ context.Context, p *entities.BudgetPayment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createErr != nil {
		err := r.createErr
		r.createErr = nil
		return err
	}
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.Status == "" {
		p.Status = entities.PaymentPending
	}
	cp := *p
	r.payments[p.ID] = &cp
	return nil
}

func (r *fakeRepo) UpdatePaymentFields(_ context.Context, id uuid.UUID, fields map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.payments[id]; ok {
		applyPaymentFields(p, fields)
	}
	return nil
}

func (r *fakeRepo) UpdatePendingFields(_ context.Context, id uuid.UUID, fields map[string]interface{}) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.payments[id]
	if !ok || p.Status != entities.PaymentPending {
		return 0, nil
	}
	applyPaymentFields(p, fields)
	return 1, nil
}

func (r *fakeRepo) MarkPaidGuarded(_ context.Context, id uuid.UUID, _ string, fields map[string]interface{}) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.payments[id]
	if !ok {
		return 0, nil
	}
	if p.Status != entities.PaymentPending && p.Status != entities.PaymentCanceled {
		return 0, nil
	}
	applyPaymentFields(p, fields)
	return 1, nil
}

func (r *fakeRepo) CancelPendingByID(_ context.Context, id uuid.UUID, reason string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.payments[id]
	if !ok || p.Status != entities.PaymentPending {
		return 0, nil
	}
	p.Status = entities.PaymentCanceled
	p.StatusReason = reason
	return 1, nil
}

func applyPaymentFields(p *entities.BudgetPayment, fields map[string]interface{}) {
	if v, ok := fields["status"].(string); ok {
		p.Status = entities.PaymentRecordStatus(v)
	}
	if v, ok := fields["status_reason"].(string); ok {
		p.StatusReason = v
	}
	if v, ok := fields["checkout_url"].(string); ok {
		p.CheckoutURL = v
	}
	if v, ok := fields["slug"].(string); ok {
		p.Slug = &v
	}
	if v, ok := fields["paid_amount_cents"].(int64); ok {
		p.PaidAmountCents = v
	}
	if v, ok := fields["capture_method"].(string); ok {
		p.CaptureMethod = v
	}
	if v, ok := fields["installments"].(int); ok {
		p.Installments = v
	}
	if v, ok := fields["transaction_nsu"].(string); ok {
		p.TransactionNSU = &v
	}
	if v, ok := fields["receipt_url"].(string); ok {
		p.ReceiptURL = &v
	}
	if v, ok := fields["needs_review"].(bool); ok {
		p.NeedsReview = v
	}
	if v, ok := fields["review_reason"].(string); ok {
		p.ReviewReason = v
	}
	if raw, ok := fields["link_creating_at"]; ok {
		if t, ok := raw.(time.Time); ok {
			p.LinkCreatingAt = &t
		} else {
			p.LinkCreatingAt = nil
		}
	}
}

func (r *fakeRepo) CancelPendingForBudget(_ context.Context, budgetID uuid.UUID, _ string, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.payments {
		if p.BudgetID == budgetID && p.Status == entities.PaymentPending {
			p.Status = entities.PaymentCanceled
			p.StatusReason = reason
		}
	}
	return nil
}

func (r *fakeRepo) CancelPendingForOrg(_ context.Context, org, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.payments {
		if p.OrganizationID == org && p.Status == entities.PaymentPending {
			p.Status = entities.PaymentCanceled
			p.StatusReason = reason
		}
	}
	return nil
}

func (r *fakeRepo) ListExpiredPendingLinks(_ context.Context, _ time.Time, _ int) ([]*entities.BudgetPayment, error) {
	return nil, nil
}

func (r *fakeRepo) ListPendingForRecheck(_ context.Context, _ time.Time, _ int) ([]*entities.BudgetPayment, error) {
	return nil, nil
}

func (r *fakeRepo) GetBudgetChargeContext(_ context.Context, budgetID uuid.UUID, _ string) (*entities.BudgetChargeContext, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.budgets[budgetID]; ok {
		cp := *b
		return &cp, nil
	}
	return nil, nil
}

func (r *fakeRepo) GetBudgetChargeContextByToken(_ context.Context, token string) (*entities.BudgetChargeContext, error) {
	r.mu.Lock()
	id, ok := r.tokens[token]
	r.mu.Unlock()
	if !ok {
		return nil, nil
	}
	return r.GetBudgetChargeContext(context.Background(), id, "")
}

func (r *fakeRepo) SumPaidPrincipal(_ context.Context, budgetID uuid.UUID) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var sum int64
	for _, p := range r.payments {
		if p.BudgetID == budgetID && p.Status == entities.PaymentPaid && p.StatusReason != entities.ReasonPaidAfterCancel {
			sum += p.AmountCents
		}
	}
	return sum, nil
}

func (r *fakeRepo) RecalcBudgetAggregate(_ context.Context, budgetID uuid.UUID, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var paid, refund int64
	for _, p := range r.payments {
		if p.BudgetID == budgetID && p.Status == entities.PaymentPaid {
			if p.StatusReason == entities.ReasonPaidAfterCancel {
				refund += p.AmountCents
			} else {
				paid += p.AmountCents
			}
		}
	}
	if b, ok := r.budgets[budgetID]; ok {
		b.AmountPaidCents = paid
		b.RefundDueCents = refund
	}
	return nil
}

func (r *fakeRepo) HasBlockingPayments(_ context.Context, budgetID uuid.UUID, _ string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.payments {
		if p.BudgetID == budgetID && (p.Status == entities.PaymentPaid || p.Status == entities.PaymentPending) {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeRepo) CreateWebhookEvent(_ context.Context, _ string, _ []byte, _ time.Time) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (r *fakeRepo) MarkWebhookProcessed(_ context.Context, _ uuid.UUID) error { return nil }
func (r *fakeRepo) MarkWebhookAttempt(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (r *fakeRepo) ListUnprocessedWebhooks(_ context.Context, _ time.Time, _, _ int) ([]*entities.WebhookEventRecord, error) {
	return nil, nil
}

// seedIntegration adds an active integration for org with the given settings.
func (r *fakeRepo) seedIntegration(org string, box *crypto.SecretBox, pixPct, assured int, deposit *float64) *entities.PaymentIntegration {
	key, _ := box.EncryptString("sk_live_testkey1234")
	tok, _ := box.EncryptString("webhooktoken")
	i := &entities.PaymentIntegration{
		ID:                     uuid.New(),
		OrganizationID:         org,
		Provider:               entities.ProviderInfinitePay,
		Handle:                 "mystore",
		APIKeyCiphertext:       key,
		APIKeyLast4:            "1234",
		WebhookTokenCiphertext: tok,
		Enabled:                true,
		AssuredInstallments:    assured,
		PixDiscountPercent:     pixPct,
		DefaultDepositPercent:  deposit,
	}
	r.mu.Lock()
	r.integrations[org] = i
	r.mu.Unlock()
	return i
}

// seedIntegrationRaw stores an integration with a caller-provided API key ciphertext
// (used to simulate a key the current box cannot decrypt).
func (r *fakeRepo) seedIntegrationRaw(org, apiKeyCiphertext, handle string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.integrations[org] = &entities.PaymentIntegration{
		ID: uuid.New(), OrganizationID: org, Provider: entities.ProviderInfinitePay,
		Handle: handle, APIKeyCiphertext: apiKeyCiphertext, WebhookTokenCiphertext: apiKeyCiphertext,
		Enabled: true, AssuredInstallments: 1,
	}
}

// seedBudget registers a payable budget context.
func (r *fakeRepo) seedBudget(org string, total, paid int64, deposit *float64, status string) *entities.BudgetChargeContext {
	b := &entities.BudgetChargeContext{
		ID:              uuid.New(),
		OrganizationID:  org,
		Status:          status,
		TotalCost:       total,
		AmountPaidCents: paid,
		DepositPercent:  deposit,
		Name:            "Vaso",
	}
	r.mu.Lock()
	r.budgets[b.ID] = b
	r.mu.Unlock()
	return b
}

// seedPayment registers an existing payment row.
func (r *fakeRepo) seedPayment(p *entities.BudgetPayment) *entities.BudgetPayment {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	r.mu.Lock()
	r.payments[p.ID] = p
	r.mu.Unlock()
	return p
}

var errUnique = errors.New("duplicate key value violates unique constraint")

// fakeNotifications captures Notify calls for assertions.
type fakeNotifications struct {
	mu   sync.Mutex
	sent []notificationEntities.NewNotification
}

func (f *fakeNotifications) Notify(_ string, n notificationEntities.NewNotification) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, n)
}
func (f *fakeNotifications) List(_ *gin.Context)        {}
func (f *fakeNotifications) UnreadCount(_ *gin.Context) {}
func (f *fakeNotifications) MarkRead(_ *gin.Context)    {}
func (f *fakeNotifications) MarkAllRead(_ *gin.Context) {}

func (f *fakeNotifications) all() []notificationEntities.NewNotification {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]notificationEntities.NewNotification, len(f.sent))
	copy(out, f.sent)
	return out
}
