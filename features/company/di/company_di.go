package di

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	dataRepositories "github.com/RodolfoBonis/spooliq/features/company/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/company/domain/usecases"
	"go.uber.org/fx"
)

// Module exports the company feature's dependencies
var Module = fx.Options(
	fx.Provide(dataRepositories.NewCompanyRepository),
	fx.Provide(dataRepositories.NewBrandingRepository),
	fx.Provide(func(repo domainRepositories.CompanyRepository, cdnService *services.CDNService, logger logger.Logger) usecases.ICompanyUseCase {
		return usecases.NewCompanyUseCase(repo, cdnService, logger)
	}),
	fx.Provide(func(repo domainRepositories.BrandingRepository, logger logger.Logger) usecases.IBrandingUseCase {
		return usecases.NewBrandingUseCase(repo, logger)
	}),
)
