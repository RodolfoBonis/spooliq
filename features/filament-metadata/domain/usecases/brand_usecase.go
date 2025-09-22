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

// BrandUseCase defines operations for brand management
type BrandUseCase interface {
	CreateBrand(c *gin.Context)
	GetBrand(c *gin.Context)
	GetAllBrands(c *gin.Context)
	UpdateBrand(c *gin.Context)
	DeleteBrand(c *gin.Context)
}

// CreateBrandRequest represents a request to create a new brand
type CreateBrandRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Description *string `json:"description,omitempty"`
}

// UpdateBrandRequest represents a request to update a brand
type UpdateBrandRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Description *string `json:"description,omitempty"`
	Active      bool    `json:"active"`
}

// BrandResponse represents a brand in API responses
type BrandResponse struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Active      bool    `json:"active"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// BrandListResponse represents the response for brand lists
type BrandListResponse struct {
	Data []*BrandResponse `json:"data"`
}

type brandUseCaseImpl struct {
	brandRepo repositories.BrandRepository
	validator *validator.Validate
	logger    logger.Logger
}

// NewBrandUseCase creates a new instance of brand use case
func NewBrandUseCase(brandRepo repositories.BrandRepository, logger logger.Logger) BrandUseCase {
	return &brandUseCaseImpl{
		brandRepo: brandRepo,
		validator: validator.New(),
		logger:    logger,
	}
}

func (uc *brandUseCaseImpl) CreateBrand(c *gin.Context) {
	var request CreateBrandRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errors.InvalidRequestResponse(err.Error()))
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		c.JSON(http.StatusBadRequest, errors.ValidationErrorResponse(err.Error()))
		return
	}

	// Check if brand already exists
	exists, err := uc.brandRepo.Exists(c.Request.Context(), request.Name)
	if err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to check brand existence", map[string]interface{}{
			"name":  request.Name,
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToCreateBrand))
		return
	}

	if exists {
		c.JSON(http.StatusConflict, errors.ErrorResponse("Brand with this name already exists"))
		return
	}

	brand := &entities.FilamentBrand{
		Name:        request.Name,
		Description: request.Description,
		Active:      true,
	}

	if err := uc.brandRepo.Create(c.Request.Context(), brand); err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to create brand", map[string]interface{}{
			"name":  request.Name,
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToCreateBrand))
		return
	}

	response := toBrandResponse(brand)
	c.JSON(http.StatusCreated, response)
}

func (uc *brandUseCaseImpl) GetBrand(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrorResponse("Invalid brand ID"))
		return
	}

	brand, err := uc.brandRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound || strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, errors.ErrorResponse(errors.ErrorMessages.BrandNotFound))
			return
		}
		uc.logger.Error(c.Request.Context(), "Failed to get brand", map[string]interface{}{
			"brand_id": id,
			"error":    err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToGetBrands))
		return
	}

	response := toBrandResponse(brand)
	c.JSON(http.StatusOK, response)
}

func (uc *brandUseCaseImpl) GetAllBrands(c *gin.Context) {
	activeOnly := c.Query("active_only") == "true"

	brands, err := uc.brandRepo.GetAll(c.Request.Context(), activeOnly)
	if err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to get brands", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToGetBrands))
		return
	}

	var responses []*BrandResponse
	for _, brand := range brands {
		responses = append(responses, toBrandResponse(brand))
	}

	c.JSON(http.StatusOK, BrandListResponse{Data: responses})
}

func (uc *brandUseCaseImpl) UpdateBrand(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrorResponse("Invalid brand ID"))
		return
	}

	var request UpdateBrandRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errors.InvalidRequestResponse(err.Error()))
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		c.JSON(http.StatusBadRequest, errors.ValidationErrorResponse(err.Error()))
		return
	}

	// Get existing brand
	brand, err := uc.brandRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound || strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, errors.ErrorResponse(errors.ErrorMessages.BrandNotFound))
			return
		}
		uc.logger.Error(c.Request.Context(), "Failed to get brand for update", map[string]interface{}{
			"brand_id": id,
			"error":    err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToUpdateBrand))
		return
	}

	// Check if name is being changed and if it conflicts
	if brand.Name != request.Name {
		exists, err := uc.brandRepo.Exists(c.Request.Context(), request.Name)
		if err != nil {
			uc.logger.Error(c.Request.Context(), "Failed to check brand existence for update", map[string]interface{}{
				"name":  request.Name,
				"error": err.Error(),
			})
			c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToUpdateBrand))
			return
		}

		if exists {
			c.JSON(http.StatusConflict, errors.ErrorResponse("Brand with this name already exists"))
			return
		}
	}

	// Update brand fields
	brand.Name = request.Name
	brand.Description = request.Description
	brand.Active = request.Active

	if err := uc.brandRepo.Update(c.Request.Context(), brand); err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to update brand", map[string]interface{}{
			"brand_id": id,
			"error":    err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToUpdateBrand))
		return
	}

	response := toBrandResponse(brand)
	c.JSON(http.StatusOK, response)
}

func (uc *brandUseCaseImpl) DeleteBrand(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrorResponse("Invalid brand ID"))
		return
	}

	// Check if brand exists
	_, err = uc.brandRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound || strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, errors.ErrorResponse(errors.ErrorMessages.BrandNotFound))
			return
		}
		uc.logger.Error(c.Request.Context(), "Failed to get brand for deletion", map[string]interface{}{
			"brand_id": id,
			"error":    err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToDeleteBrand))
		return
	}

	if err := uc.brandRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to delete brand", map[string]interface{}{
			"brand_id": id,
			"error":    err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToDeleteBrand))
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// toBrandResponse converts a brand entity to API response format
func toBrandResponse(brand *entities.FilamentBrand) *BrandResponse {
	return &BrandResponse{
		ID:          brand.ID,
		Name:        brand.Name,
		Description: brand.Description,
		Active:      brand.Active,
		CreatedAt:   brand.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   brand.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
