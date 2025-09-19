package di

import (
	"github.com/RodolfoBonis/spooliq/features/calculation/domain/services"
	"go.uber.org/fx"
)

// Module provides the fx module for calculation feature.
var Module = fx.Module("calculation",
	fx.Provide(
		services.NewCalculationService,
	),
)