package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/RodolfoBonis/spooliq/core/entities"

	"github.com/joho/godotenv"
)

// GetEnv retrieves the value of the specified environment variable.
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)

	if value != "" {
		return value
	}

	return defaultValue
}

// EnvPort returns the port from environment variables.
func EnvPort() string {
	return GetEnv("PORT", "8000")
}

// EnvKeyCloak returns the Keycloak configuration from environment variables.
func EnvKeyCloak() entities.KeyCloakDataEntity {
	return entities.KeyCloakDataEntity{
		ClientID:      GetEnv("CLIENT_ID", ""),
		ClientSecret:  GetEnv("CLIENT_SECRET", ""),
		Realm:         GetEnv("REALM", ""),
		Host:          GetEnv("KEYCLOAK_HOST", ""),
		AdminUsername: GetEnv("KEYCLOAK_ADMIN_USERNAME", "admin"),
		AdminPassword: GetEnv("KEYCLOAK_ADMIN_PASSWORD", "admin123"),
	}
}

// EnvServiceID retrieves the service ID from the environment variables.
// If not set, tries to derive from the module name automatically.
func EnvServiceID() string {
	return GetEnv("SERVICE_ID", "")
}

// EnvSentryDSN returns the Sentry DSN from environment variables.
func EnvSentryDSN() string {
	return GetEnv("SENTRY_DSN", "")
}

// EnvDBHost returns the database host from environment variables.
func EnvDBHost() string {
	return GetEnv("DB_HOST", "localhost")
}

// EnvDBPort returns the database port from environment variables.
func EnvDBPort() string {
	return GetEnv("DB_PORT", "5432")
}

// EnvDBUser returns the database user from environment variables.
func EnvDBUser() string {
	return GetEnv("DB_USER", "user")
}

// EnvDBPassword returns the database password from environment variables.
func EnvDBPassword() string {
	return GetEnv("DB_SECRET", "password")
}

// EnvDBName returns the database name from environment variables.
func EnvDBName() string {
	return GetEnv("DB_NAME", "spooliq_db")
}

// EnvDBDriver returns the database driver from environment variables.
func EnvDBDriver() string {
	return GetEnv("DB_DRIVER", "postgres")
}

// EnvRedisHost returns the Redis host from environment variables.
func EnvRedisHost() string {
	return GetEnv("REDIS_HOST", "localhost")
}

// EnvRedisPort returns the Redis port from environment variables.
func EnvRedisPort() string {
	return GetEnv("REDIS_PORT", "6379")
}

// EnvRedisPassword returns the Redis password from environment variables.
func EnvRedisPassword() string {
	return GetEnv("REDIS_PASSWORD", "")
}

// EnvRedisDB returns the Redis database number from environment variables.
func EnvRedisDB() int {
	dbStr := GetEnv("REDIS_DB", "0")
	db, err := strconv.Atoi(dbStr)
	if err != nil {
		return 0
	}
	return db
}

// EnvironmentConfig returns the environment configuration.
func EnvironmentConfig() string {
	return GetEnv("ENV", "development")
}

// EnvServiceName returns the service name from environment variables.
func EnvServiceName() string {
	return GetEnv("SERVICE_NAME", "spooliq")
}

func envUserAmqp() string {
	return GetEnv("USER_AMQP", "guest")
}

func envPasswordAmqp() string {
	return GetEnv("PASSWORD_AMQP", "guest")
}

func envHostAmqp() string {
	return GetEnv("HOST_AMQP", "localhost:5672")
}

// EnvAmqpConnection returns the AMQP connection string from environment variables.
func EnvAmqpConnection() string {
	user := envUserAmqp()
	password := envPasswordAmqp()
	host := envHostAmqp()

	return fmt.Sprintf("amqp://%s:%s@%s/", user, password, host)
}

// EnvCDNBaseURL returns the public cdn edge that serves the bucket; persisted file URLs point here
// (e.g. https://assets.spooliq.com/<key>).
func EnvCDNBaseURL() string {
	return GetEnv("CDN_PUBLIC_BASE_URL", "https://assets.spooliq.com")
}

