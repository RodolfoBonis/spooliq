package usecases_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
)

// TestIntegration_EmbedFeeWithoutReferenceRateReturns422 asserts the price-embed switch
// is rejected (422 payment_fee_rate_missing) when the reference card rate (credit at
// assured_installments) is not configured.
func TestIntegration_EmbedFeeWithoutReferenceRateReturns422(t *testing.T) {
	repo := newFakeRepo()
	client := &fakeClient{checkErr: coreServices.ErrIPInvalid} // key accepted
	uc := newIntegrationUC(repo, client)

	// assured_installments=3 but credit_installments has no "3" entry.
	body := `{"handle":"mystore","api_key":"sk_live_key_123456","enabled":true,` +
		`"assured_installments":3,"pix_discount_percent":0,` +
		`"fee_rates":{"pix":0,"credit_1x":4.2,"credit_installments":{"2":5.4}},` +
		`"embed_fee_in_price":true}`
	c, w := authCtx(http.MethodPut, body, "org1", []string{"Owner"})
	uc.Upsert(c)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d want 422; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "payment_fee_rate_missing") {
		t.Fatalf("body missing code: %s", w.Body.String())
	}
}

// TestIntegration_EmbedFeeWithReferenceRateSucceeds asserts the switch is accepted and
// the fee table is persisted and returned when the reference rate is present.
func TestIntegration_EmbedFeeWithReferenceRateSucceeds(t *testing.T) {
	repo := newFakeRepo()
	client := &fakeClient{checkErr: coreServices.ErrIPInvalid}
	uc := newIntegrationUC(repo, client)

	body := `{"handle":"mystore","api_key":"sk_live_key_123456","enabled":true,` +
		`"assured_installments":3,"pix_discount_percent":0,` +
		`"fee_rates":{"pix":0,"credit_1x":4.2,"credit_installments":{"2":5.4,"3":6.1}},` +
		`"embed_fee_in_price":true}`
	c, w := authCtx(http.MethodPut, body, "org1", []string{"Owner"})
	uc.Upsert(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d want 200; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["embed_fee_in_price"] != true {
		t.Fatalf("embed_fee_in_price = %v want true", resp["embed_fee_in_price"])
	}
	fr, ok := resp["fee_rates"].(map[string]interface{})
	if !ok {
		t.Fatalf("fee_rates missing in response: %v", resp["fee_rates"])
	}
	if fr["credit_1x"].(float64) != 4.2 {
		t.Fatalf("credit_1x = %v want 4.2", fr["credit_1x"])
	}

	// Persisted.
	integ, _ := repo.GetIntegration(c.Request.Context(), "org1", "infinitepay")
	if integ == nil || integ.FeeRates == nil || !integ.EmbedFeeInPrice {
		t.Fatalf("fee settings not persisted: %+v", integ)
	}
	if r, present := integ.FeeRates.ReferenceRate(3); !present || r != 6.1 {
		t.Fatalf("reference rate not stored: %v %v", r, present)
	}
}

// TestIntegration_InvalidFeeRateReturns422 asserts out-of-range fee rates are rejected.
func TestIntegration_InvalidFeeRateReturns422(t *testing.T) {
	repo := newFakeRepo()
	client := &fakeClient{checkErr: coreServices.ErrIPInvalid}
	uc := newIntegrationUC(repo, client)

	body := `{"handle":"mystore","api_key":"sk_live_key_123456","enabled":true,` +
		`"assured_installments":1,"pix_discount_percent":0,` +
		`"fee_rates":{"pix":0,"credit_1x":45.0}}`
	c, w := authCtx(http.MethodPut, body, "org1", []string{"Owner"})
	uc.Upsert(c)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d want 422; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "payment_fee_rate_invalid") {
		t.Fatalf("body missing code: %s", w.Body.String())
	}
}

// TestIntegration_UserSeesFeeSettings asserts the fee settings are returned for the
// User role (non-sensitive).
func TestIntegration_UserSeesFeeSettings(t *testing.T) {
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", testBox(), 0, 3, nil)
	integ.FeeRates = &entities.FeeRates{Pix: 0, Credit1x: 4.2, CreditInstallments: map[string]float64{"3": 6.1}}
	integ.EmbedFeeInPrice = true
	uc := newIntegrationUC(repo, &fakeClient{})

	c, w := authCtx(http.MethodGet, "", "org1", []string{"User"})
	uc.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d want 200; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["embed_fee_in_price"] != true {
		t.Fatalf("User should see embed_fee_in_price; got %v", resp["embed_fee_in_price"])
	}
	if _, ok := resp["fee_rates"].(map[string]interface{}); !ok {
		t.Fatalf("User should see fee_rates; got %v", resp["fee_rates"])
	}
	// Sensitive field still hidden.
	if _, ok := resp["api_key_last4"]; ok {
		t.Fatalf("User must not see api_key_last4")
	}
}

