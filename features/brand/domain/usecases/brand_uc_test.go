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
	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"
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

// fakeBrandRepo is an in-memory BrandRepository that records the arguments the
// use case passes to FindAll so tests can assert pagination/search/sort mapping.
type fakeBrandRepo struct {
	brands []entities.BrandEntity
	total  int64

	lastSearch string
	lastOrder  string
	lastLimit  int
	lastOffset int

	existsResult bool
	findByIDErr  error
	deleteErr    error
}

func (f *fakeBrandRepo) FindAll(_ string, search, order string, limit, offset int) ([]entities.BrandEntity, int64, error) {
	f.lastSearch, f.lastOrder, f.lastLimit, f.lastOffset = search, order, limit, offset
	total := f.total
	if total == 0 {
		total = int64(len(f.brands))
	}
	return f.brands, total, nil
}

func (f *fakeBrandRepo) FindByID(id uuid.UUID, _ string) (*entities.BrandEntity, error) {
	if f.findByIDErr != nil {
		return nil, f.findByIDErr
	}
	b := entities.BrandEntity{ID: id, Name: "Existing", OrganizationID: testOrg}
	return &b, nil
}
func (f *fakeBrandRepo) Create(b *entities.BrandEntity) error {
	b.ID = uuid.New()
	return nil
}
func (f *fakeBrandRepo) Update(_ *entities.BrandEntity) error { return nil }
func (f *fakeBrandRepo) Delete(_ uuid.UUID) error             { return f.deleteErr }
func (f *fakeBrandRepo) Exists(_ string, _ string) (bool, error) {
	return f.existsResult, nil
}

