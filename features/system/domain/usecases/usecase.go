package usecases

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/system/data/services"
	"github.com/gin-gonic/gin"
)

// SystemUseCase define os casos de uso do domínio de sistema.
type SystemUseCase interface {
	// GetSystemStatus retorna o status do sistema.
	GetSystemStatus(c *gin.Context)
}

// systemUseCaseImpl é a implementação de SystemUseCase.
type systemUseCaseImpl struct {
	Service services.SystemService
	Logger  logger.Logger
}

// NewSystemUseCase cria uma nova instância de SystemUseCase.
func NewSystemUseCase(service services.SystemService, logger logger.Logger) SystemUseCase {
	return &systemUseCaseImpl{
		Service: service,
		Logger:  logger,
	}
}
