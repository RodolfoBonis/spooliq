package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/presets/domain/services"
	"github.com/RodolfoBonis/spooliq/features/presets/presentation/dto"
)

type PresetHandler struct {
	presetService services.PresetService
	logger        logger.Logger
	validator     *validator.Validate
}

// NewPresetHandler creates a new preset handler
func NewPresetHandler(
	presetService services.PresetService,
	logger logger.Logger,
) *PresetHandler {
	return &PresetHandler{
		presetService: presetService,
		logger:        logger,
		validator:     validator.New(),
	}
}

// GetEnergyLocations retrieves all available energy preset locations
// @Summary Get energy locations
// @Description Retrieves all available locations for energy presets
// @Tags presets
// @Produce json
// @Success 200 {object} dto.EnergyLocationResponse
// @Failure 500 {object} errors.HTTPError
// @Router /presets/energy/locations [get]
func (h *PresetHandler) GetEnergyLocations(c *gin.Context) {
	locations, err := h.presetService.GetEnergyLocations(c.Request.Context())
	if err != nil {
		appError := errors.NewAppError(coreEntities.ErrService, "Failed to get energy locations", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to get energy locations", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	response := dto.EnergyLocationResponse{
		Locations: locations,
	}

	c.JSON(http.StatusOK, response)
}

// GetMachinePresets retrieves all machine presets
// @Summary Get machine presets
// @Description Retrieves all available machine presets
// @Tags presets
// @Produce json
// @Success 200 {object} dto.MachinePresetsResponse
// @Failure 500 {object} errors.HTTPError
// @Router /presets/machines [get]
func (h *PresetHandler) GetMachinePresets(c *gin.Context) {
	machines, err := h.presetService.GetMachinePresets(c.Request.Context())
	if err != nil {
		appError := errors.NewAppError(coreEntities.ErrService, "Failed to get machine presets", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to get machine presets", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	response := dto.MachinePresetsResponse{
		Machines: dto.FromMachinePresetEntities(machines),
	}

	c.JSON(http.StatusOK, response)
}

// GetEnergyPresets retrieves energy presets, optionally filtered by location
// @Summary Get energy presets
// @Description Retrieves energy presets, optionally filtered by location
// @Tags presets
// @Produce json
// @Param location query string false "Filter by location"
// @Success 200 {object} dto.EnergyPresetsResponse
// @Failure 500 {object} errors.HTTPError
// @Router /presets/energy [get]
func (h *PresetHandler) GetEnergyPresets(c *gin.Context) {
	location := c.Query("location")

	presets, err := h.presetService.GetEnergyPresets(c.Request.Context(), location)
	if err != nil {
		appError := errors.NewAppError(coreEntities.ErrService, "Failed to get energy presets", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to get energy presets", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	response := dto.EnergyPresetsResponse{
		Presets: dto.FromEnergyPresetEntities(presets),
	}

	c.JSON(http.StatusOK, response)
}

// CreatePreset creates a new preset (admin only)
// @Summary Create preset
// @Description Creates a new energy or machine preset (admin only)
// @Tags presets
// @Accept json
// @Produce json
// @Param type query string true "Preset type: 'energy' or 'machine'"
// @Param request body interface{} true "Preset data (CreateEnergyPresetRequest or CreateMachinePresetRequest)"
// @Success 201 "Preset created successfully"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /presets [post]
// @Security BearerAuth
func (h *PresetHandler) CreatePreset(c *gin.Context) {
	presetType := c.Query("type")
	if presetType == "" {
		appError := errors.NewAppError(coreEntities.ErrEntity, "Preset type is required (energy or machine)", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Preset type not specified", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := errors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	switch presetType {
	case "energy":
		h.createEnergyPreset(c, requesterID)
	case "machine":
		h.createMachinePreset(c, requesterID)
	default:
		appError := errors.NewAppError(coreEntities.ErrEntity, "Invalid preset type. Must be 'energy' or 'machine'", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Invalid preset type", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}
}

func (h *PresetHandler) createEnergyPreset(c *gin.Context, requesterID string) {
	// Bind request
	var req dto.CreateEnergyPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := errors.NewAppError(coreEntities.ErrEntity, "Invalid request format", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to bind energy preset request", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		appError := errors.NewAppError(coreEntities.ErrEntity, "Validation failed", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Energy preset request validation failed", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert to domain entity
	energyPreset := req.ToEntity()

	// Create preset
	err := h.presetService.CreateEnergyPreset(c.Request.Context(), energyPreset, requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to create energy preset", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	h.logger.Info(c.Request.Context(), "Energy preset created successfully", map[string]interface{}{
		"location":     energyPreset.Location,
		"year":         energyPreset.Year,
		"requester_id": requesterID,
	})

	c.Status(http.StatusCreated)
}

func (h *PresetHandler) createMachinePreset(c *gin.Context, requesterID string) {
	// Bind request
	var req dto.CreateMachinePresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := errors.NewAppError(coreEntities.ErrEntity, "Invalid request format", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to bind machine preset request", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		appError := errors.NewAppError(coreEntities.ErrEntity, "Validation failed", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Machine preset request validation failed", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert to domain entity
	machinePreset := req.ToEntity()

	// Create preset
	err := h.presetService.CreateMachinePreset(c.Request.Context(), machinePreset, requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to create machine preset", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	h.logger.Info(c.Request.Context(), "Machine preset created successfully", map[string]interface{}{
		"name":         machinePreset.Name,
		"brand":        machinePreset.Brand,
		"model":        machinePreset.Model,
		"requester_id": requesterID,
	})

	c.Status(http.StatusCreated)
}

// UpdatePreset updates an existing preset (admin only)
// @Summary Update preset
// @Description Updates an existing preset by key (admin only)
// @Tags presets
// @Accept json
// @Produce json
// @Param key path string true "Preset key"
// @Param request body dto.UpdatePresetRequest true "Updated preset data"
// @Success 200 "Preset updated successfully"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /presets/{key} [put]
// @Security BearerAuth
func (h *PresetHandler) UpdatePreset(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		appError := errors.NewAppError(coreEntities.ErrEntity, "Preset key is required", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Preset key not provided", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := errors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Bind request
	var req dto.UpdatePresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := errors.NewAppError(coreEntities.ErrEntity, "Invalid request format", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to bind update preset request", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Update preset
	err := h.presetService.UpdatePreset(c.Request.Context(), key, req.Data, requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to update preset", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	c.Status(http.StatusOK)
}

// DeletePreset deletes a preset (admin only)
// @Summary Delete preset
// @Description Deletes a preset by key (admin only)
// @Tags presets
// @Param key path string true "Preset key"
// @Success 204 "Preset deleted successfully"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /presets/{key} [delete]
// @Security BearerAuth
func (h *PresetHandler) DeletePreset(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		appError := errors.NewAppError(coreEntities.ErrEntity, "Preset key is required", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Preset key not provided", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := errors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Delete preset
	err := h.presetService.DeletePreset(c.Request.Context(), key, requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to delete preset", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	c.Status(http.StatusNoContent)
}

// Helper methods

func (h *PresetHandler) mapDomainError(err error) *errors.AppError {
	errMsg := err.Error()

	switch {
	case contains(errMsg, "not found"):
		return errors.NewAppError(coreEntities.ErrNotFound, "Preset not found", nil, err)
	case contains(errMsg, "already exists"):
		return errors.NewAppError(coreEntities.ErrConflict, "Preset already exists", nil, err)
	case contains(errMsg, "admin permissions required"):
		return errors.NewAppError(coreEntities.ErrUnauthorized, "Admin permissions required", nil, err)
	case contains(errMsg, "validation failed"):
		return errors.NewAppError(coreEntities.ErrEntity, "Validation failed", nil, err)
	default:
		return errors.NewAppError(coreEntities.ErrService, "Internal server error", nil, err)
	}
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || (len(str) > len(substr) && (str[:len(substr)] == substr || str[len(str)-len(substr):] == substr || contains(str[1:], substr))))
}