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
	"github.com/jinzhu/gorm"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

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

	// Add OpenTelemetry tracing callbacks
	addOtelCallbacks(db)

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
func RunMigrations() {
	Connector.AutoMigrate(
		&brands.BrandModel{},
	)
}

// addOtelCallbacks adds OpenTelemetry tracing callbacks to GORM
func addOtelCallbacks(db *gorm.DB) {
	tracer := otel.Tracer("database")

	// Before callback - start span
	db.Callback().Create().Before("gorm:create").Register("otel:before_create", func(scope *gorm.Scope) {
		if ctx, exists := scope.Get("otel:ctx"); exists && ctx != nil {
			if ctxVal, ok := ctx.(context.Context); ok {
				ctx, span := tracer.Start(ctxVal, "db.create")
				span.SetAttributes(
					attribute.String("db.table", scope.TableName()),
					attribute.String("db.operation", "create"),
				)
				scope.Set("otel:span", span)
				scope.Set("otel:ctx", ctx)
			}
		}
	})

	db.Callback().Update().Before("gorm:update").Register("otel:before_update", func(scope *gorm.Scope) {
		if ctx, exists := scope.Get("otel:ctx"); exists && ctx != nil {
			if ctxVal, ok := ctx.(context.Context); ok {
				ctx, span := tracer.Start(ctxVal, "db.update")
				span.SetAttributes(
					attribute.String("db.table", scope.TableName()),
					attribute.String("db.operation", "update"),
				)
				scope.Set("otel:span", span)
				scope.Set("otel:ctx", ctx)
			}
		}
	})

	db.Callback().Query().Before("gorm:query").Register("otel:before_query", func(scope *gorm.Scope) {
		if ctx, exists := scope.Get("otel:ctx"); exists && ctx != nil {
			if ctxVal, ok := ctx.(context.Context); ok {
				ctx, span := tracer.Start(ctxVal, "db.query")
				span.SetAttributes(
					attribute.String("db.table", scope.TableName()),
					attribute.String("db.operation", "query"),
				)
				scope.Set("otel:span", span)
				scope.Set("otel:ctx", ctx)
			}
		}
	})

	db.Callback().Delete().Before("gorm:delete").Register("otel:before_delete", func(scope *gorm.Scope) {
		if ctx, exists := scope.Get("otel:ctx"); exists && ctx != nil {
			if ctxVal, ok := ctx.(context.Context); ok {
				ctx, span := tracer.Start(ctxVal, "db.delete")
				span.SetAttributes(
					attribute.String("db.table", scope.TableName()),
					attribute.String("db.operation", "delete"),
				)
				scope.Set("otel:span", span)
				scope.Set("otel:ctx", ctx)
			}
		}
	})

	// After callbacks - end span
	afterCallback := func(scope *gorm.Scope) {
		if spanVal, exists := scope.Get("otel:span"); exists && spanVal != nil {
			if span, ok := spanVal.(interface{ End() }); ok {
				if scope.HasError() {
					// Add error attributes
					if span, ok := spanVal.(interface {
						End()
						SetAttributes(...attribute.KeyValue)
						RecordError(error)
					}); ok {
						span.SetAttributes(attribute.Bool("error", true))
						span.RecordError(scope.DB().Error)
					}
				}
				span.End()
			}
		}
	}

	db.Callback().Create().After("gorm:create").Register("otel:after_create", afterCallback)
	db.Callback().Update().After("gorm:update").Register("otel:after_update", afterCallback)
	db.Callback().Query().After("gorm:query").Register("otel:after_query", afterCallback)
	db.Callback().Delete().After("gorm:delete").Register("otel:after_delete", afterCallback)
}