// EnvCDNKeys returns the MinIO connection for direct uploads (creds from Vault k3s/spooliq/minio).
func EnvCDNKeys() entities.CdnKeysEntity {
	useSSL, _ := strconv.ParseBool(GetEnv("MINIO_USE_SSL", "false"))
	return entities.CdnKeysEntity{
		Bucket:    GetEnv("MINIO_BUCKET", "spooliq"),
		Endpoint:  GetEnv("MINIO_SERVER", ""),
		AccessKey: GetEnv("MINIO_ACCESS_ID", ""),
		SecretKey: GetEnv("MINIO_SECRET_KEY", ""),
		UseSSL:    useSSL,
	}
}

// EnvAsaasAPIKey returns the Asaas API key from environment variables.
func EnvAsaasAPIKey() string {
	return GetEnv("ASAAS_API_KEY", "")
}

// EnvAsaasWebhookSecret returns the Asaas webhook secret from environment variables.
func EnvAsaasWebhookSecret() string {
	return GetEnv("ASAAS_WEBHOOK_SECRET", "")
}

// EnvAsaasBaseURL returns the Asaas base URL from environment variables.
func EnvAsaasBaseURL() string {
	return GetEnv("ASAAS_BASE_URL", "https://sandbox.asaas.com/api/v3")
}

// EnvCORSAllowedOrigins returns the explicit list of allowed CORS origins parsed
// from the comma-separated CORS_ALLOWED_ORIGINS environment variable.
//
// When unset (or empty), it returns an empty slice, which the CORS middleware
// treats as "allow all origins WITHOUT credentials". When a list is provided,
// the middleware restricts origins to that list and enables credentials. Entries
// are trimmed and blank entries are dropped. A single "*" entry is treated as
// "allow all" (empty slice) to avoid the invalid wildcard+credentials combo.
func EnvCORSAllowedOrigins() []string {
	raw := GetEnv("CORS_ALLOWED_ORIGINS", "")
	if raw == "" {
		return []string{}
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		if trimmed == "*" {
			// A bare wildcard means "allow all"; return empty so the middleware
			// uses AllowAllOrigins without credentials (wildcard + credentials is
			// rejected by the CORS spec and by gin-contrib/cors).
			return []string{}
		}
		origins = append(origins, trimmed)
	}
	return origins
}

// LoadEnvVars loads all environment variables required by the application.
func LoadEnvVars() {
	env := EnvironmentConfig()
	if env == entities.Environment.Production || env == entities.Environment.Staging {
		fmt.Printf("Not using .env file in production or staging")
		return
	}

	filename := fmt.Sprintf(".env.%s", env)

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		filename = ".env"
	}

	err := godotenv.Load(filename)

	if err != nil {
		fmt.Printf(".env file not loaded")
		os.Exit(1)
	}
}

// EnvDBSSLMode returns the database SSL mode from environment variables.
func EnvDBSSLMode() string {
	return GetEnv("DB_SSLMODE", "disable")
}

// EnvDBSSLRootCert returns the database SSL root certificate path from environment variables.
func EnvDBSSLRootCert() string {
	return GetEnv("DB_SSLROOTCERT", "")
}

// EnvRedisTLSEnabled returns whether Redis TLS is enabled from environment variables.
func EnvRedisTLSEnabled() bool {
	return GetEnv("REDIS_TLS_ENABLED", "false") == "true"
}

// EnvRedisTLSCA returns the Redis TLS CA certificate path from environment variables.
func EnvRedisTLSCA() string {
	return GetEnv("REDIS_TLS_CA", "")
}

// EnvRedisTLSCert returns the Redis TLS client certificate path from environment variables.
func EnvRedisTLSCert() string {
	return GetEnv("REDIS_TLS_CERT", "")
}

// EnvRedisTLSKey returns the Redis TLS client key path from environment variables.
func EnvRedisTLSKey() string {
	return GetEnv("REDIS_TLS_KEY", "")
}
