package usecases

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	authEntities "github.com/RodolfoBonis/spooliq/features/auth/domain/entities"
	companyEntities "github.com/RodolfoBonis/spooliq/features/company/domain/entities"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	subscriptionEntities "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	subscriptionRepositories "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	userEntities "github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	userRepositories "github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// RegisterUseCase handles company and user registration
type RegisterUseCase struct {
	keycloakService   services.IKeycloakAdminService
	asaasService      services.IAsaasService
	companyRepository companyRepositories.CompanyRepository
	userRepository    userRepositories.UserRepository
	gatewayLinkRepo   subscriptionRepositories.PaymentGatewayLinkRepository
	logger            logger.Logger
	validator         *validator.Validate
}

// NewRegisterUseCase creates a new RegisterUseCase
func NewRegisterUseCase(
	keycloakService services.IKeycloakAdminService,
	asaasService services.IAsaasService,
	companyRepository companyRepositories.CompanyRepository,
	userRepository userRepositories.UserRepository,
	gatewayLinkRepo subscriptionRepositories.PaymentGatewayLinkRepository,
	logger logger.Logger,
) *RegisterUseCase {
	return &RegisterUseCase{
		keycloakService:   keycloakService,
		asaasService:      asaasService,
		companyRepository: companyRepository,
		userRepository:    userRepository,
		gatewayLinkRepo:   gatewayLinkRepo,
		logger:            logger,
		validator:         validator.New(),
	}
}

