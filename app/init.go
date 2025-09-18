package app

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/docs"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

func InitAndRun() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle, cfg *config.AppConfig, amqpService *services.AmqpService, app *gin.Engine, log logger.Logger) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				// Initialize database connection
				services.OpenConnection(log)
				log.Info(ctx, "💿 Database connected successfully")

				// Initialize AMQP connection
				amqpService.StartAmqpConnection()
				log.Info(ctx, "🔗 Amqp connected successfully")

				// setup the swagger info
				if cfg.Environment == entities.Environment.Development {
					docs.SwaggerInfo.Host = "localhost:" + cfg.Port
				} else {
					docs.SwaggerInfo.Host = "spooliq.RodolfoBonis.com"
				}

				docs.SwaggerInfo.BasePath = "/api/v1"
				docs.SwaggerInfo.Schemes = []string{"http", "https"}
				docs.SwaggerInfo.Title = "spooliq"
				docs.SwaggerInfo.Description = "SpoolIq calcula o preço real das suas impressões 3D: filamento multi-cor (g/m), energia (kWh + bandeira), desgaste, overhead e mão-de-obra. Gera pacotes (só impressão, ajustes, modelagem), exporta PDF/CSV e guarda materiais."
				docs.SwaggerInfo.Version = "1.0"

				host := cfg.Environment
				if cfg.Environment == entities.Environment.Development {
					host = "localhost"
				} else {
					host = "spooliq.RodolfoBonis.com"
				}

				docs.SwaggerInfo.Host = host + ":" + cfg.Port

				// Run the Gin server
				go app.Run(":" + cfg.Port)

				return nil
			},
			OnStop: func(ctx context.Context) error {
				log.Info(ctx, "🛑 Shutting down gracefully")
				return nil
			},
		})
	})
}
