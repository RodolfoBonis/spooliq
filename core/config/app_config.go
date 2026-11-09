package config

import (
	"github.com/RodolfoBonis/spooliq/core/entities"
	"go.uber.org/fx"
)

// AppConfig holds the application configuration.
type AppConfig struct {
	Port               string
	Keycloak           entities.KeyCloakDataEntity
	ServiceID          string
	SentryDSN          string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	Environment        string
	ServiceName        string
	AmqpConnection     string
	RedisHost          string
	RedisPort          string
	RedisPassword      string
	RedisDB            int
	CDNBaseURL         string
	CDNKeys            entities.CdnKeysEntity
	AsaasAPIKey        string
	AsaasBaseURL       string
	AsaasWebhookSecret string

	// InfinitePay (per-company checkout links). Credentials are per-organization and
	// stored encrypted; only the API host and the global kill switch live here.
	InfinitePayBaseURL string
	InfinitePayEnabled bool
	// PublicAPIBaseURL / PublicWebBaseURL build the webhook and redirect URLs sent to
	// InfinitePay when a checkout link is created.
	PublicAPIBaseURL string
	PublicWebBaseURL string
	// PaymentsEncryptionKey is the AES key material for encrypting payment
	// credentials at rest (from Vault). Empty => payment endpoints return 503.
	PaymentsEncryptionKey string
}

// NewAppConfig creates and returns a new AppConfig instance.
func NewAppConfig() *AppConfig {
	// Load environment variables from .env file
	LoadEnvVars()

	return &AppConfig{
		Port:                  EnvPort(),
		Keycloak:              EnvKeyCloak(),
		ServiceID:             EnvServiceID(),
		SentryDSN:             EnvSentryDSN(),
		DBHost:                EnvDBHost(),
		DBPort:                EnvDBPort(),
		DBUser:                EnvDBUser(),
		DBPassword:            EnvDBPassword(),
		DBName:                EnvDBName(),
		Environment:           EnvironmentConfig(),
		ServiceName:           EnvServiceName(),
		AmqpConnection:        EnvAmqpConnection(),
		RedisHost:             EnvRedisHost(),
		RedisPort:             EnvRedisPort(),
		RedisPassword:         EnvRedisPassword(),
		RedisDB:               EnvRedisDB(),
		CDNBaseURL:            EnvCDNBaseURL(),
		CDNKeys:               EnvCDNKeys(),
		AsaasAPIKey:           EnvAsaasAPIKey(),
		AsaasBaseURL:          EnvAsaasBaseURL(),
		AsaasWebhookSecret:    EnvAsaasWebhookSecret(),
		InfinitePayBaseURL:    EnvInfinitePayBaseURL(),
		InfinitePayEnabled:    EnvInfinitePayEnabled(),
		PublicAPIBaseURL:      EnvPublicAPIBaseURL(),
		PublicWebBaseURL:      EnvPublicWebBaseURL(),
		PaymentsEncryptionKey: EnvPaymentsEncryptionKey(),
	}
}

// Module provides the fx module for AppConfig.
var Module = fx.Module("config", fx.Provide(NewAppConfig))
