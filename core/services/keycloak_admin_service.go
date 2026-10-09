package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	apperrors "github.com/RodolfoBonis/spooliq/core/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// tokenExpiryBuffer is how long before the real expiry we proactively refresh
// the admin token, to avoid racing the Keycloak clock on in-flight requests.
const tokenExpiryBuffer = 30 * time.Second

// errAdminNotConfigured is returned (as the cause) when the admin service-account
// credentials are missing. It lets admin calls fail with a clear message instead
// of crashing the whole application at startup.
var errAdminNotConfigured = errors.New("keycloak admin service account credentials are not configured (KEYCLOAK_CLIENT_ID / KEYCLOAK_CLIENT_SECRET)")

// IKeycloakAdminService defines the interface for Keycloak Admin API interactions.
type IKeycloakAdminService interface {
	CreateUser(ctx context.Context, req KeycloakUserRequest) (string, *apperrors.AppError)
	SetUserPassword(ctx context.Context, userID, password string) *apperrors.AppError
	AssignRoleToUser(ctx context.Context, userID, roleName string) *apperrors.AppError
	AddUserToGroup(ctx context.Context, userID, groupID string) *apperrors.AppError
	SetUserAttributes(ctx context.Context, userID string, attributes map[string][]string) *apperrors.AppError
	GetOrCreateGroup(ctx context.Context, groupName string) (string, *apperrors.AppError)
	SetGroupAttributes(ctx context.Context, groupID string, attributes map[string][]string) *apperrors.AppError
	GetUserByEmail(ctx context.Context, email string) (*KeycloakUserResponse, *apperrors.AppError)
	ExecuteActionsEmail(ctx context.Context, userID string, actions []string, lifespan time.Duration) *apperrors.AppError
	UpdateUserName(ctx context.Context, userID, firstName, lastName string) *apperrors.AppError
}

// KeycloakAdminService implements IKeycloakAdminService.
//
// It authenticates to the Keycloak Admin REST API using the confidential client
// `spooliq-admin-svc` via the client_credentials grant on the application realm
// (REALM), NOT a human admin on the master realm. The service account must hold
// the realm-management roles manage-users, view-users, query-groups and
// view-realm.
type KeycloakAdminService struct {
	baseURL      string
	realm        string
	clientID     string
	clientSecret string
	logger       logger.Logger
	client       *http.Client

	// now is injectable so tests can control token expiry. Defaults to time.Now.
	now func() time.Time

	// mu guards the cached token below, making token acquisition concurrency-safe
	// and ensuring concurrent callers trigger at most one token fetch.
	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// KeycloakUserRequest represents a request to create a user in Keycloak
type KeycloakUserRequest struct {
	Username      string              `json:"username"`
	Email         string              `json:"email"`
	EmailVerified bool                `json:"emailVerified"`
	Enabled       bool                `json:"enabled"`
	FirstName     string              `json:"firstName"`
	LastName      string              `json:"lastName"`
	Attributes    map[string][]string `json:"attributes,omitempty"`
}

// KeycloakUserResponse represents a user response from Keycloak
type KeycloakUserResponse struct {
	ID               string              `json:"id"`
	Username         string              `json:"username"`
	Email            string              `json:"email"`
	EmailVerified    bool                `json:"emailVerified"`
	Enabled          bool                `json:"enabled"`
	FirstName        string              `json:"firstName"`
	LastName         string              `json:"lastName"`
	Attributes       map[string][]string `json:"attributes,omitempty"`
	CreatedTimestamp int64               `json:"createdTimestamp"`
}

// KeycloakPasswordRequest represents a request to set a user password
type KeycloakPasswordRequest struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	Temporary bool   `json:"temporary"`
}

// KeycloakTokenResponse represents an access token response from Keycloak
type KeycloakTokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
}

// KeycloakGroupRequest represents a request to create a group
type KeycloakGroupRequest struct {
	Name       string              `json:"name"`
	Attributes map[string][]string `json:"attributes,omitempty"`
}

// KeycloakGroupResponse represents a group response from Keycloak
type KeycloakGroupResponse struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Path       string              `json:"path"`
	Attributes map[string][]string `json:"attributes,omitempty"`
}

