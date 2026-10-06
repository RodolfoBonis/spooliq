// Package repositories defines the data-access contracts for the slicer feature.
package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/slicer/domain/suggest"
)

// FilamentCatalogRepository loads the organization's filaments as suggestion
// candidates. It is intentionally narrow: the slicer feature only needs the
// fields used for color/material matching, loaded once per request.
type FilamentCatalogRepository interface {
	// LoadCandidates returns all active filaments of the organization reduced to
	// suggestion candidates. Returns an empty slice when the org has none.
	LoadCandidates(ctx context.Context, organizationID string) ([]suggest.Candidate, error)
}
