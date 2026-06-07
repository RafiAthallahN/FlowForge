package repository

import (
	"context"
	"testing"
	"time"
	"github.com/flow-forger/flow-forger/backend/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDBAndRepository(t *testing.T) {
	db, err := InitDB(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	repo := NewRunRepository(db)
	ctx := context.Background()

	tenantID := "tenant-test"
	db.Create(&domain.Tenant{ID: tenantID, TenantID: tenantID, Name: "Test Tenant"})
	db.Create(&domain.Workflow{ID: "w-test", TenantID: tenantID, Version: 1, Name: "Test Workflow", Definition: "{}"})

	now := time.Now()
	run := domain.WorkflowRun{
		ID:              "run-test",
		TenantID:        tenantID,
		WorkflowID:      "w-test",
		WorkflowVersion: 1,
		Status:          "Running",
		StartedAt:       &now,
	}

	if err := repo.CreateRun(ctx, &run); err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	errMsg := "some error"
	completedAt := time.Now()
	if err := repo.UpdateRun(ctx, "run-test", tenantID, "Failed", &completedAt, &errMsg); err != nil {
		t.Fatalf("failed to update run: %v", err)
	}

	// Update again with identical values to verify RowsAffected is still non-zero in SQLite
	if err := repo.UpdateRun(ctx, "run-test", tenantID, "Failed", &completedAt, &errMsg); err != nil {
		t.Fatalf("failed to update run again: %v", err)
	}

	log := domain.ExecutionLog{
		ID:            "log-test",
		TenantID:      tenantID,
		WorkflowRunID: "run-test",
		StepID:        "step-1",
		Status:        "Success",
		RetryCount:    0,
		DurationMS:    150,
		LogOutput:     "test log output",
	}

	if err := repo.CreateLog(ctx, &log); err != nil {
		t.Fatalf("failed to create log: %v", err)
	}
}

func TestForeignKeysAndNotFound(t *testing.T) {
	db, err := InitDB(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	repo := NewRunRepository(db)
	ctx := context.Background()

	// 1. Verify UpdateRun returns gorm.ErrRecordNotFound if run does not exist
	errMsg := "some error"
	completedAt := time.Now()
	err = repo.UpdateRun(ctx, "non-existent-run", "tenant-test", "Failed", &completedAt, &errMsg)
	if err == nil {
		t.Error("expected error updating non-existent run, got nil")
	} else if err != gorm.ErrRecordNotFound {
		t.Errorf("expected gorm.ErrRecordNotFound, got: %v", err)
	}

	// 2. Verify foreign key constraint on WorkflowRun referencing Tenant
	// Create run referencing non-existent tenant (invalid FK)
	now := time.Now()
	runInvalidTenant := domain.WorkflowRun{
		ID:              "run-invalid-tenant",
		TenantID:        "non-existent-tenant",
		WorkflowID:      "w-test",
		WorkflowVersion: 1,
		Status:          "Running",
		StartedAt:       &now,
	}
	err = repo.CreateRun(ctx, &runInvalidTenant)
	if err == nil {
		t.Error("expected foreign key violation error for non-existent Tenant, got nil")
	}

	// Now create a valid Tenant, but referencing a non-existent Workflow
	tenantID := "tenant-test"
	db.Create(&domain.Tenant{ID: tenantID, TenantID: tenantID, Name: "Test Tenant"})

	runInvalidWorkflow := domain.WorkflowRun{
		ID:              "run-invalid-workflow",
		TenantID:        tenantID,
		WorkflowID:      "non-existent-workflow",
		WorkflowVersion: 1,
		Status:          "Running",
		StartedAt:       &now,
	}
	err = repo.CreateRun(ctx, &runInvalidWorkflow)
	if err == nil {
		t.Error("expected foreign key violation error for non-existent Workflow, got nil")
	}

	// Now create a valid Workflow, and try to create an ExecutionLog referencing a non-existent WorkflowRun
	db.Create(&domain.Workflow{ID: "w-test", TenantID: tenantID, Version: 1, Name: "Test Workflow", Definition: "{}"})

	logInvalidRun := domain.ExecutionLog{
		ID:            "log-invalid-run",
		TenantID:      tenantID,
		WorkflowRunID: "non-existent-run",
		StepID:        "step-1",
		Status:        "Success",
		RetryCount:    0,
		DurationMS:    150,
		LogOutput:     "test log output",
	}
	err = repo.CreateLog(ctx, &logInvalidRun)
	if err == nil {
		t.Error("expected foreign key violation error for non-existent WorkflowRun, got nil")
	}
}

