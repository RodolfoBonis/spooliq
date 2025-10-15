package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
)

// IKeycloakAdminService defines the interface for Keycloak Admin API interactions.
type IKeycloakAdminService interface {
	CreateUser(ctx context.Context, req KeycloakUserRequest) (string, *errors.AppError)
	SetUserPassword(ctx context.Context, userID, password string) *errors.AppError
	AssignRoleToUser(ctx context.Context, userID, roleName string) *errors.AppError
	AddUserToGroup(ctx context.Context, userID, groupID string) *errors.AppError
	SetUserAttributes(ctx context.Context, userID string, attributes map[string][]string) *errors.AppError
	GetOrCreateGroup(ctx context.Context, groupName string) (string, *errors.AppError)
	SetGroupAttributes(ctx context.Context, groupID string, attributes map[string][]string) *errors.AppError
	GetUserByEmail(ctx context.Context, email string) (*KeycloakUserResponse, *errors.AppError)
}

// KeycloakAdminService implements IKeycloakAdminService.
type KeycloakAdminService struct {
	baseURL       string
	realm         string
	clientID      string
	clientSecret  string
	adminUsername string
	adminPassword string
	logger        logger.Logger
	client        *http.Client
	accessToken   string
	tokenExpiry   time.Time
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
func NewKeycloakAdminService(cfg *config.AppConfig, logger logger.Logger) IKeycloakAdminService {
	return &KeycloakAdminService{
		baseURL:       cfg.Keycloak.Host,
		realm:         cfg.Keycloak.Realm,
		clientID:      cfg.Keycloak.ClientID,
		clientSecret:  cfg.Keycloak.ClientSecret,
		adminUsername: cfg.Keycloak.AdminUsername,
		adminPassword: cfg.Keycloak.AdminPassword,
		logger:        logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// getAccessToken obtains an access token from Keycloak using admin credentials
func (s *KeycloakAdminService) getAccessToken(ctx context.Context) *errors.AppError {
	// Check if token is still valid
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
		return nil
	}

	// Use master realm for admin authentication
	tokenURL := fmt.Sprintf("%s/realms/master/protocol/openid-connect/token", s.baseURL)

	// Use password grant type with admin credentials (URL-encoded)
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("client_id", "admin-cli")
	data.Set("username", s.adminUsername)
	data.Set("password", s.adminPassword)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		s.logger.Error(ctx, "Failed to create token request", map[string]interface{}{"error": err.Error()})
		return errors.NewAppError(entities.ErrService, "Failed to authenticate with Keycloak", nil, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error(ctx, "Failed to get access token", map[string]interface{}{"error": err.Error()})
		return errors.NewAppError(entities.ErrService, "Failed to authenticate with Keycloak", nil, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Error(ctx, "Keycloak token request failed", map[string]interface{}{
			"status": resp.StatusCode,
			"body":   string(body),
		})
		return errors.NewAppError(entities.ErrService, "Failed to authenticate with Keycloak", nil, fmt.Errorf("status: %d", resp.StatusCode))
	}

	var tokenResp KeycloakTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		s.logger.Error(ctx, "Failed to decode token response", map[string]interface{}{"error": err.Error()})
		return errors.NewAppError(entities.ErrService, "Failed to process Keycloak response", nil, err)
	}

	s.accessToken = tokenResp.AccessToken
	s.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second) // Refresh 60s before expiry

	return nil
}

// doRequest performs an authenticated request to Keycloak Admin API
func (s *KeycloakAdminService) doRequest(ctx context.Context, method, path string, body interface{}, response interface{}) *errors.AppError {
	if err := s.getAccessToken(ctx); err != nil {
		return err
	}

	url := fmt.Sprintf("%s/admin/realms/%s/%s", s.baseURL, s.realm, path)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			s.logger.Error(ctx, "Failed to marshal request body", map[string]interface{}{"error": err.Error()})
			return errors.NewAppError(entities.ErrService, "Failed to process request", nil, err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		s.logger.Error(ctx, "Failed to create HTTP request", map[string]interface{}{"error": err.Error()})
		return errors.NewAppError(entities.ErrService, "Failed to communicate with Keycloak", nil, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.accessToken))

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error(ctx, "Failed to send HTTP request to Keycloak", map[string]interface{}{"error": err.Error()})
		return errors.NewAppError(entities.ErrService, "Failed to communicate with Keycloak", nil, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error(ctx, "Failed to read response body", map[string]interface{}{"error": err.Error()})
		return errors.NewAppError(entities.ErrService, "Failed to process Keycloak response", nil, err)
	}

	if resp.StatusCode >= 400 {
		s.logger.Error(ctx, "Keycloak API returned an error", map[string]interface{}{
			"status_code":   resp.StatusCode,
			"response_body": string(respBody),
			"url":           url,
		})
		return errors.NewAppError(entities.ErrService, fmt.Sprintf("Keycloak API error: %s", string(respBody)), nil, fmt.Errorf("status: %d", resp.StatusCode))
	}

	if response != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, response); err != nil {
			s.logger.Error(ctx, "Failed to unmarshal response", map[string]interface{}{"error": err.Error()})
			return errors.NewAppError(entities.ErrService, "Failed to process Keycloak response", nil, err)
		}
	}

	// Handle 201 Created with Location header (for user creation)
	if resp.StatusCode == http.StatusCreated && response == nil {
		location := resp.Header.Get("Location")
		if location != "" {
			// Extract user ID from location header
			// Location format: .../users/{user-id}
			parts := bytes.Split([]byte(location), []byte("/"))
			if len(parts) > 0 {
				userID := string(parts[len(parts)-1])
				if strPtr, ok := response.(*string); ok {
					*strPtr = userID
				}
			}
		}
	}

	return nil
}

