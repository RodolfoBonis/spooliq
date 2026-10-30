package entities

// ForgotPasswordRequest asks for a password reset e-mail.
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
}

// UpdateMeRequest updates the authenticated user's own profile.
type UpdateMeRequest struct {
	Name string `json:"name" validate:"required,min=2,max=255"`
}

// ChangePasswordRequest changes the authenticated user's own password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}

// MeResponse is the authenticated user's own profile.
type MeResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	UserType       string `json:"user_type,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
}
