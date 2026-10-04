package preset

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/preset/mocks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noopActivityService satisfies activityUc.IActivityService without side effects.
type noopActivityService struct{}

func (noopActivityService) Record(_ context.Context, _ activityEntities.ActivityEntity) {}
func (noopActivityService) ListActivities(_ *gin.Context)                               {}
func (noopActivityService) FindRecentByOrganization(_ string, _ int) ([]activityEntities.ActivityEntity, error) {
	return nil, nil
}

const (
	orgA = "org-a"
	orgB = "org-b"
)

func newTestHandler(repo *mocks.InMemoryPresetRepository) *Handler {
	return NewPresetHandler(
		usecases.NewCreatePresetUseCase(repo),
		usecases.NewFindPresetUseCase(repo),
		usecases.NewUpdatePresetUseCase(repo),
		usecases.NewDeletePresetUseCase(repo),
		noopActivityService{},
	)
}

func newTestContext(method, path string, orgID string) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	if orgID != "" {
		c.Set("organization_id", orgID)
	}
	return w, c
}

func seedMachine(repo *mocks.InMemoryPresetRepository, org string, isDefault bool) uuid.UUID {
	id := uuid.New()
	repo.Seed(&entities.PresetEntity{
		ID:             id,
		Name:           "Printer",
		Type:           entities.PresetTypeMachine,
		IsActive:       true,
		IsDefault:      isDefault,
		OrganizationID: org,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, &entities.MachinePresetEntity{
		ID:               id,
		OrganizationID:   org,
		BuildVolumeX:     200,
		BuildVolumeY:     200,
		BuildVolumeZ:     200,
		NozzleDiameter:   0.4,
		LayerHeightMin:   0.1,
		LayerHeightMax:   0.3,
		PrintSpeedMax:    120,
		PowerConsumption: 150,
		FilamentDiameter: 1.75,
	})
	return id
}

// Cross-org GET must return 404, not 403, to avoid leaking existence.
func TestGetPresetByID_CrossOrgReturns404(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	handler := newTestHandler(repo)

	id := seedMachine(repo, orgB, false)

	w, c := newTestContext(http.MethodGet, "/presets/"+id.String(), orgA)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}
	handler.GetPresetByID(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetPresetByID_OwnedReturns200(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	handler := newTestHandler(repo)

	id := seedMachine(repo, orgA, false)

	w, c := newTestContext(http.MethodGet, "/presets/"+id.String(), orgA)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}
	handler.GetPresetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetPresetByID_InvalidIDReturns400(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	handler := newTestHandler(repo)

	w, c := newTestContext(http.MethodGet, "/presets/not-a-uuid", orgA)
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	handler.GetPresetByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetPresetByID_MissingOrgReturns400(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	handler := newTestHandler(repo)

	id := seedMachine(repo, orgA, false)
	w, c := newTestContext(http.MethodGet, "/presets/"+id.String(), "")
	c.Params = gin.Params{{Key: "id", Value: id.String()}}
	handler.GetPresetByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Cross-org DELETE must return 404 and leave the other org's data intact.
func TestDeletePreset_CrossOrgReturns404(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	handler := newTestHandler(repo)

	id := seedMachine(repo, orgB, false)

	w, c := newTestContext(http.MethodDelete, "/presets/"+id.String(), orgA)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}
	handler.DeletePreset(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Nil(t, repo.StoredPreset(id).DeletedAt)
}

// Deleting a default preset must return 409 Conflict.
func TestDeletePreset_DefaultReturns409(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	handler := newTestHandler(repo)

	id := seedMachine(repo, orgA, true)

	w, c := newTestContext(http.MethodDelete, "/presets/"+id.String(), orgA)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}
	handler.DeletePreset(c)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// Deleting an owned, non-default preset returns 204.
func TestDeletePreset_OwnedReturns204(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	handler := newTestHandler(repo)

	id := seedMachine(repo, orgA, false)

	w, c := newTestContext(http.MethodDelete, "/presets/"+id.String(), orgA)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}
	handler.DeletePreset(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, repo.StoredPreset(id).DeletedAt)
}

// Cross-org typed GET must return 404.
func TestGetMachinePresetByID_CrossOrgReturns404(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	handler := newTestHandler(repo)

	id := seedMachine(repo, orgB, false)

	w, c := newTestContext(http.MethodGet, "/presets/machines/"+id.String(), orgA)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}
	handler.GetMachinePresetByID(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Listing presets must be organization-scoped (never leak other orgs).
func TestGetPresets_ScopedToOrganization(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	handler := newTestHandler(repo)

	seedMachine(repo, orgA, false)
	seedMachine(repo, orgB, false)

	w, c := newTestContext(http.MethodGet, "/presets", orgA)
	handler.GetPresets(c)

	assert.Equal(t, http.StatusOK, w.Code)
	// Only org A's single preset is visible.
	require.Len(t, repo.OrgScopes, 1)
	assert.Equal(t, orgA, repo.OrgScopes[0])
}
