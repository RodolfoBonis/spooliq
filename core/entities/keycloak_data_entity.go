package entities

// KeyCloakDataEntity represents the data structure for Keycloak integration.
type KeyCloakDataEntity struct {
	ClientID     string
	ClientSecret string
	Realm        string
	Host         string
}
