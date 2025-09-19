package services

import (
	"github.com/jinzhu/gorm"
	"go.uber.org/fx"
)

// Module provides the fx module for services.
var Module = fx.Module("services",
	fx.Provide(
		NewAmqpService,
		NewRedisService,
		NewAuthService,
		func() *gorm.DB {
			return Connector
		},
	),
)
