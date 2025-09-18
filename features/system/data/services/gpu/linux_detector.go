package gpu

import (
	"context"
	"os/exec"
	"strings"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/system/domain/entities"
)

// LinuxDetector provides GPU detection for Linux systems.
type LinuxDetector struct {
	logger logger.Logger
}

// NewLinuxDetector creates a new LinuxDetector instance.
func NewLinuxDetector(logger logger.Logger) Detector {
	return &LinuxDetector{logger: logger}
}

// GetGPUInfo retrieves GPU information on Linux systems.
func (d *LinuxDetector) GetGPUInfo() (entities.GPU, *errors.AppError) {
	cmd := exec.Command("lspci", "-v")
	output, err := cmd.Output()
	if err != nil {
		appErr := errors.ServiceError(err.Error(), map[string]interface{}{"cmd": "lspci -v"})
		d.logger.LogError(context.Background(), "Failed to execute lspci", appErr)
		return entities.GPU{}, appErr
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "vga compatible controller") ||
			strings.Contains(strings.ToLower(line), "3d controller") {
			parts := strings.Split(line, ": ")
			if len(parts) > 1 {
				model := strings.TrimSpace(parts[1])

				memory := d.getLinuxGPUMemory(model)

				return entities.GPU{
					Model:     model,
					Memory:    memory,
					Available: true,
				}, nil
			}
		}
	}

	return entities.GPU{}, errors.ServiceError("No GPU found on Linux")
}

func (d *LinuxDetector) getLinuxGPUMemory(model string) string {
	if strings.Contains(strings.ToLower(model), "amd") {
		if cmd := exec.Command("rocm-smi", "--showmeminfo"); cmd != nil {
			output, err := cmd.Output()
			if err == nil {
				return d.parseAMDMemoryOutput(string(output))
			}
			appErr := errors.ServiceError(err.Error(), map[string]interface{}{"cmd": "rocm-smi --showmeminfo"})
			d.logger.LogError(context.Background(), "Failed to execute rocm-smi", appErr)
		}
	}

	return "Memory info unavailable"
}

func (d *LinuxDetector) parseAMDMemoryOutput(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Total") && strings.Contains(line, "MB") {
			return strings.TrimSpace(line)
		}
	}
	return "AMD GPU detected - Memory info unavailable"
}
