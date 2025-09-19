package usecases

import (
	"net/http"
	"strconv"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/filaments/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/filaments/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type filamentUseCaseImpl struct {
	filamentRepo repositories.FilamentRepository
	logger       logger.Logger
	validator    *validator.Validate
}

func NewFilamentUseCase(filamentRepo repositories.FilamentRepository, logger logger.Logger) FilamentUseCase {
	return &filamentUseCaseImpl{
		filamentRepo: filamentRepo,
		logger:       logger,
		validator:    validator.New(),
	}
}

func (uc *filamentUseCaseImpl) CreateFilament(c *gin.Context) {
	var request CreateFilamentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errors.InvalidRequestResponse(err.Error()))
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		c.JSON(http.StatusBadRequest, errors.ValidationErrorResponse(err.Error()))
		return
	}

	userID := c.GetString("user_id")
	var ownerUserID *string
	if userID != "" {
		ownerUserID = &userID
	}

	filament := &entities.Filament{
		Name:          request.Name,
		Brand:         request.Brand,
		Material:      request.Material,
		Color:         request.Color,
		ColorHex:      request.ColorHex,
		PricePerKg:    request.PricePerKg,
		PricePerMeter: request.PricePerMeter,
		URL:           request.URL,
		OwnerUserID:   ownerUserID,
	}

	if err := uc.filamentRepo.Create(c.Request.Context(), filament); err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to create filament", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToCreateFilament))
		return
	}

	response := ToFilamentResponse(filament)
	c.JSON(http.StatusCreated, response)
}

func (uc *filamentUseCaseImpl) GetFilament(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrorResponse("ID do filamento inválido"))
		return
	}

	userID := c.GetString("user_id")
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	filament, err := uc.filamentRepo.GetByID(c.Request.Context(), uint(id), userIDPtr)
	if err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to get filament", map[string]interface{}{
			"filament_id": id,
			"error":       err.Error(),
		})
		c.JSON(http.StatusNotFound, errors.ErrorResponse(errors.ErrorMessages.FilamentNotFound))
		return
	}

	response := ToFilamentResponse(filament)
	c.JSON(http.StatusOK, response)
}

func (uc *filamentUseCaseImpl) GetAllFilaments(c *gin.Context) {
	userID := c.GetString("user_id")
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	filaments, err := uc.filamentRepo.GetAll(c.Request.Context(), userIDPtr)
	if err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to get filaments", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToGetFilaments))
		return
	}

	responses := make([]*FilamentResponse, 0, len(filaments))
	for _, filament := range filaments {
		responses = append(responses, ToFilamentResponse(filament))
	}

	c.JSON(http.StatusOK, gin.H{"filaments": responses})
}

func (uc *filamentUseCaseImpl) UpdateFilament(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrorResponse("ID do filamento inválido"))
		return
	}

	var request UpdateFilamentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errors.InvalidRequestResponse(err.Error()))
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		c.JSON(http.StatusBadRequest, errors.ValidationErrorResponse(err.Error()))
		return
	}

	userID := c.GetString("user_id")
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	filament := &entities.Filament{
		ID:            uint(id),
		Name:          request.Name,
		Brand:         request.Brand,
		Material:      request.Material,
		Color:         request.Color,
		ColorHex:      request.ColorHex,
		PricePerKg:    request.PricePerKg,
		PricePerMeter: request.PricePerMeter,
		URL:           request.URL,
	}

	if err := uc.filamentRepo.Update(c.Request.Context(), filament, userIDPtr); err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to update filament", map[string]interface{}{
			"filament_id": id,
			"error":       err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToUpdateFilament))
		return
	}

	response := ToFilamentResponse(filament)
	c.JSON(http.StatusOK, response)
}

func (uc *filamentUseCaseImpl) DeleteFilament(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrorResponse("ID do filamento inválido"))
		return
	}

	userID := c.GetString("user_id")
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	if err := uc.filamentRepo.Delete(c.Request.Context(), uint(id), userIDPtr); err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to delete filament", map[string]interface{}{
			"filament_id": id,
			"error":       err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToDeleteFilament))
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (uc *filamentUseCaseImpl) GetUserFilaments(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, errors.ErrorResponse(errors.ErrorMessages.UserNotAuthenticated))
		return
	}

	filaments, err := uc.filamentRepo.GetByOwner(c.Request.Context(), userID)
	if err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to get user filaments", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToGetUserFilaments))
		return
	}

	responses := make([]*FilamentResponse, 0, len(filaments))
	for _, filament := range filaments {
		responses = append(responses, ToFilamentResponse(filament))
	}

	c.JSON(http.StatusOK, gin.H{"filaments": responses})
}

func (uc *filamentUseCaseImpl) GetGlobalFilaments(c *gin.Context) {
	filaments, err := uc.filamentRepo.GetGlobal(c.Request.Context())
	if err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to get global filaments", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToGetGlobalFilaments))
		return
	}

	responses := make([]*FilamentResponse, 0, len(filaments))
	for _, filament := range filaments {
		responses = append(responses, ToFilamentResponse(filament))
	}

	c.JSON(http.StatusOK, gin.H{"filaments": responses})
}