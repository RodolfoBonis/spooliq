package usecases

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RodolfoBonis/spooliq/features/model3d/domain/entities"
	slicerentities "github.com/RodolfoBonis/spooliq/features/slicer/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSliceAnalysis_InvalidID(t *testing.T) {
	uc := newUC(newFakeRepo())
	c, rec := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodGet, "/models3d/bad/slice-analysis", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad"}}
	uc.GetSliceAnalysis(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_model3d_id", decodeEnvelope(t, rec)["code"])
}

func TestGetSliceAnalysis_ModelNotFound(t *testing.T) {
	uc := newUC(newFakeRepo())
	c, rec := ctxWithOrg(testOrg)
	id := uuid.New()
	c.Request = httptest.NewRequest(http.MethodGet, "/models3d/"+id.String()+"/slice-analysis", nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}
	uc.GetSliceAnalysis(c)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "model3d_not_found", decodeEnvelope(t, rec)["code"])
}

func TestGetSliceAnalysis_NotSlicedModel(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.byID[key(testOrg, id)] = &entities.Model3DEntity{ID: id, OrganizationID: testOrg, Name: "no-analysis"}

	uc := newUC(repo)
	c, rec := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodGet, "/models3d/"+id.String()+"/slice-analysis", nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}
	uc.GetSliceAnalysis(c)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "slice_analysis_not_found", decodeEnvelope(t, rec)["code"])
}

func TestGetSliceAnalysis_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.byID[key(testOrg, id)] = &entities.Model3DEntity{
		ID:             id,
		OrganizationID: testOrg,
		Name:           "sliced",
		SliceAnalysis: &slicerentities.Analysis{
			Source: slicerentities.Source3MF,
			Plates: []slicerentities.Plate{{
				Index:            1,
				PrintTimeSeconds: 3600,
				Filaments: []slicerentities.Filament{
					{Slot: 1, Grams: 12.34, ColorHex: "#FF0000", Material: "PLA"},
				},
			}},
			Warnings: []string{},
		},
	}

	uc := newUC(repo)
	c, rec := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodGet, "/models3d/"+id.String()+"/slice-analysis", nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}
	uc.GetSliceAnalysis(c)
	require.Equal(t, http.StatusOK, rec.Code)

	body := decodeEnvelope(t, rec)
	assert.Equal(t, "3mf", body["source"])
	plates := body["plates"].([]any)
	require.Len(t, plates, 1)
	plate := plates[0].(map[string]any)
	assert.EqualValues(t, 3600, plate["print_time_seconds"])
	// Suggestion present (null) since the test catalog is empty.
	fil := plate["filaments"].([]any)[0].(map[string]any)
	assert.Contains(t, fil, "suggestion")
	assert.Nil(t, fil["suggestion"])
}
