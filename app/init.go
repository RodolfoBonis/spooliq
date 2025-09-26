package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	appErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/docs"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"go.uber.org/fx"
)

// InitAndRun initializes and runs the application using Fx lifecycle
func InitAndRun() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle, cfg *config.AppConfig, amqpService *services.AmqpService, app *gin.Engine, log logger.Logger, db *gorm.DB) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
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

				docs.SwaggerInfo.BasePath = "/v1"

				docs.SwaggerInfo.Title = "spooliq"
				docs.SwaggerInfo.Description = "SpoolIq calcula o preço real das suas impressões 3D: filamento multi-cor (g/m), energia (kWh + bandeira), desgaste, overhead e mão-de-obra. Gera pacotes (só impressão, ajustes, modelagem), exporta PDF/CSV e guarda materiais."
				docs.SwaggerInfo.Version = "1.0"

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
