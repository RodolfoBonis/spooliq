package system

import (
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/features/system/domain/usecases"
	"github.com/gin-gonic/gin"
)

// GetSystemStatusHandler returns the current system status, including OS, CPU, memory, GPU, storage, and server info.
// @Summary Get System Status
// @Schemes
// @Description Returns the current system status (OS, CPU, memory, GPU, storage, server)
// @Tags System
// @Accept json
// @Produce json
// @Success 200 {object} entities.SystemStatus "System status info"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /system [get]
//
//	@Example response {
//	  "OS": "Darwin",
//	  "CPU": {"Model": "Intel(R) Core(TM) i7", "Cores": 8, "Threads": 16, "Usage": "15%"},
//	  "Memory": {"Total": "16GB", "Available": "8GB", "Used": "8GB", "Percentage": "50%"},
//	  "GPU": {"Model": "AMD Radeon Pro", "Memory": "4GB", "Available": true},
//	  "Storage": {"Used": "200GB", "Total": "500GB", "Percentage": "40%"},
//	  "Server": {"Version": "1.0.0", "Active": true}
//	}
func GetSystemStatusHandler(systemUc usecases.SystemUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		systemUc.GetSystemStatus(c)
	}
}

// Routes registers system-related routes for the application.
func Routes(route *gin.RouterGroup, systemUc usecases.SystemUseCase, cacheMiddleware *middlewares.CacheMiddleware) {
	systemRoute := route.Group("/system")

	// Endpoint sem cache (sempre busca dados frescos)
	systemRoute.GET("", GetSystemStatusHandler(systemUc))
}
