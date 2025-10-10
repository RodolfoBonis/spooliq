package services

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides the fx module for services.
var Module = fx.Module("services",
	fx.Provide(
		NewAmqpService,
		NewRedisService,
		NewAuthService,
		NewAsaasService,
		NewKeycloakAdminService,
		func(logger logger.Logger) *gorm.DB {
			if Connector == nil {
				_ = OpenConnection(logger)
			}
			return Connector
		},
	),
)
