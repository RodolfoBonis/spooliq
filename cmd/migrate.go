package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/migrations"
	"github.com/RodolfoBonis/spooliq/core/services"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Initialize logger
	log := logger.NewLogger()
	ctx := context.Background()

	// Get migrations path
	migrationsPath := getMigrationsPath()

	// Commands that don't need database connection
	switch command {
	case "create":
		if len(os.Args) < 3 {
			fmt.Println("Error: Migration name required")
			fmt.Println("Usage: migrate create <name>")
			os.Exit(1)
		}
		createMigration(migrationsPath, os.Args[2], log)
		return
	}

	// Initialize database connection for commands that need it
	if err := services.OpenConnection(log); err != nil {
		log.LogError(ctx, "Failed to connect to database", err)
		os.Exit(1)
	}

	// Create SQL migration executor
	executor := migrations.NewSQLMigrationExecutor(services.Connector, log, migrationsPath)

	switch command {
	case "up":
		// Run all pending migrations
		if err := executor.RunAll(); err != nil {
			log.Error(ctx, "Migration failed", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
		log.Info(ctx, "✅ Migrations completed successfully", nil)

	case "up:one":
		// Run next migration
		if err := executor.RunUp(1); err != nil {
			log.Error(ctx, "Migration failed", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
		log.Info(ctx, "✅ Migration completed successfully", nil)

	case "down":
		// Rollback last migration (or specified number)
		count := 1
		if len(os.Args) > 2 {
			if n, err := strconv.Atoi(os.Args[2]); err == nil {
				count = n
			}
		}
		if err := executor.RunDown(count); err != nil {
			log.Error(ctx, "Rollback failed", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
		log.Info(ctx, "✅ Rollback completed successfully", nil)

	case "status":
		// Show migration status
		statuses, err := executor.GetStatus()
		if err != nil {
			log.Error(ctx, "Failed to get migration status", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
		printMigrationStatus(statuses)

	case "fresh":
		// Drop all tables and rerun migrations
		fmt.Println("⚠️  WARNING: This will DROP ALL TABLES and data will be lost!")
		fmt.Print("Are you sure? (yes/no): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Aborted")
			os.Exit(0)
		}
		if err := executor.Fresh(); err != nil {
			log.Error(ctx, "Fresh migration failed", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
		log.Info(ctx, "✅ Fresh migration completed successfully", nil)

	case "reset":
		// Rollback all and rerun
		fmt.Println("⚠️  WARNING: This will rollback and rerun all migrations!")
		fmt.Print("Are you sure? (yes/no): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Aborted")
			os.Exit(0)
		}
		if err := executor.Reset(); err != nil {
			log.Error(ctx, "Reset failed", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
		log.Info(ctx, "✅ Reset completed successfully", nil)

	case "list":
		// List all migrations
		scanner := migrations.NewSQLMigrationScanner(migrationsPath)
		migrations, err := scanner.ScanMigrations()
		if err != nil {
			log.Error(ctx, "Failed to scan migrations", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
		printMigrationList(migrations)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("SpoolIQ Migration Tool")
	fmt.Println("======================")
	fmt.Println("")
	fmt.Println("Usage: migrate <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  create <name>    - Create a new migration with the given name")
	fmt.Println("  up               - Run all pending migrations")
	fmt.Println("  up:one           - Run the next pending migration")
	fmt.Println("  down [n]         - Rollback last n migrations (default: 1)")
	fmt.Println("  status           - Show status of all migrations")
	fmt.Println("  list             - List all available migrations")
	fmt.Println("  fresh            - Drop all tables and rerun all migrations")
	fmt.Println("  reset            - Rollback all migrations and rerun them")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  migrate create add_users_table")
	fmt.Println("  migrate up")
	fmt.Println("  migrate down 2")
	fmt.Println("  migrate status")
}

func getMigrationsPath() string {
	// Check if custom path is set
	if path := os.Getenv("MIGRATIONS_PATH"); path != "" {
		return path
	}

	// Default to migrations directory in project root
	return "migrations"
}

func createMigration(migrationsPath, name string, log logger.Logger) {
	ctx := context.Background()

	scanner := migrations.NewSQLMigrationScanner(migrationsPath)
	migration, err := scanner.CreateMigration(name)
	if err != nil {
		log.Error(ctx, "Failed to create migration", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	fmt.Println("✅ Migration created successfully!")
	fmt.Printf("📁 Location: %s\n", migration.Path)
	fmt.Printf("📝 Version: %s\n", migration.Version)
	fmt.Printf("🏷️  Name: %s\n", migration.Name)
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Printf("1. Edit the migration files:\n")
	fmt.Printf("   - %s\n", filepath.Join(migration.Path, "up.sql"))
	fmt.Printf("   - %s\n", filepath.Join(migration.Path, "down.sql"))
	fmt.Println("2. Run: migrate up")
}

func printMigrationStatus(statuses []migrations.MigrationStatus) {
	fmt.Println("\nMigration Status")
	fmt.Println("================")

	if len(statuses) == 0 {
		fmt.Println("No migrations found")
		return
	}

	for _, status := range statuses {
		icon := "⬜"
		if status.Applied {
			icon = "✅"
		}

		fmt.Printf("%s %s - %s", icon, status.Version, status.Name)

		if status.AppliedAt != nil {
			fmt.Printf(" (applied: %s)", status.AppliedAt.Format("2006-01-02 15:04:05"))
		}

		fmt.Println()
	}

	// Count summary
	applied := 0
	pending := 0
	for _, s := range statuses {
		if s.Applied {
			applied++
		} else {
			pending++
		}
	}

	fmt.Println("\n---")
	fmt.Printf("Total: %d | Applied: %d | Pending: %d\n", len(statuses), applied, pending)
}

func printMigrationList(migrations []migrations.SQLMigration) {
	fmt.Println("\nAvailable Migrations")
	fmt.Println("====================")

	if len(migrations) == 0 {
		fmt.Println("No migrations found")
		fmt.Println("\nCreate your first migration with: migrate create <name>")
		return
	}

	for _, m := range migrations {
		fmt.Printf("📄 %s - %s\n", m.Version, m.Name)
		fmt.Printf("   Path: %s\n", m.Path)
		fmt.Printf("   Created: %s\n", m.Timestamp.Format("2006-01-02 15:04:05"))

		hasUp := m.UpSQL != ""
		hasDown := m.DownSQL != ""

		if hasUp && hasDown {
			fmt.Println("   Files: ✅ up.sql ✅ down.sql")
		} else if hasUp {
			fmt.Println("   Files: ✅ up.sql ⬜ down.sql")
		} else {
			fmt.Println("   Files: ⬜ up.sql ⬜ down.sql")
		}
		fmt.Println()
	}
}
