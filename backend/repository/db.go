package repository

import (
	"fmt"
	"github.com/flow-forger/flow-forger/backend/domain"
	"gorm.io/gorm"
)

func InitDB(dialector gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Use(&TenantIsolationPlugin{}); err != nil {
		return nil, fmt.Errorf("failed to register tenant isolation plugin: %w", err)
	}

	// Enable SQLite foreign key constraints at connection start if using SQLite
	if dialector.Name() == "sqlite" {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("failed to get sql database handle: %w", err)
		}
		if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON;"); err != nil {
			return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
		}
	}

	err = db.AutoMigrate(
		&domain.Tenant{},
		&domain.User{},
		&domain.Workflow{},
		&domain.WorkflowRun{},
		&domain.ExecutionLog{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := RunMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run safe custom migrations: %w", err)
	}

	return db, nil
}
