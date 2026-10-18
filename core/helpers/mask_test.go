package helpers

import "testing"

func TestMaskPublicBudgetToken(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain token", "/v1/public/budgets/abc123TOKEN", "/v1/public/budgets/***"},
		{"pdf suffix", "/v1/public/budgets/abc123/pdf", "/v1/public/budgets/***/pdf"},
		{"approve suffix", "/v1/public/budgets/abc123/approve", "/v1/public/budgets/***/approve"},
		{"reject suffix", "/v1/public/budgets/abc123/reject", "/v1/public/budgets/***/reject"},
		{"with query", "/v1/public/budgets/abc123?foo=bar", "/v1/public/budgets/***?foo=bar"},
		{"full url", "https://api.example.com/v1/public/budgets/secretTok/pdf", "https://api.example.com/v1/public/budgets/***/pdf"},
		{"no prefix", "/v1/budgets/123/share", "/v1/budgets/123/share"},
		{"collection path only", "/v1/public/budgets/", "/v1/public/budgets/"},
		{"unrelated", "/v1/health", "/v1/health"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaskPublicBudgetToken(tc.in); got != tc.want {
				t.Errorf("MaskPublicBudgetToken(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
