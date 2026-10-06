package profile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	presetEntities "github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	presetMocks "github.com/RodolfoBonis/spooliq/features/preset/mocks"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/usecases"
	profileMocks "github.com/RodolfoBonis/spooliq/features/profile/mocks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const orgA = "org-a"

type envelope struct {
	Data       []json.RawMessage `json:"data"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

func newHandler() (*Handler, *presetMocks.InMemoryPresetRepository, *profileMocks.InMemoryProfileRepository) {
	presetRepo := presetMocks.NewInMemoryPresetRepository()
	profileRepo := profileMocks.NewInMemoryProfileRepository()
	uc := usecases.NewProfileUseCase(profileRepo, presetRepo)
	return NewProfileHandler(uc), presetRepo, profileRepo
}

func seedProfile(uc *usecases.ProfileUseCase, presetRepo *presetMocks.InMemoryPresetRepository, name string) {
	mid := uuid.New()
	presetRepo.Seed(&presetEntities.PresetEntity{ID: mid, Name: "M", Type: presetEntities.PresetTypeMachine, IsActive: true, OrganizationID: orgA}, nil)
	eid := uuid.New()
	presetRepo.Seed(&presetEntities.PresetEntity{ID: eid, Name: "E", Type: presetEntities.PresetTypeEnergy, IsActive: true, OrganizationID: orgA}, nil)
	_, err := uc.Create(&entities.CreateProfileRequest{Name: name, MachinePresetID: mid, EnergyPresetID: eid}, orgA, "user-1")
	if err != nil {
		panic(err)
	}
}

func testCtx(method, path, org string) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	if org != "" {
		c.Set("organization_id", org)
	}
	return w, c
}

// GET /profiles returns the standard envelope with default page size 100.
func TestList_ReturnsEnvelope(t *testing.T) {
	handler, presetRepo, _ := newHandler()
	seedProfile(handler.useCase, presetRepo, "Alpha")
	seedProfile(handler.useCase, presetRepo, "Beta")

	w, c := testCtx(http.MethodGet, "/profiles", orgA)
	handler.List(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var env envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, int64(2), env.Total)
	assert.Len(t, env.Data, 2)
	assert.Equal(t, 100, env.PageSize)
	assert.Equal(t, 1, env.Page)
}

// q filters by name; page_size paginates.
func TestList_SearchAndPagination(t *testing.T) {
	handler, presetRepo, _ := newHandler()
	seedProfile(handler.useCase, presetRepo, "Alpha")
	seedProfile(handler.useCase, presetRepo, "Alphabet")
	seedProfile(handler.useCase, presetRepo, "Gamma")

	w, c := testCtx(http.MethodGet, "/profiles?q=alpha&page_size=1", orgA)
	handler.List(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var env envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, int64(2), env.Total)
	assert.Len(t, env.Data, 1)
	assert.Equal(t, 2, env.TotalPages)
}

// Invalid UUID path returns a 400 with a stable code (no leaked error text).
func TestGet_InvalidIDCode(t *testing.T) {
	handler, _, _ := newHandler()

	w, c := testCtx(http.MethodGet, "/profiles/nope", orgA)
	c.Params = gin.Params{{Key: "id", Value: "nope"}}
	handler.Get(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "invalid_profile_id", body.Code)
}
