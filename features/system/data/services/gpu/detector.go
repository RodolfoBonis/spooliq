package gpu

import (
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/system/domain/entities"
)

// Detector provides GPU detection capabilities.
type Detector interface {
	GetGPUInfo() (entities.GPU, *errors.AppError)
}