// NewKeycloakAdminService creates a new KeycloakAdminService instance.
//
// If the service-account credentials are missing it logs a clear error but does
// NOT crash the application; admin calls will then fail with a clear error.
func NewKeycloakAdminService(cfg *config.AppConfig, log logger.Logger) IKeycloakAdminService {
	if cfg.Keycloak.AdminClientID == "" || cfg.Keycloak.AdminClientSecret == "" {
		log.Error(context.Background(), "Keycloak admin service account is not configured; admin operations (sign-up, user creation) will fail", map[string]interface{}{
			"missing_env":            "KEYCLOAK_CLIENT_ID and/or KEYCLOAK_CLIENT_SECRET",
			"admin_client_id_set":    cfg.Keycloak.AdminClientID != "",
			"admin_client_secret_ok": cfg.Keycloak.AdminClientSecret != "",
		})
	}

	return &KeycloakAdminService{
		baseURL:      cfg.Keycloak.Host,
		realm:        cfg.Keycloak.Realm,
		clientID:     cfg.Keycloak.AdminClientID,
		clientSecret: cfg.Keycloak.AdminClientSecret,
		logger:       log,
		now:          time.Now,
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

// getAccessToken returns a valid admin access token, fetching a new one via the
// client_credentials grant when the cache is empty or near expiry.
//
// The whole check-and-fetch runs under s.mu so that concurrent callers wait for
// a single in-flight fetch instead of each triggering their own.
func (s *KeycloakAdminService) getAccessToken(ctx context.Context) (string, *apperrors.AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return the cached token while it is still valid.
	if s.accessToken != "" && s.now().Before(s.tokenExpiry) {
		return s.accessToken, nil
	}

	if s.clientID == "" || s.clientSecret == "" {
		s.logger.Error(ctx, "Keycloak admin service account is not configured", map[string]interface{}{
			"missing_env": "KEYCLOAK_CLIENT_ID / KEYCLOAK_CLIENT_SECRET",
		})
		return "", apperrors.NewAppError(entities.ErrService, "Serviço administrativo do Keycloak não configurado", nil, errAdminNotConfigured)
	}

	// client_credentials grant on the APPLICATION realm (REALM), using the
	// confidential service-account client. No human admin, no master realm.
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", s.baseURL, s.realm)

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", s.clientID)
	data.Set("client_secret", s.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		s.logger.Error(ctx, "Failed to create Keycloak token request", map[string]interface{}{"error": err.Error()})
		return "", apperrors.NewAppError(entities.ErrService, "Falha ao autenticar com o Keycloak", nil, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error(ctx, "Failed to get Keycloak access token", map[string]interface{}{"error": err.Error()})
		return "", apperrors.NewAppError(entities.ErrService, "Falha ao autenticar com o Keycloak", nil, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		// Log status and body for observability; the body here never contains our
		// secret (it is an OAuth error payload).
		s.logger.Error(ctx, "Keycloak token request failed", map[string]interface{}{
			"status": resp.StatusCode,
			"body":   string(body),
		})
		return "", apperrors.NewAppError(entities.ErrService, "Falha ao autenticar com o Keycloak", nil, fmt.Errorf("token request returned status %d", resp.StatusCode))
	}

	var tokenResp KeycloakTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		s.logger.Error(ctx, "Failed to decode Keycloak token response", map[string]interface{}{"error": err.Error()})
		return "", apperrors.NewAppError(entities.ErrService, "Falha ao processar resposta do Keycloak", nil, err)
	}

	if tokenResp.AccessToken == "" {
		s.logger.Error(ctx, "Keycloak token response had no access_token", map[string]interface{}{"status": resp.StatusCode})
		return "", apperrors.NewAppError(entities.ErrService, "Falha ao autenticar com o Keycloak", nil, errors.New("empty access_token in token response"))
	}

	s.accessToken = tokenResp.AccessToken
	// Refresh shortly before the real expiry. Guard tiny lifetimes.
	ttl := time.Duration(tokenResp.ExpiresIn) * time.Second
	if ttl > tokenExpiryBuffer {
		ttl -= tokenExpiryBuffer
	} else if ttl <= 0 {
		ttl = 0
	}
	s.tokenExpiry = s.now().Add(ttl)

	return s.accessToken, nil
}

// doRequest performs an authenticated request to the Keycloak Admin API on the
// application realm (/admin/realms/{REALM}/...).
func (s *KeycloakAdminService) doRequest(ctx context.Context, method, path string, body interface{}, response interface{}) *apperrors.AppError {
	token, tokenErr := s.getAccessToken(ctx)
	if tokenErr != nil {
		return tokenErr
	}

	requestURL := fmt.Sprintf("%s/admin/realms/%s/%s", s.baseURL, s.realm, path)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			s.logger.Error(ctx, "Failed to marshal request body", map[string]interface{}{"error": err.Error()})
			return apperrors.NewAppError(entities.ErrService, "Falha ao processar requisição", nil, err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, reqBody)
	if err != nil {
		s.logger.Error(ctx, "Failed to create HTTP request", map[string]interface{}{"error": err.Error()})
		return apperrors.NewAppError(entities.ErrService, "Falha na comunicação com o Keycloak", nil, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error(ctx, "Failed to send HTTP request to Keycloak", map[string]interface{}{"error": err.Error()})
		return apperrors.NewAppError(entities.ErrService, "Falha na comunicação com o Keycloak", nil, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error(ctx, "Failed to read response body", map[string]interface{}{"error": err.Error()})
		return apperrors.NewAppError(entities.ErrService, "Falha ao processar resposta do Keycloak", nil, err)
	}

	if resp.StatusCode >= 400 {
		s.logger.Error(ctx, "Keycloak API returned an error", map[string]interface{}{
			"status_code":   resp.StatusCode,
			"response_body": string(respBody),
			"url":           requestURL,
		})
		return apperrors.NewAppError(entities.ErrService, "Falha na operação administrativa do Keycloak", nil, fmt.Errorf("keycloak admin API returned status %d", resp.StatusCode))
	}

	if response != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, response); err != nil {
			s.logger.Error(ctx, "Failed to unmarshal response", map[string]interface{}{"error": err.Error()})
			return apperrors.NewAppError(entities.ErrService, "Falha ao processar resposta do Keycloak", nil, err)
		}
	}

	// Handle 201 Created with Location header (for resource creation).
	if resp.StatusCode == http.StatusCreated {
		if strPtr, ok := response.(*string); ok {
			*strPtr = extractIDFromLocation(resp.Header.Get("Location"))
		}
	}

	return nil
}

// extractIDFromLocation extracts the trailing ID from a Keycloak Location header
// of the form ".../users/{id}".
func extractIDFromLocation(location string) string {
	if location == "" {
		return ""
	}
	parts := bytes.Split([]byte(location), []byte("/"))
	if len(parts) == 0 {
		return ""
	}
	return string(parts[len(parts)-1])
}

// CreateUser creates a new user in Keycloak on the application realm.
func (s *KeycloakAdminService) CreateUser(ctx context.Context, req KeycloakUserRequest) (string, *apperrors.AppError) {
	// First, check if user already exists.
	existingUser, err := s.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return "", apperrors.NewAppError(entities.ErrConflict, "Já existe um usuário com este e-mail", nil, nil)
	}

	var userID string
	if createErr := s.doRequest(ctx, http.MethodPost, "users", req, &userID); createErr != nil {
		return "", createErr
	}

	return userID, nil
}

// SetUserPassword sets a password for a user
func (s *KeycloakAdminService) SetUserPassword(ctx context.Context, userID, password string) *apperrors.AppError {
	path := fmt.Sprintf("users/%s/reset-password", userID)

	passwordReq := KeycloakPasswordRequest{
		Type:      "password",
		Value:     password,
		Temporary: false,
	}

	return s.doRequest(ctx, http.MethodPut, path, passwordReq, nil)
}

// AssignRoleToUser assigns a realm role to a user
func (s *KeycloakAdminService) AssignRoleToUser(ctx context.Context, userID, roleName string) *apperrors.AppError {
	// First, get the role representation
	var role map[string]interface{}
	if err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("roles/%s", roleName), nil, &role); err != nil {
		return err
	}

	if role == nil {
		return apperrors.NewAppError(entities.ErrNotFound, "Papel (role) não encontrado", nil, nil)
	}

	// Assign the role to the user
	path := fmt.Sprintf("users/%s/role-mappings/realm", userID)
	return s.doRequest(ctx, http.MethodPost, path, []map[string]interface{}{role}, nil)
}

// AddUserToGroup adds a user to a group
func (s *KeycloakAdminService) AddUserToGroup(ctx context.Context, userID, groupID string) *apperrors.AppError {
	path := fmt.Sprintf("users/%s/groups/%s", userID, groupID)
	return s.doRequest(ctx, http.MethodPut, path, nil, nil)
}

// SetUserAttributes sets custom attributes for a user
func (s *KeycloakAdminService) SetUserAttributes(ctx context.Context, userID string, attributes map[string][]string) *apperrors.AppError {
	// Get current user
	var user KeycloakUserResponse
	if err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("users/%s", userID), nil, &user); err != nil {
		return err
	}

	// Update attributes
	if user.Attributes == nil {
		user.Attributes = make(map[string][]string)
	}
	for k, v := range attributes {
		user.Attributes[k] = v
	}

	// Update user
	path := fmt.Sprintf("users/%s", userID)
	return s.doRequest(ctx, http.MethodPut, path, user, nil)
}

