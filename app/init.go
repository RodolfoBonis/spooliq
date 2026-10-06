package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	appErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/docs"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// InitAndRun initializes and runs the application using Fx lifecycle
func InitAndRun() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle, cfg *config.AppConfig, amqpService *services.AmqpService, app *gin.Engine, log logger.Logger, db *gorm.DB) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				// Validate database connection
				if db == nil {
					log.Error(ctx, "📊 Database instance is nil", map[string]interface{}{})
					return fmt.Errorf("database instance is nil - check database initialization logs")
				}

				// Test database connection
				sqlDB, err := db.DB()
				if err != nil {
					log.Error(ctx, "📊 Failed to get database instance", map[string]interface{}{
						"error": err.Error(),
					})
					return fmt.Errorf("failed to get database instance: %w", err)
				}
				if err := sqlDB.Ping(); err != nil {
					log.Error(ctx, "📊 Database ping failed", map[string]interface{}{
						"error": err.Error(),
					})
					return fmt.Errorf("database not accessible: %w", err)
				}
				log.Info(ctx, "📊 Database connection verified")

				log.Info(ctx, "Running migrations...")

				services.RunMigrations()

				log.Info(ctx, "Migrations done")

				// Setup the swagger info
				if cfg.Environment == entities.Environment.Development {
					docs.SwaggerInfo.Host = "localhost:" + cfg.Port
					docs.SwaggerInfo.Schemes = []string{"http", "https"}
				} else {
					docs.SwaggerInfo.Host = "api.spooliq.rodolfodebonis.com.br"
					docs.SwaggerInfo.Schemes = []string{"https"}
				}

				// Title, Description, Version and BasePath are defined once as
				// general annotations in main.go and baked into docs at generation
				// time. Only Host and Schemes are environment-specific and set here.

				runPort := fmt.Sprintf(":%s", cfg.Port)

				go func() {
					err := app.Run(runPort)
					if err != nil && !errors.Is(err, http.ErrServerClosed) {
						appError := appErrors.RootError(err.Error(), nil)
						log.LogError(ctx, "Erro ao subir servidor HTTP", appError)
						panic(err)
					}
				}()

				return nil
			},
			OnStop: func(ctx context.Context) error {
				log.Info(ctx, "🛑 Shutting down gracefully")
				return nil
			},
		})
	})
}
