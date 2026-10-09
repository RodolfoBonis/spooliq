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
	notifications "github.com/RodolfoBonis/spooliq/features/notification/data/models"
	presets "github.com/RodolfoBonis/spooliq/features/preset/data/models"
	presetRepos "github.com/RodolfoBonis/spooliq/features/preset/data/repositories"
	profiles "github.com/RodolfoBonis/spooliq/features/profile/data/models"
	profileRepos "github.com/RodolfoBonis/spooliq/features/profile/data/repositories"
	stocks "github.com/RodolfoBonis/spooliq/features/stock/data/models"
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

	// 5.2. Notifications (FK: OrganizationID -> Companies)
	if err := Connector.AutoMigrate(&notifications.NotificationModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING NOTIFICATION MIGRATION: %s", err.Error()))
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

	// 9.1. Models 3D (FK: OrganizationID -> Companies, CustomerID -> Customers SET NULL,
	// OwnerUserID -> Users). Migrated right after customers so the customer FK below
	// has both tables available.
	if err := Connector.AutoMigrate(&models3d.Model3DModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING MODEL3D MIGRATION: %s", err.Error()))
	}

	// 9.2. models_3d.customer_id FK -> customers(id) ON DELETE SET NULL. FK creation
	// is disabled during AutoMigrate (see newGormConfig), so add it here idempotently:
	// only when both tables exist and the constraint is not already present. ON DELETE
	// SET NULL detaches the customer from its models when the customer is deleted.
	{
		var customersExists, models3dExists bool
		Connector.Raw("SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_name = 'customers')").Scan(&customersExists)
		Connector.Raw("SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_name = 'models_3d')").Scan(&models3dExists)
		if customersExists && models3dExists {
			var fkExists bool
			Connector.Raw(`
				SELECT EXISTS(
					SELECT 1 FROM information_schema.table_constraints
					WHERE table_name = 'models_3d' AND constraint_name = 'fk_models_3d_customer'
				)
			`).Scan(&fkExists)
			if !fkExists {
				sql := `
					ALTER TABLE models_3d
					ADD CONSTRAINT fk_models_3d_customer
					FOREIGN KEY (customer_id)
					REFERENCES customers(id)
					ON UPDATE CASCADE
					ON DELETE SET NULL
				`
				if err := Connector.Exec(sql).Error; err != nil {
					fmt.Printf("Warning: FK constraint for models_3d.customer_id failed: %v\n", err)
				} else {
					fmt.Println("Added FK constraint for models_3d.customer_id")
				}
			}
		}
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

	// Enforce a single default preset per (organization, type): dedupe existing
	// data then create the partial unique index. Safe and idempotent.
	if err := presetRepos.MigrateDefaults(Connector); err != nil {
		panic(fmt.Sprintf("ERROR DURING PRESET DEFAULT CONSTRAINT MIGRATION: %s", err.Error()))
	}

	// 13.1. Print Profiles (FK: OrganizationID -> Companies; references presets by id)
	if err := Connector.AutoMigrate(&profiles.PrintProfileModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING PRINT_PROFILE MIGRATION: %s", err.Error()))
	}

	// Enforce a single default print profile per organization (dedupe + index).
	if err := profileRepos.MigrateDefaults(Connector); err != nil {
		panic(fmt.Sprintf("ERROR DURING PRINT_PROFILE DEFAULT CONSTRAINT MIGRATION: %s", err.Error()))
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

	// 15.1. Budgets.profile_id FK -> print_profiles (ON DELETE SET NULL). AutoMigrate
	// adds the nullable profile_id column from the model; FK creation is disabled
	// during AutoMigrate (see newGormConfig), so add it here, idempotently: only
	// when the budgets table exists and the constraint is not already present. ON
	// DELETE SET NULL means deleting a profile simply detaches it from past budgets.
	{
		var budgetsExists bool
		Connector.Raw("SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_name = 'budgets')").Scan(&budgetsExists)
		if budgetsExists {
			var fkExists bool
			Connector.Raw(`
				SELECT EXISTS(
					SELECT 1 FROM information_schema.table_constraints
					WHERE table_name = 'budgets' AND constraint_name = 'fk_budgets_profile'
				)
			`).Scan(&fkExists)
			if !fkExists {
				sql := `
					ALTER TABLE budgets
					ADD CONSTRAINT fk_budgets_profile
					FOREIGN KEY (profile_id)
					REFERENCES print_profiles(id)
					ON UPDATE CASCADE
					ON DELETE SET NULL
				`
				if err := Connector.Exec(sql).Error; err != nil {
					fmt.Printf("Warning: FK constraint for budgets.profile_id failed: %v\n", err)
				} else {
					fmt.Println("Added FK constraint for budgets.profile_id")
				}
			}
		}
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

	// 18.1. Filament Stock Movements (Phase 4C). FK: OrganizationID -> Companies,
	// FilamentID -> Filaments CASCADE, BudgetID -> Budgets SET NULL. Migrated after
	// filaments and budgets exist; the non-org FKs are added idempotently below.
	if err := Connector.AutoMigrate(&stocks.StockMovementModel{}); err != nil {
		panic(fmt.Sprintf("ERROR DURING FILAMENT_STOCK_MOVEMENT MIGRATION: %s", err.Error()))
	}

	// 18.2. filament_stock_movements.filament_id FK -> filaments(id) ON DELETE CASCADE,
	// and budget_id FK -> budgets(id) ON DELETE SET NULL. FK creation is disabled during
	// AutoMigrate (see newGormConfig), so add them here, idempotently.
	{
		var tableExists bool
		Connector.Raw("SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_name = 'filament_stock_movements')").Scan(&tableExists)
		if tableExists {
			var filamentFKExists bool
			Connector.Raw(`
				SELECT EXISTS(
					SELECT 1 FROM information_schema.table_constraints
					WHERE table_name = 'filament_stock_movements' AND constraint_name = 'fk_stock_movements_filament'
				)
			`).Scan(&filamentFKExists)
			if !filamentFKExists {
				sql := `
					ALTER TABLE filament_stock_movements
					ADD CONSTRAINT fk_stock_movements_filament
					FOREIGN KEY (filament_id)
					REFERENCES filaments(id)
					ON UPDATE CASCADE
					ON DELETE CASCADE
				`
				if err := Connector.Exec(sql).Error; err != nil {
					fmt.Printf("Warning: FK constraint for filament_stock_movements.filament_id failed: %v\n", err)
				} else {
					fmt.Println("Added FK constraint for filament_stock_movements.filament_id")
				}
			}

			var budgetFKExists bool
			Connector.Raw(`
				SELECT EXISTS(
					SELECT 1 FROM information_schema.table_constraints
					WHERE table_name = 'filament_stock_movements' AND constraint_name = 'fk_stock_movements_budget'
				)
			`).Scan(&budgetFKExists)
			if !budgetFKExists {
				sql := `
					ALTER TABLE filament_stock_movements
					ADD CONSTRAINT fk_stock_movements_budget
					FOREIGN KEY (budget_id)
					REFERENCES budgets(id)
					ON UPDATE CASCADE
					ON DELETE SET NULL
				`
				if err := Connector.Exec(sql).Error; err != nil {
					fmt.Printf("Warning: FK constraint for filament_stock_movements.budget_id failed: %v\n", err)
				} else {
					fmt.Println("Added FK constraint for filament_stock_movements.budget_id")
				}
			}
		}
	}

	// 17.1. budget_items.model_3d_id FK -> models_3d(id) ON DELETE SET NULL. Added
	// here (after both budget_items and models_3d exist), idempotently: only when the
	// constraint is not already present. Deleting a model simply detaches it from the
	// budget items that referenced it.
	{
		var budgetItemsExists, models3dExists bool
		Connector.Raw("SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_name = 'budget_items')").Scan(&budgetItemsExists)
		Connector.Raw("SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_name = 'models_3d')").Scan(&models3dExists)
		if budgetItemsExists && models3dExists {
			var fkExists bool
			Connector.Raw(`
				SELECT EXISTS(
					SELECT 1 FROM information_schema.table_constraints
					WHERE table_name = 'budget_items' AND constraint_name = 'fk_budget_items_model3d'
				)
			`).Scan(&fkExists)
			if !fkExists {
				sql := `
					ALTER TABLE budget_items
					ADD CONSTRAINT fk_budget_items_model3d
					FOREIGN KEY (model_3d_id)
					REFERENCES models_3d(id)
					ON UPDATE CASCADE
					ON DELETE SET NULL
				`
				if err := Connector.Exec(sql).Error; err != nil {
					fmt.Printf("Warning: FK constraint for budget_items.model_3d_id failed: %v\n", err)
				} else {
					fmt.Println("Added FK constraint for budget_items.model_3d_id")
				}
			}
		}
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
		"activities": true, "notifications": true, "users": true, "brands": true, "materials": true, "filaments": true,
		"customers": true, "models_3d": true, "presets": true, "budgets": true, "budget_items": true,
		"budget_item_filaments": true, "budget_status_history": true,
		"payment_methods": true, "subscription_payments": true, "company_branding": true,
		"print_profiles": true, "filament_stock_movements": true,
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
			// Dashboard profit is filtered by approval date.
			"idx_budgets_org_approved",
			"CREATE INDEX IF NOT EXISTS idx_budgets_org_approved ON budgets(organization_id, approved_at) WHERE deleted_at IS NULL AND approved_at IS NOT NULL",
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
		{
			// Dedup guard for 3D model uploads: one live row per (org, file_hash).
			// Partial so soft-deleted rows don't block re-uploading the same content.
			"uq_models_3d_org_file_hash",
			"CREATE UNIQUE INDEX IF NOT EXISTS uq_models_3d_org_file_hash ON models_3d(organization_id, file_hash) WHERE deleted_at IS NULL",
		},
		{
			// Sequential quote number is unique per organization (partial so legacy
			// NULLs and soft-deleted rows do not collide).
			"uq_budgets_org_quote_number",
			"CREATE UNIQUE INDEX IF NOT EXISTS uq_budgets_org_quote_number ON budgets(organization_id, quote_number) WHERE deleted_at IS NULL AND quote_number IS NOT NULL",
		},
		{
			// Public share token is globally unique (partial: only non-NULL tokens).
			"uq_budgets_public_token",
			"CREATE UNIQUE INDEX IF NOT EXISTS uq_budgets_public_token ON budgets(public_token) WHERE public_token IS NOT NULL",
		},
		{
			// Ledger lookups: newest movement first, per filament within an org.
			"idx_stock_movements_org_filament_created",
			"CREATE INDEX IF NOT EXISTS idx_stock_movements_org_filament_created ON filament_stock_movements(organization_id, filament_id, created_at DESC)",
		},
		{
			// Idempotency guard for auto-deduction: at most one consumption movement
			// per (budget, filament). Partial so purchases/adjustments/waste are free.
			"uq_stock_movements_budget_filament_consumption",
			"CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_movements_budget_filament_consumption ON filament_stock_movements(budget_id, filament_id) WHERE type = 'consumption'",
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

	runDataMigrations()
}

// dataMigrations are one-off data fixes. Each runs exactly once per database:
// its name is recorded in data_migrations after it succeeds.
var dataMigrations = []struct{ name, sql string }{
	{
		// Filaments were created with is_active=false since the module was added
		// (no default, and create/update ignored the field). Nothing ever let users
		// deactivate a filament, so every existing row is meant to be active.
		"2026-10-06_activate_legacy_filaments",
		"UPDATE filaments SET is_active = true WHERE is_active = false",
	},
	{
		// Phase 4A added budgets.include_machine_cost with a column DEFAULT of TRUE so
		// that NEW budgets charge machine time. AutoMigrate's ADD COLUMN ... DEFAULT true
		// backfills EXISTING rows to TRUE, which would silently reinterpret already-quoted
		// budgets. This one-time fix flips every pre-existing budget back to FALSE so
		// stored totals are not changed; new budgets keep the TRUE default going forward.
		"2026-10-07_backfill_budget_include_machine_cost_false",
		"UPDATE budgets SET include_machine_cost = false",
	},
	{
		// Phase 4B added budgets.quote_number (sequential per organization). Backfill
		// existing rows deterministically in (created_at, id) order so every org's
		// quotes are numbered 1..N. Runs once; new budgets allocate via the companies
		// counter going forward.
		"2026-10-08_backfill_budget_quote_number",
		`UPDATE budgets b SET quote_number = n.rn
		 FROM (
			SELECT id, ROW_NUMBER() OVER (PARTITION BY organization_id ORDER BY created_at, id) AS rn
			FROM budgets
			WHERE quote_number IS NULL
		 ) n
		 WHERE b.id = n.id AND b.quote_number IS NULL`,
	},
	{
		// After backfilling quote_number, align each company's next_quote_number to
		// max(quote_number)+1 so freshly allocated numbers never collide with backfilled
		// ones. Companies with no budgets keep the default (1).
		"2026-10-08_init_company_next_quote_number",
		`UPDATE companies c SET next_quote_number = sub.max_qn + 1
		 FROM (
			SELECT organization_id, MAX(quote_number) AS max_qn
			FROM budgets
			WHERE quote_number IS NOT NULL
			GROUP BY organization_id
		 ) sub
		 WHERE c.organization_id = sub.organization_id AND sub.max_qn >= c.next_quote_number`,
	},
	{
		// Customer responses via the public link never wrote status history, so the
		// funnel and response times missed them. Add one row per response, inferring
		// approve vs reject from the current status (skips reopened/cancelled ones).
		"2026-10-09_backfill_public_response_history",
		`INSERT INTO budget_status_history
			(id, budget_id, organization_id, previous_status, new_status, changed_by, notes, created_at)
		 SELECT gen_random_uuid(), b.id, b.organization_id, 'sent',
			CASE WHEN b.status = 'rejected' THEN 'rejected' ELSE 'approved' END,
			COALESCE(b.customer_response_name, 'Cliente') || ' (cliente)',
			'Resposta do cliente pelo link', b.customer_response_at
		 FROM budgets b
		 WHERE b.customer_response_at IS NOT NULL
		   AND b.status IN ('approved', 'printing', 'completed', 'rejected')
		   AND NOT EXISTS (
			SELECT 1 FROM budget_status_history h
			WHERE h.budget_id = b.id AND h.new_status IN ('approved', 'rejected')
			  AND h.created_at BETWEEN b.customer_response_at - interval '1 minute'
			                       AND b.customer_response_at + interval '1 minute'
		   )`,
	},
	{
		// The expiry job never wrote status history either.
		"2026-10-09_backfill_expired_history",
		`INSERT INTO budget_status_history
			(id, budget_id, organization_id, previous_status, new_status, changed_by, notes, created_at)
		 SELECT gen_random_uuid(), b.id, b.organization_id, 'sent', 'expired', 'system',
			'Validade expirada', COALESCE(b.valid_until, b.updated_at)
		 FROM budgets b
		 WHERE b.status = 'expired'
		   AND NOT EXISTS (SELECT 1 FROM budget_status_history h WHERE h.budget_id = b.id AND h.new_status = 'expired')`,
	},
	{
		// approved_at = the latest approval in the history (now complete thanks to the
		// backfills above), falling back to the last update for older budgets.
		"2026-10-09_backfill_budget_approved_at",
		`UPDATE budgets b SET approved_at = COALESCE(
			(SELECT MAX(h.created_at) FROM budget_status_history h
			 WHERE h.budget_id = b.id AND h.new_status = 'approved'),
			b.updated_at) -- legacy budgets without history: best available date
		 WHERE b.approved_at IS NULL AND b.status IN ('approved', 'printing', 'completed')`,
	},
	{
		"2026-10-09_backfill_budget_completed_at",
		`UPDATE budgets b SET completed_at = COALESCE(
			(SELECT MAX(h.created_at) FROM budget_status_history h
			 WHERE h.budget_id = b.id AND h.new_status = 'completed'),
			b.updated_at)
		 WHERE b.completed_at IS NULL AND b.status = 'completed'`,
	},
}

// runDataMigrations applies pending dataMigrations, each in its own transaction
// together with its marker row, so a failure leaves it pending for the next start.
func runDataMigrations() {
	if err := Connector.Exec(`CREATE TABLE IF NOT EXISTS data_migrations (
		name varchar(255) PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`).Error; err != nil {
		fmt.Printf("Warning: data_migrations table creation failed: %v\n", err)
		return
	}
	// Migrations run in order and later ones may depend on earlier ones (the
	// approved_at backfill reads history rows inserted by the history
	// backfills), so the first failure stops the run; it is retried on the next
	// boot. An advisory lock serializes replicas starting at the same time, and
	// the applied check is repeated under the lock.
	for _, m := range dataMigrations {
		err := Connector.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext('data_migrations'))").Error; err != nil {
				return err
			}
			var applied int64
			if err := tx.Raw("SELECT count(*) FROM data_migrations WHERE name = ?", m.name).Scan(&applied).Error; err != nil {
				return err
			}
			if applied > 0 {
				return nil
			}
			res := tx.Exec(m.sql)
			if res.Error != nil {
				return res.Error
			}
			fmt.Printf("Data migration %s applied (%d rows)\n", m.name, res.RowsAffected)
			return tx.Exec("INSERT INTO data_migrations (name) VALUES (?) ON CONFLICT (name) DO NOTHING", m.name).Error
		})
		if err != nil {
			fmt.Printf("Warning: data migration %s failed, skipping the remaining ones until next start: %v\n", m.name, err)
			return
		}
	}
}
