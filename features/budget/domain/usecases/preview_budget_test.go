package usecases

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	pricing "github.com/RodolfoBonis/spooliq/features/budget/domain/services"
	"github.com/google/uuid"
)

// TestPreview_ReturnsBreakdownAndNeverPersists proves the preview endpoint returns
// the full cost + sale breakdown from the pricing engine and writes NOTHING
// (no Create, no AddItem, no CalculateCosts).
func TestPreview_ReturnsBreakdownAndNeverPersists(t *testing.T) {
	repo := &fakeBudgetRepo{
		pricingResult: pricing.PricingResult{
			Items: []pricing.PricingItemResult{
				{FilamentCost: 1200, ItemTotalCost: 1200, UnitCost: 1200, SaleTotal: 1320, SaleUnitPrice: 1320},
			},
			FilamentCost: 1200,
			Subtotal:     1200,
			Total:        1320,
		},
	}
	uc := newUseCaseWith(repo)

	fil := uuid.New()
	body := `{
		"name": "Preview test",
		"items": [
			{"product_name":"Cube","product_quantity":1,"filaments":[{"filament_id":"` + fil.String() + `","quantity":100,"order":1}]}
		]
	}`

	c, w := newJSONContext(http.MethodPost, "/v1/budgets/preview", body, nil)
	uc.Preview(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp entities.BudgetResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v (body: %s)", err, w.Body.String())
	}

	if resp.TotalCost != 1320 {
		t.Errorf("total_cost = %d, want 1320", resp.TotalCost)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].SaleTotal != 1320 {
		t.Errorf("item sale_total = %d, want 1320", resp.Items[0].SaleTotal)
	}
	if resp.Items[0].SaleUnitPrice != 1320 {
		t.Errorf("item sale_unit_price = %d, want 1320", resp.Items[0].SaleUnitPrice)
	}
	if resp.Items[0].ItemTotalCost != 1200 {
		t.Errorf("item item_total_cost = %d, want 1200", resp.Items[0].ItemTotalCost)
	}

	// Core guarantee: a preview must not persist anything.
	if repo.createCalls != 0 || repo.addItemCalls != 0 || repo.calcCalls != 0 {
		t.Errorf("preview persisted data: create=%d addItem=%d calc=%d", repo.createCalls, repo.addItemCalls, repo.calcCalls)
	}
}

// TestPreview_RejectsEmptyItems proves invalid input yields a 400 (pt-BR friendly)
// and no computation/persistence is attempted.
func TestPreview_RejectsEmptyItems(t *testing.T) {
	repo := &fakeBudgetRepo{}
	uc := newUseCaseWith(repo)

	c, w := newJSONContext(http.MethodPost, "/v1/budgets/preview", `{"items":[]}`, nil)
	uc.Preview(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", w.Code, w.Body.String())
	}
	if repo.createCalls != 0 || repo.calcCalls != 0 {
		t.Errorf("preview with invalid input must not persist: create=%d calc=%d", repo.createCalls, repo.calcCalls)
	}
}

// TestPreview_ValidatesFilamentPresetTypes proves the preview validates referenced
// presets with the correct slot type before computing.
func TestPreview_ValidatesPresetTypes(t *testing.T) {
	machine := uuid.New()
	energy := uuid.New()
	itemCost := uuid.New()
	fil := uuid.New()

	repo := &fakeBudgetRepo{
		pricingResult: pricing.PricingResult{
			Items: []pricing.PricingItemResult{{ItemTotalCost: 100, SaleTotal: 100}},
			Total: 100,
		},
	}
	uc := newUseCaseWith(repo)

	body := `{
		"machine_preset_id":"` + machine.String() + `",
		"energy_preset_id":"` + energy.String() + `",
		"items":[
			{"product_name":"Cube","product_quantity":1,"cost_preset_id":"` + itemCost.String() + `","filaments":[{"filament_id":"` + fil.String() + `","quantity":50,"order":1}]}
		]
	}`

	c, w := newJSONContext(http.MethodPost, "/v1/budgets/preview", body, nil)
	uc.Preview(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	want := map[uuid.UUID]string{machine: "machine", energy: "energy", itemCost: "cost"}
	if len(repo.validatedPresets) != len(want) {
		t.Fatalf("expected %d preset validations, got %d: %+v", len(want), len(repo.validatedPresets), repo.validatedPresets)
	}
	for _, v := range repo.validatedPresets {
		if want[v.id] != v.presetType {
			t.Errorf("preset %s: got type %q want %q", v.id, v.presetType, want[v.id])
		}
	}
}
