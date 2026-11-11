package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
)

// recordedRequest captures the salient parts of an inbound admin request.
type recordedRequest struct {
	method string
	path   string
	auth   string
}

// --- test logger -------------------------------------------------------------

// kcNoopLogger satisfies logger.Logger without side effects.
type kcNoopLogger struct{}

func (kcNoopLogger) Debug(context.Context, string, ...logger.Fields)   {}
func (kcNoopLogger) Info(context.Context, string, ...logger.Fields)    {}
func (kcNoopLogger) Warning(context.Context, string, ...logger.Fields) {}
func (kcNoopLogger) Error(context.Context, string, ...logger.Fields)   {}
func (kcNoopLogger) Fatal(context.Context, string, ...logger.Fields)   {}
func (kcNoopLogger) Panic(context.Context, string, ...logger.Fields)   {}
func (n kcNoopLogger) With(logger.Fields) logger.Logger                { return n }
func (kcNoopLogger) LogError(context.Context, string, error)           {}

const (
	testRealm        = "spooliq"
	testClientID     = "spooliq-admin-svc"
	testClientSecret = "super-secret"
	testAccessToken  = "test-access-token-1"
)

// newTestService builds a KeycloakAdminService pointed at the given base URL with
// the admin service-account credentials configured.
func newTestService(t *testing.T, baseURL string) *KeycloakAdminService {
	t.Helper()
	cfg := &config.AppConfig{
		Keycloak: entities.KeyCloakDataEntity{
			Host:              baseURL,
			Realm:             testRealm,
			AdminClientID:     testClientID,
			AdminClientSecret: testClientSecret,
		},
	}
	svc, ok := NewKeycloakAdminService(cfg, kcNoopLogger{}).(*KeycloakAdminService)
	if !ok {
		t.Fatalf("expected *KeycloakAdminService")
	}
	return svc
}

// --- token acquisition -------------------------------------------------------

func TestGetAccessToken_UsesClientCredentials(t *testing.T) {
	var gotPath, gotGrant, gotClientID, gotClientSecret, gotContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		gotGrant = r.Form.Get("grant_type")
		gotClientID = r.Form.Get("client_id")
		gotClientSecret = r.Form.Get("client_secret")
		writeToken(w, testAccessToken, 300)
	}))
	defer server.Close()

	svc := newTestService(t, server.URL)

	token, err := svc.getAccessToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != testAccessToken {
		t.Fatalf("token = %q, want %q", token, testAccessToken)
	}

	wantPath := "/realms/" + testRealm + "/protocol/openid-connect/token"
	if gotPath != wantPath {
		t.Errorf("token path = %q, want %q", gotPath, wantPath)
	}
	if gotGrant != "client_credentials" {
		t.Errorf("grant_type = %q, want client_credentials", gotGrant)
	}
	if gotClientID != testClientID {
		t.Errorf("client_id = %q, want %q", gotClientID, testClientID)
	}
	if gotClientSecret != testClientSecret {
		t.Errorf("client_secret = %q, want %q", gotClientSecret, testClientSecret)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("content-type = %q", gotContentType)
	}
}

func TestGetAccessToken_CachedAndRefreshedAfterExpiry(t *testing.T) {
	var tokenCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&tokenCalls, 1)
		writeToken(w, "token-"+strconv.Itoa(int(n)), 300)
	}))
	defer server.Close()

	svc := newTestService(t, server.URL)

	// Controllable clock.
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base
	var mu sync.Mutex
	svc.now = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return current
	}

	// First call fetches.
	tok1, err := svc.getAccessToken(context.Background())
	if err != nil {
		t.Fatalf("first token: %v", err)
	}
	// Second call within validity uses cache.
	tok2, err := svc.getAccessToken(context.Background())
	if err != nil {
		t.Fatalf("second token: %v", err)
	}
	if tok1 != tok2 {
		t.Fatalf("expected cached token, got %q then %q", tok1, tok2)
	}
	if got := atomic.LoadInt32(&tokenCalls); got != 1 {
		t.Fatalf("token fetches = %d, want 1 (cache hit)", got)
	}

	// Advance past expiry (300s - 30s buffer = 270s valid window).
	mu.Lock()
	current = base.Add(271 * time.Second)
	mu.Unlock()

	tok3, err := svc.getAccessToken(context.Background())
	if err != nil {
		t.Fatalf("third token: %v", err)
	}
	if tok3 == tok1 {
		t.Fatalf("expected refreshed token after expiry, still got %q", tok3)
	}
	if got := atomic.LoadInt32(&tokenCalls); got != 2 {
		t.Fatalf("token fetches = %d, want 2 (refresh)", got)
	}
}

