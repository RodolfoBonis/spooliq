package main

import (
	"context"
	"fmt"
	"os"

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

	// Initialize database connection
	if err := services.OpenConnection(log); err != nil {
		log.LogError(ctx, "Failed to connect to database", err)
		os.Exit(1)
	}

	// Create migration service
	migrationService := migrations.NewMigrationService(services.Connector, log)
	allMigrations := migrations.GetAllMigrations()
	migrationService.RegisterMigrations(allMigrations)

	switch command {
	case "up":
		if err := migrationService.Run(); err != nil {
			log.Error(ctx, "Migration failed", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
		log.Info(ctx, "Migrations completed successfully", nil)

	case "down":
		if err := migrationService.Rollback(); err != nil {
			log.Error(ctx, "Rollback failed", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
		log.Info(ctx, "Rollback completed successfully", nil)

	case "status":
		statuses, err := migrationService.Status()
		if err != nil {
			log.Error(ctx, "Failed to get migration status", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
		printMigrationStatus(statuses)

	case "dry-run":
		pending, err := migrationService.DryRun()
		if err != nil {
			log.Error(ctx, "Failed to perform dry run", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
		printPendingMigrations(pending)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: go run cmd/migrate.go <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  up       - Run all pending migrations")
	fmt.Println("  down     - Rollback the last migration")
	fmt.Println("  status   - Show status of all migrations")
	fmt.Println("  dry-run  - Show what migrations would be run without executing them")
}

func printMigrationStatus(statuses []migrations.MigrationStatus) {
	fmt.Println("Migration Status:")
	fmt.Println("=================")
	for _, status := range statuses {
		appliedStr := "[ ]"
		if status.Applied {
			appliedStr = "[✓]"
		}
		fmt.Printf("%s %s - %s", appliedStr, status.Version, status.Name)
		if status.AppliedAt != nil {
			fmt.Printf(" (applied: %s)", status.AppliedAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}
}

func printPendingMigrations(pending []migrations.Migration) {
	if len(pending) == 0 {
		fmt.Println("No pending migrations")
		return
	}

	fmt.Println("Pending Migrations:")
	fmt.Println("==================")
	for _, migration := range pending {
		fmt.Printf("- %s: %s\n", migration.Version, migration.Name)
	}
}