// TestIntegration_UpsertWithoutFeeKeysKeepsThem asserts that a PUT which omits
// fee_rates and embed_fee_in_price (an older desktop) KEEPS the stored table and flag
// rather than wiping them.
func TestIntegration_UpsertWithoutFeeKeysKeepsThem(t *testing.T) {
	repo := newFakeRepo()
	// Seed an existing integration with a fee table and embed on.
	integ := repo.seedIntegration("org1", testBox(), 0, 3, nil)
	integ.FeeRates = &entities.FeeRates{Pix: 0, Credit1x: 4.2, CreditInstallments: map[string]float64{"3": 6.1}}
	integ.EmbedFeeInPrice = true
	uc := newIntegrationUC(repo, &fakeClient{checkErr: coreServices.ErrIPInvalid})

	// Body WITHOUT fee_rates / embed_fee_in_price (keeps the key via handle change only).
	body := `{"handle":"mystore","enabled":true,"assured_installments":3,"pix_discount_percent":0}`
	c, w := authCtx(http.MethodPut, body, "org1", []string{"Owner"})
	uc.Upsert(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d want 200; body=%s", w.Code, w.Body.String())
	}
	got, _ := repo.GetIntegration(c.Request.Context(), "org1", "infinitepay")
	if got.FeeRates == nil {
		t.Fatalf("fee table was wiped by a PUT that omitted fee_rates")
	}
	if !got.EmbedFeeInPrice {
		t.Fatalf("embed flag was reset by a PUT that omitted embed_fee_in_price")
	}
	if r, ok := got.FeeRates.ReferenceRate(3); !ok || r != 6.1 {
		t.Fatalf("stored rate changed: %v %v", r, ok)
	}
}

// TestIntegration_UpsertExplicitNullClearsFeeTable asserts an explicit null clears the
// table (with embed turned off in the same request).
func TestIntegration_UpsertExplicitNullClearsFeeTable(t *testing.T) {
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", testBox(), 0, 3, nil)
	integ.FeeRates = &entities.FeeRates{Credit1x: 4.2, CreditInstallments: map[string]float64{"3": 6.1}}
	integ.EmbedFeeInPrice = false
	uc := newIntegrationUC(repo, &fakeClient{checkErr: coreServices.ErrIPInvalid})

	body := `{"handle":"mystore","enabled":true,"assured_installments":3,"pix_discount_percent":0,` +
		`"fee_rates":null,"embed_fee_in_price":false}`
	c, w := authCtx(http.MethodPut, body, "org1", []string{"Owner"})
	uc.Upsert(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d want 200; body=%s", w.Code, w.Body.String())
	}
	got, _ := repo.GetIntegration(c.Request.Context(), "org1", "infinitepay")
	if got.FeeRates != nil {
		t.Fatalf("explicit null should clear the table, got %+v", got.FeeRates)
	}
}

// TestIntegration_UpsertClearingTableWhileEmbedStaysOnIs422 asserts you cannot clear the
// table (explicit null) while the embed flag remains on (kept from storage).
func TestIntegration_UpsertClearingTableWhileEmbedStaysOnIs422(t *testing.T) {
	repo := newFakeRepo()
	integ := repo.seedIntegration("org1", testBox(), 0, 3, nil)
	integ.FeeRates = &entities.FeeRates{Credit1x: 4.2, CreditInstallments: map[string]float64{"3": 6.1}}
	integ.EmbedFeeInPrice = true // stays on (not sent in the request)
	uc := newIntegrationUC(repo, &fakeClient{checkErr: coreServices.ErrIPInvalid})

	body := `{"handle":"mystore","enabled":true,"assured_installments":3,"pix_discount_percent":0,"fee_rates":null}`
	c, w := authCtx(http.MethodPut, body, "org1", []string{"Owner"})
	uc.Upsert(c)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d want 422; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "payment_fee_rate_missing") {
		t.Fatalf("body missing code: %s", w.Body.String())
	}
}