// CreateUser creates a new user in Keycloak
func (s *KeycloakAdminService) CreateUser(ctx context.Context, req KeycloakUserRequest) (string, *errors.AppError) {
	var userID string

	// First, check if user already exists
	existingUser, err := s.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return "", errors.NewAppError(entities.ErrConflict, "User with this email already exists", nil, nil)
	}

	httpReq, buildErr := s.buildCreateUserRequest(ctx, req)
	if buildErr != nil {
		s.logger.Error(ctx, "Failed to build create user request", map[string]interface{}{"error": buildErr.Error()})
		return "", errors.NewAppError(entities.ErrService, "Failed to create user in Keycloak", nil, buildErr)
	}

	resp, httpErr := s.client.Do(httpReq)
	if httpErr != nil {
		s.logger.Error(ctx, "Failed to create user", map[string]interface{}{"error": httpErr.Error()})
		return "", errors.NewAppError(entities.ErrService, "Failed to create user in Keycloak", nil, httpErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Error(ctx, "Keycloak user creation failed", map[string]interface{}{
			"status": resp.StatusCode,
			"body":   string(body),
		})
		return "", errors.NewAppError(entities.ErrService, fmt.Sprintf("Failed to create user: %s", string(body)), nil, fmt.Errorf("status: %d", resp.StatusCode))
	}

	// Extract user ID from Location header
	location := resp.Header.Get("Location")
	if location != "" {
		parts := bytes.Split([]byte(location), []byte("/"))
		if len(parts) > 0 {
			userID = string(parts[len(parts)-1])
		}
	}

	return userID, nil
}

func (s *KeycloakAdminService) buildCreateUserRequest(ctx context.Context, req KeycloakUserRequest) (*http.Request, error) {
	if err := s.getAccessToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to get access token: %v", err)
	}

	url := fmt.Sprintf("%s/admin/realms/%s/users", s.baseURL, s.realm)

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.accessToken))

	return httpReq, nil
}

// SetUserPassword sets a password for a user
func (s *KeycloakAdminService) SetUserPassword(ctx context.Context, userID, password string) *errors.AppError {
	path := fmt.Sprintf("users/%s/reset-password", userID)

	passwordReq := KeycloakPasswordRequest{
		Type:      "password",
		Value:     password,
		Temporary: false,
	}

	return s.doRequest(ctx, http.MethodPut, path, passwordReq, nil)
}

// AssignRoleToUser assigns a realm role to a user
func (s *KeycloakAdminService) AssignRoleToUser(ctx context.Context, userID, roleName string) *errors.AppError {
	// First, get the role representation
	var role map[string]interface{}
	if err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("roles/%s", roleName), nil, &role); err != nil {
		return err
	}

	if role == nil {
		return errors.NewAppError(entities.ErrNotFound, "Role not found", nil, nil)
	}

	// Assign the role to the user
	path := fmt.Sprintf("users/%s/role-mappings/realm", userID)
	return s.doRequest(ctx, http.MethodPost, path, []map[string]interface{}{role}, nil)
}

// AddUserToGroup adds a user to a group
func (s *KeycloakAdminService) AddUserToGroup(ctx context.Context, userID, groupID string) *errors.AppError {
	path := fmt.Sprintf("users/%s/groups/%s", userID, groupID)
	return s.doRequest(ctx, http.MethodPut, path, nil, nil)
}

// SetUserAttributes sets custom attributes for a user
func (s *KeycloakAdminService) SetUserAttributes(ctx context.Context, userID string, attributes map[string][]string) *errors.AppError {
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
func (s *KeycloakAdminService) GetOrCreateGroup(ctx context.Context, groupName string) (string, *errors.AppError) {
	// Search for existing group
	var groups []KeycloakGroupResponse
	if err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("groups?search=%s", groupName), nil, &groups); err != nil {
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
		if err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("groups?search=%s", groupName), nil, &groups); err != nil {
			return "", err
		}
		if len(groups) > 0 {
			groupID = groups[0].ID
		}
	}

	return groupID, nil
}

// SetGroupAttributes sets custom attributes for a group
func (s *KeycloakAdminService) SetGroupAttributes(ctx context.Context, groupID string, attributes map[string][]string) *errors.AppError {
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
func (s *KeycloakAdminService) GetUserByEmail(ctx context.Context, email string) (*KeycloakUserResponse, *errors.AppError) {
	var users []KeycloakUserResponse
	if err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("users?email=%s&exact=true", email), nil, &users); err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, nil
	}

	return &users[0], nil
}
