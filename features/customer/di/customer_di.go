package di

import (
	"github.com/RodolfoBonis/spooliq/features/customer/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/usecases"
	"go.uber.org/fx"
)

// Module exports the customer feature's dependency injection module
var Module = fx.Module(
	"customer",
	fx.Provide(
		repositories.NewCustomerRepository,
		usecases.NewCustomerUseCase,
	),
)
