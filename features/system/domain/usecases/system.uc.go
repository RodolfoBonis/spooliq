package usecases

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/system/domain/entities"
	"github.com/gin-gonic/gin"
)

// GetSystemStatus returns the current system status, including OS, CPU, memory, GPU, storage, and server info.
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
// @Router /system/ [get]
// @Example response {
//   "OS": "Darwin",
//   "CPU": {"Model": "Intel(R) Core(TM) i7", "Cores": 8, "Threads": 16, "Usage": "15%"},
//   "Memory": {"Total": "16GB", "Available": "8GB", "Used": "8GB", "Percentage": "50%"},
//   "GPU": {"Model": "AMD Radeon Pro", "Memory": "4GB", "Available": true},
//   "Storage": {"Used": "200GB", "Total": "500GB", "Percentage": "40%"},
//   "Server": {"Version": "1.0.0", "Active": true}
// }

func (uc *systemUseCaseImpl) GetSystemStatus(c *gin.Context) {
	ctx := c.Request.Context()
	uc.Logger.Info(ctx, "System status requested", logger.Fields{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})
	systemStatus := entities.SystemStatus{}

	cpu, appError := uc.Service.GetCPUInfo()
	if appError != nil {
		httpError := appError.ToHTTPError()
		uc.Logger.LogError(ctx, "Failed to get CPU info", appError)
		c.JSON(httpError.StatusCode, httpError.ToMap())
		return
	}

	memory, appError := uc.Service.GetMemoryInfo()
	if appError != nil {
		httpError := appError.ToHTTPError()
		uc.Logger.LogError(ctx, "Failed to get memory info", appError)
		c.JSON(httpError.StatusCode, httpError.ToMap())
		return
	}

	storage, appError := uc.Service.GetStorageInfo()
	if appError != nil {
		httpError := appError.ToHTTPError()
		uc.Logger.LogError(ctx, "Failed to get storage info", appError)
		c.JSON(httpError.StatusCode, httpError.ToMap())
		return
	}

	hostInfo, appError := uc.Service.GetHostInfo()
	if appError != nil {
		httpError := appError.ToHTTPError()
		uc.Logger.LogError(ctx, "Failed to get host info", appError)
		c.JSON(httpError.StatusCode, httpError.ToMap())
		return
	}

	gpuInfo, appError := uc.Service.GetGPUInfo()
	if appError != nil {
		httpError := appError.ToHTTPError()
		uc.Logger.LogError(ctx, "Failed to get GPU info", appError)
		c.JSON(httpError.StatusCode, httpError.ToMap())
		return
	}

	serverInfo, appError := uc.Service.GetServerInfo()
	if appError != nil {
		httpError := appError.ToHTTPError()
		uc.Logger.LogError(ctx, "Failed to get server info", appError)
		c.JSON(httpError.StatusCode, httpError.ToMap())
		return
	}

	systemStatus.OS = hostInfo
	systemStatus.CPU = cpu
	systemStatus.Memory = memory
	systemStatus.Storage = storage
	systemStatus.GPU = gpuInfo
	systemStatus.Server = serverInfo

	uc.Logger.Info(ctx, "System status returned", logger.Fields{
		"ip": c.ClientIP(),
	})
	c.JSON(200, systemStatus)
}
