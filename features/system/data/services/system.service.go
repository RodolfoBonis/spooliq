package services

import (
	"fmt"
	"os"
	"strings"

	"github.com/RodolfoBonis/spooliq/core/config"
	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/system/data/services/gpu"
	"github.com/RodolfoBonis/spooliq/features/system/domain/entities"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// SystemService provides system-related services.
type SystemService interface {
	GetCPUInfo() (entities.CPU, *errors.AppError)
	GetMemoryInfo() (entities.Memory, *errors.AppError)
	GetStorageInfo() (entities.Storage, *errors.AppError)
	GetGPUInfo() (entities.GPU, *errors.AppError)
	GetHostInfo() (string, *errors.AppError)
	GetServerInfo() (entities.Server, *errors.AppError)
}

type systemService struct {
	gpuService gpu.Service
}

// NewSystemService creates a new SystemService instance.
func NewSystemService(gpuService gpu.Service) SystemService {
	return &systemService{
		gpuService: gpuService,
	}
}

func (s *systemService) GetCPUInfo() (entities.CPU, *errors.AppError) {
	infos, err := cpu.Info()

	if err != nil {
		return entities.CPU{}, errors.NewAppError(coreEntities.ErrService, err.Error(), nil, err)
	}

	if len(infos) == 0 {
		return entities.CPU{}, errors.NewAppError(coreEntities.ErrService, "no CPU information available", nil, nil)
	}

	cpuInfo := infos[0]

	percent, _ := cpu.Percent(0, false)
	return entities.CPU{
		Model: cpuInfo.ModelName,
		Cores: cpuInfo.Cores,
		Usage: fmt.Sprintf("%.2f%%", percent[0]),
	}, nil
}

func (s *systemService) GetMemoryInfo() (entities.Memory, *errors.AppError) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return entities.Memory{}, errors.NewAppError(coreEntities.ErrService, err.Error(), nil, err)
	}

	return entities.Memory{
		Total:      fmt.Sprintf("%.2f GB", float64(memInfo.Total)/(1024*1024*1024)),
		Available:  fmt.Sprintf("%.2f GB", float64(memInfo.Available)/(1024*1024*1024)),
		Used:       fmt.Sprintf("%.2f GB", float64(memInfo.Used)/(1024*1024*1024)),
		Percentage: fmt.Sprintf("%.2f%%", memInfo.UsedPercent),
	}, nil
}

func (s *systemService) GetStorageInfo() (entities.Storage, *errors.AppError) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return entities.Storage{}, errors.NewAppError(coreEntities.ErrService, err.Error(), nil, err)
	}

	var totalUsed, totalTotal uint64
	partition := partitions[0]
	usage, _ := disk.Usage(partition.Mountpoint)

	totalUsed += usage.Used
	totalTotal += usage.Total

	var usagePercentage float64
	if totalTotal > 0 {
		usagePercentage = float64(totalUsed) / float64(totalTotal) * 100
	}

	return entities.Storage{
		Used:       fmt.Sprintf("%v GB", totalUsed/(1024*1024*1024)),
		Total:      fmt.Sprintf("%v GB", totalTotal/(1024*1024*1024)),
		Percentage: fmt.Sprintf("%.2f%%", usagePercentage),
	}, nil
}

func (s *systemService) GetHostInfo() (string, *errors.AppError) {
	info, err := host.Info()

	if err != nil {
		return "", errors.NewAppError(coreEntities.ErrService, err.Error(), nil, err)
	}

	return fmt.Sprintf("Platform: %s %s (%s)",
		info.Platform,
		info.PlatformVersion,
		info.PlatformFamily,
	), nil
}

func (s *systemService) GetGPUInfo() (entities.GPU, *errors.AppError) {
	gpuInfo, err := s.gpuService.GetGPUInfo()
	if err == nil {
		return gpuInfo, nil
	}
	return entities.GPU{
		Model:     "No dedicated GPU detected",
		Memory:    "N/A",
		Available: false,
	}, nil
}

func (s *systemService) GetServerInfo() (entities.Server, *errors.AppError) {
	versionFileName := "version.txt"
	if config.EnvironmentConfig() == coreEntities.Environment.Production {
		versionFileName = "/version.txt"
	}
	version := "unknown"
	if content, err := os.ReadFile(versionFileName); err == nil {
		version = strings.TrimSpace(string(content))
	}
	return entities.Server{
		Version: version,
		Active:  true,
	}, nil
}
