package services

import (
	"context"
	"fmt"
	"time"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"

	// Entidades do SpoolIq
	filamentsEntities "github.com/RodolfoBonis/spooliq/features/filaments/domain/entities"
	presetsEntities "github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
	quotesEntities "github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"

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

// RunMigrations runs the database migrations.
func RunMigrations() {
	// Executar migrações automáticas
	Connector.AutoMigrate(
		// Filaments
		&filamentsEntities.Filament{},

		// Quotes e relacionados
		&quotesEntities.Quote{},
		&quotesEntities.QuoteFilamentLine{},
		&quotesEntities.MachineProfile{},
		&quotesEntities.EnergyProfile{},
		&quotesEntities.CostProfile{},
		&quotesEntities.MarginProfile{},

		// Presets
		&presetsEntities.Preset{},
	)
}
