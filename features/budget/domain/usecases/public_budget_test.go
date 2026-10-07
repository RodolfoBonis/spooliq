package usecases

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// publicFake is a stateful BudgetRepository fake for the public endpoint tests.
type publicFake struct {
	*fakeBudgetRepo
	stored        *entities.BudgetEntity
	token         string
	customerEmail string
	items         []*entities.BudgetItemEntity
}

func (p *publicFake) GetItems(_ context.Context, _ uuid.UUID) ([]*entities.BudgetItemEntity, error) {
	return p.items, nil
}

func (p *publicFake) FindByPublicToken(_ context.Context, token string) (*entities.BudgetEntity, error) {
	if p.stored == nil || token != p.token {
		return nil, entities.ErrBudgetNotFound
	}
	cp := *p.stored
	return &cp, nil
}

func (p *publicFake) GetCustomerInfo(_ context.Context, id uuid.UUID, _ string) (*entities.CustomerInfo, error) {
	email := p.customerEmail
	phone := "+55 11 99999-9999"
	doc := "123.456.789-00"
	return &entities.CustomerInfo{ID: id.String(), Name: "Cliente Teste", Email: &email, Phone: &phone, Document: &doc}, nil
}

func (p *publicFake) RespondToPublicBudget(_ context.Context, _ uuid.UUID, newStatus entities.BudgetStatus, name, _, _ string, reason *string, at time.Time) (int64, error) {
	if p.stored.Status == entities.StatusSent && !p.stored.IsExpired(time.Now()) {
		p.stored.Status = newStatus
		t := at
		p.stored.CustomerResponseAt = &t
		n := name
		p.stored.CustomerResponseName = &n
		p.stored.RejectionReason = reason
		return 1, nil
	}
	return 0, nil
}

func newPublicUC(repo *publicFake) *PublicBudgetUseCase {
	return &PublicBudgetUseCase{
		budgetRepository: repo,
		activityService:  noopActivity{},
		logger:           logger.NewLogger("test"),
	}
}

func newPublicFake(b *entities.BudgetEntity, token string) *publicFake {
	return &publicFake{fakeBudgetRepo: &fakeBudgetRepo{}, stored: b, token: token, customerEmail: "secret@example.com"}
}

func TestPublicGet_SanitizesSensitiveFields(t *testing.T) {
	id := uuid.New()
	future := time.Now().Add(48 * time.Hour)
	b := &entities.BudgetEntity{
		ID:             id,
		OrganizationID: "org-secret-12345",
		Name:           "Projeto X",
		Status:         entities.StatusSent,
		ValidUntil:     &future,
		FilamentCost:   9999,
		ProfitAmount:   5555,
		TotalCost:      12000,
		QuoteNumber:    intPtr(42),
	}
	repo := newPublicFake(b, "tok123")
	uc := newPublicUC(repo)

	c, w := newJSONContext(http.MethodGet, "/v1/public/budgets/tok123", "", gin.Params{{Key: "token", Value: "tok123"}})
	uc.GetByToken(c)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d body %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, forbidden := range []string{"filament_cost", "organization_id", "profit", "secret@example.com", "123.456.789-00"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("sanitized view must NOT contain %q; body=%s", forbidden, body)
		}
	}

	var view entities.PublicBudgetView
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode view: %v", err)
	}
	if view.QuoteNumber == nil || *view.QuoteNumber != 42 {
		t.Errorf("quote_number not surfaced: %v", view.QuoteNumber)
	}
	if !view.CanRespond {
		t.Error("a non-expired sent budget should be answerable")
	}
	if view.Status != string(entities.StatusSent) {
		t.Errorf("status: got %q want sent", view.Status)
	}
}

