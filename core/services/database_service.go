package services

import (
	"context"
	"fmt"
	"time"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	brands "github.com/RodolfoBonis/spooliq/features/brand/data/models"
	materials "github.com/RodolfoBonis/spooliq/features/material/data/models"
	presets "github.com/RodolfoBonis/spooliq/features/preset/data/models"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/plugin/opentelemetry/tracing"

	// Register postgres driver for otelsql
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

	// Register and open instrumented SQL driver
	sqlDB, err := otelsql.Open("postgres", dbConfig,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
		otelsql.WithDBName(connConfig.DBName),
		otelsql.WithTracerProvider(otel.GetTracerProvider()),
		otelsql.WithMeterProvider(otel.GetMeterProvider()),
	)
	if err != nil {
		appErr := errors.NewAppError(entities.ErrDatabase, err.Error(), map[string]interface{}{"db_config": dbConfig}, err)
		logger.LogError(context.Background(), "Failed to open instrumented database connection", appErr)
		return appErr
	}

	// Report DB stats to OpenTelemetry
	otelsql.ReportDBStatsMetrics(sqlDB, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
		semconv.DBName(connConfig.DBName),
	))

	// Configure GORM with custom logger for better integration
	gormConfig := &gorm.Config{
		Logger: gormlogger.New(
			nil, // Use default logger writer
			gormlogger.Config{
				SlowThreshold:             time.Second,       // Log slow queries
				LogLevel:                  gormlogger.Silent, // Use Silent to avoid duplicate logs
				IgnoreRecordNotFoundError: true,              // Don't log RecordNotFound errors
				Colorful:                  false,             // Disable color for structured logging
			},
		),
	}

	// Use the instrumented sql.DB with GORM
	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), gormConfig)

	if err != nil {
		appErr := errors.NewAppError(entities.ErrDatabase, err.Error(), map[string]interface{}{"db_config": dbConfig}, err)
		logger.LogError(context.Background(), "Failed to connect to database", appErr)
		return appErr
	}

	// Test the connection immediately
	testSQLDB, err := db.DB()
	if err != nil {
		appErr := errors.NewAppError(entities.ErrDatabase, "Failed to get SQL DB instance", map[string]interface{}{"error": err.Error()}, err)
		logger.LogError(context.Background(), "Database SQL instance failed", appErr)
		return appErr
	}

	if err = testSQLDB.Ping(); err != nil {
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

	// Configure connection pool settings
	sqlDB, err = db.DB()
	if err == nil {
		sqlDB.SetConnMaxLifetime(10 * time.Second)
		sqlDB.SetMaxIdleConns(30)
		sqlDB.SetMaxOpenConns(100)
	}

	// Add OpenTelemetry tracing callbacks
	addOtelCallbacks(db)

	Connector = db

	go func(dbConfig string) {
		var intervals = []time.Duration{3 * time.Second, 3 * time.Second, 15 * time.Second, 30 * time.Second, 60 * time.Second}
		for {
			time.Sleep(60 * time.Second)
			sqlDB, _ := Connector.DB()
			if e := sqlDB.Ping(); e != nil {
				appErr := errors.NewAppError(entities.ErrDatabase, e.Error(), nil, e)
				logger.LogError(context.Background(), "Database ping failed", appErr)
			L:
				for i := 0; i < len(intervals); i++ {
					e2 := RetryHandler(3, func() (bool, error) {
						var e error

						// Reopen instrumented SQL connection
						retrySQLDB, e := otelsql.Open("postgres", dbConfig,
							otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
							otelsql.WithDBName(connConfig.DBName),
							otelsql.WithTracerProvider(otel.GetTracerProvider()),
							otelsql.WithMeterProvider(otel.GetMeterProvider()),
						)
						if e != nil {
							appErr := errors.NewAppError(entities.ErrDatabase, e.Error(), nil, e)
							logger.LogError(context.Background(), "Database retry failed", appErr)
							return false, e
						}

						// Register DB stats metrics
						otelsql.ReportDBStatsMetrics(retrySQLDB, otelsql.WithAttributes(
							semconv.DBSystemPostgreSQL,
							semconv.DBName(connConfig.DBName),
						))

						// Create gorm config for retry connections
						retryGormConfig := &gorm.Config{
							Logger: gormlogger.New(
								nil,
								gormlogger.Config{
									SlowThreshold:             time.Second,
									LogLevel:                  gormlogger.Silent,
									IgnoreRecordNotFoundError: true,
									Colorful:                  false,
								},
							),
						}
						Connector, e = gorm.Open(postgres.New(postgres.Config{
							Conn: retrySQLDB,
						}), retryGormConfig)
						if e != nil {
							appErr := errors.NewAppError(entities.ErrDatabase, e.Error(), nil, e)
							logger.LogError(context.Background(), "Database retry failed", appErr)
							return false, e
						}

						// Re-add OpenTelemetry tracing after reconnection
						addOtelCallbacks(Connector)

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
func RunMigrations() {
	// Check if tables exist before migrating to avoid conflicts with existing data
	if !Connector.Migrator().HasTable(&brands.BrandModel{}) {
		err := Connector.AutoMigrate(&brands.BrandModel{})
		if err != nil {
			panic(fmt.Sprintf("ERROR DURING BRAND MIGRATION: %s", err.Error()))
		}
	}

	if !Connector.Migrator().HasTable(&materials.MaterialModel{}) {
		err := Connector.AutoMigrate(&materials.MaterialModel{})
		if err != nil {
			panic(fmt.Sprintf("ERROR DURING MATERIAL MIGRATION: %s", err.Error()))
		}
	}

	// Preset migrations
	if !Connector.Migrator().HasTable(&presets.PresetModel{}) {
		err := Connector.AutoMigrate(&presets.PresetModel{})
		if err != nil {
			panic(fmt.Sprintf("ERROR DURING PRESET MIGRATION: %s", err.Error()))
		}
	}

	if !Connector.Migrator().HasTable(&presets.MachinePresetModel{}) {
		err := Connector.AutoMigrate(&presets.MachinePresetModel{})
		if err != nil {
			panic(fmt.Sprintf("ERROR DURING MACHINE_PRESET MIGRATION: %s", err.Error()))
		}
	}

	if !Connector.Migrator().HasTable(&presets.EnergyPresetModel{}) {
		err := Connector.AutoMigrate(&presets.EnergyPresetModel{})
		if err != nil {
			panic(fmt.Sprintf("ERROR DURING ENERGY_PRESET MIGRATION: %s", err.Error()))
		}
	}

	if !Connector.Migrator().HasTable(&presets.CostPresetModel{}) {
		err := Connector.AutoMigrate(&presets.CostPresetModel{})
		if err != nil {
			panic(fmt.Sprintf("ERROR DURING COST_PRESET MIGRATION: %s", err.Error()))
		}
	}
}

// addOtelCallbacks adds OpenTelemetry tracing callbacks to GORM using the official plugin
func addOtelCallbacks(db *gorm.DB) {
	// Use the official GORM OpenTelemetry plugin which provides better instrumentation
	if err := db.Use(tracing.NewPlugin()); err != nil {
		// Log error but don't fail the application if tracing setup fails
		fmt.Printf("Failed to register GORM OpenTelemetry plugin: %v\n", err)
	}
}
