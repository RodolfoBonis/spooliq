package services

import (
	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"github.com/RodolfoBonis/go-otel-agent/logger"
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
		func(logger logger.Logger, agent *otelagent.Agent) *gorm.DB {
			if Connector == nil {
				_ = OpenConnection(logger, agent)
			}
			return Connector
		},
	),
)
