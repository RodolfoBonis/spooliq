package entities

// KeyCloakDataEntity represents the data structure for Keycloak integration.
type KeyCloakDataEntity struct {
	// ClientID / ClientSecret identify the public/confidential client used for
	// END-USER login (CLIENT_ID / CLIENT_SECRET). Do NOT reuse these for the
	// Admin REST API.
	ClientID     string
	ClientSecret string
	Realm        string
	Host         string
	// AdminClientID / AdminClientSecret are the credentials of the confidential
	// client `spooliq-admin-svc` whose SERVICE ACCOUNT holds the realm-management
	// roles (manage-users, view-users, query-groups, view-realm). They are used
	// to obtain an admin token via the client_credentials grant on the app realm.
	// Sourced from KEYCLOAK_CLIENT_ID / KEYCLOAK_CLIENT_SECRET and are distinct
	// from ClientID / ClientSecret above.
	AdminClientID     string
	AdminClientSecret string
}
