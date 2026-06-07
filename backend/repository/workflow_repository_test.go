package repository

import (
	"context"
	"testing"

	"github.com/flow-forger/flow-forger/backend/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWorkflowRepository(t *testing.T) {
	db, err := InitDB(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	repo := NewWorkflowRepository(db)

	t1 := "tenant-a"
	t2 := "tenant-b"

	// Create tenants
	db.Create(&domain.Tenant{ID: t1, TenantID: t1, Name: "Tenant A"})
	db.Create(&domain.Tenant{ID: t2, TenantID: t2, Name: "Tenant B"})

	ctxA := context.WithValue(context.Background(), domain.ContextKeyTenantID, t1)
	ctxB := context.WithValue(context.Background(), domain.ContextKeyTenantID, t2)

	t.Run("Create Workflow", func(t *testing.T) {
		wf := &domain.Workflow{
			ID:         "flow-1",
			Name:       "Flow One",
			Definition: `{"steps": []}`,
		}
		err := repo.CreateWorkflow(ctxA, wf)
		if err != nil {
			t.Fatalf("failed to create workflow: %v", err)
		}

		if wf.Version != 1 {
			t.Errorf("expected version 1, got %d", wf.Version)
		}
		if wf.TenantID != t1 {
			t.Errorf("expected tenant_id to be %s, got %s", t1, wf.TenantID)
		}
	})

	t.Run("Get Workflow (Latest vs Specific)", func(t *testing.T) {
		// Fetch latest (should be version 1)
		latest, err := repo.GetWorkflow(ctxA, "flow-1", nil)
		if err != nil {
			t.Fatalf("failed to get latest workflow: %v", err)
		}
		if latest.Version != 1 {
			t.Errorf("expected version 1, got %d", latest.Version)
		}

		// Update to version 2
		v2, err := repo.UpdateWorkflow(ctxA, "flow-1", "Flow One v2", `{"steps": [{"id": "1"}]}`)
		if err != nil {
			t.Fatalf("failed to update workflow: %v", err)
		}
		if v2.Version != 2 {
			t.Errorf("expected version 2, got %d", v2.Version)
		}

		// Fetch latest again (should be version 2)
		latest2, err := repo.GetWorkflow(ctxA, "flow-1", nil)
		if err != nil {
			t.Fatalf("failed to get latest workflow: %v", err)
		}
		if latest2.Version != 2 {
			t.Errorf("expected version 2, got %d", latest2.Version)
		}

		// Fetch version 1 explicitly
		ver1 := 1
		wf1, err := repo.GetWorkflow(ctxA, "flow-1", &ver1)
		if err != nil {
			t.Fatalf("failed to get version 1 workflow: %v", err)
		}
		if wf1.Version != 1 {
			t.Errorf("expected version 1, got %d", wf1.Version)
		}
		if wf1.Name != "Flow One" {
			t.Errorf("expected name 'Flow One', got '%s'", wf1.Name)
		}
	})

	t.Run("Rollback Workflow", func(t *testing.T) {
		// Rollback to version 1
		rolled, err := repo.RollbackWorkflow(ctxA, "flow-1", 1)
		if err != nil {
			t.Fatalf("failed to rollback workflow: %v", err)
		}
		// Rollback creates a new version, so it should be version 3
		if rolled.Version != 3 {
			t.Errorf("expected version 3, got %d", rolled.Version)
		}
		if rolled.Name != "Flow One" {
			t.Errorf("expected rolled name to be 'Flow One', got '%s'", rolled.Name)
		}

		// Fetch latest (should be version 3, with version 1 content)
		latest, err := repo.GetWorkflow(ctxA, "flow-1", nil)
		if err != nil {
			t.Fatalf("failed to get latest: %v", err)
		}
		if latest.Version != 3 {
			t.Errorf("expected latest version 3, got %d", latest.Version)
		}
		if latest.Definition != `{"steps": []}` {
			t.Errorf("expected rolled definition to match v1, got '%s'", latest.Definition)
		}
	})

	t.Run("List Workflows with Pagination", func(t *testing.T) {
		// Create a second workflow for Tenant A
		wf2 := &domain.Workflow{
			ID:         "flow-2",
			Name:       "Flow Two",
			Definition: `{"steps": []}`,
		}
		if err := repo.CreateWorkflow(ctxA, wf2); err != nil {
			t.Fatalf("failed to create workflow 2: %v", err)
		}

		// List workflows page 1, limit 1
		list, total, err := repo.ListWorkflows(ctxA, 1, 1)
		if err != nil {
			t.Fatalf("failed to list workflows: %v", err)
		}
		if total != 2 {
			t.Errorf("expected total count 2, got %d", total)
		}
		if len(list) != 1 {
			t.Errorf("expected page list length 1, got %d", len(list))
		}

		// List workflows page 2, limit 1
		list2, total2, err := repo.ListWorkflows(ctxA, 2, 1)
		if err != nil {
			t.Fatalf("failed to list workflows: %v", err)
		}
		if total2 != 2 {
			t.Errorf("expected total count 2, got %d", total2)
		}
		if len(list2) != 1 {
			t.Errorf("expected page list length 1, got %d", len(list2))
		}
		if list[0].ID == list2[0].ID {
			t.Errorf("expected page 1 and page 2 to have different workflows, both have %s", list[0].ID)
		}
	})

	t.Run("Tenant Isolation Verification", func(t *testing.T) {
		// Tenant B attempts to fetch flow-1
		_, err := repo.GetWorkflow(ctxB, "flow-1", nil)
		if err == nil {
			t.Fatal("expected error when fetching workflow of another tenant, got nil")
		}
		if err != gorm.ErrRecordNotFound {
			t.Errorf("expected ErrRecordNotFound, got %v", err)
		}

		// Tenant B attempts to update flow-1
		_, err = repo.UpdateWorkflow(ctxB, "flow-1", "Hacked", "{}")
		if err == nil {
			t.Fatal("expected error when updating workflow of another tenant, got nil")
		}

		// Tenant B lists workflows (should see 0, since we haven't created any for B)
		list, total, err := repo.ListWorkflows(ctxB, 1, 10)
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}
		if total != 0 {
			t.Errorf("expected 0 workflows for Tenant B, got %d", total)
		}
		if len(list) != 0 {
			t.Errorf("expected empty list for Tenant B, got %d", len(list))
		}
	})
}