func TestGetAccessToken_401ReturnsCleanError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized_client"}`))
	}))
	defer server.Close()

	svc := newTestService(t, server.URL)

	token, err := svc.getAccessToken(context.Background())
	if err == nil {
		t.Fatalf("expected error on 401, got token %q", token)
	}
	if err.Type != entities.ErrService {
		t.Errorf("error type = %v, want ErrService", err.Type)
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestGetAccessToken_MissingCredentials(t *testing.T) {
	cfg := &config.AppConfig{
		Keycloak: entities.KeyCloakDataEntity{
			Host:  "http://example.invalid",
			Realm: testRealm,
			// No AdminClientID / AdminClientSecret.
		},
	}
	svc := NewKeycloakAdminService(cfg, kcNoopLogger{}).(*KeycloakAdminService)

	_, err := svc.getAccessToken(context.Background())
	if err == nil {
		t.Fatal("expected error when admin credentials are missing")
	}
	if err.Type != entities.ErrService {
		t.Errorf("error type = %v, want ErrService", err.Type)
	}
}

func TestGetAccessToken_ConcurrentSingleFetch(t *testing.T) {
	var tokenCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tokenCalls, 1)
		// Simulate latency so goroutines pile up on the mutex.
		time.Sleep(20 * time.Millisecond)
		writeToken(w, testAccessToken, 300)
	}))
	defer server.Close()

	svc := newTestService(t, server.URL)

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if _, err := svc.getAccessToken(context.Background()); err != nil {
				t.Errorf("concurrent token fetch: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&tokenCalls); got != 1 {
		t.Fatalf("token fetches = %d, want 1 (single fetch under concurrency)", got)
	}
}

// --- admin methods -----------------------------------------------------------

