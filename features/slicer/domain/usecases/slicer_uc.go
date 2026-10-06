package usecases

import (
	log "github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/service"
	"github.com/gin-gonic/gin"
)

// ISlicerUseCase defines the slicer feature's HTTP operations.
type ISlicerUseCase interface {
	// Analyze parses an uploaded sliced file and returns the analysis enriched
	// with per-slot filament suggestions.
	Analyze(c *gin.Context)
}

// SlicerUseCase implements the slicer HTTP operations.
type SlicerUseCase struct {
	service service.Service
	logger  log.Logger
}

// NewSlicerUseCase creates a new slicer use case.
func NewSlicerUseCase(svc service.Service, logger log.Logger) ISlicerUseCase {
	return &SlicerUseCase{service: svc, logger: logger}
}
