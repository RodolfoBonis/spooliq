package migrations

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/jinzhu/gorm"
	"github.com/schollz/progressbar/v3"
)

// SQLMigrationExecutor handles execution of SQL migrations
type SQLMigrationExecutor struct {
	db         *gorm.DB
	logger     logger.Logger
	repository *MigrationRepository
	scanner    *SQLMigrationScanner
}

// NewSQLMigrationExecutor creates a new SQL migration executor
func NewSQLMigrationExecutor(db *gorm.DB, logger logger.Logger, migrationsPath string) *SQLMigrationExecutor {
	return &SQLMigrationExecutor{
		db:         db,
		logger:     logger,
		repository: NewMigrationRepository(db, logger),
		scanner:    NewSQLMigrationScanner(migrationsPath),
	}
}

// RunAll executes all pending SQL migrations
func (e *SQLMigrationExecutor) RunAll() error {
	ctx := context.Background()

	// Temporarily disable SQL logging
	e.db.LogMode(false)
	defer func() {
		e.db.LogMode(true)
	}()

	// Ensure migration table exists
	if err := e.repository.EnsureMigrationTableExists(); err != nil {
		return fmt.Errorf("failed to ensure migration table exists: %w", err)
	}

	// Scan for migrations
	migrations, err := e.scanner.ScanMigrations()
	if err != nil {
		return fmt.Errorf("failed to scan migrations: %w", err)
	}

	// Count pending migrations
	pendingCount := 0
	for _, migration := range migrations {
		applied, err := e.repository.IsApplied(migration.Version)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}
		if !applied {
			pendingCount++
		}
	}

	if pendingCount == 0 {
		e.logger.Info(ctx, "✅ No pending migrations", nil)
		return nil
	}

	// Create progress bar
	bar := progressbar.NewOptions(pendingCount,
		progressbar.OptionSetDescription("📊 Running migrations..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerHead:    "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(100*time.Millisecond),
	)

	executed := 0
	for _, migration := range migrations {
		// Check if already applied
		applied, err := e.repository.IsApplied(migration.Version)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if applied {
			continue
		}

		// Update progress bar description
		bar.Describe(fmt.Sprintf("📊 Running migration: %s...", migration.Name))

		// Execute migration
		if err := e.executeMigrationSilent(migration, true); err != nil {
			bar.Finish()
			return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
		}

		executed++
		bar.Add(1)
	}

	bar.Describe("✅ Migrations completed successfully!")
	bar.Finish()
	fmt.Println()

	e.logger.Info(ctx, "Migrations completed", map[string]interface{}{
		"executed": executed,
		"total":    len(migrations),
	})

	return nil
}

// RunUp executes specific number of migrations
func (e *SQLMigrationExecutor) RunUp(count int) error {
	ctx := context.Background()

	// Ensure migration table exists
	if err := e.repository.EnsureMigrationTableExists(); err != nil {
		return fmt.Errorf("failed to ensure migration table exists: %w", err)
	}

	// Scan for migrations
	migrations, err := e.scanner.ScanMigrations()
	if err != nil {
		return fmt.Errorf("failed to scan migrations: %w", err)
	}

	executed := 0
	for _, migration := range migrations {
		if count > 0 && executed >= count {
			break
		}

		// Check if already applied
		applied, err := e.repository.IsApplied(migration.Version)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if applied {
			continue
		}

		// Execute migration
		if err := e.executeMigration(migration, true); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
		}

		executed++
	}

	e.logger.Info(ctx, "Migrations completed", map[string]interface{}{
		"executed": executed,
		"requested": count,
	})

	return nil
}

// RunDown rolls back specific number of migrations
func (e *SQLMigrationExecutor) RunDown(count int) error {
	ctx := context.Background()

	// Get applied migrations in reverse order
	appliedMigrations, err := e.repository.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	if len(appliedMigrations) == 0 {
		e.logger.Info(ctx, "No migrations to rollback", nil)
		return nil
	}

	// Scan for migration files
	migrationFiles, err := e.scanner.ScanMigrations()
	if err != nil {
		return fmt.Errorf("failed to scan migrations: %w", err)
	}

	// Create map for quick lookup
	migrationMap := make(map[string]SQLMigration)
	for _, m := range migrationFiles {
		migrationMap[m.Version] = m
	}

	// Rollback migrations
	rolled := 0
	for i := len(appliedMigrations) - 1; i >= 0 && rolled < count; i-- {
		applied := appliedMigrations[i]
		
		migration, exists := migrationMap[applied.Version]
		if !exists {
			e.logger.Warning(ctx, "Migration file not found for rollback", map[string]interface{}{
				"version": applied.Version,
			})
			continue
		}

		if migration.DownSQL == "" {
			e.logger.Warning(ctx, "No down migration available", map[string]interface{}{
				"version": migration.Version,
			})
			continue
		}

		// Execute rollback
		if err := e.executeMigration(migration, false); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", migration.Version, err)
		}

		rolled++
	}

	e.logger.Info(ctx, "Rollback completed", map[string]interface{}{
		"rolled_back": rolled,
	})

	return nil
}

