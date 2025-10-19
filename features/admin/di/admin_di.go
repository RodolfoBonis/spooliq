package di

import (
	"github.com/RodolfoBonis/spooliq/features/admin"
	"github.com/RodolfoBonis/spooliq/features/admin/domain/usecases"
	subscriptionUsecases "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/usecases"
	"go.uber.org/fx"
)

// AdminModule provides the fx module for admin dependencies
var AdminModule = fx.Module("admin",
	fx.Provide(
		// Company & Subscription Use Cases
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
			subscriptionPlanUC *subscriptionUsecases.SubscriptionPlanUseCase,
		) *admin.Handler {
			return admin.NewAdminHandler(
				listCompaniesUC,
				getCompanyDetailsUC,
				updateCompanyStatusUC,
				listSubscriptionsUC,
				getSubscriptionDetailsUC,
				getPaymentHistoryUC,
				subscriptionPlanUC,
			)
		},
	),
)
