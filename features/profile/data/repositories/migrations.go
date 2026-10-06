package repositories

import "gorm.io/gorm"

// profileDefaultIndexName is the partial unique index guaranteeing at most one
// default print profile per organization among non-deleted rows.
const profileDefaultIndexName = "uniq_print_profiles_org_default"

// profileDefaultMigrationLockKey is a fixed advisory-lock key serializing this
// migration across concurrently-starting replicas.
const profileDefaultMigrationLockKey = 918273646

// MigrateDefaults makes the "single default profile per organization" invariant
// durable at the database level. It is safe on existing data, idempotent, and
// safe to run concurrently from multiple replicas: the whole operation runs in a
// transaction holding a fixed advisory lock, and the index uses IF NOT EXISTS.
//  1. DEDUPE: keep only the most recently updated default per organization.
//  2. Create a PARTIAL UNIQUE INDEX on (organization_id) WHERE is_default AND
//     deleted_at IS NULL.
func MigrateDefaults(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", profileDefaultMigrationLockKey).Error; err != nil {
			return err
		}

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
		if err := tx.Exec(dedupe).Error; err != nil {
			return err
		}

		createIndex := `
			CREATE UNIQUE INDEX IF NOT EXISTS ` + profileDefaultIndexName + `
			ON print_profiles (organization_id)
			WHERE is_default AND deleted_at IS NULL
		`
		return tx.Exec(createIndex).Error
	})
}
