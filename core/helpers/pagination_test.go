package helpers

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
)

func newCtx(rawQuery string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?"+rawQuery, nil)
	return c
}

func defaultOpts() ListQueryOptions {
	return ListQueryOptions{
		DefaultPageSize: 20,
		SortWhitelist: map[string]string{
			"created_at": "created_at",
			"name":       "lower(name)",
		},
		DefaultSort: "created_at",
	}
}

func TestParseListQuery(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		opts         ListQueryOptions
		wantPage     int
		wantPageSize int
		wantSearch   string
		wantOrder    string
	}{
		{
			name:         "defaults when empty",
			query:        "",
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 20,
			wantSearch:   "",
			wantOrder:    "created_at desc",
		},
		{
			name:         "page below minimum clamps to 1",
			query:        "page=0",
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 20,
			wantOrder:    "created_at desc",
		},
		{
			name:         "negative page clamps to 1",
			query:        "page=-5",
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 20,
			wantOrder:    "created_at desc",
		},
		{
			name:         "page_size above max clamps to 100",
			query:        "page_size=9999",
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 100,
			wantOrder:    "created_at desc",
		},
		{
			name:         "page_size below 1 clamps to 1",
			query:        "page_size=0",
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 1,
			wantOrder:    "created_at desc",
		},
		{
			name:         "limit alias works",
			query:        "limit=37",
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 37,
			wantOrder:    "created_at desc",
		},
		{
			name:         "pageSize camelCase alias works",
			query:        "pageSize=45",
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 45,
			wantOrder:    "created_at desc",
		},
		{
			name:         "search alias q is trimmed",
			query:        "q=" + url.QueryEscape("  hello  "),
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 20,
			wantSearch:   "hello",
			wantOrder:    "created_at desc",
		},
		{
			name:         "search alias search works",
			query:        "search=" + url.QueryEscape("world"),
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 20,
			wantSearch:   "world",
			wantOrder:    "created_at desc",
		},
		{
			name:         "whitelisted sort_by resolves to column",
			query:        "sort_by=name&sort_dir=asc",
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 20,
			wantOrder:    "lower(name) asc",
		},
		{
			name:         "injection attempt falls back to default sort",
			query:        "sort_by=" + url.QueryEscape("name; DROP TABLE users"),
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 20,
			wantOrder:    "created_at desc",
		},
		{
			name:         "unknown sort_by falls back to default",
			query:        "sort_by=totally_unknown",
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 20,
			wantOrder:    "created_at desc",
		},
		{
			name:         "invalid sort_dir falls back to desc",
			query:        "sort_by=name&sort_dir=sideways",
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 20,
			wantOrder:    "lower(name) desc",
		},
		{
			name:         "sort_dir is case-insensitive",
			query:        "sort_by=name&sort_dir=ASC",
			opts:         defaultOpts(),
			wantPage:     1,
			wantPageSize: 20,
			wantOrder:    "lower(name) asc",
		},
		{
			name:         "default page size falls back to 20 when opts zero",
			query:        "",
			opts:         ListQueryOptions{SortWhitelist: map[string]string{"created_at": "created_at"}, DefaultSort: "created_at"},
			wantPage:     1,
			wantPageSize: 20,
			wantOrder:    "created_at desc",
		},
		{
			name:         "custom default page size honored",
			query:        "",
			opts:         ListQueryOptions{DefaultPageSize: 50, SortWhitelist: map[string]string{"created_at": "created_at"}, DefaultSort: "created_at"},
			wantPage:     1,
			wantPageSize: 50,
			wantOrder:    "created_at desc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseListQuery(newCtx(tt.query), tt.opts)
			if got.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", got.Page, tt.wantPage)
			}
			if got.PageSize != tt.wantPageSize {
				t.Errorf("PageSize = %d, want %d", got.PageSize, tt.wantPageSize)
			}
			if got.Search != tt.wantSearch {
				t.Errorf("Search = %q, want %q", got.Search, tt.wantSearch)
			}
			if got.OrderClause() != tt.wantOrder {
				t.Errorf("OrderClause() = %q, want %q", got.OrderClause(), tt.wantOrder)
			}
		})
	}
}

