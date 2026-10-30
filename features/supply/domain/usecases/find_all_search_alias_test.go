package usecases_test

import (
	"net/http"
	"testing"

	"github.com/RodolfoBonis/spooliq/core/validation"
)

// TestSupplyFindAll_SearchAlias verifies the list accepts `search` as an alias of
// `q` (q wins when both are present), matching the other catalog lists.
func TestSupplyFindAll_SearchAlias(t *testing.T) {
	validation.Register()

	t.Run("search alias forwarded when q absent", func(t *testing.T) {
		repo := &fakeSupplyRepo{}
		_, c := testCtx(http.MethodGet, "/supplies?search=led", "")
		newSupplyUC(repo).FindAll(c)
		if repo.lastSearch != "led" {
			t.Errorf("search alias = %q, want %q", repo.lastSearch, "led")
		}
	})

	t.Run("q wins over search", func(t *testing.T) {
		repo := &fakeSupplyRepo{}
		_, c := testCtx(http.MethodGet, "/supplies?q=creme&search=led", "")
		newSupplyUC(repo).FindAll(c)
		if repo.lastSearch != "creme" {
			t.Errorf("search = %q, want %q (q must win)", repo.lastSearch, "creme")
		}
	})
}
