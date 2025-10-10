package config

import (
	"github.com/RodolfoBonis/spooliq/core/entities"
	"go.uber.org/fx"
)

// AppConfig holds the application configuration.
type AppConfig struct {
	Port           string
	Keycloak       entities.KeyCloakDataEntity
	ServiceID      string
	SentryDSN      string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	Environment    string
	ServiceName    string
	AmqpConnection string
	RedisHost      string
	RedisPort      string
	RedisPassword  string
	RedisDB        int
	CDNBaseURL          string
	CDNAPIKey           string
	CDNBucket           string
	AsaasAPIKey         string
	AsaasBaseURL        string
	AsaasWebhookSecret  string
}

// NewAppConfig creates and returns a new AppConfig instance.
func NewAppConfig() *AppConfig {
	// Load environment variables from .env file
	LoadEnvVars()

	return &AppConfig{
		Port:           EnvPort(),
		Keycloak:       EnvKeyCloak(),
		ServiceID:      EnvServiceID(),
		SentryDSN:      EnvSentryDSN(),
		DBHost:         EnvDBHost(),
		DBPort:         EnvDBPort(),
		DBUser:         EnvDBUser(),
		DBPassword:     EnvDBPassword(),
		DBName:         EnvDBName(),
		Environment:    EnvironmentConfig(),
		ServiceName:    EnvServiceName(),
		AmqpConnection: EnvAmqpConnection(),
		RedisHost:      EnvRedisHost(),
		RedisPort:      EnvRedisPort(),
		RedisPassword:  EnvRedisPassword(),
		RedisDB:        EnvRedisDB(),
		CDNBaseURL:         EnvCDNBaseURL(),
		CDNAPIKey:          EnvCDNAPIKey(),
		CDNBucket:          EnvCDNBucket(),
		AsaasAPIKey:        EnvAsaasAPIKey(),
		AsaasBaseURL:       EnvAsaasBaseURL(),
		AsaasWebhookSecret: EnvAsaasWebhookSecret(),
	}
}

// Module provides the fx module for AppConfig.
var Module = fx.Module("config", fx.Provide(NewAppConfig))
