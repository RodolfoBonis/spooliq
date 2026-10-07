package helpers

import "strings"

// publicBudgetsPrefix is the path segment that precedes a public budget share
// token. Anything between it and the next "/" or "?" is the secret token.
const publicBudgetsPrefix = "/public/budgets/"

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
