package gpu

import (
	"context"
	"fmt"

	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/system/domain/entities"
	"github.com/mindprince/gonvml"
)

// NvidiaDetector provides GPU detection for NVIDIA GPUs.
type NvidiaDetector struct {
	logger logger.Logger
}

// NewNvidiaDetector creates a new NvidiaDetector instance.
func NewNvidiaDetector(logger logger.Logger) Detector {
	return &NvidiaDetector{logger: logger}
}

// GetGPUInfo retrieves GPU information for NVIDIA GPUs.
func (d *NvidiaDetector) GetGPUInfo() (entities.GPU, *errors.AppError) {
	if err := gonvml.Initialize(); err != nil {
		appErr := errors.NewAppError(coreEntities.ErrService, err.Error(), map[string]interface{}{"step": "NVML Initialize"}, err)
		d.logger.LogError(context.Background(), "Failed to initialize NVML", appErr)
		return entities.GPU{}, appErr
	}
	defer func() {
		err := gonvml.Shutdown()
		if err != nil {
			appErr := errors.NewAppError(coreEntities.ErrService, err.Error(), map[string]interface{}{"step": "NVML Shutdown"}, err)
			d.logger.LogError(context.Background(), "Error shutting down NVML", appErr)
		}
	}()

	count, err := gonvml.DeviceCount()
	if err != nil || count == 0 {
		appErr := errors.NewAppError(coreEntities.ErrService, "Nenhuma GPU NVIDIA encontrada ou erro ao obter contagem", map[string]interface{}{"error": err}, err)
		d.logger.LogError(context.Background(), "Failed to get NVIDIA device count", appErr)
		return entities.GPU{}, appErr
	}

	device, err := gonvml.DeviceHandleByIndex(0)
	if err != nil {
		appErr := errors.NewAppError(coreEntities.ErrService, err.Error(), map[string]interface{}{"step": "DeviceHandleByIndex"}, err)
		d.logger.LogError(context.Background(), "Failed to get NVIDIA device handle", appErr)
		return entities.GPU{}, appErr
	}

	name, err := device.Name()
	if err != nil {
		name = "GPU NVIDIA (Modelo Desconhecido)"
	}

	total, used, err := device.MemoryInfo()
	var memoryStr string
	if err != nil {
		memoryStr = "Informações de memória indisponíveis"
		appErr := errors.NewAppError(coreEntities.ErrService, err.Error(), map[string]interface{}{"step": "MemoryInfo"}, err)
		d.logger.LogError(context.Background(), "Failed to get NVIDIA memory info", appErr)
	} else {
		totalMemoryGB := float64(total) / (1024 * 1024 * 1024)
		usedMemoryGB := float64(used) / (1024 * 1024 * 1024)
		memoryStr = fmt.Sprintf("%.2f GB / %.2f GB", usedMemoryGB, totalMemoryGB)
	}

	return entities.GPU{
		Model:     name,
		Memory:    memoryStr,
		Available: true,
	}, nil
}
