package repositories

import "gorm.io/gorm"

// profileDefaultIndexName is the partial unique index guaranteeing at most one
// default print profile per organization among non-deleted rows.
const profileDefaultIndexName = "uniq_print_profiles_org_default"

// MigrateDefaults makes the "single default profile per organization" invariant
// durable at the database level. It is safe to run on existing data and
// idempotent:
//  1. DEDUPE: for every organization with more than one default profile, keep
//     only the most recently updated one and clear is_default on the rest.
//  2. Create a PARTIAL UNIQUE INDEX on (organization_id) WHERE is_default AND
//     deleted_at IS NULL.
func MigrateDefaults(db *gorm.DB) error {
	dedupe := `
		UPDATE print_profiles
		SET is_default = false
		WHERE is_default = true
		  AND deleted_at IS NULL
		  AND id NOT IN (
			SELECT DISTINCT ON (organization_id) id
			FROM print_profiles
			WHERE is_default = true AND deleted_at IS NULL
			ORDER BY organization_id, updated_at DESC, id
		  )
	`
	if err := db.Exec(dedupe).Error; err != nil {
		return err
	}

	createIndex := `
		CREATE UNIQUE INDEX IF NOT EXISTS ` + profileDefaultIndexName + `
		ON print_profiles (organization_id)
		WHERE is_default AND deleted_at IS NULL
	`
	return db.Exec(createIndex).Error
}
