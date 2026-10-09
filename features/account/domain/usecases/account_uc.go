package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	coreerrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/core/validation"
	"github.com/RodolfoBonis/spooliq/features/account/domain/entities"
	userEntities "github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	userRepos "github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
)

// resetLinkLifespan is how long the password reset link stays valid.
const resetLinkLifespan = time.Hour

// Rate limits for the public forgot-password endpoint.
const (
	forgotPerIPLimit    = 10
	forgotPerEmailLimit = 3
	forgotWindow        = time.Hour
)

// PasswordVerifier checks e-mail + password against the identity provider.
type PasswordVerifier func(ctx context.Context, email, password string) error

// RateLimiter reports whether another request is allowed for key within window.
// Implementations should fail open when their backend is unavailable.
type RateLimiter func(ctx context.Context, key string, limit int, window time.Duration) bool

// Claims is the subset of the authenticated user's token used here.
type Claims struct {
	KeycloakID     string
	Email          string
	Name           string
	OrganizationID string
}

// AccountUseCase implements self-service account operations: password reset,
// own profile and own password.
type AccountUseCase struct {
	keycloak       services.IKeycloakAdminService
	users          userRepos.UserRepository
	verifyPassword PasswordVerifier
	allow          RateLimiter
	logger         logger.Logger
}

// NewAccountUseCase creates an AccountUseCase.
func NewAccountUseCase(
	keycloak services.IKeycloakAdminService,
	users userRepos.UserRepository,
	verifyPassword PasswordVerifier,
	allow RateLimiter,
	log logger.Logger,
) *AccountUseCase {
	return &AccountUseCase{
		keycloak:       keycloak,
		users:          users,
		verifyPassword: verifyPassword,
		allow:          allow,
		logger:         log,
	}
}

// ForgotPassword e-mails a reset link (via Keycloak) when the e-mail belongs to
// a user. It never reveals whether the e-mail exists: unknown e-mails and
// provider failures return nil, and only invalid input or rate limits error.
func (uc *AccountUseCase) ForgotPassword(ctx context.Context, req *entities.ForgotPasswordRequest, clientIP string) error {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := validation.Validate(req); err != nil {
		return err
	}
	email := req.Email

	if !uc.allow(ctx, "forgot_password:ip:"+clientIP, forgotPerIPLimit, forgotWindow) ||
		!uc.allow(ctx, "forgot_password:email:"+email, forgotPerEmailLimit, forgotWindow) {
		return coreerrors.TooManyRequests("too_many_requests", "Muitas solicitações. Tente novamente mais tarde.")
	}

	user, err := uc.keycloak.GetUserByEmail(ctx, email)
	if err != nil {
		uc.logger.Error(ctx, "Forgot password: user lookup failed", map[string]interface{}{"error": err.Error()})
		return nil
	}
	if user == nil || !user.Enabled {
		uc.logger.Info(ctx, "Forgot password: no active user for e-mail", nil)
		return nil
	}

	if err := uc.keycloak.ExecuteActionsEmail(ctx, user.ID, []string{"UPDATE_PASSWORD"}, resetLinkLifespan); err != nil {
		// Usually SMTP misconfiguration on the realm: alert, but keep the
		// response identical so the endpoint can't be used to probe e-mails.
		uc.logger.Error(ctx, "Forgot password: Keycloak failed to send the e-mail", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
		})
		return nil
	}
	uc.logger.Info(ctx, "Forgot password: reset e-mail sent", map[string]interface{}{"user_id": user.ID})
	return nil
}

// GetMe returns the authenticated user's profile. Platform admins without a
// local users row are answered from the token claims.
func (uc *AccountUseCase) GetMe(ctx context.Context, claims Claims) (*entities.MeResponse, error) {
	user, err := uc.users.FindByKeycloakUserID(ctx, claims.KeycloakID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return &entities.MeResponse{
			ID:             claims.KeycloakID,
			Name:           claims.Name,
			Email:          claims.Email,
			OrganizationID: claims.OrganizationID,
		}, nil
	}
	return toMe(user), nil
}

// UpdateMe changes the authenticated user's own name in Keycloak and in the
// local users table.
func (uc *AccountUseCase) UpdateMe(ctx context.Context, claims Claims, req *entities.UpdateMeRequest) (*entities.MeResponse, error) {
	if err := validation.Validate(req); err != nil {
		return nil, err
	}
	name := strings.Join(strings.Fields(req.Name), " ")
	first, last := splitName(name)

	if err := uc.keycloak.UpdateUserName(ctx, claims.KeycloakID, first, last); err != nil {
		uc.logger.Error(ctx, "Update profile: Keycloak update failed", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	user, err := uc.users.FindByKeycloakUserID(ctx, claims.KeycloakID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return &entities.MeResponse{
			ID:             claims.KeycloakID,
			Name:           name,
			Email:          claims.Email,
			OrganizationID: claims.OrganizationID,
		}, nil
	}
	user.Name = name
	if err := uc.users.Update(ctx, user.ID, user.OrganizationID, user); err != nil {
		return nil, err
	}
	return toMe(user), nil
}

// ChangePassword verifies the current password and sets a new one.
func (uc *AccountUseCase) ChangePassword(ctx context.Context, claims Claims, req *entities.ChangePasswordRequest) error {
	if err := validation.Validate(req); err != nil {
		return err
	}
	if req.NewPassword == req.CurrentPassword {
		return coreerrors.BadRequest("same_password", "A nova senha deve ser diferente da atual")
	}
	if err := uc.verifyPassword(ctx, claims.Email, req.CurrentPassword); err != nil {
		uc.logger.Warning(ctx, "Change password: wrong current password", map[string]interface{}{"user_id": claims.KeycloakID})
		return coreerrors.BadRequest("invalid_current_password", "Senha atual incorreta")
	}
	if err := uc.keycloak.SetUserPassword(ctx, claims.KeycloakID, req.NewPassword); err != nil {
		// Keycloak rejects passwords that break the realm's password policy.
		uc.logger.Error(ctx, "Change password: Keycloak rejected the new password", map[string]interface{}{"error": err.Error()})
		return coreerrors.BadRequest("password_rejected", "A nova senha não atende aos requisitos de segurança")
	}
	uc.logger.Info(ctx, "Password changed", map[string]interface{}{"user_id": claims.KeycloakID})
	return nil
}

func toMe(user *userEntities.UserEntity) *entities.MeResponse {
	return &entities.MeResponse{
		ID:             user.KeycloakUserID,
		Name:           user.Name,
		Email:          user.Email,
		UserType:       user.UserType,
		OrganizationID: user.OrganizationID,
	}
}

// splitName mirrors registration: Keycloak requires both first and last name.
func splitName(name string) (string, string) {
	parts := strings.SplitN(name, " ", 2)
	if len(parts) == 1 {
		return parts[0], "User"
	}
	return parts[0], parts[1]
}
