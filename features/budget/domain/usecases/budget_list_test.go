package usecases

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// errorEnvelope is the shared error body produced by core/errors.Respond.
type errorEnvelope struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// pageEnvelope mirrors helpers.Page for assertions over the list endpoints.
type pageEnvelope struct {
	Data       []json.RawMessage `json:"data"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

func decodeError(t *testing.T, body []byte) errorEnvelope {
	t.Helper()
	var e errorEnvelope
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("decode error body: %v (%s)", err, body)
	}
	return e
}

func sampleBudgets(n int) []*entities.BudgetEntity {
	budgets := make([]*entities.BudgetEntity, n)
	for i := range budgets {
		budgets[i] = &entities.BudgetEntity{
			ID:             uuid.New(),
			OrganizationID: "org-a",
			Name:           "Budget",
			CustomerID:     uuid.New(),
			Status:         entities.StatusDraft,
		}
	}
	return budgets
}

// --- GET /budgets : filters & validation ---------------------------------------

func TestFindAll_InvalidStatusFilter_Returns400(t *testing.T) {
	uc := newUseCaseWith(&fakeBudgetRepo{})
	c, w := newJSONContext(http.MethodGet, "/v1/budgets?status=bogus", ``, nil)
	uc.FindAll(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", w.Code, w.Body.String())
	}
	if code := decodeError(t, w.Body.Bytes()).Code; code != CodeInvalidStatusFilter {
		t.Fatalf("expected code %q, got %q", CodeInvalidStatusFilter, code)
	}
}

func TestFindAll_InvalidCustomerID_Returns400(t *testing.T) {
	uc := newUseCaseWith(&fakeBudgetRepo{})
	c, w := newJSONContext(http.MethodGet, "/v1/budgets?customer_id=not-a-uuid", ``, nil)
	uc.FindAll(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", w.Code, w.Body.String())
	}
	if code := decodeError(t, w.Body.Bytes()).Code; code != CodeInvalidCustomerID {
		t.Fatalf("expected code %q, got %q", CodeInvalidCustomerID, code)
	}
}

func TestFindAll_InvalidFromDate_Returns400(t *testing.T) {
	uc := newUseCaseWith(&fakeBudgetRepo{})
	c, w := newJSONContext(http.MethodGet, "/v1/budgets?from=13-13-2026", ``, nil)
	uc.FindAll(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", w.Code, w.Body.String())
	}
	if code := decodeError(t, w.Body.Bytes()).Code; code != CodeInvalidDate {
		t.Fatalf("expected code %q, got %q", CodeInvalidDate, code)
	}
}

func TestFindAll_InvalidToDate_Returns400(t *testing.T) {
	uc := newUseCaseWith(&fakeBudgetRepo{})
	c, w := newJSONContext(http.MethodGet, "/v1/budgets?to=nope", ``, nil)
	uc.FindAll(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", w.Code, w.Body.String())
	}
	if code := decodeError(t, w.Body.Bytes()).Code; code != CodeInvalidDate {
		t.Fatalf("expected code %q, got %q", CodeInvalidDate, code)
	}
}

func TestFindAll_ValidFiltersAcceptedAndPassedToRepo(t *testing.T) {
	repo := &fakeBudgetRepo{searchResult: sampleBudgets(1), searchTotal: 1}
	uc := newUseCaseWith(repo)
	customerID := uuid.New()
	url := "/v1/budgets?status=approved&customer_id=" + customerID.String() + "&from=2026-01-01&to=2026-12-31&q=cadeira&sort_by=name&sort_dir=asc"
	c, w := newJSONContext(http.MethodGet, url, ``, nil)
	uc.FindAll(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if repo.lastSearchFilters["status"] != "approved" {
		t.Errorf("status filter not forwarded: %+v", repo.lastSearchFilters)
	}
	if repo.lastSearchFilters["customer_id"] != customerID {
		t.Errorf("customer_id filter not forwarded: %+v", repo.lastSearchFilters)
	}
	if repo.lastSearchFilters["name"] != "cadeira" {
		t.Errorf("name/search filter not forwarded: %+v", repo.lastSearchFilters)
	}
	if _, ok := repo.lastSearchFilters["start_date"]; !ok {
		t.Errorf("from filter not forwarded: %+v", repo.lastSearchFilters)
	}
	if _, ok := repo.lastSearchFilters["end_date"]; !ok {
		t.Errorf("to filter not forwarded: %+v", repo.lastSearchFilters)
	}
	// name sort maps to the whitelisted LOWER(name) column, ascending.
	if repo.lastSearchOrderBy != "LOWER(name) asc" {
		t.Errorf("unexpected order clause: %q", repo.lastSearchOrderBy)
	}
}

func TestFindAll_Envelope(t *testing.T) {
	repo := &fakeBudgetRepo{searchResult: sampleBudgets(2), searchTotal: 2}
	uc := newUseCaseWith(repo)
	c, w := newJSONContext(http.MethodGet, "/v1/budgets", ``, nil)
	uc.FindAll(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	var page pageEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if len(page.Data) != 2 || page.Total != 2 || page.Page != 1 || page.PageSize != budgetListPageSize || page.TotalPages != 1 {
		t.Fatalf("unexpected envelope: %+v", page)
	}
}

func TestFindAll_DefaultSortIsCreatedAtDesc(t *testing.T) {
	repo := &fakeBudgetRepo{searchResult: sampleBudgets(1), searchTotal: 1}
	uc := newUseCaseWith(repo)
	c, w := newJSONContext(http.MethodGet, "/v1/budgets", ``, nil)
	uc.FindAll(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if repo.lastSearchOrderBy != "created_at desc" {
		t.Fatalf("expected default order 'created_at desc', got %q", repo.lastSearchOrderBy)
	}
}

// --- GET /budgets/by-customer/:customer_id -------------------------------------

func TestFindByCustomer_InvalidCustomerID_Returns400(t *testing.T) {
	uc := newUseCaseWith(&fakeBudgetRepo{})
	c, w := newJSONContext(http.MethodGet, "/v1/budgets/by-customer/not-a-uuid", ``, gin.Params{{Key: "customer_id", Value: "not-a-uuid"}})
	uc.FindByCustomer(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", w.Code, w.Body.String())
	}
	if code := decodeError(t, w.Body.Bytes()).Code; code != CodeInvalidCustomerID {
		t.Fatalf("expected code %q, got %q", CodeInvalidCustomerID, code)
	}
}

// TestFindByCustomer_NoPanic_OnResults is the regression guard for the former
// nil-pointer dereference (response, _ := buildBudgetResponse(...); *response). The
// handler must build the page without panicking and scope the query to the path
// customer_id.
func TestFindByCustomer_NoPanic_OnResults(t *testing.T) {
	repo := &fakeBudgetRepo{searchResult: sampleBudgets(3), searchTotal: 3}
	uc := newUseCaseWith(repo)
	customerID := uuid.New()
	c, w := newJSONContext(http.MethodGet, "/v1/budgets/by-customer/"+customerID.String(), ``, gin.Params{{Key: "customer_id", Value: customerID.String()}})
	uc.FindByCustomer(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	var page pageEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if len(page.Data) != 3 || page.Total != 3 {
		t.Fatalf("unexpected envelope: %+v", page)
	}
	if repo.lastSearchFilters["customer_id"] != customerID {
		t.Fatalf("path customer_id must scope the query, got %+v", repo.lastSearchFilters)
	}
}

// --- GET /budgets/:id/history --------------------------------------------------

func TestGetHistory_Envelope_DefaultPageSize(t *testing.T) {
	repo := &fakeBudgetRepo{budget: &entities.BudgetEntity{ID: uuid.New(), OrganizationID: "org-a", Status: entities.StatusDraft}}
	uc := newUseCaseWith(repo)
	id := repo.budget.ID
	c, w := newJSONContext(http.MethodGet, "/v1/budgets/"+id.String()+"/history", ``, gin.Params{{Key: "id", Value: id.String()}})
	uc.GetHistory(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	var page pageEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if page.Page != 1 || page.PageSize != budgetHistoryPageSize {
		t.Fatalf("expected default history page size %d, got %+v", budgetHistoryPageSize, page)
	}
}

// --- error codes ----------------------------------------------------------------

func TestFindByID_InvalidBudgetID_Returns400Code(t *testing.T) {
	uc := newUseCaseWith(&fakeBudgetRepo{})
	c, w := newJSONContext(http.MethodGet, "/v1/budgets/bad", ``, gin.Params{{Key: "id", Value: "bad"}})
	uc.FindByID(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if code := decodeError(t, w.Body.Bytes()).Code; code != CodeInvalidBudgetID {
		t.Fatalf("expected code %q, got %q", CodeInvalidBudgetID, code)
	}
}

func TestUpdateStatus_InvalidTransition_Returns400Code(t *testing.T) {
	id := uuid.New()
	repo := &fakeBudgetRepo{budget: &entities.BudgetEntity{ID: id, OrganizationID: "org-a", Status: entities.StatusDraft}}
	uc := newUseCaseWith(repo)
	// draft -> completed is not an allowed transition.
	c, w := newJSONContext(http.MethodPatch, "/v1/budgets/"+id.String()+"/status", `{"status":"completed"}`, gin.Params{{Key: "id", Value: id.String()}})
	uc.UpdateStatus(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", w.Code, w.Body.String())
	}
	if code := decodeError(t, w.Body.Bytes()).Code; code != CodeInvalidStatusTransition {
		t.Fatalf("expected code %q, got %q", CodeInvalidStatusTransition, code)
	}
}

func TestDelete_PrintingBudget_Returns409Code(t *testing.T) {
	id := uuid.New()
	repo := &fakeBudgetRepo{budget: &entities.BudgetEntity{ID: id, OrganizationID: "org-a", Status: entities.StatusPrinting}}
	uc := newUseCaseWith(repo)
	c, w := newJSONContext(http.MethodDelete, "/v1/budgets/"+id.String(), ``, gin.Params{{Key: "id", Value: id.String()}})
	uc.Delete(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (%s)", w.Code, w.Body.String())
	}
	if code := decodeError(t, w.Body.Bytes()).Code; code != CodeBudgetNotDeletable {
		t.Fatalf("expected code %q, got %q", CodeBudgetNotDeletable, code)
	}
}
