package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	otellogger "github.com/RodolfoBonis/go-otel-agent/logger"
	apperrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/account/domain/entities"
	userEntities "github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopLogger struct{}

func (noopLogger) Debug(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Info(context.Context, string, ...otellogger.Fields)    {}
func (noopLogger) Warning(context.Context, string, ...otellogger.Fields) {}
func (noopLogger) Error(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Fatal(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Panic(context.Context, string, ...otellogger.Fields)   {}
func (n noopLogger) With(otellogger.Fields) otellogger.Logger            { return n }
func (noopLogger) LogError(context.Context, string, error)               {}

// fakeKeycloak records the admin calls the use case makes.
type fakeKeycloak struct {
	services.IKeycloakAdminService
	users       map[string]*services.KeycloakUserResponse
	actionsFor  []string
	lifespan    time.Duration
	sendErr     *apperrors.AppError
	nameUpdate  [2]string
	newPassword string
	passwordErr *apperrors.AppError
}

func (k *fakeKeycloak) GetUserByEmail(_ context.Context, email string) (*services.KeycloakUserResponse, *apperrors.AppError) {
	return k.users[email], nil
}

func (k *fakeKeycloak) ExecuteActionsEmail(_ context.Context, userID string, _ []string, lifespan time.Duration) *apperrors.AppError {
	k.actionsFor = append(k.actionsFor, userID)
	k.lifespan = lifespan
	return k.sendErr
}

func (k *fakeKeycloak) UpdateUserName(_ context.Context, _, first, last string) *apperrors.AppError {
	k.nameUpdate = [2]string{first, last}
	return nil
}

func (k *fakeKeycloak) SetUserPassword(_ context.Context, _, password string) *apperrors.AppError {
	k.newPassword = password
	return k.passwordErr
}

type fakeUsers struct {
	byKeycloak map[string]*userEntities.UserEntity
	updated    *userEntities.UserEntity
}

func (r *fakeUsers) FindAll(context.Context, string, helpers.ListQuery) ([]*userEntities.UserEntity, int64, error) {
	return nil, 0, nil
}
func (r *fakeUsers) FindByID(context.Context, uuid.UUID, string) (*userEntities.UserEntity, error) {
	return nil, nil
}
func (r *fakeUsers) FindByEmail(context.Context, string) (*userEntities.UserEntity, error) {
	return nil, nil
}
func (r *fakeUsers) FindByKeycloakUserID(_ context.Context, id string) (*userEntities.UserEntity, error) {
	return r.byKeycloak[id], nil
}
func (r *fakeUsers) FindOwner(context.Context, string) (*userEntities.UserEntity, error) {
	return nil, nil
}
func (r *fakeUsers) Create(context.Context, *userEntities.UserEntity) error { return nil }
func (r *fakeUsers) Update(_ context.Context, _ uuid.UUID, _ string, u *userEntities.UserEntity) error {
	r.updated = u
	return nil
}
func (r *fakeUsers) Delete(context.Context, uuid.UUID, string) error { return nil }

func allowAll(context.Context, string, int, time.Duration) bool { return true }

func newUC(kc *fakeKeycloak, users *fakeUsers, verify PasswordVerifier, allow RateLimiter) *AccountUseCase {
	if verify == nil {
		verify = func(context.Context, string, string) error { return nil }
	}
	if allow == nil {
		allow = allowAll
	}
	return NewAccountUseCase(kc, users, verify, allow, noopLogger{})
}

var claims = Claims{KeycloakID: "kc-1", Email: "ana@x.com", Name: "Ana", OrganizationID: "org"}

func TestForgotPassword_SendsResetEmailForActiveUser(t *testing.T) {
	kc := &fakeKeycloak{users: map[string]*services.KeycloakUserResponse{
		"ana@x.com": {ID: "kc-1", Enabled: true},
	}}
	err := newUC(kc, &fakeUsers{}, nil, nil).ForgotPassword(
		context.Background(), &entities.ForgotPasswordRequest{Email: " Ana@X.com "}, "1.1.1.1")

	require.NoError(t, err)
	assert.Equal(t, []string{"kc-1"}, kc.actionsFor)
	assert.Equal(t, time.Hour, kc.lifespan)
}

func TestForgotPassword_DoesNotRevealUnknownEmailsOrFailures(t *testing.T) {
	kc := &fakeKeycloak{
		users:   map[string]*services.KeycloakUserResponse{"off@x.com": {ID: "kc-2", Enabled: false}},
		sendErr: apperrors.ServiceError("smtp down"),
	}
	uc := newUC(kc, &fakeUsers{}, nil, nil)

	for _, email := range []string{"nobody@x.com", "off@x.com"} {
		err := uc.ForgotPassword(context.Background(), &entities.ForgotPasswordRequest{Email: email}, "ip")
		require.NoError(t, err, email)
	}
	assert.Empty(t, kc.actionsFor)
}

func TestForgotPassword_RateLimited(t *testing.T) {
	deny := func(context.Context, string, int, time.Duration) bool { return false }
	err := newUC(&fakeKeycloak{}, &fakeUsers{}, nil, deny).ForgotPassword(
		context.Background(), &entities.ForgotPasswordRequest{Email: "ana@x.com"}, "ip")

	var apiErr *apperrors.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, "too_many_requests", apiErr.Code)
}

func TestGetMe_FallsBackToClaimsWithoutLocalUser(t *testing.T) {
	me, err := newUC(&fakeKeycloak{}, &fakeUsers{}, nil, nil).GetMe(context.Background(), claims)
	require.NoError(t, err)
	assert.Equal(t, "Ana", me.Name)
	assert.Equal(t, "ana@x.com", me.Email)
}

func TestUpdateMe_UpdatesKeycloakAndLocalUser(t *testing.T) {
	local := &userEntities.UserEntity{ID: uuid.New(), KeycloakUserID: "kc-1", Name: "Ana", Email: "ana@x.com", OrganizationID: "org", UserType: "owner"}
	users := &fakeUsers{byKeycloak: map[string]*userEntities.UserEntity{"kc-1": local}}
	kc := &fakeKeycloak{}

	me, err := newUC(kc, users, nil, nil).UpdateMe(context.Background(), claims, &entities.UpdateMeRequest{Name: "  Ana   Maria  Souza "})

	require.NoError(t, err)
	assert.Equal(t, [2]string{"Ana", "Maria Souza"}, kc.nameUpdate)
	assert.Equal(t, "Ana Maria Souza", users.updated.Name)
	assert.Equal(t, "Ana Maria Souza", me.Name)
	assert.Equal(t, "owner", me.UserType)
}

func TestChangePassword_RequiresCurrentPassword(t *testing.T) {
	kc := &fakeKeycloak{}
	wrong := func(context.Context, string, string) error { return errors.New("401") }

	err := newUC(kc, &fakeUsers{}, wrong, nil).ChangePassword(context.Background(), claims,
		&entities.ChangePasswordRequest{CurrentPassword: "old-pass", NewPassword: "new-pass-123"})

	var apiErr *apperrors.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, "invalid_current_password", apiErr.Code)
	assert.Empty(t, kc.newPassword)
}

func TestChangePassword_SetsNewPassword(t *testing.T) {
	kc := &fakeKeycloak{}
	err := newUC(kc, &fakeUsers{}, nil, nil).ChangePassword(context.Background(), claims,
		&entities.ChangePasswordRequest{CurrentPassword: "old-pass", NewPassword: "new-pass-123"})

	require.NoError(t, err)
	assert.Equal(t, "new-pass-123", kc.newPassword)
}

func TestChangePassword_PolicyRejection(t *testing.T) {
	kc := &fakeKeycloak{passwordErr: apperrors.ServiceError("policy")}
	err := newUC(kc, &fakeUsers{}, nil, nil).ChangePassword(context.Background(), claims,
		&entities.ChangePasswordRequest{CurrentPassword: "old-pass", NewPassword: "new-pass-123"})

	var apiErr *apperrors.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, "password_rejected", apiErr.Code)
}
