package usecases

import (
	log "github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/repositories"
	slicerservice "github.com/RodolfoBonis/spooliq/features/slicer/domain/service"
	"github.com/gin-gonic/gin"
)

// IModel3DUseCase defines the contract for 3D model use case operations.
type IModel3DUseCase interface {
	Upload(c *gin.Context)
	FindAll(c *gin.Context)
	FindByID(c *gin.Context)
	FindByCustomer(c *gin.Context)
	StreamFile(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	GetSliceAnalysis(c *gin.Context)
}

// Model3DUseCase implements 3D model business logic operations.
type Model3DUseCase struct {
	repository       repositories.Model3DRepository
	cdnService       *services.CDNService
	thumbnailService *services.ThumbnailService
	slicerService    slicerservice.Service
	logger           log.Logger
	activityService  activityUc.IActivityService
}

// NewModel3DUseCase creates a new instance of the 3D model use case.
func NewModel3DUseCase(
	repository repositories.Model3DRepository,
	cdnService *services.CDNService,
	thumbnailService *services.ThumbnailService,
	slicerService slicerservice.Service,
	logger log.Logger,
	activityService activityUc.IActivityService,
) IModel3DUseCase {
	return &Model3DUseCase{
		repository:       repository,
		cdnService:       cdnService,
		thumbnailService: thumbnailService,
		slicerService:    slicerService,
		logger:           logger,
		activityService:  activityService,
	}
}
