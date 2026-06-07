package domain

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAutoMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Enable foreign key constraints in SQLite for verification
	err = db.Exec("PRAGMA foreign_keys = ON;").Error
	if err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Auto migrate all entities
	err = db.AutoMigrate(
		&Tenant{},
		&User{},
		&Workflow{},
		&WorkflowRun{},
		&ExecutionLog{},
	)
	if err != nil {
		t.Fatalf("failed to auto migrate: %v", err)
	}

	// Schema validation helpers
	type FKInfo struct {
		ID       int    `gorm:"column:id"`
		Seq      int    `gorm:"column:seq"`
		Table    string `gorm:"column:table"`
		From     string `gorm:"column:from"`
		To       string `gorm:"column:to"`
		OnUpdate string `gorm:"column:on_update"`
		OnDelete string `gorm:"column:on_delete"`
		Match    string `gorm:"column:match"`
	}

	getFKs := func(table string) []FKInfo {
		var fks []FKInfo
		if err := db.Raw("PRAGMA foreign_key_list(" + table + ")").Scan(&fks).Error; err != nil {
			t.Fatalf("failed to query foreign keys for %s: %v", table, err)
		}
		return fks
	}

	// 1. Verify 'tenants' table has no outgoing foreign keys
	tenantsFKs := getFKs("tenants")
	if len(tenantsFKs) > 0 {
		t.Errorf("expected 'tenants' table to have 0 foreign keys, got %d", len(tenantsFKs))
		for _, fk := range tenantsFKs {
			t.Errorf("  - Tenant outgoing FK: From %s To %s(%s)", fk.From, fk.Table, fk.To)
		}
	}

	// 2. Verify 'users' table has a foreign key to 'tenants'
	usersFKs := getFKs("users")
	var hasTenantFK bool
	for _, fk := range usersFKs {
		if fk.Table == "tenants" && fk.From == "tenant_id" && fk.To == "id" {
			hasTenantFK = true
			if fk.OnDelete != "CASCADE" || fk.OnUpdate != "CASCADE" {
				t.Errorf("expected CASCADE constraints on user tenant foreign key, got Delete:%s Update:%s", fk.OnDelete, fk.OnUpdate)
			}
		}
	}
	if !hasTenantFK {
		t.Error("expected 'users' table to have a foreign key to 'tenants'")
	}

	// 3. Verify 'workflows' table has a foreign key to 'tenants'
	workflowsFKs := getFKs("workflows")
	hasTenantFK = false
	for _, fk := range workflowsFKs {
		if fk.Table == "tenants" && fk.From == "tenant_id" && fk.To == "id" {
			hasTenantFK = true
			if fk.OnDelete != "CASCADE" || fk.OnUpdate != "CASCADE" {
				t.Errorf("expected CASCADE constraints on workflow tenant foreign key, got Delete:%s Update:%s", fk.OnDelete, fk.OnUpdate)
			}
		}
	}
	if !hasTenantFK {
		t.Error("expected 'workflows' table to have a foreign key to 'tenants'")
	}

	// 4. Verify 'workflow_runs' table has foreign keys to 'tenants' and 'workflows'
	runFKs := getFKs("workflow_runs")
	var hasRunTenantFK, hasRunWorkflowFK bool
	for _, fk := range runFKs {
		if fk.Table == "tenants" && fk.From == "tenant_id" && fk.To == "id" {
			hasRunTenantFK = true
		}
		if fk.Table == "workflows" {
			// Composite foreign key consists of multiple parts
			hasRunWorkflowFK = true
		}
	}
	if !hasRunTenantFK {
		t.Error("expected 'workflow_runs' table to have a foreign key to 'tenants'")
	}
	if !hasRunWorkflowFK {
		t.Error("expected 'workflow_runs' table to have a composite foreign key to 'workflows'")
	}

	// 5. Verify 'execution_logs' table has foreign keys to 'tenants' and 'workflow_runs'
	logFKs := getFKs("execution_logs")
	var hasLogTenantFK, hasLogRunFK bool
	for _, fk := range logFKs {
		if fk.Table == "tenants" && fk.From == "tenant_id" && fk.To == "id" {
			hasLogTenantFK = true
		}
		if fk.Table == "workflow_runs" && fk.From == "workflow_run_id" && fk.To == "id" {
			hasLogRunFK = true
		}
	}
	if !hasLogTenantFK {
		t.Error("expected 'execution_logs' table to have a foreign key to 'tenants'")
	}
	if !hasLogRunFK {
		t.Error("expected 'execution_logs' table to have a foreign key to 'workflow_runs'")
	}
}
