package migrations

import (
	"time"

	"github.com/jinzhu/gorm"
)

// SchemaMigration represents a database migration record
// Tracks which migrations have been applied to the database
type SchemaMigration struct {
	ID           uint      `gorm:"primary_key;auto_increment" json:"id"`
	Version      string    `gorm:"type:varchar(255);unique;not null" json:"version"`
	Name         string    `gorm:"type:varchar(255);not null" json:"name"`
	AppliedAt    time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"applied_at"`
	ExecutionTime int      `gorm:"type:integer;default:0" json:"execution_time"` // in milliseconds
}

// TableName specifies the table name for GORM
func (SchemaMigration) TableName() string {
	return "schema_migrations"
}

// BeforeCreate is a GORM hook executed before creating a migration record
func (sm *SchemaMigration) BeforeCreate(scope *gorm.Scope) error {
	sm.AppliedAt = time.Now()
	return nil
}

// Migration represents a database migration with up and down operations
type Migration struct {
	Version string
	Name    string
	Up      func(*gorm.DB) error
	Down    func(*gorm.DB) error
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version   string `json:"version"`
	Name      string `json:"name"`
	Applied   bool   `json:"applied"`
	AppliedAt *time.Time `json:"applied_at,omitempty"`
}