package usecases

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
)

// ISubscriptionUseCase defines the interface for subscription use cases
type ISubscriptionUseCase interface {
	// Public endpoints
	GetPlanFeatures(c *gin.Context)

	// User endpoints
	GetPaymentHistory(c *gin.Context)
	CancelSubscription(c *gin.Context)
	ChangePlan(c *gin.Context)
}

// SubscriptionUseCase implements the subscription use cases
type SubscriptionUseCase struct {
	subscriptionRepository repositories.SubscriptionRepository
	planRepository         repositories.PlanRepository
	companyRepository      companyRepositories.CompanyRepository
	asaasService           services.IAsaasService
	logger                 logger.Logger
}

// NewSubscriptionUseCase creates a new instance of SubscriptionUseCase
func NewSubscriptionUseCase(
	subscriptionRepository repositories.SubscriptionRepository,
	planRepository repositories.PlanRepository,
	companyRepository companyRepositories.CompanyRepository,
	asaasService services.IAsaasService,
	logger logger.Logger,
) ISubscriptionUseCase {
	return &SubscriptionUseCase{
		subscriptionRepository: subscriptionRepository,
		planRepository:         planRepository,
		companyRepository:      companyRepository,
		asaasService:           asaasService,
		logger:                 logger,
	}
}

// GetPlanFeatures retrieves detailed plan information (public)
func (uc *SubscriptionUseCase) GetPlanFeatures(c *gin.Context) {
	getPlanFeaturesUC := NewGetPlanFeaturesUseCase(uc.planRepository, uc.logger)
	getPlanFeaturesUC.Execute(c)
}

// GetPaymentHistory retrieves payment history for the current organization
func (uc *SubscriptionUseCase) GetPaymentHistory(c *gin.Context) {
	getPaymentHistoryUC := NewGetPaymentHistoryUseCase(uc.subscriptionRepository, uc.logger)
	getPaymentHistoryUC.Execute(c)
}

// CancelSubscription cancels the current subscription
func (uc *SubscriptionUseCase) CancelSubscription(c *gin.Context) {
	cancelUC := NewCancelSubscriptionUseCase(uc.asaasService, uc.companyRepository, uc.logger)
	cancelUC.Execute(c)
}

// ChangePlan changes the subscription plan
func (uc *SubscriptionUseCase) ChangePlan(c *gin.Context) {
	changePlanUC := NewChangePlanUseCase(uc.asaasService, uc.companyRepository, uc.planRepository, uc.logger)
	changePlanUC.Execute(c)
}