// GetOrCreateGroup gets an existing group by name or creates it if it doesn't exist
func (s *KeycloakAdminService) GetOrCreateGroup(ctx context.Context, groupName string) (string, *apperrors.AppError) {
	// Search for existing group
	var groups []KeycloakGroupResponse
	if err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("groups?search=%s", url.QueryEscape(groupName)), nil, &groups); err != nil {
		return "", err
	}

	// Check if group exists
	for _, group := range groups {
		if group.Name == groupName {
			return group.ID, nil
		}
	}

	// Create new group
	groupReq := KeycloakGroupRequest{
		Name: groupName,
	}

	var groupID string
	if err := s.doRequest(ctx, http.MethodPost, "groups", groupReq, &groupID); err != nil {
		return "", err
	}

	// If groupID is empty, fetch the created group
	if groupID == "" {
		if err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("groups?search=%s", url.QueryEscape(groupName)), nil, &groups); err != nil {
			return "", err
		}
		if len(groups) > 0 {
			groupID = groups[0].ID
		}
	}

	return groupID, nil
}

// SetGroupAttributes sets custom attributes for a group
func (s *KeycloakAdminService) SetGroupAttributes(ctx context.Context, groupID string, attributes map[string][]string) *apperrors.AppError {
	// Get current group
	var group KeycloakGroupResponse
	if err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("groups/%s", groupID), nil, &group); err != nil {
		return err
	}

	// Update attributes
	if group.Attributes == nil {
		group.Attributes = make(map[string][]string)
	}
	for k, v := range attributes {
		group.Attributes[k] = v
	}

	// Update group
	path := fmt.Sprintf("groups/%s", groupID)
	return s.doRequest(ctx, http.MethodPut, path, group, nil)
}

