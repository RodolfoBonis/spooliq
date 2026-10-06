package repositories

import "gorm.io/gorm"

// presetDefaultIndexName is the partial unique index guaranteeing at most one
// default preset per (organization, type) among non-deleted rows.
const presetDefaultIndexName = "uniq_presets_org_type_default"

// presetDefaultMigrationLockKey is a fixed advisory-lock key serializing this
// migration across concurrently-starting replicas, so two instances never race
// on the dedupe + index creation.
const presetDefaultMigrationLockKey = 918273645

// MigrateDefaults makes the "single default per (organization, type)" invariant
// durable at the database level.
//
// It is safe to run on existing data, idempotent, and safe to run concurrently
// from multiple replicas: the whole operation runs inside a transaction holding
// a fixed advisory lock, and the index is created with IF NOT EXISTS.
//  1. DEDUPE: for every (organization_id, type) with more than one default, keep
//     only the most recently updated one and clear is_default on the rest.
//     Without this, creating the unique index would fail on legacy data.
//  2. Create a PARTIAL UNIQUE INDEX on (organization_id, type) WHERE is_default
//     AND deleted_at IS NULL.
func MigrateDefaults(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Serialize concurrent replicas running this migration.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", presetDefaultMigrationLockKey).Error; err != nil {
			return err
		}

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
		if err := tx.Exec(dedupe).Error; err != nil {
			return err
		}

		createIndex := `
			CREATE UNIQUE INDEX IF NOT EXISTS ` + presetDefaultIndexName + `
			ON presets (organization_id, type)
			WHERE is_default AND deleted_at IS NULL
		`
		return tx.Exec(createIndex).Error
	})
}
