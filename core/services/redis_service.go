package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
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
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", r.cfg.RedisHost, r.cfg.RedisPort),
		Password: r.cfg.RedisPassword,
		DB:       r.cfg.RedisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		appErr := errors.NewAppError(entities.ErrService, err.Error(), map[string]interface{}{
			"redis_host": config.EnvRedisHost(),
			"redis_port": config.EnvRedisPort(),
		}, err)
		r.logger.LogError(context.Background(), "Failed to connect to Redis", appErr)
		return appErr
	}

	r.client = rdb
	r.logger.Info(context.Background(), "Redis connected successfully", map[string]interface{}{
		"redis_host": config.EnvRedisHost(),
		"redis_port": config.EnvRedisPort(),
	})

	return nil
}

// GetClient returns the Redis client instance.
func (r *RedisService) GetClient() *redis.Client {
	return r.client
}

// Set stores a key-value pair with optional expiration.
func (r *RedisService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *errors.AppError {
	err := r.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		appErr := errors.NewAppError(entities.ErrService, err.Error(), map[string]interface{}{
			"key": key,
		}, err)
		r.logger.LogError(ctx, "Failed to set Redis key", appErr)
		return appErr
	}
	return nil
}

// Get retrieves a value by key.
func (r *RedisService) Get(ctx context.Context, key string) (string, *errors.AppError) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Key does not exist
	}
	if err != nil {
		appErr := errors.NewAppError(entities.ErrService, err.Error(), map[string]interface{}{
			"key": key,
		}, err)
		r.logger.LogError(ctx, "Failed to get Redis key", appErr)
		return "", appErr
	}
	return val, nil
}

// Delete removes a key from Redis.
func (r *RedisService) Delete(ctx context.Context, key string) *errors.AppError {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		appErr := errors.NewAppError(entities.ErrService, err.Error(), map[string]interface{}{
			"key": key,
		}, err)
		r.logger.LogError(ctx, "Failed to delete Redis key", appErr)
		return appErr
	}
	return nil
}

// Exists checks if a key exists in Redis.
func (r *RedisService) Exists(ctx context.Context, key string) (bool, *errors.AppError) {
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
