package services

import (
	"github.com/Nerzal/gocloak/v13"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/logger"
)

// AuthService provides authentication capabilities.
type AuthService struct {
	client *gocloak.GoCloak
	logger logger.Logger
	cfg    *config.AppConfig
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(logger logger.Logger, cfg *config.AppConfig) *AuthService {
	client := gocloak.NewClient(cfg.Keycloak.Host)

	// Get the internal Resty client and configure it with instrumented HTTP transport
	restyClient := client.RestyClient()
	instrumentedHTTPClient := NewInstrumentedHTTPClient()
	restyClient.SetTransport(instrumentedHTTPClient.Transport)

	return &AuthService{
		client: client,
		logger: logger,
		cfg:    cfg,
	}
}

// GetClient returns the Keycloak client instance.
func (s *AuthService) GetClient() *gocloak.GoCloak {
	return s.client
}

// GetConfig returns the application configuration.
func (s *AuthService) GetConfig() *config.AppConfig {
	return s.cfg
}
