package services

import (
	"context"
	"fmt"
	"time"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"github.com/RodolfoBonis/go-otel-agent/integration/gormplugin"
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	activities "github.com/RodolfoBonis/spooliq/features/activity/data/models"
	brands "github.com/RodolfoBonis/spooliq/features/brand/data/models"
	budgets "github.com/RodolfoBonis/spooliq/features/budget/data/models"
	companies "github.com/RodolfoBonis/spooliq/features/company/data/models"
	customers "github.com/RodolfoBonis/spooliq/features/customer/data/models"
	filaments "github.com/RodolfoBonis/spooliq/features/filament/data/models"
	materials "github.com/RodolfoBonis/spooliq/features/material/data/models"
	models3d "github.com/RodolfoBonis/spooliq/features/model3d/data/models"
	presets "github.com/RodolfoBonis/spooliq/features/preset/data/models"
	subscriptions "github.com/RodolfoBonis/spooliq/features/subscriptions/data/models"
	users "github.com/RodolfoBonis/spooliq/features/users/data/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
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
	sslMode := config.EnvDBSSLMode()
	sslRootCert := config.EnvDBSSLRootCert()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		connectorConfig.Host,
		connectorConfig.Port,
		connectorConfig.User,
		connectorConfig.DBName,
		connectorConfig.Password,
		sslMode,
	)

	if sslRootCert != "" {
		dsn += fmt.Sprintf(" sslrootcert=%s", sslRootCert)
	}

	return dsn
}

