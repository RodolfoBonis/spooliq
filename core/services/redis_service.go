package services

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// RedisService provides Redis caching capabilities.
type RedisService struct {
	client *redis.Client
	logger logger.Logger
	cfg    *config.AppConfig
}

// NewRedisService creates a new RedisService instance.
func NewRedisService(logger logger.Logger, cfg *config.AppConfig) *RedisService {
	return &RedisService{
		logger: logger,
		cfg:    cfg,
	}
}

// Init initializes the Redis connection.
func (r *RedisService) Init() *errors.AppError {
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", r.cfg.RedisHost, r.cfg.RedisPort),
		Password: r.cfg.RedisPassword,
		DB:       r.cfg.RedisDB,
	}

	// Configure TLS if enabled
	if config.EnvRedisTLSEnabled() {
		tlsCA := config.EnvRedisTLSCA()

		if tlsCA != "" {
			caCert, err := os.ReadFile(tlsCA)
			if err != nil {
				appErr := errors.NewAppError(entities.ErrService, fmt.Sprintf("failed to read CA cert: %v", err), map[string]interface{}{
					"redis_tls_ca": tlsCA,
				}, err)
				r.logger.LogError(context.Background(), "Failed to load Redis TLS CA certificate", appErr)
				return appErr
			}

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)

			opts.TLSConfig = &tls.Config{
				RootCAs: caCertPool,
			}
		}
	}

	rdb := redis.NewClient(opts)

	// Add OpenTelemetry instrumentation for Redis
	if err := redisotel.InstrumentTracing(rdb); err != nil {
		r.logger.Warning(context.Background(), "Failed to instrument Redis tracing", map[string]interface{}{
			"error": err.Error(),
		})
	}

	if err := redisotel.InstrumentMetrics(rdb); err != nil {
		r.logger.Warning(context.Background(), "Failed to instrument Redis metrics", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Assign the client before pinging. A *redis.Client is safe to use
	// immediately and reconnects lazily on each command, so a transient failure
	// at boot must not leave the cache permanently disabled until a pod restart.
	// Ping is only a startup diagnostic; the cache layer already fails open when
	// commands error (see cache_middleware.go).
	r.client = rdb

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		appErr := errors.NewAppError(entities.ErrService, err.Error(), map[string]interface{}{
			"redis_host": config.EnvRedisHost(),
			"redis_port": config.EnvRedisPort(),
		}, err)
		r.logger.LogError(context.Background(), "Failed to connect to Redis at startup; cache will retry lazily", appErr)
		return appErr
	}

	r.logger.Info(context.Background(), "Redis connected successfully with OpenTelemetry instrumentation", map[string]interface{}{
		"redis_host": config.EnvRedisHost(),
		"redis_port": config.EnvRedisPort(),
	})

	return nil
}

// GetClient returns the Redis client instance.
func (r *RedisService) GetClient() *redis.Client {
	return r.client
}

// available reports whether the Redis client is initialized and usable.
//
// When Redis fails to initialize (see Init), the client stays nil. Instead of
// dereferencing it and panicking, the cache layer degrades gracefully
// (fail-open): reads behave as a cache miss and writes become no-ops, so
// business endpoints keep serving even while the cache is down. Init already
// logs the connection failure loudly at boot, so here we only emit a debug line
// to avoid per-request log spam.
func (r *RedisService) available(ctx context.Context) bool {
	if r.client == nil {
		r.logger.Debug(ctx, "Redis client unavailable; bypassing cache")
		return false
	}
	return true
}

// Set stores a key-value pair with optional expiration.
func (r *RedisService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *errors.AppError {
	if !r.available(ctx) {
		return nil // fail-open: cache unavailable, skip write
	}
	// Automatic instrumentation is now handled by the observability system
	err := r.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		r.logger.Error(ctx, "Failed to set Redis key", map[string]interface{}{
			"key":        key,
			"expiration": expiration.String(),
			"error":      err.Error(),
		})

		appErr := errors.NewAppError(entities.ErrService, err.Error(), map[string]interface{}{
			"key": key,
		}, err)
		return appErr
	}
	return nil
}

// Get retrieves a value by key.
func (r *RedisService) Get(ctx context.Context, key string) (string, *errors.AppError) {
	if !r.available(ctx) {
		return "", nil // fail-open: cache unavailable, behave as a miss
	}
	// Automatic instrumentation is now handled by the observability system
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Key does not exist
	}
	if err != nil {
		r.logger.Error(ctx, "Failed to get Redis key", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})

		appErr := errors.NewAppError(entities.ErrService, err.Error(), map[string]interface{}{
			"key": key,
		}, err)
		return "", appErr
	}

	return val, nil
}

// Delete removes a key from Redis.
func (r *RedisService) Delete(ctx context.Context, key string) *errors.AppError {
	if !r.available(ctx) {
		return nil // fail-open: cache unavailable, nothing to delete
	}
	// Automatic instrumentation is now handled by the observability system
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		r.logger.Error(ctx, "Failed to delete Redis key", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})

		appErr := errors.NewAppError(entities.ErrService, err.Error(), map[string]interface{}{
			"key": key,
		}, err)
		return appErr
	}
	return nil
}

// Exists checks if a key exists in Redis.
func (r *RedisService) Exists(ctx context.Context, key string) (bool, *errors.AppError) {
	if !r.available(ctx) {
		return false, nil // fail-open: cache unavailable, key considered absent
	}
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		appErr := errors.NewAppError(entities.ErrService, err.Error(), map[string]interface{}{
			"key": key,
		}, err)
		r.logger.LogError(ctx, "Failed to check Redis key existence", appErr)
		return false, appErr
	}
	return count > 0, nil
}

// SetWithJSON stores a JSON object with optional expiration.
func (r *RedisService) SetWithJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) *errors.AppError {
	jsonData, err := json.Marshal(value)
	if err != nil {
		appErr := errors.NewAppError(entities.ErrService, err.Error(), map[string]interface{}{
			"key": key,
		}, err)
		r.logger.LogError(ctx, "Failed to marshal JSON for Redis", appErr)
		return appErr
	}

	return r.Set(ctx, key, jsonData, expiration)
}

// GetWithJSON retrieves and unmarshals a JSON object.
func (r *RedisService) GetWithJSON(ctx context.Context, key string, dest interface{}) *errors.AppError {
	val, appErr := r.Get(ctx, key)
	if appErr != nil {
		return appErr
	}
	if val == "" {
		return nil // Key does not exist
	}

	err := json.Unmarshal([]byte(val), dest)
	if err != nil {
		appErr := errors.NewAppError(entities.ErrService, err.Error(), map[string]interface{}{
			"key": key,
		}, err)
		r.logger.LogError(ctx, "Failed to unmarshal JSON from Redis", appErr)
		return appErr
	}
	return nil
}

// Close closes the Redis connection.
func (r *RedisService) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}
