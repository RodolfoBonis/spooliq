package company

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	otellogger "github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/features/company/domain/usecases"
	subscriptionEntities "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopLogger struct{}

func (noopLogger) Debug(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Info(context.Context, string, ...otellogger.Fields)    {}
func (noopLogger) Warning(context.Context, string, ...otellogger.Fields) {}
func (noopLogger) Error(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Fatal(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Panic(context.Context, string, ...otellogger.Fields)   {}
func (n noopLogger) With(otellogger.Fields) otellogger.Logger            { return n }
func (noopLogger) LogError(context.Context, string, error)               {}

// stubSubscriptionRepo implements subscriptions.SubscriptionRepository for the
// payment-history test, returning a fixed page of payments scoped by org.
type stubSubscriptionRepo struct {
	payments  []*subscriptionEntities.SubscriptionEntity
	lastOrg   uuid.UUID
	lastLimit int
	lastOff   int
}

func (r *stubSubscriptionRepo) FindAll(_ context.Context, org uuid.UUID, limit, offset int) ([]*subscriptionEntities.SubscriptionEntity, error) {
	r.lastOrg = org
	r.lastLimit = limit
	r.lastOff = offset
	end := offset + limit
	if end > len(r.payments) {
		end = len(r.payments)
	}
	if offset > len(r.payments) {
		offset = len(r.payments)
	}
	return r.payments[offset:end], nil
}
func (r *stubSubscriptionRepo) FindByID(context.Context, uuid.UUID, uuid.UUID) (*subscriptionEntities.SubscriptionEntity, error) {
	return nil, nil
}
func (r *stubSubscriptionRepo) FindByAsaasPaymentID(context.Context, string) (*subscriptionEntities.SubscriptionEntity, error) {
	return nil, nil
}
func (r *stubSubscriptionRepo) Create(context.Context, *subscriptionEntities.SubscriptionEntity) error {
	return nil
}
func (r *stubSubscriptionRepo) Update(context.Context, uuid.UUID, uuid.UUID, *subscriptionEntities.SubscriptionEntity) error {
	return nil
}
func (r *stubSubscriptionRepo) UpdateByEntity(context.Context, *subscriptionEntities.SubscriptionEntity) error {
	return nil
}
func (r *stubSubscriptionRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (r *stubSubscriptionRepo) CountByOrganizationID(context.Context, uuid.UUID) (int64, error) {
	return int64(len(r.payments)), nil
}

// The owner payment-history endpoint returns subscription PAYMENTS (not methods)
// in the standard envelope, with the documented item fields.
func TestListMyPayments_Envelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	org := uuid.New()
	paid := time.Now()
	repo := &stubSubscriptionRepo{payments: []*subscriptionEntities.SubscriptionEntity{
		{ID: uuid.New(), OrganizationID: org.String(), Amount: 100, Status: "CONFIRMED", DueDate: time.Now(), PaymentDate: &paid, InvoiceURL: "http://x/1"},
		{ID: uuid.New(), OrganizationID: org.String(), Amount: 200, Status: "PENDING", DueDate: time.Now()},
	}}
	uc := usecases.NewSubscriptionPaymentsUseCase(repo, noopLogger{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/company/subscription/payments", nil)
	c.Set("organization_id", org.String())

	uc.ListMyPayments(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var env struct {
		Data []struct {
			ID         string  `json:"id"`
			Amount     float64 `json:"amount"`
			Status     string  `json:"status"`
			InvoiceURL string  `json:"invoice_url"`
		} `json:"data"`
		Total    int64 `json:"total"`
		PageSize int   `json:"page_size"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, int64(2), env.Total)
	require.Len(t, env.Data, 2)
	assert.Equal(t, float64(100), env.Data[0].Amount)
	assert.Equal(t, "CONFIRMED", env.Data[0].Status)
	assert.Equal(t, org, repo.lastOrg)
	assert.Equal(t, 20, env.PageSize)
}

// Missing organization yields a 401 with a stable code.
func TestListMyPayments_MissingOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := usecases.NewSubscriptionPaymentsUseCase(&stubSubscriptionRepo{}, noopLogger{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/company/subscription/payments", nil)

	uc.ListMyPayments(c)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "organization_id_missing", body.Code)
}
