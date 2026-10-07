package usecases

import (
	"net/http"
	"testing"
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// draftWithTriState returns a draft budget with every tri-state field populated, so a
// test can assert that an absent field keeps it, a null clears it and a value replaces it.
func draftWithTriState(id uuid.UUID) *entities.BudgetEntity {
	taxRate := 10.0
	discType := entities.DiscountTypePercent
	discVal := 5.0
	var ship int64 = 1500
	vu := time.Date(2026, 1, 2, 23, 59, 59, 0, time.UTC)
	return &entities.BudgetEntity{
		ID:               id,
		OrganizationID:   "org-a",
		Status:           entities.StatusDraft,
		TaxRate:          &taxRate,
		DiscountType:     &discType,
		DiscountValue:    &discVal,
		ShippingOverride: &ship,
		ValidUntil:       &vu,
	}
}

func runUpdate(t *testing.T, budget *entities.BudgetEntity, body string) (*entities.BudgetEntity, int) {
	t.Helper()
	repo := &fakeBudgetRepo{budget: budget}
	uc := newUseCaseWith(repo)
	c, w := newJSONContext(http.MethodPut, "/v1/budgets/"+budget.ID.String(), body, gin.Params{{Key: "id", Value: budget.ID.String()}})
	uc.Update(c)
	return repo.budget, w.Code
}

// --- tax_rate -------------------------------------------------------------------

func TestUpdate_TaxRate_Absent_Unchanged(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"name":"x"}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.TaxRate == nil || *b.TaxRate != 10.0 {
		t.Errorf("tax_rate = %v, want unchanged 10", b.TaxRate)
	}
}

func TestUpdate_TaxRate_Null_Clears(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"tax_rate":null}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.TaxRate != nil {
		t.Errorf("tax_rate = %v, want nil (cleared)", *b.TaxRate)
	}
}

func TestUpdate_TaxRate_Value_Sets(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"tax_rate":7.5}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.TaxRate == nil || *b.TaxRate != 7.5 {
		t.Errorf("tax_rate = %v, want 7.5", b.TaxRate)
	}
}

func TestUpdate_TaxRate_OutOfRange_400(t *testing.T) {
	_, code := runUpdate(t, draftWithTriState(uuid.New()), `{"tax_rate":150}`)
	if code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", code)
	}
}

// --- shipping_override ----------------------------------------------------------

func TestUpdate_ShippingOverride_Absent_Unchanged(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"name":"x"}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.ShippingOverride == nil || *b.ShippingOverride != 1500 {
		t.Errorf("shipping_override = %v, want unchanged 1500", b.ShippingOverride)
	}
}

func TestUpdate_ShippingOverride_Null_Clears(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"shipping_override":null}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.ShippingOverride != nil {
		t.Errorf("shipping_override = %v, want nil (cleared)", *b.ShippingOverride)
	}
}

func TestUpdate_ShippingOverride_Value_Sets(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"shipping_override":2000}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.ShippingOverride == nil || *b.ShippingOverride != 2000 {
		t.Errorf("shipping_override = %v, want 2000", b.ShippingOverride)
	}
}

func TestUpdate_ShippingOverride_Negative_400(t *testing.T) {
	_, code := runUpdate(t, draftWithTriState(uuid.New()), `{"shipping_override":-5}`)
	if code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", code)
	}
}

// --- valid_until ----------------------------------------------------------------

func TestUpdate_ValidUntil_Absent_Unchanged(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"name":"x"}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.ValidUntil == nil {
		t.Errorf("valid_until cleared unexpectedly")
	}
}

func TestUpdate_ValidUntil_Null_Clears(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"valid_until":null}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.ValidUntil != nil {
		t.Errorf("valid_until = %v, want nil (cleared)", *b.ValidUntil)
	}
}

func TestUpdate_ValidUntil_Value_Sets(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"valid_until":"2026-03-10T10:00:00Z"}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.ValidUntil == nil {
		t.Fatalf("valid_until nil, want set")
	}
	// normalizeValidUntil snaps to end of day in America/Sao_Paulo; the calendar day
	// (10 March) must survive.
	if got := b.ValidUntil.In(saoPauloLocation); got.Day() != 10 || got.Month() != time.March {
		t.Errorf("valid_until = %v, want 2026-03-10 (SP)", got)
	}
}

// --- discount pair --------------------------------------------------------------

func TestUpdate_Discount_Absent_Unchanged(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"name":"x"}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.DiscountType == nil || *b.DiscountType != entities.DiscountTypePercent || b.DiscountValue == nil || *b.DiscountValue != 5.0 {
		t.Errorf("discount changed: type=%v value=%v", b.DiscountType, b.DiscountValue)
	}
}

func TestUpdate_Discount_TypeNull_ClearsBoth(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"discount_type":null}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.DiscountType != nil || b.DiscountValue != nil {
		t.Errorf("want both cleared, got type=%v value=%v", b.DiscountType, b.DiscountValue)
	}
}

func TestUpdate_Discount_ValueNull_ClearsBoth(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"discount_value":null}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.DiscountType != nil || b.DiscountValue != nil {
		t.Errorf("want both cleared, got type=%v value=%v", b.DiscountType, b.DiscountValue)
	}
}

// Null wins: discount_type:null together with a discount_value clears both.
func TestUpdate_Discount_TypeNullWithValue_ClearsBoth(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"discount_type":null,"discount_value":20}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.DiscountType != nil || b.DiscountValue != nil {
		t.Errorf("null must win: got type=%v value=%v", b.DiscountType, b.DiscountValue)
	}
}

func TestUpdate_Discount_SetBoth(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"discount_type":"fixed","discount_value":100}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.DiscountType == nil || *b.DiscountType != entities.DiscountTypeFixed || b.DiscountValue == nil || *b.DiscountValue != 100 {
		t.Errorf("want fixed/100, got type=%v value=%v", b.DiscountType, b.DiscountValue)
	}
}

// Only discount_value while the stored type is nil -> 400 via the paired validation.
func TestUpdate_Discount_ValueOnly_NoStoredType_400(t *testing.T) {
	id := uuid.New()
	budget := &entities.BudgetEntity{ID: id, OrganizationID: "org-a", Status: entities.StatusDraft}
	_, code := runUpdate(t, budget, `{"discount_value":20}`)
	if code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", code)
	}
}

// Updating only the value when a type is already stored is valid (paired against the
// effective pair).
func TestUpdate_Discount_ValueOnly_WithStoredType_OK(t *testing.T) {
	b, code := runUpdate(t, draftWithTriState(uuid.New()), `{"discount_value":12}`)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	if b.DiscountType == nil || *b.DiscountType != entities.DiscountTypePercent || b.DiscountValue == nil || *b.DiscountValue != 12 {
		t.Errorf("want percent/12, got type=%v value=%v", b.DiscountType, b.DiscountValue)
	}
}
