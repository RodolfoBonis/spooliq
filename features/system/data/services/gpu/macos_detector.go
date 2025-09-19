package gpu

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/system/domain/entities"
	"github.com/shirou/gopsutil/v3/mem"
)

// MacOSDetector provides GPU detection for macOS systems.
type MacOSDetector struct {
	logger logger.Logger
}

// NewMacOSDetector creates a new MacOSDetector instance.
func NewMacOSDetector(logger logger.Logger) Detector {
	return &MacOSDetector{logger: logger}
}

// GetGPUInfo retrieves GPU information on macOS systems.
func (d *MacOSDetector) GetGPUInfo() (entities.GPU, *errors.AppError) {
	d.logger.Info(context.Background(), "Entrou no MacOSDetector.GetGPUInfo")
	cmd := exec.Command("system_profiler", "SPDisplaysDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		appErr := errors.NewAppError(coreEntities.ErrService, err.Error(), map[string]interface{}{"cmd": "system_profiler SPDisplaysDataType -json"}, err)
		d.logger.LogError(context.Background(), "Failed to execute system_profiler", appErr)
		return entities.GPU{}, appErr
	}

	var displays map[string]interface{}
	if err := json.Unmarshal(output, &displays); err != nil {
		appErr := errors.NewAppError(coreEntities.ErrService, err.Error(), map[string]interface{}{"cmd": "system_profiler output unmarshal", "raw": string(output)}, err)
		d.logger.LogError(context.Background(), "Failed to unmarshal system_profiler output", appErr)
		return entities.GPU{}, appErr
	}

	if items, ok := displays["SPDisplaysDataType"].([]interface{}); ok && len(items) > 0 {
		if gpu, ok := items[0].(map[string]interface{}); ok {
			model := "Unknown GPU"
			memory := "N/A"
			cores := 0

			if name, ok := gpu["sppci_model"].(string); ok {
				model = name
			} else if name, ok := gpu["_name"].(string); ok {
				model = name
			}

			if c, ok := gpu["sppci_cores"].(string); ok {
				_, _ = fmt.Sscanf(c, "%d", &cores)
			}

			if vm, err := mem.VirtualMemory(); err == nil {
				memory = fmt.Sprintf("%.2f GB (Shared)", float64(vm.Total)/(1024*1024*1024))
			}

			d.logger.Info(context.Background(), "GPU info detected", map[string]interface{}{"model": model, "memory": memory})
			return entities.GPU{
				Model:     model,
				Memory:    memory,
				Available: true,
				Cores:     cores,
			}, nil
		}
	}
	d.logger.Info(context.Background(), "No GPU found on MacOS after parsing")
	return entities.GPU{}, errors.NewAppError(coreEntities.ErrService, "No GPU found on MacOS", nil, nil)
}
