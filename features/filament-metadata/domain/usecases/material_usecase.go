package usecases

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/gorm"
)

// MaterialUseCase defines operations for material management
type MaterialUseCase interface {
	CreateMaterial(c *gin.Context)
	GetMaterial(c *gin.Context)
	GetAllMaterials(c *gin.Context)
	UpdateMaterial(c *gin.Context)
	DeleteMaterial(c *gin.Context)
}

// CreateMaterialRequest represents a request to create a new material
type CreateMaterialRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=50"`
	Description *string `json:"description,omitempty"`
	Properties  *string `json:"properties,omitempty"`
}

// UpdateMaterialRequest represents a request to update a material
type UpdateMaterialRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=50"`
	Description *string `json:"description,omitempty"`
	Properties  *string `json:"properties,omitempty"`
	Active      bool    `json:"active"`
}

// MaterialResponse represents a material in API responses
type MaterialResponse struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Properties  *string `json:"properties,omitempty"`
	Active      bool    `json:"active"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// MaterialListResponse represents the response for material lists
type MaterialListResponse struct {
	Data []*MaterialResponse `json:"data"`
}

type materialUseCaseImpl struct {
	materialRepo repositories.MaterialRepository
	validator    *validator.Validate
	logger       logger.Logger
}

// NewMaterialUseCase creates a new instance of material use case
func NewMaterialUseCase(materialRepo repositories.MaterialRepository, validator *validator.Validate, logger logger.Logger) MaterialUseCase {
	return &materialUseCaseImpl{
		materialRepo: materialRepo,
		validator:    validator,
		logger:       logger,
	}
}

func (uc *materialUseCaseImpl) CreateMaterial(c *gin.Context) {
	var request CreateMaterialRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errors.InvalidRequestResponse(err.Error()))
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		c.JSON(http.StatusBadRequest, errors.ValidationErrorResponse(err.Error()))
		return
	}

	// Check if material already exists
	exists, err := uc.materialRepo.Exists(c.Request.Context(), request.Name)
	if err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to check material existence", map[string]interface{}{
			"name":  request.Name,
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToCreateMaterial))
		return
	}

	if exists {
		c.JSON(http.StatusConflict, errors.ErrorResponse("Material with this name already exists"))
		return
	}

	material := &entities.FilamentMaterial{
		Name:        request.Name,
		Description: request.Description,
		Properties:  request.Properties,
		Active:      true,
	}

	if err := uc.materialRepo.Create(c.Request.Context(), material); err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to create material", map[string]interface{}{
			"name":  request.Name,
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToCreateMaterial))
		return
	}

	response := toMaterialResponse(material)
	c.JSON(http.StatusCreated, response)
}

func (uc *materialUseCaseImpl) GetMaterial(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrorResponse("Invalid material ID"))
		return
	}

	material, err := uc.materialRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound || strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, errors.ErrorResponse(errors.ErrorMessages.MaterialNotFound))
			return
		}
		uc.logger.Error(c.Request.Context(), "Failed to get material", map[string]interface{}{
			"material_id": id,
			"error":       err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToGetMaterials))
		return
	}

	response := toMaterialResponse(material)
	c.JSON(http.StatusOK, response)
}

func (uc *materialUseCaseImpl) GetAllMaterials(c *gin.Context) {
	activeOnly := c.Query("active_only") == "true"

	materials, err := uc.materialRepo.GetAll(c.Request.Context(), activeOnly)
	if err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to get materials", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToGetMaterials))
		return
	}

	var responses []*MaterialResponse
	for _, material := range materials {
		responses = append(responses, toMaterialResponse(material))
	}

	c.JSON(http.StatusOK, MaterialListResponse{Data: responses})
}

func (uc *materialUseCaseImpl) UpdateMaterial(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrorResponse("Invalid material ID"))
		return
	}

	var request UpdateMaterialRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errors.InvalidRequestResponse(err.Error()))
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		c.JSON(http.StatusBadRequest, errors.ValidationErrorResponse(err.Error()))
		return
	}

	// Get existing material
	material, err := uc.materialRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound || strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, errors.ErrorResponse(errors.ErrorMessages.MaterialNotFound))
			return
		}
		uc.logger.Error(c.Request.Context(), "Failed to get material for update", map[string]interface{}{
			"material_id": id,
			"error":       err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToUpdateMaterial))
		return
	}

	// Check if name is being changed and if it conflicts
	if material.Name != request.Name {
		exists, err := uc.materialRepo.Exists(c.Request.Context(), request.Name)
		if err != nil {
			uc.logger.Error(c.Request.Context(), "Failed to check material existence for update", map[string]interface{}{
				"name":  request.Name,
				"error": err.Error(),
			})
			c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToUpdateMaterial))
			return
		}

		if exists {
			c.JSON(http.StatusConflict, errors.ErrorResponse("Material with this name already exists"))
			return
		}
	}

	// Update material fields
	material.Name = request.Name
	material.Description = request.Description
	material.Properties = request.Properties
	material.Active = request.Active

	if err := uc.materialRepo.Update(c.Request.Context(), material); err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to update material", map[string]interface{}{
			"material_id": id,
			"error":       err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToUpdateMaterial))
		return
	}

	response := toMaterialResponse(material)
	c.JSON(http.StatusOK, response)
}

func (uc *materialUseCaseImpl) DeleteMaterial(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrorResponse("Invalid material ID"))
		return
	}

	// Check if material exists
	_, err = uc.materialRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound || strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, errors.ErrorResponse(errors.ErrorMessages.MaterialNotFound))
			return
		}
		uc.logger.Error(c.Request.Context(), "Failed to get material for deletion", map[string]interface{}{
			"material_id": id,
			"error":       err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToDeleteMaterial))
		return
	}

	if err := uc.materialRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to delete material", map[string]interface{}{
			"material_id": id,
			"error":       err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToDeleteMaterial))
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// toMaterialResponse converts a material entity to API response format
func toMaterialResponse(material *entities.FilamentMaterial) *MaterialResponse {
	return &MaterialResponse{
		ID:          material.ID,
		Name:        material.Name,
		Description: material.Description,
		Properties:  material.Properties,
		Active:      material.Active,
		CreatedAt:   material.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   material.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
