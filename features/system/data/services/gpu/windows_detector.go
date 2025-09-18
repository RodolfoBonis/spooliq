package gpu

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/system/domain/entities"
)

// WindowsDetector provides GPU detection for Windows systems.
type WindowsDetector struct {
	logger logger.Logger
}

// NewWindowsDetector creates a new WindowsDetector instance.
func NewWindowsDetector(logger logger.Logger) Detector {
	return &WindowsDetector{logger: logger}
}

// GetGPUInfo retrieves GPU information on Windows systems.
func (d *WindowsDetector) GetGPUInfo() (entities.GPU, *errors.AppError) {
	cmd := exec.Command("wmic", "path", "win32_VideoController", "get", "name,AdapterRAM", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		appErr := errors.NewAppError(coreEntities.ErrService, "Failed to execute wmic", map[string]interface{}{"cmd": "wmic path win32_VideoController get name,AdapterRAM /format:csv"}, err)
		d.logger.LogError(context.Background(), "Failed to execute wmic", appErr)
		return entities.GPU{}, appErr
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ",") && !strings.Contains(line, "Node") {
			parts := strings.Split(line, ",")
			if len(parts) >= 3 {
				name := strings.TrimSpace(parts[2])
				ramStr := strings.TrimSpace(parts[1])

				if name != "" && name != "Name" {
					memory := "N/A"
					if ramStr != "" && ramStr != "AdapterRAM" {
						if ram, err := strconv.ParseInt(ramStr, 10, 64); err == nil && ram > 0 {
							memory = fmt.Sprintf("%.2f GB", float64(ram)/(1024*1024*1024))
						} else if err != nil {
							appErr := errors.NewAppError(coreEntities.ErrService, err.Error(), map[string]interface{}{"ramStr": ramStr}, err)
							d.logger.LogError(context.Background(), "Failed to parse AdapterRAM", appErr)
						}
					}

					return entities.GPU{
						Model:     name,
						Memory:    memory,
						Available: true,
					}, nil
				}
			}
		}
	}

	return entities.GPU{}, errors.NewAppError(coreEntities.ErrService, "No GPU found on Windows", nil, nil)
}
