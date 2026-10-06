package usecases

import (
	"net/http"
	"testing"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBudgetAPIError_InvalidModel3DReference(t *testing.T) {
	apiErr := budgetAPIError(entities.ErrInvalidModel3DReference)
	if assert.NotNil(t, apiErr) {
		assert.Equal(t, http.StatusBadRequest, apiErr.Status)
		assert.Equal(t, CodeInvalidModel3DReference, apiErr.Code)
		assert.Equal(t, "invalid_model3d_reference", apiErr.Code)
		assert.NotEmpty(t, apiErr.Message)
	}
}

func TestCollectModel3DIDs_DedupAndSkipNil(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	reqs := []entities.BudgetItemRequest{
		{Model3DID: &id1},
		{Model3DID: nil},
		{Model3DID: &id1}, // duplicate
		{Model3DID: &id2},
	}
	got := collectModel3DIDs(reqs)
	assert.ElementsMatch(t, []uuid.UUID{id1, id2}, got)
	assert.Len(t, got, 2)
}
