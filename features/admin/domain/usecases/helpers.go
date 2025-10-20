package usecases

import "github.com/google/uuid"

// ptrToStr converts *string to string, returning empty string if nil
func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// uuidPtrToStrPtr converts *uuid.UUID to *string, returning nil if input is nil
func uuidPtrToStrPtr(u *uuid.UUID) *string {
	if u == nil {
		return nil
	}
	str := u.String()
	return &str
}

// contains checks if a slice contains a specific value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
