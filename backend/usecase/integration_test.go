package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/flow-forger/flow-forger/backend/domain"
	"github.com/flow-forger/flow-forger/backend/repository"
	"github.com/flow-forger/flow-forger/backend/usecase"
	"gorm.io/driver/sqlite"
)

func TestE2EIntegration(t *testing.T) {
	dbConn, err := repository.InitDB(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	sqlDB, err := dbConn.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql db: %v", err)
	}
	defer sqlDB.Close()

	repo := repository.NewRunRepository(dbConn)
	tenantID := "tenant-1"
	ctx := context.Background()

	if err := dbConn.Create(&domain.Tenant{ID: tenantID, TenantID: tenantID, Name: "Tenant 1"}).Error; err != nil {
		t.Fatalf("failed to seed tenant: %v", err)
	}
	if err := dbConn.Create(&domain.Workflow{ID: "w-1", TenantID: tenantID, Version: 1, Name: "Flow 1", Definition: "{}"}).Error; err != nil {
		t.Fatalf("failed to seed workflow: %v", err)
	}

	// Create run
	startedAt := time.Now()
	err = repo.CreateRun(ctx, &domain.WorkflowRun{
		ID:              "run-1",
		TenantID:        tenantID,
		WorkflowID:      "w-1",
		WorkflowVersion: 1,
		Status:          string(usecase.StatusRunning),
		StartedAt:       &startedAt,
	})
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	def := usecase.WorkflowDefinition{
		Steps: []usecase.Step{{ID: "step-1"}},
	}

	eng := usecase.NewEngine(nil)
	result, err := eng.Execute(ctx, def, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	completedAt := time.Now()
	var errMsg *string
	if result.ErrorMessage != "" {
		errMsg = &result.ErrorMessage
	}

	err = repo.UpdateRun(ctx, "run-1", tenantID, string(result.Status), &completedAt, errMsg)
	if err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	for id, log := range result.StepLogs {
		err = repo.CreateLog(ctx, &domain.ExecutionLog{
			ID:            "log-" + id,
			TenantID:      tenantID,
			WorkflowRunID: "run-1",
			StepID:        id,
			Status:        string(log.Status),
			RetryCount:    log.RetryCount,
			DurationMS:    log.DurationMS,
			LogOutput:     log.LogOutput,
		})
		if err != nil {
			t.Fatalf("failed to create log: %v", err)
		}
	}

	// Verify run record updates
	var runRecord domain.WorkflowRun
	if err := dbConn.First(&runRecord, "id = ?", "run-1").Error; err != nil {
		t.Fatalf("failed to fetch run: %v", err)
	}
	if runRecord.Status != string(usecase.StatusSuccess) {
		t.Errorf("expected Success status, got %v", runRecord.Status)
	}
	if runRecord.CompletedAt == nil {
		t.Error("expected CompletedAt to be set, got nil")
	}
	if runRecord.ErrorMessage != nil {
		t.Errorf("expected ErrorMessage to be nil, got %s", *runRecord.ErrorMessage)
	}

	// Verify execution logs are present
	var logRecord domain.ExecutionLog
	if err := dbConn.First(&logRecord, "workflow_run_id = ? AND step_id = ?", "run-1", "step-1").Error; err != nil {
		t.Fatalf("failed to fetch step log: %v", err)
	}
	if logRecord.Status != string(usecase.StatusSuccess) {
		t.Errorf("expected step log status Success, got %s", logRecord.Status)
	}
}
