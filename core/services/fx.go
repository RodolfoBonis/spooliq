package services

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/jinzhu/gorm"
	"go.uber.org/fx"
)

// Module provides the fx module for services.
var Module = fx.Module("services",
	fx.Provide(
		NewAmqpService,
		NewRedisService,
		NewAuthService,
		NewTelemetryService,
		func(logger logger.Logger) *gorm.DB {
			if Connector == nil {
				_ = OpenConnection(logger)
			}
			return Connector
		},
	),
)
