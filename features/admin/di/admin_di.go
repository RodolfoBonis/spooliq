package di

import (
	"github.com/RodolfoBonis/spooliq/features/admin"
	"github.com/RodolfoBonis/spooliq/features/admin/domain/usecases"
	"go.uber.org/fx"
)

// AdminModule provides the fx module for admin dependencies
var AdminModule = fx.Module("admin",
	fx.Provide(
		// Use Cases
		usecases.NewListCompaniesUseCase,
		usecases.NewGetCompanyDetailsUseCase,
		usecases.NewUpdateCompanyStatusUseCase,
		usecases.NewListSubscriptionsUseCase,
		usecases.NewGetSubscriptionDetailsUseCase,
		usecases.NewGetPaymentHistoryUseCase,

		// Handler
		func(
			listCompaniesUC *usecases.ListCompaniesUseCase,
			getCompanyDetailsUC *usecases.GetCompanyDetailsUseCase,
			updateCompanyStatusUC *usecases.UpdateCompanyStatusUseCase,
			listSubscriptionsUC *usecases.ListSubscriptionsUseCase,
			getSubscriptionDetailsUC *usecases.GetSubscriptionDetailsUseCase,
			getPaymentHistoryUC *usecases.GetPaymentHistoryUseCase,
		) *admin.Handler {
			return admin.NewAdminHandler(
				listCompaniesUC,
				getCompanyDetailsUC,
				updateCompanyStatusUC,
				listSubscriptionsUC,
				getSubscriptionDetailsUC,
				getPaymentHistoryUC,
			)
		},
	),
)