// executeMigration executes a single migration (up or down)
func (e *SQLMigrationExecutor) executeMigration(migration SQLMigration, up bool) error {
	ctx := context.Background()

	sql := migration.UpSQL
	action := "applying"
	if !up {
		sql = migration.DownSQL
		action = "rolling back"
	}

	e.logger.Info(ctx, fmt.Sprintf("Migration %s", action), map[string]interface{}{
		"version": migration.Version,
		"name":    migration.Name,
	})

	start := time.Now()

	// Split SQL into individual statements
	statements := splitSQLStatements(sql)

	// Start transaction
	tx := e.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Execute each statement
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if err := tx.Exec(stmt).Error; err != nil {
			tx.Rollback()
			e.logger.Error(ctx, "Migration statement failed", map[string]interface{}{
				"version":   migration.Version,
				"statement": i + 1,
				"error":     err.Error(),
			})
			return fmt.Errorf("statement %d failed: %w", i+1, err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	duration := time.Since(start)

	// Record or remove migration
	if up {
		if err := e.repository.RecordAppliedMigration(migration.Version, migration.Name, duration); err != nil {
			e.logger.Warning(ctx, "Failed to record migration", map[string]interface{}{
				"version": migration.Version,
				"error":   err.Error(),
			})
		}
	} else {
		if err := e.repository.RemoveAppliedMigration(migration.Version); err != nil {
			e.logger.Warning(ctx, "Failed to remove migration record", map[string]interface{}{
				"version": migration.Version,
				"error":   err.Error(),
			})
		}
	}

	e.logger.Info(ctx, fmt.Sprintf("Migration %s successfully", action), map[string]interface{}{
		"version":  migration.Version,
		"duration": duration.String(),
	})

	return nil
}

// executeMigrationSilent executes a single migration without logging (for progress bar)
func (e *SQLMigrationExecutor) executeMigrationSilent(migration SQLMigration, up bool) error {
	sql := migration.UpSQL
	if !up {
		sql = migration.DownSQL
	}

	start := time.Now()

	// Split SQL into individual statements
	statements := splitSQLStatements(sql)

	// Start transaction
	tx := e.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Execute each statement
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if err := tx.Exec(stmt).Error; err != nil {
			tx.Rollback()
			preview := stmt
			if len(stmt) > 100 {
				preview = stmt[:100] + "..."
			}
			return fmt.Errorf("statement %d failed (SQL: %s): %w", i+1, preview, err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	duration := time.Since(start)

	// Record or remove migration
	if up {
		if err := e.repository.RecordAppliedMigration(migration.Version, migration.Name, duration); err != nil {
			return fmt.Errorf("failed to record migration: %w", err)
		}
	} else {
		if err := e.repository.RemoveAppliedMigration(migration.Version); err != nil {
			return fmt.Errorf("failed to remove migration record: %w", err)
		}
	}

	return nil
}

// GetStatus returns the status of all migrations
func (e *SQLMigrationExecutor) GetStatus() ([]MigrationStatus, error) {
	// Scan for migrations
	migrations, err := e.scanner.ScanMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to scan migrations: %w", err)
	}

	// Get applied migrations
	appliedMigrations, err := e.repository.GetAppliedMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Create map for quick lookup
	appliedMap := make(map[string]*SchemaMigration)
	for i := range appliedMigrations {
		appliedMap[appliedMigrations[i].Version] = &appliedMigrations[i]
	}

	// Build status list
	var statuses []MigrationStatus
	for _, migration := range migrations {
		status := MigrationStatus{
			Version: migration.Version,
			Name:    migration.Name,
			Applied: false,
		}

		if applied, exists := appliedMap[migration.Version]; exists {
			status.Applied = true
			status.AppliedAt = &applied.AppliedAt
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// Fresh drops all tables and reruns all migrations
func (e *SQLMigrationExecutor) Fresh() error {
	ctx := context.Background()
	
	e.logger.Warning(ctx, "Dropping all tables and starting fresh", nil)
	
	// Get all table names
	var tables []string
	query := `
		SELECT tablename 
		FROM pg_tables 
		WHERE schemaname = 'public'
	`
	
	rows, err := e.db.Raw(query).Rows()
	if err != nil {
		return fmt.Errorf("failed to get table list: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}
		tables = append(tables, tableName)
	}
	
	// Drop all tables
	for _, table := range tables {
		e.logger.Info(ctx, "Dropping table", map[string]interface{}{"table": table})
		if err := e.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error; err != nil {
			e.logger.Warning(ctx, "Failed to drop table", map[string]interface{}{
				"table": table,
				"error": err.Error(),
			})
		}
	}
	
	// Run all migrations
	return e.RunAll()
}

// Reset rolls back all migrations and reruns them
func (e *SQLMigrationExecutor) Reset() error {
	ctx := context.Background()
	
	e.logger.Info(ctx, "Resetting all migrations", nil)
	
	// Get count of applied migrations
	appliedMigrations, err := e.repository.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}
	
	// Rollback all
	if len(appliedMigrations) > 0 {
		if err := e.RunDown(len(appliedMigrations)); err != nil {
			return fmt.Errorf("failed to rollback migrations: %w", err)
		}
	}
	
	// Run all migrations
	return e.RunAll()
}

// splitSQLStatements splits SQL content into individual statements
func splitSQLStatements(sql string) []string {
	// Simple split by semicolon
	// TODO: Improve to handle semicolons inside strings
	statements := strings.Split(sql, ";")
	
	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" && !isComment(stmt) {
			result = append(result, stmt)
		}
	}
	
	return result
}

// isComment checks if a statement is entirely a comment (no SQL code)
func isComment(stmt string) bool {
	lines := strings.Split(strings.TrimSpace(stmt), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "--") && !strings.HasPrefix(line, "/*") {
			// Found a non-comment line, so this statement contains SQL
			return false
		}
	}
	// All lines are either empty or comments
	return true
}

// createDir creates a directory if it doesn't exist
func createDir(path string) error {
	return os.MkdirAll(path, 0755)
}