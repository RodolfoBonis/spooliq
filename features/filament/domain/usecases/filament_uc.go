package usecases

import (
	"context"

	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"

	log "github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/filament/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FilamentUseCase implements filament business logic operations.
type FilamentUseCase struct {
	repository      repositories.FilamentRepository
	logger          log.Logger
	activityService activityUc.IActivityService
}

// IFilamentUseCase defines the contract for filament use case operations.
type IFilamentUseCase interface {
	Create(c *gin.Context)
	Update(c *gin.Context)
	FindByID(c *gin.Context)
	FindAll(c *gin.Context)
	Delete(c *gin.Context)
	Search(c *gin.Context)
}

// NewFilamentUseCase creates a new instance of the filament use case.
func NewFilamentUseCase(repository repositories.FilamentRepository, logger log.Logger, activityService activityUc.IActivityService) IFilamentUseCase {
	return &FilamentUseCase{
		repository:      repository,
		logger:          logger,
		activityService: activityService,
	}
}

// filamentSortWhitelist maps public sort names to safe SQL column expressions.
// Columns are qualified with the filaments table so they stay unambiguous when
// the search joins (brands/materials) are present. Default sort is created_at
// descending.
var filamentSortWhitelist = map[string]string{
	"name":         "lower(filaments.name)",
	"created_at":   "filaments.created_at",
	"price_per_kg": "filaments.price_per_kg",
}

// buildFilamentResponses assembles FilamentResponse values for a page of
// filaments, batch-loading the related brand and material info in exactly one
// query each (per page) regardless of row count. This is the N+1 fix for the
// list/search endpoints, which previously issued two queries per row.
func (uc *FilamentUseCase) buildFilamentResponses(ctx context.Context, organizationID string, filaments []*entities.FilamentEntity) []entities.FilamentResponse {
	responses := make([]entities.FilamentResponse, len(filaments))

	brandIDs := make([]uuid.UUID, 0, len(filaments))
	materialIDs := make([]uuid.UUID, 0, len(filaments))
	for _, f := range filaments {
		brandIDs = append(brandIDs, f.BrandID)
		materialIDs = append(materialIDs, f.MaterialID)
	}

	brands, err := uc.repository.GetBrandsInfo(ctx, brandIDs, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to batch-load brand info", map[string]interface{}{"error": err.Error()})
		brands = map[uuid.UUID]*entities.BrandInfo{}
	}
	materials, err := uc.repository.GetMaterialsInfo(ctx, materialIDs, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to batch-load material info", map[string]interface{}{"error": err.Error()})
		materials = map[uuid.UUID]*entities.MaterialInfo{}
	}

	for i, f := range filaments {
		responses[i] = entities.FilamentResponse{FilamentEntity: f}
		if b, ok := brands[f.BrandID]; ok {
			responses[i].Brand = b
		}
		if m, ok := materials[f.MaterialID]; ok {
			responses[i].Material = m
		}
	}

	return responses
}
