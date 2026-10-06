package preset

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/mocks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// envelope mirrors the standard list envelope for assertions.
type envelope struct {
	Data       []json.RawMessage `json:"data"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

func seedActiveMachine(repo *mocks.InMemoryPresetRepository, org, name string) uuid.UUID {
	id := uuid.New()
	repo.Seed(&entities.PresetEntity{
		ID:             id,
		Name:           name,
		Type:           entities.PresetTypeMachine,
		IsActive:       true,
		OrganizationID: org,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, &entities.MachinePresetEntity{
		ID:             id,
		OrganizationID: org,
		Brand:          "Prusa",
	})
	return id
}

// GetPresets returns the standard envelope with correct totals and page defaults.
func TestGetPresets_ReturnsEnvelope(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	handler := newTestHandler(repo)

	seedActiveMachine(repo, orgA, "Alpha")
	seedActiveMachine(repo, orgA, "Beta")
	seedActiveMachine(repo, orgB, "Other")

	w, c := newTestContext(http.MethodGet, "/presets", orgA)
	handler.GetPresets(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var env envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, int64(2), env.Total)
	assert.Len(t, env.Data, 2)
	assert.Equal(t, 1, env.Page)
	assert.Equal(t, 100, env.PageSize) // default page size for presets
	assert.Equal(t, 1, env.TotalPages)
}

// page_size and q are honored, and data is never null.
func TestGetPresets_PaginationAndSearch(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	handler := newTestHandler(repo)

	seedActiveMachine(repo, orgA, "Alpha")
	seedActiveMachine(repo, orgA, "Alphabet")
	seedActiveMachine(repo, orgA, "Gamma")

	// q=alpha matches two; page_size=1 returns the first page of one.
	w, c := newTestContext(http.MethodGet, "/presets?q=alpha&page_size=1", orgA)
	handler.GetPresets(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var env envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, int64(2), env.Total)
	assert.Len(t, env.Data, 1)
	assert.Equal(t, 1, env.PageSize)
	assert.Equal(t, 2, env.TotalPages)
}

// Deleting a default preset returns the stable snake_case code.
func TestDeletePreset_DefaultConflictCode(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	handler := newTestHandler(repo)

	id := seedMachine(repo, orgA, true)

	w, c := newTestContext(http.MethodDelete, "/presets/"+id.String(), orgA)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}
	handler.DeletePreset(c)

	require.Equal(t, http.StatusConflict, w.Code)
	var body struct {
		Code    string `json:"code"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "default_preset_cannot_be_deleted", body.Code)
	assert.NotEmpty(t, body.Message)
	assert.Equal(t, body.Error, body.Message)
}

// An invalid UUID path param returns a 400 with a stable code (no leaked error string).
func TestGetPresetByID_InvalidIDCode(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	handler := newTestHandler(repo)

	w, c := newTestContext(http.MethodGet, "/presets/nope", orgA)
	c.Params = gin.Params{{Key: "id", Value: "nope"}}
	handler.GetPresetByID(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "invalid_preset_id", body.Code)
}
