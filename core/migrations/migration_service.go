package migrations

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/jinzhu/gorm"
)

// MigrationService handles running and managing database migrations
type MigrationService struct {
	db         *gorm.DB
	repository *MigrationRepository
	logger     logger.Logger
	migrations []Migration
}

// NewMigrationService creates a new migration service instance
func NewMigrationService(db *gorm.DB, logger logger.Logger) *MigrationService {
	repository := NewMigrationRepository(db, logger)
	return &MigrationService{
		db:         db,
		repository: repository,
		logger:     logger,
		migrations: []Migration{}, // Will be populated by RegisterMigrations
	}
}

// RegisterMigrations registers all available migrations
func (s *MigrationService) RegisterMigrations(migrations []Migration) {
	// Sort migrations by version to ensure correct order
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	s.migrations = migrations
	s.logger.Info(context.Background(), "Registered migrations", map[string]interface{}{
		"count": len(migrations),
	})
}

// Run executes all pending migrations
func (s *MigrationService) Run() error {
	ctx := context.Background()

	// Ensure migration table exists
	if err := s.repository.EnsureMigrationTableExists(); err != nil {
		return fmt.Errorf("failed to ensure migration table exists: %w", err)
	}

	s.logger.Info(ctx, "Starting migrations", map[string]interface{}{
		"total_migrations": len(s.migrations),
	})

	executed := 0
	for _, migration := range s.migrations {
		applied, err := s.repository.IsApplied(migration.Version)
		if err != nil {
			return fmt.Errorf("failed to check if migration %s is applied: %w", migration.Version, err)
		}

		if applied {
			s.logger.Info(ctx, "Skipping already applied migration", map[string]interface{}{
				"version": migration.Version,
				"name":    migration.Name,
			})
			continue
		}

		if err := s.runSingleMigration(migration); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration.Version, err)
		}
		executed++
	}

	s.logger.Info(ctx, "Migrations completed", map[string]interface{}{
		"executed": executed,
		"total":    len(s.migrations),
	})

	return nil
}

// RunTo executes migrations up to a specific version
func (s *MigrationService) RunTo(targetVersion string) error {
	ctx := context.Background()

	// Ensure migration table exists
	if err := s.repository.EnsureMigrationTableExists(); err != nil {
		return fmt.Errorf("failed to ensure migration table exists: %w", err)
	}

	s.logger.Info(ctx, "Running migrations to version", map[string]interface{}{
		"target_version": targetVersion,
	})

	executed := 0
	for _, migration := range s.migrations {
		if migration.Version > targetVersion {
			break
		}

		applied, err := s.repository.IsApplied(migration.Version)
		if err != nil {
			return fmt.Errorf("failed to check if migration %s is applied: %w", migration.Version, err)
		}

		if applied {
			continue
		}

		if err := s.runSingleMigration(migration); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration.Version, err)
		}
		executed++
	}

	s.logger.Info(ctx, "Migrations to version completed", map[string]interface{}{
		"target_version": targetVersion,
		"executed":       executed,
	})

	return nil
}

// Rollback rolls back the last migration
func (s *MigrationService) Rollback() error {
	ctx := context.Background()

	latest, err := s.repository.GetLatestMigration()
	if err != nil {
		return fmt.Errorf("failed to get latest migration: %w", err)
	}

	if latest == nil {
		s.logger.Info(ctx, "No migrations to rollback", nil)
		return nil
	}

	// Find the migration definition
	var targetMigration *Migration
	for _, migration := range s.migrations {
		if migration.Version == latest.Version {
			targetMigration = &migration
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("migration definition not found for version %s", latest.Version)
	}

	if targetMigration.Down == nil {
		return fmt.Errorf("migration %s does not support rollback (no Down function)", latest.Version)
	}

	s.logger.Info(ctx, "Rolling back migration", map[string]interface{}{
		"version": latest.Version,
		"name":    latest.Name,
	})

	start := time.Now()

	// Start transaction
	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Execute rollback
	if err := targetMigration.Down(tx); err != nil {
		tx.Rollback()
		s.logger.Error(ctx, "Migration rollback failed", map[string]interface{}{
			"version": latest.Version,
			"name":    latest.Name,
			"error":   err.Error(),
		})
		return fmt.Errorf("migration rollback failed: %w", err)
	}

	// Remove migration record
	if err := s.repository.RemoveAppliedMigration(latest.Version); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit rollback transaction: %w", err)
	}

	duration := time.Since(start)
	s.logger.Info(ctx, "Migration rolled back successfully", map[string]interface{}{
		"version":        latest.Version,
		"name":           latest.Name,
		"execution_time": duration.String(),
	})

	return nil
}

// Status returns the status of all migrations
func (s *MigrationService) Status() ([]MigrationStatus, error) {
	return s.repository.GetMigrationStatuses(s.migrations)
}

// DryRun shows what migrations would be executed without actually running them
func (s *MigrationService) DryRun() ([]Migration, error) {
	var pendingMigrations []Migration

	for _, migration := range s.migrations {
		applied, err := s.repository.IsApplied(migration.Version)
		if err != nil {
			return nil, fmt.Errorf("failed to check if migration %s is applied: %w", migration.Version, err)
		}

		if !applied {
			pendingMigrations = append(pendingMigrations, migration)
		}
	}

	return pendingMigrations, nil
}

// runSingleMigration executes a single migration
func (s *MigrationService) runSingleMigration(migration Migration) error {
	ctx := context.Background()

	s.logger.Info(ctx, "Running migration", map[string]interface{}{
		"version": migration.Version,
		"name":    migration.Name,
	})

	start := time.Now()

	// Start transaction
	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Execute migration
	if err := migration.Up(tx); err != nil {
		tx.Rollback()
		s.logger.Error(ctx, "Migration failed", map[string]interface{}{
			"version": migration.Version,
			"name":    migration.Name,
			"error":   err.Error(),
		})
		return fmt.Errorf("migration failed: %w", err)
	}

	// Record migration
	duration := time.Since(start)
	if err := s.repository.RecordAppliedMigration(migration.Version, migration.Name, duration); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	s.logger.Info(ctx, "Migration completed successfully", map[string]interface{}{
		"version":        migration.Version,
		"name":           migration.Name,
		"execution_time": duration.String(),
	})

	return nil
}
