package usecases

// ptrToStr converts *string to string, returning empty string if nil
func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
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
