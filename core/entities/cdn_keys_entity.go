package entities

// CdnKeysEntity holds the credentials and configuration for CDN authentication.
type CdnKeysEntity struct {
	ClientID     string
	ClientSecret string
	Bucket       string
}
