package usecases

import (
	"github.com/Nerzal/gocloak/v13"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/gin-gonic/gin"
)

// AuthUseCase define os casos de uso relacionados à autenticação.
type AuthUseCase interface {
	// ValidateLogin realiza a validação do login do usuário.
	ValidateLogin(c *gin.Context)
	// Logout realiza o logout do usuário.
	Logout(c *gin.Context)
	// RefreshAuthToken renova o token de autenticação do usuário.
	RefreshAuthToken(c *gin.Context)
	// ValidateToken valida o token de autenticação atual.
	ValidateToken(c *gin.Context)
	// RegisterUser creates a new user account.
	RegisterUser(c *gin.Context)
	// ForgotPassword handles password reset requests.
	ForgotPassword(c *gin.Context)
}

// authUseCaseImpl é a implementação de AuthUseCase.
type authUseCaseImpl struct {
	KeycloakClient     *gocloak.GoCloak
	KeycloakAccessData entities.KeyCloakDataEntity
	Logger             logger.Logger
}

// NewAuthUseCase cria uma nova instância de AuthUseCase.
func NewAuthUseCase(client *gocloak.GoCloak, keycloakData entities.KeyCloakDataEntity, logger logger.Logger) AuthUseCase {
	return &authUseCaseImpl{
		KeycloakClient:     client,
		KeycloakAccessData: keycloakData,
		Logger:             logger,
	}
}
