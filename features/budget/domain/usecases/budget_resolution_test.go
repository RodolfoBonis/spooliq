package usecases

import (
	"net/http"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	pricing "github.com/RodolfoBonis/spooliq/features/budget/domain/services"
	"github.com/google/uuid"
)

// newUseCaseWithProviders builds a BudgetUseCase with the given repo + resolver
// providers so resolution-aware flows (create/preview) can be exercised.
func newUseCaseWithProviders(repo *fakeBudgetRepo, profiles ProfilePresetProvider, presets DefaultPresetProvider) *BudgetUseCase {
	return &BudgetUseCase{
		budgetRepository:   repo,
		customerRepository: fakeCustomerRepo{},
		logger:             logger.NewLogger("test"),
		activityService:    noopActivity{},
		profileProvider:    profiles,
		presetProvider:     presets,
	}
}

// TestCreate_ResolvesOrgDefaultsWhenNoPresetsProvided proves a budget created with
// no profile and no explicit presets stores the org's default presets.
func TestCreate_ResolvesOrgDefaultsWhenNoPresetsProvided(t *testing.T) {
	orgM, orgE, orgC := uuid.New(), uuid.New(), uuid.New()
	fil := uuid.New()
	customer := uuid.New()

	repo := &fakeBudgetRepo{
		pricingResult: pricing.PricingResult{
			Items: []pricing.PricingItemResult{{ItemTotalCost: 100, SaleTotal: 100}},
			Total: 100,
		},
	}
	profiles := &fakeProfileProvider{} // no default profile
	presets := &fakeDefaultPresetProvider{byType: map[string]*uuid.UUID{"machine": &orgM, "energy": &orgE, "cost": &orgC}}
	uc := newUseCaseWithProviders(repo, profiles, presets)

	body := `{
		"name":"B","customer_id":"` + customer.String() + `",
		"items":[{"product_name":"Cube","product_quantity":1,"filaments":[{"filament_id":"` + fil.String() + `","quantity":100,"order":1}]}]
	}`
	c, w := newJSONContext(http.MethodPost, "/v1/budgets", body, nil)
	uc.Create(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", w.Code, w.Body.String())
	}
	if repo.lastCreated == nil {
		t.Fatal("budget was not created")
	}
	if !eqPtr(repo.lastCreated.MachinePresetID, &orgM) {
		t.Errorf("machine = %v, want %v", repo.lastCreated.MachinePresetID, orgM)
	}
	if !eqPtr(repo.lastCreated.EnergyPresetID, &orgE) {
		t.Errorf("energy = %v, want %v", repo.lastCreated.EnergyPresetID, orgE)
	}
	if !eqPtr(repo.lastCreated.CostPresetID, &orgC) {
		t.Errorf("cost = %v, want %v", repo.lastCreated.CostPresetID, orgC)
	}
	if repo.lastCreated.ProfileID != nil {
		t.Errorf("profile = %v, want nil", repo.lastCreated.ProfileID)
	}
}

// TestCreate_ResolvesFromRequestedProfile proves profile_id drives the stored
// presets (and the stored profile_id) when no explicit preset overrides it.
func TestCreate_ResolvesFromRequestedProfile(t *testing.T) {
	profM, profE, profC := uuid.New(), uuid.New(), uuid.New()
	profileID := uuid.New()
	fil := uuid.New()
	customer := uuid.New()

	repo := &fakeBudgetRepo{
		pricingResult: pricing.PricingResult{
			Items: []pricing.PricingItemResult{{ItemTotalCost: 100, SaleTotal: 100}},
			Total: 100,
		},
	}
	profiles := &fakeProfileProvider{byID: map[uuid.UUID]*ProfilePresets{
		profileID: {Name: "Prof", MachinePresetID: &profM, EnergyPresetID: &profE, CostPresetID: &profC},
	}}
	presets := &fakeDefaultPresetProvider{}
	uc := newUseCaseWithProviders(repo, profiles, presets)

	body := `{
		"name":"B","customer_id":"` + customer.String() + `","profile_id":"` + profileID.String() + `",
		"items":[{"product_name":"Cube","product_quantity":1,"filaments":[{"filament_id":"` + fil.String() + `","quantity":100,"order":1}]}]
	}`
	c, w := newJSONContext(http.MethodPost, "/v1/budgets", body, nil)
	uc.Create(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", w.Code, w.Body.String())
	}
	if !eqPtr(repo.lastCreated.MachinePresetID, &profM) || !eqPtr(repo.lastCreated.EnergyPresetID, &profE) || !eqPtr(repo.lastCreated.CostPresetID, &profC) {
		t.Errorf("stored presets did not come from the profile: %+v", repo.lastCreated)
	}
	if !eqPtr(repo.lastCreated.ProfileID, &profileID) {
		t.Errorf("profile_id = %v, want %v", repo.lastCreated.ProfileID, profileID)
	}
}

// TestCreate_RejectsUnknownProfile proves a profile_id outside the org yields 400
// and nothing is persisted.
func TestCreate_RejectsUnknownProfile(t *testing.T) {
	profileID := uuid.New()
	fil := uuid.New()
	customer := uuid.New()

	repo := &fakeBudgetRepo{}
	profiles := &fakeProfileProvider{missing: map[uuid.UUID]bool{profileID: true}}
	presets := &fakeDefaultPresetProvider{}
	uc := newUseCaseWithProviders(repo, profiles, presets)

	body := `{
		"name":"B","customer_id":"` + customer.String() + `","profile_id":"` + profileID.String() + `",
		"items":[{"product_name":"Cube","product_quantity":1,"filaments":[{"filament_id":"` + fil.String() + `","quantity":100,"order":1}]}]
	}`
	c, w := newJSONContext(http.MethodPost, "/v1/budgets", body, nil)
	uc.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", w.Code, w.Body.String())
	}
	if repo.createCalls != 0 {
		t.Errorf("nothing should be persisted on a bad profile, create=%d", repo.createCalls)
	}
}

// TestPreview_ResolvedCostPresetDrivesOverheadProfit proves the resolved budget-level
// cost preset (here from the org default) is passed to the pricing engine as the
// overhead/profit source, even when the request sends no presets.
func TestPreview_ResolvedCostPresetDrivesOverheadProfit(t *testing.T) {
	orgC := uuid.New()
	fil := uuid.New()

	repo := &fakeBudgetRepo{
		pricingResult: pricing.PricingResult{
			Items: []pricing.PricingItemResult{{ItemTotalCost: 100, SaleTotal: 115}},
			Total: 115,
		},
	}
	profiles := &fakeProfileProvider{}
	presets := &fakeDefaultPresetProvider{byType: map[string]*uuid.UUID{"cost": &orgC}}
	uc := newUseCaseWithProviders(repo, profiles, presets)

	body := `{
		"name":"P",
		"items":[{"product_name":"Cube","product_quantity":1,"filaments":[{"filament_id":"` + fil.String() + `","quantity":100,"order":1}]}]
	}`
	c, w := newJSONContext(http.MethodPost, "/v1/budgets/preview", body, nil)
	uc.Preview(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if !eqPtr(repo.lastPricingInput.BudgetCostPresetID, &orgC) {
		t.Errorf("BudgetCostPresetID = %v, want %v (org default cost preset must drive overhead/profit)", repo.lastPricingInput.BudgetCostPresetID, orgC)
	}
}