// Register handles the company registration process
// @Summary Register a new company
// @Description Register a new company with owner user, starts 14-day trial
// @Tags auth
// @Accept json
// @Produce json
// @Param request body authEntities.RegisterRequest true "Registration data"
// @Success 201 {object} authEntities.RegisterResponse
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /register [post]
func (uc *RegisterUseCase) Register(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Registration attempt started", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	var request authEntities.RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Invalid registration payload", map[string]interface{}{
			"error": err.Error(),
		})
		appError := errors.UsecaseError("Invalid request format")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Validate request
	if err := uc.validator.Struct(request); err != nil {
		uc.logger.Error(ctx, "Registration validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		appError := errors.UsecaseError("Validation failed: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Check if email already exists
	existingUser, err := uc.userRepository.FindByEmail(ctx, request.Email)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check existing user", map[string]interface{}{
			"error": err.Error(),
			"email": request.Email,
		})
		appError := errors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	if existingUser != nil {
		uc.logger.Error(ctx, "Email already registered", map[string]interface{}{
			"email": request.Email,
		})
		appError := errors.UsecaseError("Email already registered")
		c.JSON(http.StatusConflict, gin.H{"error": appError.Message})
		return
	}

	// Generate organization UUID
	organizationID := uuid.New().String()

	uc.logger.Info(ctx, "Creating new organization", map[string]interface{}{
		"organization_id": organizationID,
		"company_name":    request.CompanyName,
	})

	// Create customer in Asaas
	asaasCustomer, err := uc.createAsaasCustomer(ctx, request, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create Asaas customer", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		appError := errors.UsecaseError("Failed to create payment account: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": appError.Message})
		return
	}

	// Create user in Keycloak
	keycloakUserID, err := uc.createKeycloakUser(ctx, request, organizationID, request.CompanyName)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create Keycloak user", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		appError := errors.UsecaseError("Failed to create user account: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": appError.Message})
		return
	}

	// Calculate trial end date (14 days from now)
	trialEndsAt := time.Now().Add(14 * 24 * time.Hour)

	// Get Trial plan ID
	trialPlanID := uuid.MustParse("7e45a0e9-0a8c-4843-b519-c208d150e41c")

	// Create company in database
	company := &companyEntities.CompanyEntity{
		ID:                    uuid.New(),
		OrganizationID:        organizationID,
		Name:                  request.CompanyName,
		Document:              &request.CompanyDocument,
		Phone:                 &request.CompanyPhone,
		Address:               &request.Address,
		City:                  &request.City,
		State:                 &request.State,
		ZipCode:               &request.ZipCode,
		SubscriptionStatus:    "trial",
		SubscriptionPlanID:    &trialPlanID,
		StatusUpdatedAt:       time.Now(),
		IsPlatformCompany:     false,
		TrialEndsAt:           &trialEndsAt,
		SubscriptionStartedAt: nil,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if request.CompanyTradeName != "" {
		company.TradeName = &request.CompanyTradeName
	}

	err = uc.companyRepository.Create(ctx, company)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create company", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		appError := errors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Create PaymentGatewayLink record to store Asaas customer ID
	gatewayLink := &subscriptionEntities.PaymentGatewayLinkEntity{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Gateway:        "asaas",
		CustomerID:     asaasCustomer.ID,
		SubscriptionID: nil, // Will be set when subscription is created
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err = uc.gatewayLinkRepo.Create(ctx, gatewayLink)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create PaymentGatewayLink", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		// Non-fatal error, continue
		uc.logger.Warning(ctx, "PaymentGatewayLink will need to be created manually", map[string]interface{}{
			"organization_id":   organizationID,
			"asaas_customer_id": asaasCustomer.ID,
		})
	} else {
		uc.logger.Info(ctx, "PaymentGatewayLink created successfully", map[string]interface{}{
			"organization_id":   organizationID,
			"asaas_customer_id": asaasCustomer.ID,
		})
	}

	// Create user in database
	user := &userEntities.UserEntity{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		KeycloakUserID: keycloakUserID,
		Email:          request.Email,
		Name:           request.Name,
		UserType:       "owner",
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err = uc.userRepository.Create(ctx, user)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create user", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		appError := errors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Note: For trial users, we don't create subscription in Asaas yet
	// Subscriptions will be created when user upgrades to Starter+ plans
	uc.logger.Info(ctx, "Trial user registered - Asaas subscription will be created on plan upgrade", map[string]interface{}{
		"organization_id":   organizationID,
		"asaas_customer_id": asaasCustomer.ID,
		"trial_ends_at":     trialEndsAt.Format("2006-01-02"),
	})

	uc.logger.Info(ctx, "Registration completed successfully", map[string]interface{}{
		"user_id":         user.ID.String(),
		"organization_id": organizationID,
		"email":           request.Email,
	})

	response := authEntities.RegisterResponse{
		UserID:         user.ID.String(),
		OrganizationID: organizationID,
		TrialEndsAt:    trialEndsAt.Format("2006-01-02T15:04:05Z07:00"),
		Message:        "Registration successful! Your 14-day trial has started.",
	}

	c.JSON(http.StatusCreated, response)
}

func (uc *RegisterUseCase) createKeycloakUser(ctx context.Context, request authEntities.RegisterRequest, organizationID string, companyName string) (string, error) {
	uc.logger.Info(ctx, "Creating Keycloak user", map[string]interface{}{
		"email":           request.Email,
		"organization_id": organizationID,
	})

	// 1. Create user in Keycloak
	// Split name into first and last (Keycloak requires both)
	nameParts := strings.Split(strings.TrimSpace(request.Name), " ")
	firstName := nameParts[0]
	lastName := ""
	if len(nameParts) > 1 {
		lastName = strings.Join(nameParts[1:], " ")
	} else {
		lastName = "User" // Default last name if not provided
	}

	keycloakReq := services.KeycloakUserRequest{
		Username:      request.Email,
		Email:         request.Email,
		EmailVerified: true, // Allow immediate login without email verification
		Enabled:       true,
		FirstName:     firstName,
		LastName:      lastName,
		Attributes: map[string][]string{
			"organization_id": {organizationID},
			"user_type":       {"owner"},
		},
	}

	userID, err := uc.keycloakService.CreateUser(ctx, keycloakReq)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create Keycloak user", map[string]interface{}{
			"error": err.Error(),
			"email": request.Email,
		})
		return "", fmt.Errorf("failed to create user in Keycloak: %w", err)
	}

	uc.logger.Info(ctx, "Keycloak user created", map[string]interface{}{
		"user_id": userID,
		"email":   request.Email,
	})

	// 2. Set user password
	if err := uc.keycloakService.SetUserPassword(ctx, userID, request.Password); err != nil {
		uc.logger.Error(ctx, "Failed to set user password", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
		return "", fmt.Errorf("failed to set user password: %w", err)
	}

	// 3. Assign Owner role to user
	if err := uc.keycloakService.AssignRoleToUser(ctx, userID, "Owner"); err != nil {
		uc.logger.Error(ctx, "Failed to assign Owner role", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
		// Non-fatal error, continue
		uc.logger.Info(ctx, "Owner role will need to be assigned manually", map[string]interface{}{
			"user_id": userID,
		})
	}

	// 4. Get or create organization group
	groupName := fmt.Sprintf("org-%s", organizationID)
	groupID, err := uc.keycloakService.GetOrCreateGroup(ctx, groupName)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get/create organization group", map[string]interface{}{
			"error":      err.Error(),
			"group_name": groupName,
		})
		// Non-fatal error, continue
		uc.logger.Info(ctx, "User will need to be added to group manually", map[string]interface{}{
			"user_id": userID,
		})
	} else {
		// Set organization_id and company_name attributes on group
		if err := uc.keycloakService.SetGroupAttributes(ctx, groupID, map[string][]string{
			"organization_id": {organizationID},
			"company_name":    {companyName},
		}); err != nil {
			uc.logger.Error(ctx, "Failed to set group attributes", map[string]interface{}{
				"error":    err.Error(),
				"group_id": groupID,
			})
		}

		// 5. Add user to organization group
		if err := uc.keycloakService.AddUserToGroup(ctx, userID, groupID); err != nil {
			uc.logger.Error(ctx, "Failed to add user to group", map[string]interface{}{
				"error":    err.Error(),
				"user_id":  userID,
				"group_id": groupID,
			})
			// Non-fatal error, continue
			uc.logger.Info(ctx, "User will need to be added to group manually", map[string]interface{}{
				"user_id": userID,
			})
		}
	}

	uc.logger.Info(ctx, "Keycloak user setup completed", map[string]interface{}{
		"user_id": userID,
		"email":   request.Email,
	})

	return userID, nil
}

func (uc *RegisterUseCase) createAsaasCustomer(ctx context.Context, request authEntities.RegisterRequest, organizationID string) (*services.AsaasCustomerResponse, error) {
	uc.logger.Info(ctx, "Creating Asaas customer", map[string]interface{}{
		"organization_id": organizationID,
		"company_name":    request.CompanyName,
	})

	// Prepare customer request
	asaasRequest := services.AsaasCustomerRequest{
		Name:              request.CompanyName,
		Email:             request.Email,
		CpfCnpj:           request.CompanyDocument,
		Phone:             request.CompanyPhone,
		Address:           request.Address,
		AddressNumber:     "", // Not in RegisterRequest, could be extracted from address
		Province:          request.City,
		PostalCode:        request.ZipCode,
		ExternalReference: organizationID,
	}

	// Create customer in Asaas
	customer, err := uc.asaasService.CreateCustomer(ctx, asaasRequest)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create Asaas customer", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		return nil, fmt.Errorf("failed to create Asaas customer: %w", err)
	}

	uc.logger.Info(ctx, "Asaas customer created successfully", map[string]interface{}{
		"customer_id":     customer.ID,
		"organization_id": organizationID,
	})

	return customer, nil
}