func newGormConfig() *gorm.Config {
	return &gorm.Config{
		Logger: gormlogger.New(
			nil,
			gormlogger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  gormlogger.Silent,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
		// FK constraints are added manually in RunMigrations() for full control.
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	}
}

// OpenConnection opens a new database connection with OTel instrumentation via gormplugin.
func OpenConnection(logger logger.Logger, agent *otelagent.Agent) *errors.AppError {
	connConfig := buildConnectorConfig()
	dbConfig := connectorURL(connConfig)

	db, err := gorm.Open(postgres.Open(dbConfig), newGormConfig())
	if err != nil {
		appErr := errors.NewAppError(entities.ErrDatabase, err.Error(), map[string]interface{}{"db_config": dbConfig}, err)
		logger.LogError(context.Background(), "Failed to connect to database", appErr)
		return appErr
	}

	// Test the connection
	sqlDB, err := db.DB()
	if err != nil {
		appErr := errors.NewAppError(entities.ErrDatabase, "Failed to get SQL DB instance", map[string]interface{}{"error": err.Error()}, err)
		logger.LogError(context.Background(), "Database SQL instance failed", appErr)
		return appErr
	}

	if err = sqlDB.Ping(); err != nil {
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

	// Configure connection pool
	sqlDB.SetConnMaxLifetime(10 * time.Second)
	sqlDB.SetMaxIdleConns(30)
	sqlDB.SetMaxOpenConns(100)

	// Add OTel instrumentation via go-otel-agent's GORM plugin
	if err := gormplugin.Instrument(db, agent,
		gormplugin.WithDBName(connConfig.DBName),
		gormplugin.WithDBUser(connConfig.User),
	); err != nil {
		logger.Warning(context.Background(), "Failed to instrument database with OTel", map[string]interface{}{
			"error": err.Error(),
		})
	}

	Connector = db

	// Background reconnection loop
	go func(dsn string) {
		intervals := []time.Duration{3 * time.Second, 3 * time.Second, 15 * time.Second, 30 * time.Second, 60 * time.Second}
		for {
			time.Sleep(60 * time.Second)
			sdb, _ := Connector.DB()
			if e := sdb.Ping(); e != nil {
				appErr := errors.NewAppError(entities.ErrDatabase, e.Error(), nil, e)
				logger.LogError(context.Background(), "Database ping failed", appErr)
			L:
				for i := 0; i < len(intervals); i++ {
					e2 := RetryHandler(3, func() (bool, error) {
						retryDB, e := gorm.Open(postgres.Open(dsn), newGormConfig())
						if e != nil {
							appErr := errors.NewAppError(entities.ErrDatabase, e.Error(), nil, e)
							logger.LogError(context.Background(), "Database retry failed", appErr)
							return false, e
						}

						if err := gormplugin.Instrument(retryDB, agent,
							gormplugin.WithDBName(connConfig.DBName),
							gormplugin.WithDBUser(connConfig.User),
						); err != nil {
							logger.Warning(context.Background(), "Failed to instrument reconnected database", map[string]interface{}{
								"error": err.Error(),
							})
						}

						Connector = retryDB
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

// RunMigrations runs the database migrations using GORM AutoMigrate.
// Order is critical to respect foreign key dependencies.
func RunMigrations() {
	// MIGRATION STRATEGY:
	// Tables are migrated in dependency order (parents before children).
	// FK constraints are added manually below for organization_id references.

	// ========================================
	// LEVEL 0: Subscription Plans (no dependencies)
	// ========================================

	// 1. Subscription Plans and Features (catalog tables with no FKs to other app tables)
	if err := Connector.AutoMigrate(&subscriptions.SubscriptionPlanModel{}, &subscriptions.PlanFeatureModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING SUBSCRIPTION_PLAN MIGRATION: %s", err.Error()))
	}

	// 1.1. Plan Templates (no FKs to other app tables)
	if err := Connector.AutoMigrate(&subscriptions.PlanTemplateModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING PLAN_TEMPLATE MIGRATION: %s", err.Error()))
	}

	// 1.2. Plan Audit Logs (FK: PlanID -> SubscriptionPlans)
	if err := Connector.AutoMigrate(&subscriptions.PlanAuditModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING PLAN_AUDIT MIGRATION: %s", err.Error()))
	}

	// 1.3. Plan Migrations (FK: FromPlanID/ToPlanID -> SubscriptionPlans)
	if err := Connector.AutoMigrate(&subscriptions.PlanMigrationModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING PLAN_MIGRATION MIGRATION: %s", err.Error()))
	}

	// ========================================
	// LEVEL 1: Companies (now depends on SubscriptionPlan for subscription_plan_id FK)
	// ========================================

	// 2. Companies (referenced by ALL tables via OrganizationID, has FK to SubscriptionPlan)
	if err := Connector.Migrator().AutoMigrate(&companies.CompanyModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING COMPANY MIGRATION: %s", err.Error()))
	}

	// 3. CompanyBranding (1:1 with Company via organization_id)
	if err := Connector.Migrator().AutoMigrate(&companies.CompanyBrandingModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING COMPANY_BRANDING MIGRATION: %s", err.Error()))
	}

	// 4. Payment Gateway Links (1:1 with Company via organization_id)
	if err := Connector.AutoMigrate(&subscriptions.PaymentGatewayLinkModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING PAYMENT_GATEWAY_LINK MIGRATION: %s", err.Error()))
	}

	// ========================================
	// LEVEL 2: Tables with FK to Companies only
	// ========================================

	// 5. Users (FK: OrganizationID -> Companies, referenced by many tables via OwnerUserID/KeycloakUserID)
	if err := Connector.AutoMigrate(&users.UserModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING USER MIGRATION: %s", err.Error()))
	}

	// 5.1. Activities (FK: OrganizationID -> Companies)
	if err := Connector.AutoMigrate(&activities.ActivityModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING ACTIVITY MIGRATION: %s", err.Error()))
	}

	// 6. Brands (FK: OrganizationID -> Companies, referenced by FilamentModel)
	if err := Connector.AutoMigrate(&brands.BrandModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING BRAND MIGRATION: %s", err.Error()))
	}

	// 7. Materials (FK: OrganizationID -> Companies, referenced by FilamentModel)
	if err := Connector.AutoMigrate(&materials.MaterialModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING MATERIAL MIGRATION: %s", err.Error()))
	}

	// ========================================
	// LEVEL 3: Tables with FK to Companies AND Users/Brands/Materials
	// ========================================

	// 8. Filaments (FK: OrganizationID -> Companies, BrandID -> Brands, MaterialID -> Materials, OwnerUserID -> Users)
	if err := Connector.AutoMigrate(&filaments.FilamentModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING FILAMENT MIGRATION: %s", err.Error()))
	}

	// 9. Customers (FK: OrganizationID -> Companies, OwnerUserID -> Users)
	if err := Connector.AutoMigrate(&customers.CustomerModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING CUSTOMER MIGRATION: %s", err.Error()))
	}

	// 3D Models (FK: OrganizationID -> Companies, CustomerID -> Customers)
	if err := Connector.AutoMigrate(&models3d.Model3DModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING MODEL3D MIGRATION: %s", err.Error()))
	}

	// ========================================
	// LEVEL 3 (continued): Presets
	// ========================================

	// 10. Presets (FK: OrganizationID -> Companies, UserID -> Users)
	if err := Connector.AutoMigrate(&presets.PresetModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING PRESET MIGRATION: %s", err.Error()))
	}

	// 11-13. Specific Preset Types (1:1 with Preset via shared ID)
	if err := Connector.AutoMigrate(&presets.MachinePresetModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING MACHINE_PRESET MIGRATION: %s", err.Error()))
	}

	if err := Connector.AutoMigrate(&presets.EnergyPresetModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING ENERGY_PRESET MIGRATION: %s", err.Error()))
	}

	if err := Connector.AutoMigrate(&presets.CostPresetModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING COST_PRESET MIGRATION: %s", err.Error()))
	}

	// ========================================
	// LEVEL 3 (continued): Payment Methods (depends on Companies only)
	// ========================================

	// 14. Payment Methods (FK: OrganizationID -> Companies)
	if err := Connector.AutoMigrate(&subscriptions.PaymentMethodModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING PAYMENT_METHOD MIGRATION: %s", err.Error()))
	}

	// ========================================
	// LEVEL 4: Budget System (complex hierarchy)
	// ========================================

	// 15. Budgets (FK: OrganizationID -> Companies, CustomerID -> Customers, OwnerUserID -> Users, Preset FKs)
	if err := Connector.AutoMigrate(&budgets.BudgetModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING BUDGET MIGRATION: %s", err.Error()))
	}

	// 16. BudgetStatusHistory (FK: OrganizationID -> Companies, BudgetID -> Budgets) CASCADE
	if err := Connector.AutoMigrate(&budgets.BudgetStatusHistoryModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING BUDGET_STATUS_HISTORY MIGRATION: %s", err.Error()))
	}

	// 17. BudgetItems (FK: OrganizationID -> Companies, BudgetID -> Budgets CASCADE, FilamentID, CostPresetID)
	if err := Connector.AutoMigrate(&budgets.BudgetItemModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING BUDGET_ITEM MIGRATION: %s", err.Error()))
	}

	// 18. BudgetItemFilaments (FK: OrganizationID -> Companies, BudgetItemID -> BudgetItems CASCADE, FilamentID)
	if err := Connector.AutoMigrate(&budgets.BudgetItemFilamentModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING BUDGET_ITEM_FILAMENT MIGRATION: %s", err.Error()))
	}

	// ========================================
	// LEVEL 5: Subscription Payments (depends on Companies, SubscriptionPlan, PaymentMethod)
	// ========================================

	// 19. Subscription Payment History (FK: OrganizationID -> Companies, SubscriptionPlanID -> SubscriptionPlans, PaymentMethodID -> PaymentMethods)
	if err := Connector.AutoMigrate(&subscriptions.SubscriptionModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING SUBSCRIPTION MIGRATION: %s", err.Error()))
	}

	// ========================================
	// FOREIGN KEY CONSTRAINTS FOR ORGANIZATION_ID
	// ========================================
	fmt.Println("Adding organization_id foreign key constraints...")

	orgFKTables := map[string]bool{
		"activities": true, "users": true, "brands": true, "materials": true, "filaments": true,
		"customers": true, "presets": true, "budgets": true, "budget_items": true,
		"budget_item_filaments": true, "budget_status_history": true,
		"payment_methods": true, "subscription_payments": true, "company_branding": true,
	}

	for table := range orgFKTables {
		var exists bool
		Connector.Raw("SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_name = ?)", table).Scan(&exists)

		if !exists {
			continue
		}

		var fkExists bool
		Connector.Raw(`
			SELECT EXISTS(
				SELECT 1 FROM information_schema.table_constraints
				WHERE table_name = ? AND constraint_name LIKE '%organization%'
			)
		`, table).Scan(&fkExists)

		if !fkExists {
			sql := fmt.Sprintf(`
				ALTER TABLE %s
				ADD CONSTRAINT fk_%s_organization
				FOREIGN KEY (organization_id)
				REFERENCES companies(organization_id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT
			`, table, table)

			if err := Connector.Exec(sql).Error; err != nil {
				fmt.Printf("Warning: FK constraint for %s.organization_id failed: %v\n", table, err)
			} else {
				fmt.Printf("Added FK constraint for %s.organization_id\n", table)
			}
		}
	}

	fmt.Println("Organization FK constraints setup completed")

	// ========================================
	// DASHBOARD PERFORMANCE INDEXES
	// ========================================
	fmt.Println("Adding dashboard performance indexes...")

	dashboardIndexes := []struct {
		name string
		sql  string
	}{
		{
			"idx_budgets_org_status_created",
			"CREATE INDEX IF NOT EXISTS idx_budgets_org_status_created ON budgets(organization_id, status, created_at) WHERE deleted_at IS NULL",
		},
		{
			"idx_budgets_customer_status",
			"CREATE INDEX IF NOT EXISTS idx_budgets_customer_status ON budgets(customer_id, status) WHERE deleted_at IS NULL",
		},
		{
			"idx_budget_items_budget",
			"CREATE INDEX IF NOT EXISTS idx_budget_items_budget ON budget_items(budget_id, organization_id)",
		},
		{
			"idx_bif_org_filament",
			"CREATE INDEX IF NOT EXISTS idx_bif_org_filament ON budget_item_filaments(organization_id, filament_id)",
		},
		{
			"idx_customers_org_created",
			"CREATE INDEX IF NOT EXISTS idx_customers_org_created ON customers(organization_id, created_at) WHERE deleted_at IS NULL",
		},
		{
			"idx_activities_org_created",
			"CREATE INDEX IF NOT EXISTS idx_activities_org_created ON activities(organization_id, created_at DESC)",
		},
		{
			"idx_activities_org_type",
			"CREATE INDEX IF NOT EXISTS idx_activities_org_type ON activities(organization_id, entity_type)",
		},
		{
			"idx_bsh_org_status_budget",
			"CREATE INDEX IF NOT EXISTS idx_bsh_org_status_budget ON budget_status_history(organization_id, new_status, budget_id)",
		},
		{
			"idx_budgets_org_created",
			"CREATE INDEX IF NOT EXISTS idx_budgets_org_created ON budgets(organization_id, created_at) WHERE deleted_at IS NULL",
		},
		{
			"idx_budgets_org_status_updated",
			"CREATE INDEX IF NOT EXISTS idx_budgets_org_status_updated ON budgets(organization_id, status, updated_at) WHERE deleted_at IS NULL",
		},
		{
			"idx_bif_budget_item",
			"CREATE INDEX IF NOT EXISTS idx_bif_budget_item ON budget_item_filaments(budget_item_id, filament_id)",
		},
		{
			"idx_filaments_brand_material",
			"CREATE INDEX IF NOT EXISTS idx_filaments_brand_material ON filaments(brand_id, material_id) WHERE deleted_at IS NULL",
		},
	}

	for _, idx := range dashboardIndexes {
		if err := Connector.Exec(idx.sql).Error; err != nil {
			fmt.Printf("Warning: Index %s creation failed: %v\n", idx.name, err)
		} else {
			fmt.Printf("Created index %s\n", idx.name)
		}
	}

	fmt.Println("Dashboard indexes setup completed")
}
