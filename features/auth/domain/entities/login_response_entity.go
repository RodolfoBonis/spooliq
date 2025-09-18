package entities

// LoginResponseEntity represents the login response payload.
// @Description Login response data
// @Example {"accessToken": "jwt-token", "refreshToken": "refresh-token", "expiresIn": 3600}

// LoginResponseEntity model info
// @description LoginResponseEntity model data
type LoginResponseEntity struct {
	// Token to access this API
	AccessToken string `json:"accessToken"`
	// Token to refresh Access Token
	RefreshToken string `json:"refreshToken"`
	// Time to expires token in int
	ExpiresIn int `json:"expiresIn"`
}
