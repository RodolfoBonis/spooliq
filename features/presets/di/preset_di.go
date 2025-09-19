package di

import (
	"go.uber.org/fx"

	presetRepositories "github.com/RodolfoBonis/spooliq/features/presets/data/repositories"
	presetServices "github.com/RodolfoBonis/spooliq/features/presets/data/services"
	"github.com/RodolfoBonis/spooliq/features/presets/domain/repositories"
	domainServices "github.com/RodolfoBonis/spooliq/features/presets/domain/services"
)

// Module provides the presets feature module
var Module = fx.Module("presets",
	fx.Provide(
		// Repositories
		fx.Annotate(
			presetRepositories.NewPresetRepository,
			fx.As(new(repositories.PresetRepository)),
		),

		// Services
		fx.Annotate(
			presetServices.NewPresetService,
			fx.As(new(domainServices.PresetService)),
		),
	),
)
