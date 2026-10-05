package usecases

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	pricing "github.com/RodolfoBonis/spooliq/features/budget/domain/services"
	customerEntities "github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	customerRepo "github.com/RodolfoBonis/spooliq/features/customer/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

type noopActivity struct{}

func (noopActivity) Record(_ context.Context, _ activityEntities.ActivityEntity) {}
func (noopActivity) ListActivities(_ *gin.Context)                               {}
func (noopActivity) FindRecentByOrganization(_ string, _ int) ([]activityEntities.ActivityEntity, error) {
	return nil, nil
}

type fakeCustomerRepo struct{}

var _ customerRepo.CustomerRepository = fakeCustomerRepo{}

func (fakeCustomerRepo) Create(_ context.Context, _ *customerEntities.CustomerEntity) error {
	return nil
}
func (fakeCustomerRepo) FindByID(_ context.Context, id uuid.UUID, org string) (*customerEntities.CustomerEntity, error) {
	return &customerEntities.CustomerEntity{ID: id, OrganizationID: org, Name: "Test"}, nil
}
func (fakeCustomerRepo) Update(_ context.Context, _ *customerEntities.CustomerEntity) error {
	return nil
}
func (fakeCustomerRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (fakeCustomerRepo) FindAll(_ context.Context, _ string, _, _ int) ([]*customerEntities.CustomerEntity, int, error) {
	return nil, 0, nil
}
func (fakeCustomerRepo) SearchCustomers(_ context.Context, _ string, _ map[string]interface{}, _, _ int) ([]*customerEntities.CustomerEntity, int, error) {
	return nil, 0, nil
}
func (fakeCustomerRepo) ExistsByEmail(_ context.Context, _ string, _ string, _ *uuid.UUID) (bool, error) {
	return false, nil
}
func (fakeCustomerRepo) CountBudgetsByCustomer(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (fakeCustomerRepo) GetCustomerBudgets(_ context.Context, _ uuid.UUID) ([]customerEntities.BudgetSummary, error) {
	return nil, nil
}
func (fakeCustomerRepo) SumBudgetTotalsByCustomerAndStatus(_ context.Context, _ uuid.UUID, _ []string) (int64, error) {
	return 0, nil
}

// presetValidation records a (id,type) pair passed to ValidatePresetInOrg.
type presetValidation struct {
	id         uuid.UUID
	presetType string
}

// fakeBudgetRepo is a configurable in-memory stand-in for budgetRepo.BudgetRepository.
type fakeBudgetRepo struct {
	budget *entities.BudgetEntity

	updateErr error // returned by Update (inside the transaction)

	validatedPresets []presetValidation

	// Preview support: canned pricing result and persistence-call counters so
	// tests can assert a preview never writes anything.
	pricingResult pricing.PricingResult
	createCalls   int
	addItemCalls  int
	calcCalls     int
}

var _ budgetRepo.BudgetRepository = (*fakeBudgetRepo)(nil)

func (f *fakeBudgetRepo) WithTransaction(ctx context.Context, fn func(repo budgetRepo.BudgetRepository) error) error {
	return fn(f)
}
func (f *fakeBudgetRepo) Create(_ context.Context, _ *entities.BudgetEntity) error {
	f.createCalls++
	return nil
}
func (f *fakeBudgetRepo) FindByID(_ context.Context, id uuid.UUID, org string) (*entities.BudgetEntity, error) {
	if f.budget != nil {
		return f.budget, nil
	}
	return &entities.BudgetEntity{ID: id, OrganizationID: org, Status: entities.StatusDraft}, nil
}
func (f *fakeBudgetRepo) Update(_ context.Context, _ *entities.BudgetEntity) error {
	return f.updateErr
}
func (f *fakeBudgetRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ string, _, _ entities.BudgetStatus) error {
	return nil
}
func (f *fakeBudgetRepo) UpdatePDFURL(_ context.Context, _ uuid.UUID, _ string, _ *string) error {
	return nil
}
func (f *fakeBudgetRepo) Delete(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (f *fakeBudgetRepo) FindAll(_ context.Context, _ string, _, _ int) ([]*entities.BudgetEntity, int, error) {
	return nil, 0, nil
}
func (f *fakeBudgetRepo) FindByCustomer(_ context.Context, _ uuid.UUID, _ string) ([]*entities.BudgetEntity, error) {
	return nil, nil
}
func (f *fakeBudgetRepo) SearchBudgets(_ context.Context, _ string, _ map[string]interface{}, _, _ int) ([]*entities.BudgetEntity, int, error) {
	return nil, 0, nil
}
func (f *fakeBudgetRepo) AddItem(_ context.Context, _ *entities.BudgetItemEntity) error {
	f.addItemCalls++
	return nil
}
func (f *fakeBudgetRepo) RemoveItem(_ context.Context, _ uuid.UUID) error { return nil }
func (f *fakeBudgetRepo) UpdateItem(_ context.Context, _ *entities.BudgetItemEntity) error {
	return nil
}
func (f *fakeBudgetRepo) GetItems(_ context.Context, _ uuid.UUID) ([]*entities.BudgetItemEntity, error) {
	return nil, nil
}
func (f *fakeBudgetRepo) DeleteAllItems(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (f *fakeBudgetRepo) AddItemFilament(_ context.Context, _ *entities.BudgetItemFilamentEntity) error {
	return nil
}
func (f *fakeBudgetRepo) RemoveItemFilament(_ context.Context, _ uuid.UUID) error { return nil }
func (f *fakeBudgetRepo) GetItemFilaments(_ context.Context, _ uuid.UUID) ([]*entities.BudgetItemFilamentEntity, error) {
	return nil, nil
}
func (f *fakeBudgetRepo) DeleteAllItemFilaments(_ context.Context, _ uuid.UUID) error { return nil }
func (f *fakeBudgetRepo) GetFilamentUsageInfo(_ context.Context, _ uuid.UUID, _ string) ([]entities.FilamentUsageInfo, error) {
	return nil, nil
}
func (f *fakeBudgetRepo) AddStatusHistory(_ context.Context, _ *entities.BudgetStatusHistoryEntity) error {
	return nil
}
func (f *fakeBudgetRepo) GetStatusHistory(_ context.Context, _ uuid.UUID) ([]entities.BudgetStatusHistoryEntity, error) {
	return nil, nil
}
func (f *fakeBudgetRepo) CalculateCosts(_ context.Context, _ uuid.UUID, _ string) error {
	f.calcCalls++
	return nil
}
func (f *fakeBudgetRepo) ComputeBudgetPricing(_ context.Context, _ entities.PricingComputationInput) (pricing.PricingResult, error) {
	return f.pricingResult, nil
}
func (f *fakeBudgetRepo) ValidateFilamentsInOrg(_ context.Context, _ []uuid.UUID, _ string) error {
	return nil
}
func (f *fakeBudgetRepo) ValidatePresetInOrg(_ context.Context, presetID uuid.UUID, presetType string, _ string) error {
	f.validatedPresets = append(f.validatedPresets, presetValidation{id: presetID, presetType: presetType})
	return nil
}
func (f *fakeBudgetRepo) GetCustomerInfo(_ context.Context, id uuid.UUID, _ string) (*entities.CustomerInfo, error) {
	return &entities.CustomerInfo{ID: id.String(), Name: "Test"}, nil
}
func (f *fakeBudgetRepo) GetFilamentInfo(_ context.Context, _ uuid.UUID, _ string) (*entities.FilamentInfo, error) {
	return &entities.FilamentInfo{}, nil
}
func (f *fakeBudgetRepo) GetPresetInfo(_ context.Context, id uuid.UUID, presetType string, _ string) (*entities.PresetInfo, error) {
	return &entities.PresetInfo{ID: id.String(), Type: presetType}, nil
}
func (f *fakeBudgetRepo) GetCompanyByOrganizationID(_ context.Context, _ string) (*entities.CompanyInfo, error) {
	return &entities.CompanyInfo{}, nil
}
func (f *fakeBudgetRepo) FindItemsByBudgetID(_ context.Context, _ uuid.UUID) ([]*entities.BudgetItemEntity, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newUseCaseWith(repo budgetRepo.BudgetRepository) *BudgetUseCase {
	return &BudgetUseCase{
		budgetRepository:   repo,
		customerRepository: fakeCustomerRepo{},
		validator:          validator.New(),
		logger:             logger.NewLogger("test"),
		activityService:    noopActivity{},
	}
}

func newJSONContext(method, path, body string, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = params
	c.Set("organization_id", "org-a")
	c.Set("user_id", "user-1")
	return c, w
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestUpdate_ReturnsConflictWhenRepoReportsNotEditable proves the status-guard
// path: even though the pre-check sees a draft, a concurrent approval causes the
// transactional repo.Update to return ErrBudgetNotEditable, which the use case
// must translate into HTTP 409 (not a 500).
func TestUpdate_ReturnsConflictWhenRepoReportsNotEditable(t *testing.T) {
	id := uuid.New()
	repo := &fakeBudgetRepo{
		budget:    &entities.BudgetEntity{ID: id, OrganizationID: "org-a", Status: entities.StatusDraft},
		updateErr: entities.ErrBudgetNotEditable,
	}
	uc := newUseCaseWith(repo)

	c, w := newJSONContext(http.MethodPut, "/v1/budgets/"+id.String(), `{}`, gin.Params{{Key: "id", Value: id.String()}})
	uc.Update(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestUpdate_SucceedsOnDraft confirms the happy path still returns 200 when the
// repo accepts the write.
func TestUpdate_SucceedsOnDraft(t *testing.T) {
	id := uuid.New()
	repo := &fakeBudgetRepo{
		budget:    &entities.BudgetEntity{ID: id, OrganizationID: "org-a", Status: entities.StatusDraft},
		updateErr: nil,
	}
	uc := newUseCaseWith(repo)

	c, w := newJSONContext(http.MethodPut, "/v1/budgets/"+id.String(), `{"name":"Renamed"}`, gin.Params{{Key: "id", Value: id.String()}})
	uc.Update(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestUpdate_RejectsNonDraftUpfront confirms the fast pre-check returns 409 for an
// already-approved budget before any write is attempted.
func TestUpdate_RejectsNonDraftUpfront(t *testing.T) {
	id := uuid.New()
	repo := &fakeBudgetRepo{
		budget: &entities.BudgetEntity{ID: id, OrganizationID: "org-a", Status: entities.StatusApproved},
	}
	uc := newUseCaseWith(repo)

	c, w := newJSONContext(http.MethodPut, "/v1/budgets/"+id.String(), `{}`, gin.Params{{Key: "id", Value: id.String()}})
	uc.Update(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestRecalculate_ReturnsConflictWhenRepoReportsNotEditable proves recalculation
// surfaces a concurrent approval as 409.
func TestRecalculate_ReturnsConflictWhenRepoReportsNotEditable(t *testing.T) {
	id := uuid.New()
	repo := &fakeBudgetRepo{
		budget:    &entities.BudgetEntity{ID: id, OrganizationID: "org-a", Status: entities.StatusDraft},
		updateErr: entities.ErrBudgetNotEditable,
	}
	uc := newUseCaseWith(repo)

	c, w := newJSONContext(http.MethodPost, "/v1/budgets/"+id.String()+"/recalculate", ``, gin.Params{{Key: "id", Value: id.String()}})
	uc.Recalculate(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestUpdate_ValidatesOnlyRequestPresetsWithCorrectType proves two things:
//   - a preset provided in the request is validated with the correct slot type
//     (machine), and
//   - presets already stored on the budget are NOT re-validated (so a budget that
//     references a since-soft-deleted preset can still be renamed/edited).
func TestUpdate_ValidatesOnlyRequestPresetsWithCorrectType(t *testing.T) {
	id := uuid.New()
	storedMachine := uuid.New()
	storedEnergy := uuid.New()
	requestMachine := uuid.New()

	repo := &fakeBudgetRepo{
		budget: &entities.BudgetEntity{
			ID:              id,
			OrganizationID:  "org-a",
			Status:          entities.StatusDraft,
			MachinePresetID: &storedMachine,
			EnergyPresetID:  &storedEnergy,
		},
	}
	uc := newUseCaseWith(repo)

	body := `{"machine_preset_id":"` + requestMachine.String() + `"}`
	c, w := newJSONContext(http.MethodPut, "/v1/budgets/"+id.String(), body, gin.Params{{Key: "id", Value: id.String()}})
	uc.Update(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	if len(repo.validatedPresets) != 1 {
		t.Fatalf("expected exactly 1 preset validation, got %d: %+v", len(repo.validatedPresets), repo.validatedPresets)
	}
	got := repo.validatedPresets[0]
	if got.id != requestMachine {
		t.Errorf("validated the wrong preset id: got %s want %s", got.id, requestMachine)
	}
	if got.presetType != "machine" {
		t.Errorf("validated with wrong type: got %q want %q", got.presetType, "machine")
	}
	for _, v := range repo.validatedPresets {
		if v.id == storedMachine || v.id == storedEnergy {
			t.Errorf("stored preset %s should NOT be re-validated", v.id)
		}
	}
}

// TestValidateReferences_UsesCorrectTypePerSlot checks the per-slot preset typing
// directly: machine/energy/cost budget-level presets plus item cost presets.
func TestValidateReferences_UsesCorrectTypePerSlot(t *testing.T) {
	machine := uuid.New()
	energy := uuid.New()
	cost := uuid.New()
	itemCost := uuid.New()
	fil := uuid.New()

	repo := &fakeBudgetRepo{}
	uc := newUseCaseWith(repo)

	items := []entities.BudgetItemRequest{
		{
			ProductName:     "p",
			ProductQuantity: 1,
			CostPresetID:    &itemCost,
			Filaments:       []entities.BudgetItemFilamentRequest{{FilamentID: fil, Quantity: 10, Order: 1}},
		},
	}

	if err := uc.validateReferences(context.Background(), "org-a", &machine, &energy, &cost, items); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[uuid.UUID]string{
		machine:  "machine",
		energy:   "energy",
		cost:     "cost",
		itemCost: "cost",
	}
	if len(repo.validatedPresets) != len(want) {
		t.Fatalf("expected %d validations, got %d: %+v", len(want), len(repo.validatedPresets), repo.validatedPresets)
	}
	for _, v := range repo.validatedPresets {
		wantType, ok := want[v.id]
		if !ok {
			t.Errorf("unexpected preset validated: %s", v.id)
			continue
		}
		if v.presetType != wantType {
			t.Errorf("preset %s: got type %q want %q", v.id, v.presetType, wantType)
		}
	}
}
