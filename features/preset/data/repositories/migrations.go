package repositories

import "gorm.io/gorm"

// presetDefaultIndexName is the partial unique index guaranteeing at most one
// default preset per (organization, type) among non-deleted rows.
const presetDefaultIndexName = "uniq_presets_org_type_default"

// MigrateDefaults makes the "single default per (organization, type)" invariant
// durable at the database level.
//
// It is safe to run on existing data and idempotent:
//  1. It first DEDUPES: for every (organization_id, type) that currently has more
//     than one default, it keeps only the most recently updated one and clears
//     is_default on the rest. Without this, creating the unique index below would
//     fail on legacy data that predates the invariant.
//  2. It then creates a PARTIAL UNIQUE INDEX on (organization_id, type) WHERE
//     is_default AND deleted_at IS NULL, so the database itself rejects a second
//     live default for the same pair.
//
// Both statements use IF NOT EXISTS / idempotent patterns so repeated startups
// are no-ops.
func MigrateDefaults(db *gorm.DB) error {
	// 1. Dedupe: keep the most recently updated default per (organization, type).
	dedupe := `
		UPDATE presets
		SET is_default = false
		WHERE is_default = true
		  AND deleted_at IS NULL
		  AND id NOT IN (
			SELECT DISTINCT ON (organization_id, type) id
			FROM presets
			WHERE is_default = true AND deleted_at IS NULL
			ORDER BY organization_id, type, updated_at DESC, id
		  )
	`
	if err := db.Exec(dedupe).Error; err != nil {
		return err
	}

	// 2. Partial unique index enforcing a single live default per (org, type).
	createIndex := `
		CREATE UNIQUE INDEX IF NOT EXISTS ` + presetDefaultIndexName + `
		ON presets (organization_id, type)
		WHERE is_default AND deleted_at IS NULL
	`
	return db.Exec(createIndex).Error
}
