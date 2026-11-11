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

func TestMaskSensitivePath(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"webhook token masked, org kept",
			"/v1/webhooks/infinitepay/org-123/SECRETWEBHOOKTOKEN",
			"/v1/webhooks/infinitepay/org-123/***",
		},
		{
			"webhook token with query",
			"/v1/webhooks/infinitepay/org-123/SECRET?x=1",
			"/v1/webhooks/infinitepay/org-123/***?x=1",
		},
		{
			"webhook token with trailing segment",
			"/v1/webhooks/infinitepay/org-123/SECRET/extra",
			"/v1/webhooks/infinitepay/org-123/***/extra",
		},
		{
			"webhook full url",
			"https://api.spooliq.com/v1/webhooks/infinitepay/abc/TOK",
			"https://api.spooliq.com/v1/webhooks/infinitepay/abc/***",
		},
		{
			"webhook only org, no token",
			"/v1/webhooks/infinitepay/org-123",
			"/v1/webhooks/infinitepay/org-123",
		},
		{
			"still masks public budget token",
			"/v1/public/budgets/secretTok/pdf",
			"/v1/public/budgets/***/pdf",
		},
		{
			"asaas webhook untouched",
			"/v1/webhooks/asaas",
			"/v1/webhooks/asaas",
		},
		{"unrelated", "/v1/health", "/v1/health"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaskSensitivePath(tc.in); got != tc.want {
				t.Errorf("MaskSensitivePath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