func newBrandUC(repo *fakeBrandRepo) usecases.IBrandUseCase {
	return usecases.NewBrandUseCase(repo, &logpkg.NoopLogger{}, noopActivity{})
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

func TestBrandFindAll(t *testing.T) {
	validation.Register()

	seed := []entities.BrandEntity{
		{ID: uuid.New(), Name: "Alpha", OrganizationID: testOrg},
		{ID: uuid.New(), Name: "Beta", OrganizationID: testOrg},
	}

	t.Run("envelope shape", func(t *testing.T) {
		repo := &fakeBrandRepo{brands: seed, total: 42}
		w, c := testCtx(http.MethodGet, "/brands?page=2&page_size=10", "")
		newBrandUC(repo).FindAll(c)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var env struct {
			Data       []entities.BrandEntity `json:"data"`
			Total      int64                  `json:"total"`
			Page       int                    `json:"page"`
			PageSize   int                    `json:"page_size"`
			TotalPages int                    `json:"total_pages"`
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

	t.Run("page_size clamp", func(t *testing.T) {
		repo := &fakeBrandRepo{brands: seed}
		_, c := testCtx(http.MethodGet, "/brands?page_size=500", "")
		newBrandUC(repo).FindAll(c)
		if repo.lastLimit != 100 {
			t.Errorf("clamped limit = %d, want 100", repo.lastLimit)
		}
	})

	t.Run("default page size", func(t *testing.T) {
		repo := &fakeBrandRepo{brands: seed}
		_, c := testCtx(http.MethodGet, "/brands", "")
		newBrandUC(repo).FindAll(c)
		if repo.lastLimit != 20 {
			t.Errorf("default limit = %d, want 20", repo.lastLimit)
		}
	})

	t.Run("q filter forwarded", func(t *testing.T) {
		repo := &fakeBrandRepo{brands: seed}
		_, c := testCtx(http.MethodGet, "/brands?q=alph", "")
		newBrandUC(repo).FindAll(c)
		if repo.lastSearch != "alph" {
			t.Errorf("search = %q, want %q", repo.lastSearch, "alph")
		}
	})

	t.Run("sort whitelist", func(t *testing.T) {
		cases := []struct {
			query     string
			wantOrder string
		}{
			// The "id" tie-breaker is appended for deterministic pagination.
			{"/brands", "lower(name) asc, id asc"},
			{"/brands?sort_by=created_at&sort_dir=desc", "created_at desc, id asc"},
			{"/brands?sort_by=created_at&sort_dir=asc", "created_at asc, id asc"},
			{"/brands?sort_by=unknown_col&sort_dir=desc", "lower(name) desc, id asc"}, // falls back to default column
			{"/brands?sort_by=name", "lower(name) asc, id asc"},
		}
		for _, tc := range cases {
			repo := &fakeBrandRepo{brands: seed}
			_, c := testCtx(http.MethodGet, tc.query, "")
			newBrandUC(repo).FindAll(c)
			if repo.lastOrder != tc.wantOrder {
				t.Errorf("query %q: order = %q, want %q", tc.query, repo.lastOrder, tc.wantOrder)
			}
		}
	})
}

func TestBrandErrorMapping(t *testing.T) {
	validation.Register()

	decode := func(w *httptest.ResponseRecorder) map[string]any {
		var m map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &m)
		return m
	}

	t.Run("validation error -> 400 with fields", func(t *testing.T) {
		repo := &fakeBrandRepo{}
		w, c := testCtx(http.MethodPost, "/brands", `{"name":""}`)
		newBrandUC(repo).Create(c)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		body := decode(w)
		if body["code"] != "validation_error" {
			t.Errorf("code = %v, want validation_error", body["code"])
		}
		fields, ok := body["fields"].(map[string]any)
		if !ok || fields["name"] == nil {
			t.Errorf("expected fields.name to be present, got %v", body["fields"])
		}
	})

	t.Run("conflict -> 409 brand_name_taken", func(t *testing.T) {
		repo := &fakeBrandRepo{existsResult: true}
		w, c := testCtx(http.MethodPost, "/brands", `{"name":"Dup"}`)
		newBrandUC(repo).Create(c)
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", w.Code)
		}
		if decode(w)["code"] != "brand_name_taken" {
			t.Errorf("code = %v, want brand_name_taken", decode(w)["code"])
		}
	})

	t.Run("invalid id -> 400 invalid_brand_id", func(t *testing.T) {
		repo := &fakeBrandRepo{}
		w, c := testCtx(http.MethodGet, "/brands/not-a-uuid", "")
		c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
		newBrandUC(repo).FindByID(c)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		if decode(w)["code"] != "invalid_brand_id" {
			t.Errorf("code = %v, want invalid_brand_id", decode(w)["code"])
		}
	})

	t.Run("not found -> 404 brand_not_found", func(t *testing.T) {
		repo := &fakeBrandRepo{findByIDErr: gorm.ErrRecordNotFound}
		id := uuid.New()
		w, c := testCtx(http.MethodGet, "/brands/"+id.String(), "")
		c.Params = gin.Params{{Key: "id", Value: id.String()}}
		newBrandUC(repo).FindByID(c)
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
		if decode(w)["code"] != "brand_not_found" {
			t.Errorf("code = %v, want brand_not_found", decode(w)["code"])
		}
	})

	t.Run("delete fk violation -> 409 brand_in_use", func(t *testing.T) {
		repo := &fakeBrandRepo{deleteErr: errFakeFK}
		id := uuid.New()
		w, c := testCtx(http.MethodDelete, "/brands/"+id.String(), "")
		c.Params = gin.Params{{Key: "id", Value: id.String()}}
		newBrandUC(repo).Delete(c)
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", w.Code)
		}
		if decode(w)["code"] != "brand_in_use" {
			t.Errorf("code = %v, want brand_in_use", decode(w)["code"])
		}
	})
}

// errFakeFK is a real *pgconn.PgError carrying SQLSTATE 23503, so the shared
// database.IsForeignKeyViolation detection (errors.As on *pgconn.PgError) is
// exercised exactly as it would be with the live driver.
var errFakeFK error = &pgconn.PgError{
	Code:    "23503",
	Message: `update or delete on table "brands" violates foreign key constraint "fk_filaments_brand"`,
}