// GetUserByEmail retrieves a user by email
func (s *KeycloakAdminService) GetUserByEmail(ctx context.Context, email string) (*KeycloakUserResponse, *apperrors.AppError) {
	var users []KeycloakUserResponse
	if err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("users?email=%s&exact=true", url.QueryEscape(email)), nil, &users); err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, nil
	}

	return &users[0], nil
}

// ExecuteActionsEmail asks Keycloak to e-mail the user a link to perform the
// given required actions (e.g. UPDATE_PASSWORD). The link expires after
// lifespan. Keycloak sends the e-mail with the realm's SMTP settings.
func (s *KeycloakAdminService) ExecuteActionsEmail(ctx context.Context, userID string, actions []string, lifespan time.Duration) *apperrors.AppError {
	path := fmt.Sprintf("users/%s/execute-actions-email?lifespan=%d", url.PathEscape(userID), int(lifespan.Seconds()))
	return s.doRequest(ctx, http.MethodPut, path, actions, nil)
}

// UpdateUserName updates only the first and last name of a Keycloak user.
func (s *KeycloakAdminService) UpdateUserName(ctx context.Context, userID, firstName, lastName string) *apperrors.AppError {
	body := map[string]string{"firstName": firstName, "lastName": lastName}
	return s.doRequest(ctx, http.MethodPut, "users/"+url.PathEscape(userID), body, nil)
}
