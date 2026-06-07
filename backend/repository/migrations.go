package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// SchemaMigration tracks migrations that have been applied to the database.
type SchemaMigration struct {
	Version   string    `gorm:"primaryKey"`
	AppliedAt time.Time `gorm:"not null"`
}

type Migration struct {
	Version string
	Up      func(db *gorm.DB) error
}

// RunMigrations applies pending migrations to the database.
func RunMigrations(db *gorm.DB) error {
	// AutoMigrate the schema_migrations table itself
	if err := db.AutoMigrate(&SchemaMigration{}); err != nil {
		return fmt.Errorf("failed to migrate schema_migrations: %w", err)
	}

	migrations := []Migration{
		{
			Version: "000002_add_metadata_and_indices",
			Up: func(db *gorm.DB) error {
				// SQLite and PostgreSQL safe migration syntax:
				// Alter table to add metadata column (if not already handled by AutoMigrate)
				if !db.Migrator().HasColumn(&domain_dummy{}, "metadata") {
					// Add column manually
					err := db.Exec("ALTER TABLE workflow_runs ADD COLUMN metadata TEXT").Error
					if err != nil {
						return fmt.Errorf("failed to add metadata column: %w", err)
					}
				}

				// Create index manually if it does not exist
				if !db.Migrator().HasIndex(&domain_dummy{}, "idx_runs_tenant_workflow_created") {
					err := db.Exec("CREATE INDEX idx_runs_tenant_workflow_created ON workflow_runs (tenant_id, workflow_id, created_at DESC)").Error
					if err != nil {
						return fmt.Errorf("failed to create index: %w", err)
					}
				}
				return nil
			},
		},
	}

	for _, m := range migrations {
		var count int64
		if err := db.Model(&SchemaMigration{}).Where("version = ?", m.Version).Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check migration state for %s: %w", m.Version, err)
		}

		if count == 0 {
			fmt.Printf("Applying database migration: %s...\n", m.Version)
			if err := m.Up(db); err != nil {
				return fmt.Errorf("migration %s failed: %w", m.Version, err)
			}

			migrationRecord := SchemaMigration{
				Version:   m.Version,
				AppliedAt: time.Now(),
			}
			if err := db.Create(&migrationRecord).Error; err != nil {
				return fmt.Errorf("failed to record migration success for %s: %w", m.Version, err)
			}
			fmt.Printf("Migration %s successfully applied!\n", m.Version)
		}
	}

	return nil
}

// domain_dummy is a placeholder struct to satisfy GORM's Migrator interface for custom queries.
type domain_dummy struct{}

func (domain_dummy) TableName() string {
	return "workflow_runs"
}
