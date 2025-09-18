package entities

// RequestLoginEntity represents the login request payload.
// @Description Login request data
// @Example {"email": "user@example.com", "password": "string"}

// RequestLoginEntity model info
// @description RequestLoginEntity model data
type RequestLoginEntity struct {
	// User email
	Email string `json:"email"`
	// User password
	Password string `json:"password"`
}
