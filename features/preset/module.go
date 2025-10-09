package preset

import (
	"github.com/RodolfoBonis/spooliq/features/preset/data/repositories"
	presetRepos "github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/usecases"
	"go.uber.org/fx"
)

// Module provides the preset feature dependencies
var Module = fx.Module("preset",
	// Repositories
	fx.Provide(
		fx.Annotate(
			repositories.NewPresetRepository,
			fx.As(new(presetRepos.PresetRepository)),
		),
	),

	// Use Cases
	fx.Provide(
		usecases.NewCreatePresetUseCase,
		usecases.NewFindPresetUseCase,
		usecases.NewUpdatePresetUseCase,
		usecases.NewDeletePresetUseCase,
	),

	// Handlers
	fx.Provide(NewPresetHandler),
)
