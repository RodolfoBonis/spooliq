package di

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	presetRepositories "github.com/RodolfoBonis/spooliq/features/presets/data/repositories"
	presetServices "github.com/RodolfoBonis/spooliq/features/presets/data/services"
	"github.com/RodolfoBonis/spooliq/features/presets/domain/repositories"
	domainServices "github.com/RodolfoBonis/spooliq/features/presets/domain/services"
	"github.com/RodolfoBonis/spooliq/features/presets/presentation/handlers"
)

// Module provides the presets feature module
var Module = fx.Module("presets",
	fx.Provide(
		// Repositories
		fx.Annotate(
			presetRepositories.NewPresetRepository,
			fx.As(new(repositories.PresetRepository)),
		),

		// Services
		fx.Annotate(
			presetServices.NewPresetService,
			fx.As(new(domainServices.PresetService)),
		),

		// Handlers
		handlers.NewPresetHandler,
	),
	fx.Invoke(RegisterPresetRoutes),
)

// RegisterPresetRoutes registers the preset routes
func RegisterPresetRoutes(r *gin.Engine, handler *handlers.PresetHandler) {
	v1 := r.Group("/v1")
	{
		// Preset endpoints
		presets := v1.Group("/presets")
		{
			// Public endpoints for reading presets
			presets.GET("/energy/locations", handler.GetEnergyLocations) // GET /v1/presets/energy/locations
			presets.GET("/machines", handler.GetMachinePresets)          // GET /v1/presets/machines
			presets.GET("/energy", handler.GetEnergyPresets)             // GET /v1/presets/energy

			// Admin endpoints for managing presets
			presets.POST("", handler.CreatePreset)        // POST /v1/presets (admin)
			presets.PUT("/:key", handler.UpdatePreset)    // PUT /v1/presets/{key} (admin)
			presets.DELETE("/:key", handler.DeletePreset) // DELETE /v1/presets/{key} (admin)
		}
	}
}
