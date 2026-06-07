package repository

import (
	"testing"

	"gorm.io/driver/sqlite"
)

type ExplainRow struct {
	ID      int    `gorm:"column:id"`
	Parent  int    `gorm:"column:parent"`
	NotUsed int    `gorm:"column:notused"` // SQLite third column is historically notused/sys
	Detail  string `gorm:"column:detail"`
}

func TestExplainQueryPlan(t *testing.T) {
	db, err := InitDB(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// Drop the composite index if GORM AutoMigrate / RunMigrations created it automatically,
	// so we can test the unoptimized state.
	err = db.Exec("DROP INDEX IF EXISTS idx_runs_tenant_workflow_created").Error
	if err != nil {
		t.Fatalf("failed to drop index for testing: %v", err)
	}

	// 1. Run EXPLAIN QUERY PLAN before optimization
	queryStr := "SELECT * FROM workflow_runs WHERE tenant_id = ? AND workflow_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?"
	var explainBefore []ExplainRow
	err = db.Raw("EXPLAIN QUERY PLAN "+queryStr, "tenant-a", "workflow-1", 10, 0).Scan(&explainBefore).Error
	if err != nil {
		t.Fatalf("failed to explain query before: %v", err)
	}

	t.Log("=== EXPLAIN QUERY PLAN (Before Indexing) ===")
	for _, row := range explainBefore {
		t.Logf("ID: %d, Parent: %d, Detail: %s", row.ID, row.Parent, row.Detail)
	}

	// 2. Apply the composite index manually via SQL
	t.Log("Applying index idx_runs_tenant_workflow_created...")
	err = db.Exec("CREATE INDEX idx_runs_tenant_workflow_created ON workflow_runs (tenant_id, workflow_id, created_at DESC)").Error
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// 3. Run EXPLAIN QUERY PLAN after optimization
	var explainAfter []ExplainRow
	err = db.Raw("EXPLAIN QUERY PLAN "+queryStr, "tenant-a", "workflow-1", 10, 0).Scan(&explainAfter).Error
	if err != nil {
		t.Fatalf("failed to explain query after: %v", err)
	}

	t.Log("=== EXPLAIN QUERY PLAN (After Indexing) ===")
	for _, row := range explainAfter {
		t.Logf("ID: %d, Parent: %d, Detail: %s", row.ID, row.Parent, row.Detail)
	}
}
