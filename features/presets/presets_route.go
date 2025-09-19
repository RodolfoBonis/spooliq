package presets

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/presets/domain/services"
	"github.com/RodolfoBonis/spooliq/features/presets/presentation/handlers"
	"github.com/gin-gonic/gin"
)

// GetEnergyLocationsHandler handles getting energy preset locations.
// @Summary Get energy locations
// @Schemes
// @Description Retrieves all available locations for energy presets
// @Tags Presets
// @Produce json
// @Success 200 {object} dto.EnergyLocationResponse "Successfully retrieved energy locations"
// @Failure 500 {object} errors.HTTPError
// @Router /presets/energy/locations [get]
func GetEnergyLocationsHandler(presetService services.PresetService) gin.HandlerFunc {
	handler := handlers.NewPresetHandler(presetService, nil)
	return handler.GetEnergyLocations
}

// GetMachinePresetsHandler handles getting machine presets.
// @Summary Get machine presets
// @Schemes
// @Description Retrieves all available machine presets
// @Tags Presets
// @Produce json
// @Success 200 {object} dto.MachinePresetsResponse "Successfully retrieved machine presets"
// @Failure 500 {object} errors.HTTPError
// @Router /presets/machines [get]
func GetMachinePresetsHandler(presetService services.PresetService) gin.HandlerFunc {
	handler := handlers.NewPresetHandler(presetService, nil)
	return handler.GetMachinePresets
}

// GetEnergyPresetsHandler handles getting energy presets.
// @Summary Get energy presets
// @Schemes
// @Description Retrieves energy presets, optionally filtered by location
// @Tags Presets
// @Produce json
// @Param location query string false "Filter by location"
// @Success 200 {object} dto.EnergyPresetsResponse "Successfully retrieved energy presets"
// @Failure 500 {object} errors.HTTPError
// @Router /presets/energy [get]
func GetEnergyPresetsHandler(presetService services.PresetService) gin.HandlerFunc {
	handler := handlers.NewPresetHandler(presetService, nil)
	return handler.GetEnergyPresets
}

// CreatePresetHandler handles creating a new preset.
// @Summary Create preset
// @Schemes
// @Description Creates a new energy or machine preset (admin only)
// @Tags Presets
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
// @Security Bearer
func CreatePresetHandler(presetService services.PresetService) gin.HandlerFunc {
	handler := handlers.NewPresetHandler(presetService, nil)
	return handler.CreatePreset
}

// UpdatePresetHandler handles updating an existing preset.
// @Summary Update preset
// @Schemes
// @Description Updates an existing preset by key (admin only)
// @Tags Presets
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
// @Security Bearer
func UpdatePresetHandler(presetService services.PresetService) gin.HandlerFunc {
	handler := handlers.NewPresetHandler(presetService, nil)
	return handler.UpdatePreset
}

// DeletePresetHandler handles deleting a preset.
// @Summary Delete preset
// @Schemes
// @Description Deletes a preset by key (admin only)
// @Tags Presets
// @Param key path string true "Preset key"
// @Success 204 "Preset deleted successfully"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /presets/{key} [delete]
// @Security Bearer
func DeletePresetHandler(presetService services.PresetService) gin.HandlerFunc {
	handler := handlers.NewPresetHandler(presetService, nil)
	return handler.DeletePreset
}

// Routes registers preset routes for the application.
func Routes(route *gin.RouterGroup, presetService services.PresetService, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	presets := route.Group("/presets")

	// Public routes
	presets.GET("/energy/locations", GetEnergyLocationsHandler(presetService))
	presets.GET("/machines", GetMachinePresetsHandler(presetService))
	presets.GET("/energy", GetEnergyPresetsHandler(presetService))

	// Admin-only routes
	presets.POST("", protectFactory(CreatePresetHandler(presetService), roles.AdminRole))
	presets.PUT("/:key", protectFactory(UpdatePresetHandler(presetService), roles.AdminRole))
	presets.DELETE("/:key", protectFactory(DeletePresetHandler(presetService), roles.AdminRole))
}