func TestPublicGet_UnknownAndRevokedTokenReturn404(t *testing.T) {
	// Unknown token: repo has a budget under a different token.
	repo := newPublicFake(&entities.BudgetEntity{ID: uuid.New(), Status: entities.StatusSent}, "real-token")
	uc := newPublicUC(repo)

	for _, tok := range []string{"does-not-exist", ""} {
		c, w := newJSONContext(http.MethodGet, "/v1/public/budgets/"+tok, "", gin.Params{{Key: "token", Value: tok}})
		uc.GetByToken(c)
		if w.Code != http.StatusNotFound {
			t.Fatalf("token %q: got %d want 404", tok, w.Code)
		}
		var body map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if body["code"] != CodePublicBudgetNotFound {
			t.Errorf("token %q: got code %v want %s", tok, body["code"], CodePublicBudgetNotFound)
		}
	}
}

func TestPublicGet_ExpiredComputed(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	b := &entities.BudgetEntity{ID: uuid.New(), OrganizationID: "org-a", Name: "X", Status: entities.StatusSent, ValidUntil: &past}
	repo := newPublicFake(b, "tok")
	uc := newPublicUC(repo)

	c, w := newJSONContext(http.MethodGet, "/v1/public/budgets/tok", "", gin.Params{{Key: "token", Value: "tok"}})
	uc.GetByToken(c)

	var view entities.PublicBudgetView
	_ = json.Unmarshal(w.Body.Bytes(), &view)
	if view.Status != string(entities.StatusExpired) {
		t.Errorf("status: got %q want expired", view.Status)
	}
	if !view.IsExpired {
		t.Error("is_expired should be true")
	}
	if view.CanRespond {
		t.Error("expired budget should not be answerable")
	}
}

func TestPublicApprove_HappyPath(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	b := &entities.BudgetEntity{ID: uuid.New(), OrganizationID: "org-a", Name: "X", Status: entities.StatusSent, ValidUntil: &future, QuoteNumber: intPtr(3)}
	repo := newPublicFake(b, "tok")
	uc := newPublicUC(repo)

	c, w := newJSONContext(http.MethodPost, "/v1/public/budgets/tok/approve", `{"name":"Alice"}`, gin.Params{{Key: "token", Value: "tok"}})
	uc.Approve(c)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d body %s", w.Code, w.Body.String())
	}
	var view entities.PublicBudgetView
	_ = json.Unmarshal(w.Body.Bytes(), &view)
	if view.Status != string(entities.StatusApproved) {
		t.Errorf("status: got %q want approved", view.Status)
	}
	if view.CustomerResponseName == nil || *view.CustomerResponseName != "Alice" {
		t.Errorf("customer_response_name not recorded: %v", view.CustomerResponseName)
	}
}

func TestPublicReject_ValidatesName(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	b := &entities.BudgetEntity{ID: uuid.New(), OrganizationID: "org-a", Name: "X", Status: entities.StatusSent, ValidUntil: &future}
	repo := newPublicFake(b, "tok")
	uc := newPublicUC(repo)

	c, w := newJSONContext(http.MethodPost, "/v1/public/budgets/tok/reject", `{"name":" ","reason":"no"}`, gin.Params{{Key: "token", Value: "tok"}})
	uc.Reject(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty name should be 400, got %d", w.Code)
	}
}

