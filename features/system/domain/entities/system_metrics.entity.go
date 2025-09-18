package entities

import (
	"time"
)

// SystemMetrics represents the system metrics entity.
type SystemMetrics struct {
	Timestamp     time.Time
	CPUPercent    float64
	MemoryPercent float64
	GPUMetrics    string
}
