package migrations

import (
	"context"
	"fmt"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/jinzhu/gorm"
)

// MigrationRepository handles database operations for migrations
type MigrationRepository struct {
	db     *gorm.DB
	logger logger.Logger
}

// NewMigrationRepository creates a new migration repository instance
func NewMigrationRepository(db *gorm.DB, logger logger.Logger) *MigrationRepository {
	return &MigrationRepository{
		db:     db,
		logger: logger,
	}
}

// EnsureMigrationTableExists creates the schema_migrations table if it doesn't exist
func (r *MigrationRepository) EnsureMigrationTableExists() error {
	ctx := context.Background()

	// Check if table exists
	if r.db.HasTable(&SchemaMigration{}) {
		r.logger.Info(ctx, "Migration table already exists", nil)
		return nil
	}

	// Create the migrations table
	if err := r.db.CreateTable(&SchemaMigration{}).Error; err != nil {
		r.logger.Error(ctx, "Failed to create schema_migrations table", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	r.logger.Info(ctx, "Created schema_migrations table", nil)
	return nil
}

// IsApplied checks if a migration version has been applied
func (r *MigrationRepository) IsApplied(version string) (bool, error) {
	var count int
	err := r.db.Model(&SchemaMigration{}).Where("version = ?", version).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check if migration %s is applied: %w", version, err)
	}
	return count > 0, nil
}

// RecordAppliedMigration records that a migration has been applied
func (r *MigrationRepository) RecordAppliedMigration(version, name string, executionTime time.Duration) error {
	ctx := context.Background()

	migration := SchemaMigration{
		Version:       version,
		Name:          name,
		ExecutionTime: int(executionTime.Milliseconds()),
	}

	if err := r.db.Create(&migration).Error; err != nil {
		r.logger.Error(ctx, "Failed to record applied migration", map[string]interface{}{
			"version": version,
			"name":    name,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to record migration %s: %w", version, err)
	}

	r.logger.Info(ctx, "Recorded applied migration", map[string]interface{}{
		"version":        version,
		"name":           name,
		"execution_time": executionTime.String(),
	})

	return nil
}

// RemoveAppliedMigration removes a migration record (for rollbacks)
func (r *MigrationRepository) RemoveAppliedMigration(version string) error {
	ctx := context.Background()

	if err := r.db.Where("version = ?", version).Delete(&SchemaMigration{}).Error; err != nil {
		r.logger.Error(ctx, "Failed to remove applied migration", map[string]interface{}{
			"version": version,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to remove migration %s: %w", version, err)
	}

	r.logger.Info(ctx, "Removed applied migration", map[string]interface{}{
		"version": version,
	})

	return nil
}

// GetAppliedMigrations returns all applied migrations
func (r *MigrationRepository) GetAppliedMigrations() ([]SchemaMigration, error) {
	var migrations []SchemaMigration
	if err := r.db.Order("version ASC").Find(&migrations).Error; err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}
	return migrations, nil
}

// GetLatestMigration returns the latest applied migration
func (r *MigrationRepository) GetLatestMigration() (*SchemaMigration, error) {
	var migration SchemaMigration
	err := r.db.Order("version DESC").First(&migration).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, nil // No migrations applied yet
		}
		return nil, fmt.Errorf("failed to get latest migration: %w", err)
	}
	return &migration, nil
}

// GetMigrationStatuses returns the status of all migrations
func (r *MigrationRepository) GetMigrationStatuses(allMigrations []Migration) ([]MigrationStatus, error) {
	appliedMigrations, err := r.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	// Create a map for faster lookup
	appliedMap := make(map[string]*SchemaMigration)
	for i := range appliedMigrations {
		appliedMap[appliedMigrations[i].Version] = &appliedMigrations[i]
	}

	statuses := make([]MigrationStatus, len(allMigrations))
	for i, migration := range allMigrations {
		applied, exists := appliedMap[migration.Version]
		statuses[i] = MigrationStatus{
			Version: migration.Version,
			Name:    migration.Name,
			Applied: exists,
		}
		if exists {
			statuses[i].AppliedAt = &applied.AppliedAt
		}
	}

	return statuses, nil
}