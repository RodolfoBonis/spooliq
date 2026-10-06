package helpers

import "strings"

// likeEscaper escapes the LIKE/ILIKE wildcard metacharacters so a user-supplied
// search term is matched literally. The backslash is escaped first so it does
// not double-escape the replacements that follow.
var likeEscaper = strings.NewReplacer(
	`\`, `\\`,
	`%`, `\%`,
	`_`, `\_`,
)

// EscapeLike escapes the special characters (`\`, `%`, `_`) in a free-text
// search term so it can be safely embedded in a LIKE/ILIKE pattern. Callers must
// pair it with an explicit `ESCAPE '\'` clause, e.g.:
//
//	db.Where("name ILIKE ? ESCAPE '\\'", "%"+helpers.EscapeLike(term)+"%")
//
// Without escaping, a term containing `%` or `_` would act as a wildcard and
// return unintended matches.
func EscapeLike(s string) string {
	return likeEscaper.Replace(s)
}
