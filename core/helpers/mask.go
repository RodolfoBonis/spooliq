package helpers

import "strings"

// publicBudgetsPrefix is the path segment that precedes a public budget share
// token. Anything between it and the next "/" or "?" is the secret token.
const publicBudgetsPrefix = "/public/budgets/"

// infinitePayWebhookPrefix precedes "/<organization_id>/<token>" in the InfinitePay
// webhook path. The final segment is a per-integration secret that must never reach
// logs, Sentry or traces.
const infinitePayWebhookPrefix = "/webhooks/infinitepay/"

// MaskPublicBudgetToken replaces the share token in a request path/URL with "***"
// so it never reaches logs, Sentry events or traces. It keeps everything before the
// token and any suffix after it (e.g. "/pdf", "/approve", "/reject") and the query
// string intact. Strings without the prefix are returned unchanged.
func MaskPublicBudgetToken(path string) string {
	idx := strings.Index(path, publicBudgetsPrefix)
	if idx < 0 {
		return path
	}
	start := idx + len(publicBudgetsPrefix)
	rest := path[start:]

	end := len(rest)
	for i, r := range rest {
		if r == '/' || r == '?' {
			end = i
			break
		}
	}
	if end == 0 {
		// Nothing after the prefix (e.g. the collection path itself): leave as-is.
		return path
	}
	return path[:start] + "***" + rest[end:]
}

// maskInfinitePayWebhookToken masks the per-integration webhook token in an
// InfinitePay webhook path/URL. The path shape is
// ".../webhooks/infinitepay/<organization_id>/<token>[/...][?...]"; the
// organization_id is kept (useful for correlation) and only <token> is replaced
// with "***". Strings without the prefix are returned unchanged.
func maskInfinitePayWebhookToken(path string) string {
	idx := strings.Index(path, infinitePayWebhookPrefix)
	if idx < 0 {
		return path
	}
	start := idx + len(infinitePayWebhookPrefix)
	rest := path[start:]

	// Skip the organization_id segment (up to the next "/").
	orgEnd := -1
	for i, r := range rest {
		if r == '/' {
			orgEnd = i
			break
		}
		if r == '?' {
			// Only the org segment is present; no token to mask.
			return path
		}
	}
	if orgEnd < 0 {
		// No "/" after the org: nothing that looks like a token to mask.
		return path
	}

	tokenStart := start + orgEnd + 1
	tokenRest := path[tokenStart:]
	tokenEnd := len(tokenRest)
	for i, r := range tokenRest {
		if r == '/' || r == '?' {
			tokenEnd = i
			break
		}
	}
	if tokenEnd == 0 {
		return path
	}
	return path[:tokenStart] + "***" + tokenRest[tokenEnd:]
}

// MaskSensitivePath redacts every secret token that can appear in a request
// path/URL before it is logged, reported to Sentry or attached to a trace. It
// currently masks the public budget share token and the InfinitePay webhook token.
// Strings carrying neither are returned unchanged.
func MaskSensitivePath(path string) string {
	return maskInfinitePayWebhookToken(MaskPublicBudgetToken(path))
}
