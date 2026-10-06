// Package repositories implements the slicer feature's data access.
package repositories

import (
	"context"
	"fmt"

	domainRepositories "github.com/RodolfoBonis/spooliq/features/slicer/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/suggest"
	"gorm.io/gorm"
)

type filamentCatalogRepository struct {
	db *gorm.DB
}

// NewFilamentCatalogRepository creates a repository that loads org filaments as
// suggestion candidates.
func NewFilamentCatalogRepository(db *gorm.DB) domainRepositories.FilamentCatalogRepository {
	return &filamentCatalogRepository{db: db}
}

// candidateRow is the projection used for matching: just id, name, color and the
// material name (joined from materials).
type candidateRow struct {
	ID           string
	Name         string
	ColorHex     string
	MaterialName string
}

func (r *filamentCatalogRepository) LoadCandidates(ctx context.Context, organizationID string) ([]suggest.Candidate, error) {
	var rows []candidateRow
	err := r.db.WithContext(ctx).
		Table("filaments").
		Select("filaments.id AS id, filaments.name AS name, filaments.color_hex AS color_hex, materials.name AS material_name").
		Joins("LEFT JOIN materials ON materials.id = filaments.material_id").
		Where("filaments.organization_id = ? AND filaments.deleted_at IS NULL AND filaments.is_active = ?", organizationID, true).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load filament candidates: %w", err)
	}

	candidates := make([]suggest.Candidate, 0, len(rows))
	for _, row := range rows {
		candidates = append(candidates, suggest.Candidate{
			FilamentID: row.ID,
			Name:       row.Name,
			ColorHex:   row.ColorHex,
			Material:   row.MaterialName,
		})
	}
	return candidates, nil
}