func TestPublicApprove_AlreadyRespondedReturns409(t *testing.T) {
	b := &entities.BudgetEntity{ID: uuid.New(), OrganizationID: "org-a", Name: "X", Status: entities.StatusApproved}
	repo := newPublicFake(b, "tok")
	uc := newPublicUC(repo)

	c, w := newJSONContext(http.MethodPost, "/v1/public/budgets/tok/approve", `{"name":"Bob"}`, gin.Params{{Key: "token", Value: "tok"}})
	uc.Approve(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("got status %d want 409", w.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != CodeBudgetAlreadyResponded {
		t.Errorf("got code %v want %s", body["code"], CodeBudgetAlreadyResponded)
	}
}

func TestPublicApprove_ExpiredReturns410(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	b := &entities.BudgetEntity{ID: uuid.New(), OrganizationID: "org-a", Name: "X", Status: entities.StatusSent, ValidUntil: &past}
	repo := newPublicFake(b, "tok")
	uc := newPublicUC(repo)

	c, w := newJSONContext(http.MethodPost, "/v1/public/budgets/tok/approve", `{"name":"Bob"}`, gin.Params{{Key: "token", Value: "tok"}})
	uc.Approve(c)

	if w.Code != http.StatusGone {
		t.Fatalf("got status %d want 410", w.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != CodeBudgetExpired {
		t.Errorf("got code %v want %s", body["code"], CodeBudgetExpired)
	}
}

func TestRateLimiter_NilRedisFailsOpen(t *testing.T) {
	uc := &PublicBudgetUseCase{logger: logger.NewLogger("test")} // redisService nil
	for i := 0; i < 1000; i++ {
		if !uc.allowRequest(context.Background(), "get", "1.2.3.4", publicGetRateLimit) {
			t.Fatalf("nil Redis must fail open (allow) on request %d", i)
		}
	}
}

func TestRateLimiter_MiniredisEnforcesLimit(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	cfg := &config.AppConfig{RedisHost: mr.Host(), RedisPort: mr.Port()}
	rs := services.NewRedisService(logger.NewLogger("test"), cfg)
	_ = rs.Init()

	uc := &PublicBudgetUseCase{redisService: rs, logger: logger.NewLogger("test")}

	ctx := context.Background()
	limit := 10
	ip := "9.9.9.9"
	for i := 1; i <= limit; i++ {
		if !uc.allowRequest(ctx, "post", ip, limit) {
			t.Fatalf("request %d within limit should be allowed", i)
		}
	}
	if uc.allowRequest(ctx, "post", ip, limit) {
		t.Fatal("request over the limit should be denied")
	}
	// A different IP has its own counter.
	if !uc.allowRequest(ctx, "post", "8.8.8.8", limit) {
		t.Fatal("a different IP must have an independent bucket")
	}
}

func TestPublicGet_ItemSaleValuesReconcileToBasePrice(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	// base_price = total + discount - shipping - tax = 11000 + 1000 - 2000 - 500 = 9500
	b := &entities.BudgetEntity{
		ID:             uuid.New(),
		OrganizationID: "org-a",
		Name:           "X",
		Status:         entities.StatusSent,
		ValidUntil:     &future,
		TotalCost:      11000,
		DiscountAmount: 1000,
		ShippingCost:   2000,
		TaxAmount:      500,
	}
	repo := newPublicFake(b, "tok")
	repo.items = []*entities.BudgetItemEntity{
		{ID: uuid.New(), ProductQuantity: 1, ItemTotalCost: 3000},
		{ID: uuid.New(), ProductQuantity: 2, ItemTotalCost: 1000},
	}
	uc := newPublicUC(repo)

	c, w := newJSONContext(http.MethodGet, "/v1/public/budgets/tok", "", gin.Params{{Key: "token", Value: "tok"}})
	uc.GetByToken(c)
	if w.Code != http.StatusOK {
		t.Fatalf("got status %d body %s", w.Code, w.Body.String())
	}

	var view entities.PublicBudgetView
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode: %v", err)
	}

	var sum int64
	for _, it := range view.Items {
		sum += it.TotalPrice
	}
	if sum != view.BasePrice {
		t.Errorf("sum(item total_price)=%d must equal base_price=%d", sum, view.BasePrice)
	}
	if view.BasePrice != 9500 {
		t.Errorf("base_price: got %d want 9500", view.BasePrice)
	}
	// And the customer-facing identity holds: Subtotal - discount + shipping + tax == total.
	if view.BasePrice-view.DiscountAmount+view.ShippingCost+view.TaxAmount != view.Total {
		t.Errorf("reconciliation failed: base=%d disc=%d ship=%d tax=%d total=%d",
			view.BasePrice, view.DiscountAmount, view.ShippingCost, view.TaxAmount, view.Total)
	}
}
