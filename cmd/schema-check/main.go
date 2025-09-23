package main

import (
	"context"
	"fmt"
	"os"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
)

// SchemaInfo represents database schema information
type SchemaInfo struct {
	TableName      string `json:"table_name"`
	ConstraintName string `json:"constraint_name"`
	ConstraintType string `json:"constraint_type"`
}

// MigrationStatus represents migration status information
type MigrationStatus struct {
	Version   string `json:"version"`
	Name      string `json:"name"`
	AppliedAt string `json:"applied_at"`
}

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

	switch command {
	case "constraints":
		checkConstraints(ctx, log)
	case "migrations":
		checkMigrations(ctx, log)
	case "tables":
		checkTables(ctx, log)
	case "full":
		fmt.Println("🔍 Full Schema Diagnostic Report")
		fmt.Println("================================")
		checkTables(ctx, log)
		fmt.Println()
		checkConstraints(ctx, log)
		fmt.Println()
		checkMigrations(ctx, log)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func checkConstraints(ctx context.Context, log logger.Logger) {
	fmt.Println("📋 Foreign Key Constraints Status")
	fmt.Println("==================================")

	query := `
		SELECT 
			tc.table_name,
			tc.constraint_name,
			tc.constraint_type
		FROM information_schema.table_constraints tc
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_schema = 'public'
		ORDER BY tc.table_name, tc.constraint_name;
	`

	rows, err := services.Connector.DB().Query(query)
	if err != nil {
		log.Error(ctx, "Failed to query constraints", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer rows.Close()

	constraintCount := 0
	currentTable := ""

	for rows.Next() {
		var info SchemaInfo
		err := rows.Scan(&info.TableName, &info.ConstraintName, &info.ConstraintType)
		if err != nil {
			log.Error(ctx, "Failed to scan constraint row", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		if currentTable != info.TableName {
			if currentTable != "" {
				fmt.Println()
			}
			fmt.Printf("📁 Table: %s\n", info.TableName)
			currentTable = info.TableName
		}

		fmt.Printf("  ✅ %s (%s)\n", info.ConstraintName, info.ConstraintType)
		constraintCount++
	}

	fmt.Printf("\n📊 Total Foreign Key Constraints: %d\n", constraintCount)
}

func checkMigrations(ctx context.Context, log logger.Logger) {
	fmt.Println("📊 Applied Migrations Status")
	fmt.Println("============================")

	query := `
		SELECT version, name, applied_at 
		FROM schema_migrations 
		ORDER BY version;
	`

	rows, err := services.Connector.DB().Query(query)
	if err != nil {
		log.Error(ctx, "Failed to query migrations", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer rows.Close()

	migrationCount := 0

	for rows.Next() {
		var migration MigrationStatus
		err := rows.Scan(&migration.Version, &migration.Name, &migration.AppliedAt)
		if err != nil {
			log.Error(ctx, "Failed to scan migration row", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		fmt.Printf("✅ %s - %s (applied: %s)\n", migration.Version, migration.Name, migration.AppliedAt)
		migrationCount++
	}

	fmt.Printf("\n📊 Total Applied Migrations: %d\n", migrationCount)
}

func checkTables(ctx context.Context, log logger.Logger) {
	fmt.Println("🗄️  Database Tables Status")
	fmt.Println("===========================")

	query := `
		SELECT 
			table_name,
			(SELECT COUNT(*) FROM information_schema.columns WHERE table_name = t.table_name) as column_count
		FROM information_schema.tables t
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name;
	`

	rows, err := services.Connector.DB().Query(query)
	if err != nil {
		log.Error(ctx, "Failed to query tables", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer rows.Close()

	tableCount := 0

	for rows.Next() {
		var tableName string
		var columnCount int
		err := rows.Scan(&tableName, &columnCount)
		if err != nil {
			log.Error(ctx, "Failed to scan table row", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		fmt.Printf("📁 %s (%d columns)\n", tableName, columnCount)
		tableCount++
	}

	fmt.Printf("\n📊 Total Tables: %d\n", tableCount)
}

func printUsage() {
	fmt.Println("SpoolIQ Schema Diagnostic Tool")
	fmt.Println("==============================")
	fmt.Println("")
	fmt.Println("Usage: schema-check <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  constraints  - Show all foreign key constraints")
	fmt.Println("  migrations   - Show applied migrations")
	fmt.Println("  tables       - Show all database tables")
	fmt.Println("  full         - Complete diagnostic report")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  schema-check full")
	fmt.Println("  schema-check constraints")
}
