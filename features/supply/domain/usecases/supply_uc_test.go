package usecases_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	logpkg "github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/supply/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/supply/domain/usecases"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const testOrg = "org-a"

// noopActivity satisfies activityUc.IActivityService without side effects.
type noopActivity struct{}

func (noopActivity) Record(_ context.Context, _ activityEntities.ActivityEntity) {}
func (noopActivity) ListActivities(_ *gin.Context)                               {}
func (noopActivity) FindRecentByOrganization(_ string, _ int) ([]activityEntities.ActivityEntity, error) {
	return nil, nil
}

// fakeSupplyRepo is an in-memory SupplyRepository recording the arguments the use
// case passes to FindAll so tests can assert pagination/search/sort/is_active mapping.
type fakeSupplyRepo struct {
	supplies []entities.SupplyEntity
	total    int64

	lastSearch   string
	lastIsActive *bool
	lastOrder    string
	lastLimit    int
	lastOffset   int

	existsResult bool
	findByIDErr  error
	deleteErr    error
}

func (f *fakeSupplyRepo) FindAll(_ string, search string, isActive *bool, order string, limit, offset int) ([]entities.SupplyEntity, int64, error) {
	f.lastSearch, f.lastIsActive, f.lastOrder, f.lastLimit, f.lastOffset = search, isActive, order, limit, offset
	total := f.total
	if total == 0 {
		total = int64(len(f.supplies))
	}
	return f.supplies, total, nil
}

func (f *fakeSupplyRepo) FindByID(id uuid.UUID, _ string) (*entities.SupplyEntity, error) {
	if f.findByIDErr != nil {
		return nil, f.findByIDErr
	}
	s := entities.SupplyEntity{ID: id, Name: "Existing", Unit: "un", OrganizationID: testOrg, IsActive: true}
	return &s, nil
}
func (f *fakeSupplyRepo) Create(s *entities.SupplyEntity) error {
	s.ID = uuid.New()
	return nil
}
func (f *fakeSupplyRepo) Update(_ *entities.SupplyEntity) error { return nil }
func (f *fakeSupplyRepo) Delete(_ uuid.UUID) error              { return f.deleteErr }
func (f *fakeSupplyRepo) Exists(_ string, _ string) (bool, error) {
	return f.existsResult, nil
}

func newSupplyUC(repo *fakeSupplyRepo) usecases.ISupplyUseCase {
	return usecases.NewSupplyUseCase(repo, &logpkg.NoopLogger{}, noopActivity{})
}

func testCtx(method, target, body string) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	c.Request = r
	c.Set("organization_id", testOrg)
	c.Set("user_id", "user-1")
	return w, c
}

