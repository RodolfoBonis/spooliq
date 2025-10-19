package di

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/admin"
	"github.com/RodolfoBonis/spooliq/features/admin/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/admin/domain/usecases/plans"
	planRepositories "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
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

		// Plan Use Cases
		func(planRepo planRepositories.PlanRepository, logger logger.Logger) *plans.ListPlansUseCase {
			return plans.NewListPlansUseCase(planRepo, logger)
		},
		func(planRepo planRepositories.PlanRepository, logger logger.Logger) *plans.CreatePlanUseCase {
			return plans.NewCreatePlanUseCase(planRepo, logger)
		},
		func(planRepo planRepositories.PlanRepository, logger logger.Logger) *plans.UpdatePlanUseCase {
			return plans.NewUpdatePlanUseCase(planRepo, logger)
		},
		func(planRepo planRepositories.PlanRepository, logger logger.Logger) *plans.DeletePlanUseCase {
			return plans.NewDeletePlanUseCase(planRepo, logger)
		},
		func(planRepo planRepositories.PlanRepository, logger logger.Logger) *plans.AddPlanFeatureUseCase {
			return plans.NewAddPlanFeatureUseCase(planRepo, logger)
		},
		func(planRepo planRepositories.PlanRepository, logger logger.Logger) *plans.UpdatePlanFeatureUseCase {
			return plans.NewUpdatePlanFeatureUseCase(planRepo, logger)
		},
		func(planRepo planRepositories.PlanRepository, logger logger.Logger) *plans.DeletePlanFeatureUseCase {
			return plans.NewDeletePlanFeatureUseCase(planRepo, logger)
		},

		// Handler
		func(
			listCompaniesUC *usecases.ListCompaniesUseCase,
			getCompanyDetailsUC *usecases.GetCompanyDetailsUseCase,
			updateCompanyStatusUC *usecases.UpdateCompanyStatusUseCase,
			listSubscriptionsUC *usecases.ListSubscriptionsUseCase,
			getSubscriptionDetailsUC *usecases.GetSubscriptionDetailsUseCase,
			getPaymentHistoryUC *usecases.GetPaymentHistoryUseCase,
			listPlansUC *plans.ListPlansUseCase,
			createPlanUC *plans.CreatePlanUseCase,
			updatePlanUC *plans.UpdatePlanUseCase,
			deletePlanUC *plans.DeletePlanUseCase,
			addFeatureUC *plans.AddPlanFeatureUseCase,
			updateFeatureUC *plans.UpdatePlanFeatureUseCase,
			deleteFeatureUC *plans.DeletePlanFeatureUseCase,
		) *admin.Handler {
			return admin.NewAdminHandler(
				listCompaniesUC,
				getCompanyDetailsUC,
				updateCompanyStatusUC,
				listSubscriptionsUC,
				getSubscriptionDetailsUC,
				getPaymentHistoryUC,
				listPlansUC,
				createPlanUC,
				updatePlanUC,
				deletePlanUC,
				addFeatureUC,
				updateFeatureUC,
				deleteFeatureUC,
			)
		},
	),
)