func TestAdminMethods_HitExpectedRealmPathsWithBearer(t *testing.T) {
	var (
		mu       sync.Mutex
		requests []recordedRequest
	)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Token endpoint.
		if r.URL.Path == "/realms/"+testRealm+"/protocol/openid-connect/token" {
			writeToken(w, testAccessToken, 300)
			return
		}

		mu.Lock()
		requests = append(requests, recordedRequest{method: r.Method, path: r.URL.Path, auth: r.Header.Get("Authorization")})
		mu.Unlock()

		switch {
		// GetUserByEmail (used inside CreateUser) -> empty list means "not found".
		case r.Method == http.MethodGet && r.URL.Path == "/admin/realms/"+testRealm+"/users":
			_, _ = w.Write([]byte(`[]`))
		// CreateUser POST.
		case r.Method == http.MethodPost && r.URL.Path == "/admin/realms/"+testRealm+"/users":
			w.Header().Set("Location", server.URL+"/admin/realms/"+testRealm+"/users/new-user-id")
			w.WriteHeader(http.StatusCreated)
		// AssignRoleToUser GET role.
		case r.Method == http.MethodGet && r.URL.Path == "/admin/realms/"+testRealm+"/roles/Owner":
			_, _ = w.Write([]byte(`{"id":"role-id","name":"Owner"}`))
		// GetOrCreateGroup GET search (empty -> create).
		case r.Method == http.MethodGet && r.URL.Path == "/admin/realms/"+testRealm+"/groups":
			_, _ = w.Write([]byte(`[]`))
		// GetOrCreateGroup POST create.
		case r.Method == http.MethodPost && r.URL.Path == "/admin/realms/"+testRealm+"/groups":
			w.Header().Set("Location", server.URL+"/admin/realms/"+testRealm+"/groups/new-group-id")
			w.WriteHeader(http.StatusCreated)
		// SetGroupAttributes GET group.
		case r.Method == http.MethodGet && r.URL.Path == "/admin/realms/"+testRealm+"/groups/group-123":
			_, _ = w.Write([]byte(`{"id":"group-123","name":"org"}`))
		default:
			// All other PUT/POST writes succeed with no body.
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()

	if _, err := svc.CreateUser(ctx, KeycloakUserRequest{Email: "a@b.com", Username: "a@b.com"}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := svc.SetUserPassword(ctx, "user-1", "pw"); err != nil {
		t.Fatalf("SetUserPassword: %v", err)
	}
	if err := svc.AssignRoleToUser(ctx, "user-1", "Owner"); err != nil {
		t.Fatalf("AssignRoleToUser: %v", err)
	}
	if _, err := svc.GetOrCreateGroup(ctx, "org"); err != nil {
		t.Fatalf("GetOrCreateGroup: %v", err)
	}
	if err := svc.SetGroupAttributes(ctx, "group-123", map[string][]string{"k": {"v"}}); err != nil {
		t.Fatalf("SetGroupAttributes: %v", err)
	}
	if err := svc.AddUserToGroup(ctx, "user-1", "group-123"); err != nil {
		t.Fatalf("AddUserToGroup: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Every admin request must target /admin/realms/{REALM}/ and carry the Bearer token.
	wantBearer := "Bearer " + testAccessToken
	realmPrefix := "/admin/realms/" + testRealm + "/"
	for _, rec := range requests {
		if rec.auth != wantBearer {
			t.Errorf("%s %s: auth = %q, want %q", rec.method, rec.path, rec.auth, wantBearer)
		}
		if len(rec.path) < len(realmPrefix) || rec.path[:len(realmPrefix)] != realmPrefix {
			t.Errorf("request path %q does not target app realm prefix %q", rec.path, realmPrefix)
		}
	}

	// Spot-check that each expected method path was observed.
	assertObserved(t, requests, http.MethodPost, "/admin/realms/"+testRealm+"/users")
	assertObserved(t, requests, http.MethodPut, "/admin/realms/"+testRealm+"/users/user-1/reset-password")
	assertObserved(t, requests, http.MethodGet, "/admin/realms/"+testRealm+"/roles/Owner")
	assertObserved(t, requests, http.MethodPost, "/admin/realms/"+testRealm+"/users/user-1/role-mappings/realm")
	assertObserved(t, requests, http.MethodPost, "/admin/realms/"+testRealm+"/groups")
	assertObserved(t, requests, http.MethodPut, "/admin/realms/"+testRealm+"/groups/group-123")
	assertObserved(t, requests, http.MethodPut, "/admin/realms/"+testRealm+"/users/user-1/groups/group-123")
}

func TestCreateUser_ReturnsIDFromLocation(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/realms/"+testRealm+"/protocol/openid-connect/token":
			writeToken(w, testAccessToken, 300)
		case r.Method == http.MethodGet && r.URL.Path == "/admin/realms/"+testRealm+"/users":
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/admin/realms/"+testRealm+"/users":
			w.Header().Set("Location", server.URL+"/admin/realms/"+testRealm+"/users/abc-123")
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	svc := newTestService(t, server.URL)
	id, err := svc.CreateUser(context.Background(), KeycloakUserRequest{Email: "x@y.com"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if id != "abc-123" {
		t.Fatalf("id = %q, want abc-123", id)
	}
}

// --- helpers -----------------------------------------------------------------

func writeToken(w http.ResponseWriter, accessToken string, expiresIn int) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(KeycloakTokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   expiresIn,
		TokenType:   "Bearer",
	})
}

func assertObserved(t *testing.T, requests []recordedRequest, method, path string) {
	t.Helper()
	for _, r := range requests {
		if r.method == method && r.path == path {
			return
		}
	}
	t.Errorf("expected a %s %s request, but it was not observed", method, path)
}