func TestSupplyFindAll(t *testing.T) {
	validation.Register()

	seed := []entities.SupplyEntity{
		{ID: uuid.New(), Name: "LED", Unit: "un", OrganizationID: testOrg, IsActive: true},
		{ID: uuid.New(), Name: "Creme", Unit: "ml", OrganizationID: testOrg, IsActive: true},
	}

	t.Run("envelope shape", func(t *testing.T) {
		repo := &fakeSupplyRepo{supplies: seed, total: 42}
		w, c := testCtx(http.MethodGet, "/supplies?page=2&page_size=10", "")
		newSupplyUC(repo).FindAll(c)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var env struct {
			Data       []entities.SupplyEntity `json:"data"`
			Total      int64                   `json:"total"`
			Page       int                     `json:"page"`
			PageSize   int                     `json:"page_size"`
			TotalPages int                     `json:"total_pages"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if env.Total != 42 || env.Page != 2 || env.PageSize != 10 || env.TotalPages != 5 {
			t.Errorf("envelope = %+v, want total=42 page=2 page_size=10 total_pages=5", env)
		}
		if len(env.Data) != 2 {
			t.Errorf("len(data) = %d, want 2", len(env.Data))
		}
		if repo.lastOffset != 10 || repo.lastLimit != 10 {
			t.Errorf("limit/offset = %d/%d, want 10/10", repo.lastLimit, repo.lastOffset)
		}
	})

	t.Run("is_active filter forwarded", func(t *testing.T) {
		repo := &fakeSupplyRepo{supplies: seed}
		_, c := testCtx(http.MethodGet, "/supplies?is_active=true", "")
		newSupplyUC(repo).FindAll(c)
		if repo.lastIsActive == nil || *repo.lastIsActive != true {
			t.Errorf("is_active = %v, want true", repo.lastIsActive)
		}

		repo = &fakeSupplyRepo{supplies: seed}
		_, c = testCtx(http.MethodGet, "/supplies?is_active=false", "")
		newSupplyUC(repo).FindAll(c)
		if repo.lastIsActive == nil || *repo.lastIsActive != false {
			t.Errorf("is_active = %v, want false", repo.lastIsActive)
		}

		repo = &fakeSupplyRepo{supplies: seed}
		_, c = testCtx(http.MethodGet, "/supplies", "")
		newSupplyUC(repo).FindAll(c)
		if repo.lastIsActive != nil {
			t.Errorf("absent is_active must stay nil, got %v", *repo.lastIsActive)
		}
	})

	t.Run("invalid is_active -> 400", func(t *testing.T) {
		repo := &fakeSupplyRepo{supplies: seed}
		w, c := testCtx(http.MethodGet, "/supplies?is_active=maybe", "")
		newSupplyUC(repo).FindAll(c)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("q filter forwarded", func(t *testing.T) {
		repo := &fakeSupplyRepo{supplies: seed}
		_, c := testCtx(http.MethodGet, "/supplies?q=led", "")
		newSupplyUC(repo).FindAll(c)
		if repo.lastSearch != "led" {
			t.Errorf("search = %q, want %q", repo.lastSearch, "led")
		}
	})

	t.Run("sort whitelist", func(t *testing.T) {
		cases := []struct {
			query     string
			wantOrder string
		}{
			{"/supplies", "lower(name) asc, id asc"},
			{"/supplies?sort_by=unit_cost&sort_dir=desc", "unit_cost desc, id asc"},
			{"/supplies?sort_by=created_at&sort_dir=asc", "created_at asc, id asc"},
			{"/supplies?sort_by=unknown&sort_dir=desc", "lower(name) desc, id asc"},
		}
		for _, tc := range cases {
			repo := &fakeSupplyRepo{supplies: seed}
			_, c := testCtx(http.MethodGet, tc.query, "")
			newSupplyUC(repo).FindAll(c)
			if repo.lastOrder != tc.wantOrder {
				t.Errorf("query %q: order = %q, want %q", tc.query, repo.lastOrder, tc.wantOrder)
			}
		}
	})
}

func TestSupplyCreateValidationAndErrors(t *testing.T) {
	validation.Register()

	decode := func(w *httptest.ResponseRecorder) map[string]any {
		var m map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &m)
		return m
	}

	t.Run("missing name -> 400 validation_error", func(t *testing.T) {
		repo := &fakeSupplyRepo{}
		w, c := testCtx(http.MethodPost, "/supplies", `{"name":"","unit":"un","unit_cost":100}`)
		newSupplyUC(repo).Create(c)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		if decode(w)["code"] != "validation_error" {
			t.Errorf("code = %v, want validation_error", decode(w)["code"])
		}
	})

	t.Run("invalid unit -> 400", func(t *testing.T) {
		repo := &fakeSupplyRepo{}
		w, c := testCtx(http.MethodPost, "/supplies", `{"name":"LED","unit":"foo","unit_cost":100}`)
		newSupplyUC(repo).Create(c)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 for invalid unit", w.Code)
		}
	})

	t.Run("negative unit_cost -> 400", func(t *testing.T) {
		repo := &fakeSupplyRepo{}
		w, c := testCtx(http.MethodPost, "/supplies", `{"name":"LED","unit":"un","unit_cost":-5}`)
		newSupplyUC(repo).Create(c)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 for negative unit_cost", w.Code)
		}
	})

	t.Run("valid create -> 201", func(t *testing.T) {
		repo := &fakeSupplyRepo{}
		w, c := testCtx(http.MethodPost, "/supplies", `{"name":"LED","unit":"un","unit_cost":150}`)
		newSupplyUC(repo).Create(c)
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", w.Code)
		}
		var got entities.SupplyEntity
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Name != "LED" || got.Unit != "un" || got.UnitCost != 150 {
			t.Errorf("created = %+v", got)
		}
		if !got.IsActive {
			t.Errorf("is_active must default to true when omitted")
		}
	})

	t.Run("conflict -> 409 supply_name_taken", func(t *testing.T) {
		repo := &fakeSupplyRepo{existsResult: true}
		w, c := testCtx(http.MethodPost, "/supplies", `{"name":"Dup","unit":"un","unit_cost":100}`)
		newSupplyUC(repo).Create(c)
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", w.Code)
		}
		if decode(w)["code"] != "supply_name_taken" {
			t.Errorf("code = %v, want supply_name_taken", decode(w)["code"])
		}
	})
}

func TestSupplyIDErrors(t *testing.T) {
	validation.Register()

	decode := func(w *httptest.ResponseRecorder) map[string]any {
		var m map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &m)
		return m
	}

	t.Run("invalid id -> 400 invalid_supply_id", func(t *testing.T) {
		repo := &fakeSupplyRepo{}
		w, c := testCtx(http.MethodGet, "/supplies/not-a-uuid", "")
		c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
		newSupplyUC(repo).FindByID(c)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		if decode(w)["code"] != "invalid_supply_id" {
			t.Errorf("code = %v, want invalid_supply_id", decode(w)["code"])
		}
	})

	t.Run("not found -> 404 supply_not_found", func(t *testing.T) {
		repo := &fakeSupplyRepo{findByIDErr: gorm.ErrRecordNotFound}
		id := uuid.New()
		w, c := testCtx(http.MethodGet, "/supplies/"+id.String(), "")
		c.Params = gin.Params{{Key: "id", Value: id.String()}}
		newSupplyUC(repo).FindByID(c)
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
		if decode(w)["code"] != "supply_not_found" {
			t.Errorf("code = %v, want supply_not_found", decode(w)["code"])
		}
	})

	t.Run("delete fk violation -> 409 supply_in_use", func(t *testing.T) {
		repo := &fakeSupplyRepo{deleteErr: errFakeFK}
		id := uuid.New()
		w, c := testCtx(http.MethodDelete, "/supplies/"+id.String(), "")
		c.Params = gin.Params{{Key: "id", Value: id.String()}}
		newSupplyUC(repo).Delete(c)
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", w.Code)
		}
		if decode(w)["code"] != "supply_in_use" {
			t.Errorf("code = %v, want supply_in_use", decode(w)["code"])
		}
	})
}

// errFakeFK is a real *pgconn.PgError carrying SQLSTATE 23503 so the shared
// database.IsForeignKeyViolation detection is exercised as with the live driver.
var errFakeFK error = &pgconn.PgError{
	Code:    "23503",
	Message: `update or delete on table "supplies" violates foreign key constraint "fk_budget_item_supplies_supply"`,
}
