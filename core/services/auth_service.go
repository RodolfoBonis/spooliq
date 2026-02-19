package services

import (
	"net/http"

	"github.com/Nerzal/gocloak/v13"
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

	// Wrap Resty transport with OpenTelemetry instrumentation
	restyClient := client.RestyClient()
	restyClient.SetTransport(otelhttp.NewTransport(http.DefaultTransport))

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