func TestListQueryOffsetLimit(t *testing.T) {
	q := ParseListQuery(newCtx("page=3&page_size=25"), defaultOpts())
	if q.Offset() != 50 {
		t.Errorf("Offset() = %d, want 50", q.Offset())
	}
	if q.Limit() != 25 {
		t.Errorf("Limit() = %d, want 25", q.Limit())
	}
}

func TestOrderClauseEmptyWhenNoWhitelist(t *testing.T) {
	q := ParseListQuery(newCtx("sort_by=name"), ListQueryOptions{DefaultPageSize: 10})
	if q.OrderClause() != "" {
		t.Errorf("OrderClause() = %q, want empty", q.OrderClause())
	}
}

func TestNewPage(t *testing.T) {
	tests := []struct {
		name           string
		total          int64
		pageSize       int
		page           int
		wantTotalPages int
	}{
		{"exact multiple", 40, 20, 1, 2},
		{"with remainder rounds up", 41, 20, 1, 3},
		{"single partial page", 5, 20, 1, 1},
		{"zero rows", 0, 20, 1, 0},
		{"one over page", 21, 20, 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := ListQuery{Page: tt.page, PageSize: tt.pageSize}
			p := NewPage([]int{1, 2, 3}, tt.total, q)
			if p.TotalPages != tt.wantTotalPages {
				t.Errorf("TotalPages = %d, want %d", p.TotalPages, tt.wantTotalPages)
			}
			if p.Total != tt.total {
				t.Errorf("Total = %d, want %d", p.Total, tt.total)
			}
			if p.Page != tt.page {
				t.Errorf("Page = %d, want %d", p.Page, tt.page)
			}
			if p.PageSize != tt.pageSize {
				t.Errorf("PageSize = %d, want %d", p.PageSize, tt.pageSize)
			}
		})
	}
}

func TestNewPageDataNeverNil(t *testing.T) {
	q := ListQuery{Page: 1, PageSize: 20}
	p := NewPage[string](nil, 0, q)
	if p.Data == nil {
		t.Fatal("Data must never be nil")
	}
	if len(p.Data) != 0 {
		t.Errorf("Data len = %d, want 0", len(p.Data))
	}
}

func tieOpts(tie string) ListQueryOptions {
	o := defaultOpts()
	o.TieBreaker = tie
	return o
}

func TestOrderClauseTieBreaker(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		opts      ListQueryOptions
		wantOrder string
	}{
		{
			name:      "tie-breaker appended to default sort",
			query:     "",
			opts:      tieOpts("id"),
			wantOrder: "created_at desc, id asc",
		},
		{
			name:      "tie-breaker appended to explicit sort",
			query:     "sort_by=name&sort_dir=asc",
			opts:      tieOpts("filaments.id"),
			wantOrder: "lower(name) asc, filaments.id asc",
		},
		{
			name:      "tie-breaker skipped when equal to sort column",
			query:     "sort_by=created_at&sort_dir=asc",
			opts:      tieOpts("created_at"),
			wantOrder: "created_at asc",
		},
		{
			name:      "no tie-breaker keeps clause unchanged",
			query:     "",
			opts:      defaultOpts(),
			wantOrder: "created_at desc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseListQuery(newCtx(tt.query), tt.opts).OrderClause()
			if got != tt.wantOrder {
				t.Errorf("OrderClause() = %q, want %q", got, tt.wantOrder)
			}
		})
	}
}

func TestOrderClauseDir(t *testing.T) {
	q := ParseListQuery(newCtx(""), tieOpts("id"))
	if got := q.OrderClauseDir("asc"); got != "created_at asc, id asc" {
		t.Errorf("OrderClauseDir(asc) = %q, want %q", got, "created_at asc, id asc")
	}
	if got := q.OrderClauseDir("DESC"); got != "created_at desc, id asc" {
		t.Errorf("OrderClauseDir(DESC) = %q, want %q", got, "created_at desc, id asc")
	}
}
