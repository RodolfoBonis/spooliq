package usecases_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/crypto"
	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/usecases"
	"github.com/gin-gonic/gin"
)

func authCtx(method, body, org string, roles []string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, "/v1/company/payments/infinitepay", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Set("organization_id", org)
	c.Set("user_id", "user-1")
	c.Set("user_roles", roles)
	return c, w
}

func newIntegrationUC(repo *fakeRepo, client *fakeClient) *usecases.IntegrationUseCase {
	svc := usecases.NewService(repo, client, testBox(), testConfig(), nil, nil, nil, logger.NewLogger("test"))
	return usecases.NewIntegrationUseCase(svc)
}

func TestIntegration_RejectedKeyReturns422(t *testing.T) {
	repo := newFakeRepo()
	client := &fakeClient{checkErr: coreServices.ErrIPUnauthorized}
	uc := newIntegrationUC(repo, client)

	body := `{"handle":"mystore","api_key":"sk_live_badkey","enabled":true,"assured_installments":3,"pix_discount_percent":5}`
	c, w := authCtx(http.MethodPut, body, "org1", []string{"Owner"})
	uc.Upsert(c)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d want 422; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "infinitepay_invalid_credentials") {
		t.Fatalf("body missing code: %s", w.Body.String())
	}
}

func TestIntegration_KeyNeverReturnedAndStoredEncrypted(t *testing.T) {
	repo := newFakeRepo()
	// A 4xx that is not auth means the key was accepted (order simply not found).
	client := &fakeClient{checkErr: coreServices.ErrIPInvalid}
	uc := newIntegrationUC(repo, client)

	rawKey := "sk_live_secret_key_value_9999"
	body := `{"handle":"mystore","api_key":"` + rawKey + `","enabled":true,"assured_installments":6,"pix_discount_percent":0}`
	c, w := authCtx(http.MethodPut, body, "org1", []string{"Owner"})
	uc.Upsert(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d want 200; body=%s", w.Code, w.Body.String())
	}
	bodyStr := w.Body.String()
	if strings.Contains(bodyStr, rawKey) {
		t.Fatalf("response leaked the raw API key: %s", bodyStr)
	}
	// The response must not carry an api_key field at all.
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["api_key"]; ok {
		t.Fatalf("response must not include api_key")
	}
	if resp["api_key_last4"] != "9999" {
		t.Fatalf("api_key_last4 = %v want 9999", resp["api_key_last4"])
	}
	// Stored ciphertext must not equal the plaintext.
	integ, _ := repo.GetIntegration(c.Request.Context(), "org1", "infinitepay")
	if integ == nil || integ.APIKeyCiphertext == "" || strings.Contains(integ.APIKeyCiphertext, rawKey) {
		t.Fatalf("stored key not encrypted: %+v", integ)
	}
}

func TestIntegration_UserDoesNotSeeSensitiveFields(t *testing.T) {
	repo := newFakeRepo()
	repo.seedIntegration("org1", testBox(), 5, 3, nil)
	uc := newIntegrationUC(repo, &fakeClient{})

	c, w := authCtx(http.MethodGet, "", "org1", []string{"User"})
	uc.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d want 200", w.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["api_key_last4"]; ok {
		t.Fatalf("User must not see api_key_last4")
	}
	if _, ok := resp["pending_links"]; ok {
		t.Fatalf("User must not see pending_links")
	}
}

func TestIntegration_HandleChangeWithUndecryptableKeyReturns503(t *testing.T) {
	repo := newFakeRepo()
	// Seed an integration whose key was encrypted with a DIFFERENT box (the UC's box
	// cannot decrypt it).
	otherKey := base64.StdEncoding.EncodeToString([]byte("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ"))
	otherBox, _ := crypto.NewSecretBox(otherKey)
	ct, _ := otherBox.EncryptString("sk_live_oldkey")
	repo.seedIntegrationRaw("org1", ct, "mystore")

	uc := newIntegrationUC(repo, &fakeClient{})
	// Handle-only change (no api_key) => UC must decrypt the stored key, which fails.
	body := `{"handle":"newstore","enabled":true,"assured_installments":1,"pix_discount_percent":0}`
	c, w := authCtx(http.MethodPut, body, "org1", []string{"Owner"})
	uc.Upsert(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d want 503; body=%s", w.Code, w.Body.String())
	}
}
