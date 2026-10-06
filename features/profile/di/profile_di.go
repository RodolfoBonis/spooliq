// Package di wires the print profile feature dependencies for FX.
package di

import (
	presetRepos "github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/profile"
	"github.com/RodolfoBonis/spooliq/features/profile/data/repositories"
	profileRepos "github.com/RodolfoBonis/spooliq/features/profile/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/usecases"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides all print profile dependencies for FX dependency injection.
var Module = fx.Module("profile",
	fx.Provide(
		fx.Annotate(func(db *gorm.DB) profileRepos.PrintProfileRepository {
			return repositories.NewPrintProfileRepository(db)
		}),
		func(profileRepo profileRepos.PrintProfileRepository, presetRepo presetRepos.PresetRepository) *usecases.ProfileUseCase {
			return usecases.NewProfileUseCase(profileRepo, presetRepo)
		},
		profile.NewProfileHandler,
	),
)
