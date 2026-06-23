package entities

// CdnKeysEntity holds the MinIO connection + bucket for direct uploads (replaces the rb-cdn proxy).
type CdnKeysEntity struct {
	Bucket    string
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
}
