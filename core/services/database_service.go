package services

import (
	"context"
	"fmt"
	"time"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/migrations"

	"github.com/jinzhu/gorm"
	// Drivers de banco de dados
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/lib/pq"
)

// Connector is the global database connector instance.
var Connector *gorm.DB

// ConnectorConfig holds the configuration for the database connector.
type ConnectorConfig struct {
	Driver   string // "postgres"
	Host     string
	Port     string
	User     string
	DBName   string
	Password string
}

func buildConnectorConfig() *ConnectorConfig {
	// Usar driver da configuração ou PostgreSQL como padrão
	driver := config.EnvDBDriver()

	connectorConfig := ConnectorConfig{
		Driver:   driver,
		Host:     config.EnvDBHost(),
		Port:     config.EnvDBPort(),
		User:     config.EnvDBUser(),
		Password: config.EnvDBPassword(),
		DBName:   config.EnvDBName(),
	}
	return &connectorConfig
}

func connectorURL(connectorConfig *ConnectorConfig) string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		connectorConfig.Host,
		connectorConfig.Port,
		connectorConfig.User,
		connectorConfig.DBName,
		connectorConfig.Password,
	)
}

// OpenConnection opens a new database connection.
func OpenConnection(logger logger.Logger) *errors.AppError {
	connConfig := buildConnectorConfig()
	dbConfig := connectorURL(connConfig)

	db, err := gorm.Open(connConfig.Driver,
		dbConfig,
	)

	if err != nil {
		appErr := errors.NewAppError(entities.ErrDatabase, err.Error(), map[string]interface{}{"db_config": dbConfig}, err)
		logger.LogError(context.Background(), "Failed to connect to database", appErr)
		return appErr
	}

	// Test the connection immediately
	if err := db.DB().Ping(); err != nil {
		appErr := errors.NewAppError(entities.ErrDatabase, "Failed to ping database after connection", map[string]interface{}{"error": err.Error()}, err)
		logger.LogError(context.Background(), "Database ping failed", appErr)
		return appErr
	}

	environment := config.EnvironmentConfig()
	isDevelopment := environment == entities.Environment.Development

	if isDevelopment {
		logger.Info(context.Background(), "Database connection established", map[string]interface{}{
			"db_config": dbConfig,
		})
	} else {
		logger.Info(context.Background(), "Database connection established", map[string]interface{}{
			"host":   connConfig.Host,
			"port":   connConfig.Port,
			"dbname": connConfig.DBName,
			"user":   connConfig.User,
		})
	}

	isProduction := environment == entities.Environment.Production
	db.SingularTable(true)
	db.LogMode(!isProduction)
	db.DB().SetConnMaxLifetime(10 * time.Second)
	db.DB().SetMaxIdleConns(30)
	Connector = db

	go func(dbConfig string) {
		var intervals = []time.Duration{3 * time.Second, 3 * time.Second, 15 * time.Second, 30 * time.Second, 60 * time.Second}
		for {
			time.Sleep(60 * time.Second)
			if e := Connector.DB().Ping(); e != nil {
				appErr := errors.NewAppError(entities.ErrDatabase, e.Error(), nil, e)
				logger.LogError(context.Background(), "Database ping failed", appErr)
			L:
				for i := 0; i < len(intervals); i++ {
					e2 := RetryHandler(3, func() (bool, error) {
						var e error
						Connector, e = gorm.Open("postgres", dbConfig)
						if e != nil {
							appErr := errors.NewAppError(entities.ErrDatabase, e.Error(), nil, e)
							logger.LogError(context.Background(), "Database retry failed", appErr)
							return false, e
						}
						logger.Info(context.Background(), "Database reconnected successfully")
						return true, nil
					})
					if e2 != nil {
						appErr := errors.NewAppError(entities.ErrDatabase, e2.Error(), nil, e2)
						logger.LogError(context.Background(), "Database retry failed, will retry again", appErr)
						time.Sleep(intervals[i])
						if i == len(intervals)-1 {
							i--
						}
						continue
					}
					break L
				}
			}
		}
	}(dbConfig)

	return nil
}

// RetryHandler handles retry logic for database operations.
func RetryHandler(n int, f func() (bool, error)) error {
	ok, er := f()
	if ok && er == nil {
		return nil
	}
	if n-1 > 0 {
		return RetryHandler(n-1, f)
	}
	return er
}

// RunMigrations runs the database migrations using the new SQL migration system.
func RunMigrations(logger logger.Logger) error {
	ctx := context.Background()

	logger.Info(ctx, "Initializing SQL migration system...", nil)

	// Ensure database connection is valid
	if Connector == nil {
		return fmt.Errorf("database connection not established")
	}

	// Test database connection before running migrations
	if err := Connector.DB().Ping(); err != nil {
		logger.Error(ctx, "Database connection test failed before migrations", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("database not ready for migrations: %w", err)
	}

	// Get migrations path
	migrationsPath := getMigrationsPath()
	
	// Create SQL migration executor
	executor := migrations.NewSQLMigrationExecutor(Connector, logger, migrationsPath)

	// Run migrations with retry logic
	var migrationErr error
	for attempt := 1; attempt <= 3; attempt++ {
		logger.Info(ctx, "Attempting to run SQL migrations", map[string]interface{}{
			"attempt": attempt,
		})
		
		migrationErr = executor.RunAll()
		if migrationErr == nil {
			break
		}
		
		logger.Error(ctx, "Migration attempt failed", map[string]interface{}{
			"attempt": attempt,
			"error":   migrationErr.Error(),
		})
		
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	if migrationErr != nil {
		logger.Error(ctx, "Failed to run migrations after all attempts", map[string]interface{}{
			"error": migrationErr.Error(),
		})
		return fmt.Errorf("failed to run migrations: %w", migrationErr)
	}

	logger.Info(ctx, "SQL migration system completed successfully", nil)
	return nil
}

// getMigrationsPath returns the path to the migrations directory
func getMigrationsPath() string {
	// Check if custom path is set
	if path := config.EnvMigrationsPath(); path != "" {
		return path
	}
	
	// Default to migrations directory in project root
	return "migrations"
}
