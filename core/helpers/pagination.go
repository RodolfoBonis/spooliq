package helpers

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Pagination defaults and limits. These are the single source of truth for
// list-endpoint behavior across the API.
const (
	defaultPage          = 1
	fallbackPageSize     = 20
	maxPageSize          = 100
	sortDirectionAsc     = "asc"
	sortDirectionDesc    = "desc"
	defaultSortDirection = sortDirectionDesc
)

// ListQueryOptions configures how ParseListQuery interprets a request.
//
// SortWhitelist maps the PUBLIC sort name (what the client sends in sort_by)
// to the ACTUAL database column used in the ORDER BY clause. Only names present
// in this map are ever accepted, which is what makes OrderClause() safe against
// SQL injection: the column name is never taken from user input directly.
type ListQueryOptions struct {
	// DefaultPageSize is the page size used when the client omits page_size.
	// When zero, it falls back to 20.
	DefaultPageSize int
	// SortWhitelist maps public sort names to safe column expressions.
	// Example: {"created_at": "created_at", "name": "lower(name)"}.
	SortWhitelist map[string]string
	// DefaultSort is the public sort name applied when the client omits
	// sort_by or sends one that is not whitelisted. It should be a key present
	// in SortWhitelist; otherwise OrderClause() returns an empty string.
	DefaultSort string
}

// ListQuery is the normalized, validated result of parsing list query params.
// It is safe to use directly in repository queries.
type ListQuery struct {
	// Page is the 1-based page number (>= 1).
	Page int
	// PageSize is the clamped page size (1..maxPageSize).
	PageSize int
	// Search is the trimmed free-text search term (may be empty).
	Search string
	// SortColumn is the resolved, whitelisted column expression (never raw
	// user input). Empty only when no valid default/whitelist was configured.
	SortColumn string
	// SortDir is normalized to "asc" or "desc".
	SortDir string
	// sortName is the public sort name, kept for diagnostics/echoing.
	sortName string
}

// Offset returns the SQL OFFSET for the current page.
func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.PageSize
}

// Limit returns the SQL LIMIT for the current page.
func (q ListQuery) Limit() int {
	return q.PageSize
}

// OrderClause returns a safe "column asc|desc" string built only from the
// whitelist, suitable for gorm's Order(). Returns an empty string when no
// column was resolved (so callers can fall back to their own default).
func (q ListQuery) OrderClause() string {
	if q.SortColumn == "" {
		return ""
	}
	return q.SortColumn + " " + q.SortDir
}

// SortName returns the resolved public sort name.
func (q ListQuery) SortName() string {
	return q.sortName
}

// ParseListQuery reads pagination, search and sort parameters from the request
// and returns a normalized, validated ListQuery.
//
// Accepted query parameters (with aliases):
//   - page: 1-based page number. Default 1, minimum 1.
//   - page_size (aliases: limit, pageSize): items per page. Default
//     opts.DefaultPageSize or 20, clamped to [1, 100].
//   - q (alias: search): free-text search, trimmed.
//   - sort_by: public sort name; must be in opts.SortWhitelist, otherwise
//     opts.DefaultSort is used.
//   - sort_dir: "asc" or "desc" (case-insensitive). Default "desc".
func ParseListQuery(c *gin.Context, opts ListQueryOptions) ListQuery {
	page := parsePositiveInt(c.Query("page"), defaultPage)
	if page < 1 {
		page = defaultPage
	}

	defaultSize := opts.DefaultPageSize
	if defaultSize <= 0 {
		defaultSize = fallbackPageSize
	}

	pageSize := parsePositiveInt(firstNonEmpty(c.Query("page_size"), c.Query("limit"), c.Query("pageSize")), defaultSize)
	pageSize = clamp(pageSize, 1, maxPageSize)

	search := strings.TrimSpace(firstNonEmpty(c.Query("q"), c.Query("search")))

	sortName, sortColumn := resolveSort(c.Query("sort_by"), opts)
	sortDir := normalizeSortDir(c.Query("sort_dir"))

	return ListQuery{
		Page:       page,
		PageSize:   pageSize,
		Search:     search,
		SortColumn: sortColumn,
		SortDir:    sortDir,
		sortName:   sortName,
	}
}

// resolveSort picks a public sort name and its safe column from the whitelist.
// A requested name is only honored when it is present in the whitelist; this is
// the injection-safe boundary. Otherwise opts.DefaultSort is used.
func resolveSort(requested string, opts ListQueryOptions) (name, column string) {
	if col, ok := opts.SortWhitelist[requested]; ok {
		return requested, col
	}
	if col, ok := opts.SortWhitelist[opts.DefaultSort]; ok {
		return opts.DefaultSort, col
	}
	return "", ""
}

func normalizeSortDir(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case sortDirectionAsc:
		return sortDirectionAsc
	case sortDirectionDesc:
		return sortDirectionDesc
	default:
		return defaultSortDirection
	}
}

func parsePositiveInt(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func clamp(v, minValue, maxValue int) int {
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}

// Page is the generic envelope returned by every list endpoint. T is the item
// type (usually a response entity). Data is always a non-nil slice so the JSON
// "data" field is never null.
type Page[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// NewPage builds a Page from the queried data, the total row count and the
// ListQuery used to fetch it. Data is normalized to an empty slice when nil so
// the serialized "data" is always [] and never null. TotalPages is computed
// with a ceiling division and is 0 when there are no rows.
func NewPage[T any](data []T, total int64, q ListQuery) Page[T] {
	if data == nil {
		data = []T{}
	}

	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = fallbackPageSize
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	return Page[T]{
		Data:       data,
		Total:      total,
		Page:       q.Page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
