package usecases

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func init() { gin.SetMode(gin.TestMode) }

// fakeFilamentRepo records the SearchFilaments call so a test can assert FindAll
// forwards the structured filters through the search path.
type fakeFilamentRepo struct {
	lastFilters map[string]interface{}
	lastSearch  string
	searchCalls int
}

func (f *fakeFilamentRepo) Create(context.Context, *entities.FilamentEntity) error { return nil }
func (f *fakeFilamentRepo) FindByID(context.Context, uuid.UUID, string) (*entities.FilamentEntity, error) {
	return nil, nil
}
func (f *fakeFilamentRepo) Update(context.Context, *entities.FilamentEntity) error { return nil }
func (f *fakeFilamentRepo) Delete(context.Context, uuid.UUID) error                { return nil }
func (f *fakeFilamentRepo) FindAll(context.Context, string, string, string, int, int) ([]*entities.FilamentEntity, int64, error) {
	return nil, 0, nil
}
func (f *fakeFilamentRepo) SearchFilaments(_ context.Context, _ string, filters map[string]interface{}, search, _ string, _, _ int) ([]*entities.FilamentEntity, int64, error) {
	f.searchCalls++
	f.lastFilters = filters
	f.lastSearch = search
	return []*entities.FilamentEntity{}, 0, nil
}
func (f *fakeFilamentRepo) ExistsByNameAndBrand(context.Context, string, uuid.UUID, string, *uuid.UUID) (bool, error) {
	return false, nil
}
func (f *fakeFilamentRepo) GetBrandInfo(context.Context, uuid.UUID, string) (*entities.BrandInfo, error) {
	return nil, nil
}
func (f *fakeFilamentRepo) GetMaterialInfo(context.Context, uuid.UUID, string) (*entities.MaterialInfo, error) {
	return nil, nil
}
func (f *fakeFilamentRepo) GetBrandsInfo(context.Context, []uuid.UUID, string) (map[uuid.UUID]*entities.BrandInfo, error) {
	return map[uuid.UUID]*entities.BrandInfo{}, nil
}
func (f *fakeFilamentRepo) GetMaterialsInfo(context.Context, []uuid.UUID, string) (map[uuid.UUID]*entities.MaterialInfo, error) {
	return map[uuid.UUID]*entities.MaterialInfo{}, nil
}

type noopActivity struct{}

func (noopActivity) Record(context.Context, activityEntities.ActivityEntity) {}
func (noopActivity) ListActivities(*gin.Context)                             {}
func (noopActivity) FindRecentByOrganization(string, int) ([]activityEntities.ActivityEntity, error) {
	return nil, nil
}

func TestFindAll_HonoursSearchFilters(t *testing.T) {
	repo := &fakeFilamentRepo{}
	uc := &FilamentUseCase{repository: repo, logger: &logger.NoopLogger{}, activityService: noopActivity{}}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("organization_id", "org-1")
	brandID := uuid.New()
	materialID := uuid.New()
	c.Request = httptest.NewRequest(http.MethodGet,
		"/filaments?brand_id="+brandID.String()+
			"&material_id="+materialID.String()+
			"&color_type=solid&diameter=1.75&min_price=10&max_price=100&q=pla", nil)

	uc.FindAll(c)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, repo.searchCalls, "FindAll must use the repository search path")
	assert.Equal(t, "pla", repo.lastSearch)
	assert.Equal(t, brandID, repo.lastFilters["brand_id"])
	assert.Equal(t, materialID, repo.lastFilters["material_id"])
	assert.Equal(t, "solid", repo.lastFilters["color_type"])
	assert.EqualValues(t, 1.75, repo.lastFilters["diameter"])
	assert.EqualValues(t, 10, repo.lastFilters["min_price"])
	assert.EqualValues(t, 100, repo.lastFilters["max_price"])
}

func TestFindAll_InvalidBrandIDFilter(t *testing.T) {
	repo := &fakeFilamentRepo{}
	uc := &FilamentUseCase{repository: repo, logger: &logger.NoopLogger{}, activityService: noopActivity{}}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("organization_id", "org-1")
	c.Request = httptest.NewRequest(http.MethodGet, "/filaments?brand_id=not-a-uuid", nil)

	uc.FindAll(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, 0, repo.searchCalls)
}

func TestApplyFilamentUpdate_IsActive(t *testing.T) {
	active := true
	f := &entities.FilamentEntity{IsActive: false}
	applyFilamentUpdate(f, &entities.UpdateFilamentRequest{})
	if f.IsActive {
		t.Fatal("absent is_active must keep the stored value")
	}
	applyFilamentUpdate(f, &entities.UpdateFilamentRequest{IsActive: &active})
	if !f.IsActive {
		t.Fatal("explicit is_active must be applied")
	}
}
