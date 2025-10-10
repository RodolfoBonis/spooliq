package di

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	dataRepositories "github.com/RodolfoBonis/spooliq/features/company/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/company/domain/usecases"
	"go.uber.org/fx"
)

// Module exports the company feature's dependencies
var Module = fx.Options(
	fx.Provide(dataRepositories.NewCompanyRepository),
	fx.Provide(func(repo domainRepositories.CompanyRepository, logger logger.Logger) usecases.ICompanyUseCase {
		return usecases.NewCompanyUseCase(repo, logger)
	}),
)
