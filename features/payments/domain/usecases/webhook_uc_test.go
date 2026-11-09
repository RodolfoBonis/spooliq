package usecases_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/usecases"
	"github.com/gin-gonic/gin"
)

func newWebhookUC(repo *fakeRepo) *usecases.WebhookUseCase {
	box := testBox()
	svc := usecases.NewService(repo, &fakeClient{}, box, testConfig(), nil, nil, nil, logger.NewLogger("test"))
	return usecases.NewWebhookUseCase(svc, usecases.NewConfirmService(svc))
}

func webhookCtx(orgID, token string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/infinitepay/"+orgID+"/"+token, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = gin.Params{{Key: "orgId", Value: orgID}, {Key: "token", Value: token}}
	return c, w
}

func TestWebhook_ValidTokenReturns200(t *testing.T) {
	repo := newFakeRepo()
	repo.seedIntegration("org1", testBox(), 0, 1, nil) // webhook token = "webhooktoken"
	uc := newWebhookUC(repo)

	c, w := webhookCtx("org1", "webhooktoken", []byte(`{"order_nsu":"spq_x","transaction_nsu":"t","slug":"s"}`))
	uc.Handle(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestWebhook_BadTokenReturns401(t *testing.T) {
	repo := newFakeRepo()
	repo.seedIntegration("org1", testBox(), 0, 1, nil)
	uc := newWebhookUC(repo)

	c, w := webhookCtx("org1", "not-the-token", []byte(`{"order_nsu":"spq_x"}`))
	uc.Handle(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d want 401", w.Code)
	}
}

func TestWebhook_UnknownOrgReturns401(t *testing.T) {
	repo := newFakeRepo()
	uc := newWebhookUC(repo)

	c, w := webhookCtx("ghost", "webhooktoken", []byte(`{"order_nsu":"spq_x"}`))
	uc.Handle(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d want 401", w.Code)
	}
}

func TestWebhook_OversizedBodyReturns413(t *testing.T) {
	repo := newFakeRepo()
	repo.seedIntegration("org1", testBox(), 0, 1, nil)
	uc := newWebhookUC(repo)

	big := make([]byte, 64*1024+10)
	for i := range big {
		big[i] = 'a'
	}
	c, w := webhookCtx("org1", "webhooktoken", big)
	uc.Handle(c)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d want 413", w.Code)
	}
}
