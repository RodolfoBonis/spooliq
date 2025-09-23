package migrations

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// SQLMigration represents a migration with SQL files
type SQLMigration struct {
	Version   string
	Name      string
	Timestamp time.Time
	UpSQL     string
	DownSQL   string
	Path      string
}

// SQLMigrationScanner scans directories for SQL migration files
type SQLMigrationScanner struct {
	migrationsPath string
}

// NewSQLMigrationScanner creates a new SQL migration scanner
func NewSQLMigrationScanner(migrationsPath string) *SQLMigrationScanner {
	return &SQLMigrationScanner{
		migrationsPath: migrationsPath,
	}
}

// ScanMigrations scans the migrations directory and returns all migrations
func (s *SQLMigrationScanner) ScanMigrations() ([]SQLMigration, error) {
	var migrations []SQLMigration

	// Read directory
	entries, err := ioutil.ReadDir(s.migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Pattern to match migration directories: YYYYMMDDHHMMSS_name
	migrationPattern := regexp.MustCompile(`^(\d{14})_(.+)$`)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if directory name matches pattern
		matches := migrationPattern.FindStringSubmatch(entry.Name())
		if len(matches) != 3 {
			continue
		}

		timestamp := matches[1]
		name := matches[2]

		// Parse timestamp
		parsedTime, err := time.Parse("20060102150405", timestamp)
		if err != nil {
			continue
		}

		// Read up.sql
		upPath := filepath.Join(s.migrationsPath, entry.Name(), "up.sql")
		upSQL, err := ioutil.ReadFile(upPath)
		if err != nil {
			continue // Skip if up.sql doesn't exist
		}

		// Read down.sql (optional)
		downPath := filepath.Join(s.migrationsPath, entry.Name(), "down.sql")
		downSQL, _ := ioutil.ReadFile(downPath) // Ignore error, down.sql is optional

		migrations = append(migrations, SQLMigration{
			Version:   timestamp,
			Name:      name,
			Timestamp: parsedTime,
			UpSQL:     string(upSQL),
			DownSQL:   string(downSQL),
			Path:      filepath.Join(s.migrationsPath, entry.Name()),
		})
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// CreateMigration creates a new migration directory with template files
func (s *SQLMigrationScanner) CreateMigration(name string) (*SQLMigration, error) {
	// Generate timestamp
	timestamp := time.Now().Format("20060102150405")
	
	// Clean name (remove spaces, special chars)
	cleanName := regexp.MustCompile(`[^a-zA-Z0-9_]+`).ReplaceAllString(name, "_")
	cleanName = strings.ToLower(cleanName)
	
	// Create directory name
	dirName := fmt.Sprintf("%s_%s", timestamp, cleanName)
	dirPath := filepath.Join(s.migrationsPath, dirName)

	// Create directory
	if err := createDir(dirPath); err != nil {
		return nil, fmt.Errorf("failed to create migration directory: %w", err)
	}

	// Create up.sql template
	upPath := filepath.Join(dirPath, "up.sql")
	upTemplate := fmt.Sprintf(`-- Migration: %s
-- Description: %s
-- Generated: %s

-- Add your UP migration SQL here

`, cleanName, name, time.Now().Format("2006-01-02 15:04:05"))

	if err := ioutil.WriteFile(upPath, []byte(upTemplate), 0644); err != nil {
		return nil, fmt.Errorf("failed to create up.sql: %w", err)
	}

	// Create down.sql template
	downPath := filepath.Join(dirPath, "down.sql")
	downTemplate := fmt.Sprintf(`-- Rollback Migration: %s
-- Description: %s
-- Generated: %s

-- Add your DOWN migration SQL here

`, cleanName, name, time.Now().Format("2006-01-02 15:04:05"))

	if err := ioutil.WriteFile(downPath, []byte(downTemplate), 0644); err != nil {
		return nil, fmt.Errorf("failed to create down.sql: %w", err)
	}

	return &SQLMigration{
		Version:   timestamp,
		Name:      cleanName,
		Timestamp: time.Now(),
		Path:      dirPath,
	}, nil
}