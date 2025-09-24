package di

import (
	"context"
	
	calculationDI "github.com/RodolfoBonis/spooliq/features/calculation/di"
	filamentsRepos "github.com/RodolfoBonis/spooliq/features/filaments/domain/repositories"
	presetsDI "github.com/RodolfoBonis/spooliq/features/presets/di"
	presetsEntities "github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
	presetsRepos "github.com/RodolfoBonis/spooliq/features/presets/domain/repositories"
	quotesRepositories "github.com/RodolfoBonis/spooliq/features/quotes/data/repositories"
	quotesServices "github.com/RodolfoBonis/spooliq/features/quotes/domain/services"
	quotesUseCases "github.com/RodolfoBonis/spooliq/features/quotes/domain/usecases"
	"go.uber.org/fx"
)

// QuotesModule provides dependency injection for quote-related components.
var QuotesModule = fx.Module("quotes",
	// Include calculation service
	calculationDI.Module,
	
	// Include presets module for preset repository
	presetsDI.Module,

	// Repositories
	fx.Provide(quotesRepositories.NewQuoteRepository),

	// Services - SnapshotService depends on FilamentRepository (provided by FilamentsModule)
	fx.Provide(func(filamentRepo filamentsRepos.FilamentRepository) quotesServices.SnapshotService {
		return quotesServices.NewSnapshotService(filamentRepo)
	}),
	
	// Services - EnergyProfileService depends on PresetRepository (provided by PresetsModule)
	fx.Provide(func(presetRepo presetsRepos.PresetRepository) quotesServices.EnergyProfileService {
		// Create adapter that implements the PresetRepository interface expected by EnergyProfileService
		adapter := &presetRepositoryAdapter{repo: presetRepo}
		return quotesServices.NewEnergyProfileService(adapter)
	}),

	// Use Cases
	fx.Provide(quotesUseCases.NewQuoteUseCase),
)

// presetRepositoryAdapter adapts the presets repository to the interface expected by EnergyProfileService
type presetRepositoryAdapter struct {
	repo presetsRepos.PresetRepository
}

// GetPresetByKey implements the PresetRepository interface expected by EnergyProfileService
func (a *presetRepositoryAdapter) GetPresetByKey(ctx context.Context, key string) (*presetsEntities.Preset, error) {
	return a.repo.GetPresetByKey(ctx, key)
}
