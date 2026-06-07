package repository

import (
	"context"
	"time"

	"github.com/flow-forger/flow-forger/backend/domain"
	"gorm.io/gorm"
)

type HealthMetrics struct {
	ActiveRuns   int64   `json:"active_runs"`
	TotalRuns    int64   `json:"total_runs"`
	SuccessCount int64   `json:"success_count"`
	FailureCount int64   `json:"failure_count"`
	SuccessRate  float64 `json:"success_rate"`
	FailureRate  float64 `json:"failure_rate"`
	AvgDuration  float64 `json:"avg_duration_ms"`
}

type RunRepository struct {
	db *gorm.DB
}

func NewRunRepository(db *gorm.DB) *RunRepository {
	return &RunRepository{db: db}
}

func (r *RunRepository) CreateRun(ctx context.Context, run *domain.WorkflowRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *RunRepository) UpdateRun(ctx context.Context, id string, tenantID string, status string, completedAt *time.Time, errMsg *string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if completedAt != nil {
		updates["completed_at"] = *completedAt
	} else {
		updates["completed_at"] = nil
	}
	if errMsg != nil {
		updates["error_message"] = *errMsg
	} else {
		updates["error_message"] = nil
	}

	result := r.db.WithContext(ctx).Model(&domain.WorkflowRun{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *RunRepository) CreateLog(ctx context.Context, log *domain.ExecutionLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *RunRepository) ListRuns(ctx context.Context, workflowID string, page, limit int) ([]domain.WorkflowRun, int64, error) {
	var runs []domain.WorkflowRun
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.WorkflowRun{})
	if workflowID != "" {
		query = query.Where("workflow_id = ?", workflowID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).
		Where("workflow_id = ? OR ? = ''", workflowID, workflowID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&runs).Error

	return runs, total, err
}

func (r *RunRepository) GetRunWithLogs(ctx context.Context, runID string) (*domain.WorkflowRun, []domain.ExecutionLog, error) {
	var run domain.WorkflowRun
	if err := r.db.WithContext(ctx).First(&run, "id = ?", runID).Error; err != nil {
		return nil, nil, err
	}

	var logs []domain.ExecutionLog
	if err := r.db.WithContext(ctx).
		Where("workflow_run_id = ?", runID).
		Order("created_at ASC").
		Find(&logs).Error; err != nil {
		return nil, nil, err
	}

	return &run, logs, nil
}

func (r *RunRepository) GetHealthMetrics(ctx context.Context) (*HealthMetrics, error) {
	cutoff := time.Now().Add(-24 * time.Hour)
	metrics := &HealthMetrics{}

	// Active runs (status = Running)
	if err := r.db.WithContext(ctx).Model(&domain.WorkflowRun{}).
		Where("status = ?", "Running").
		Count(&metrics.ActiveRuns).Error; err != nil {
		return nil, err
	}

	// Total completed runs in 24h window
	if err := r.db.WithContext(ctx).Model(&domain.WorkflowRun{}).
		Where("completed_at >= ?", cutoff).
		Count(&metrics.TotalRuns).Error; err != nil {
		return nil, err
	}

	// Success count
	if err := r.db.WithContext(ctx).Model(&domain.WorkflowRun{}).
		Where("status = ? AND completed_at >= ?", "Success", cutoff).
		Count(&metrics.SuccessCount).Error; err != nil {
		return nil, err
	}

	// Failure count
	if err := r.db.WithContext(ctx).Model(&domain.WorkflowRun{}).
		Where("status = ? AND completed_at >= ?", "Failed", cutoff).
		Count(&metrics.FailureCount).Error; err != nil {
		return nil, err
	}

	// Rates
	if metrics.TotalRuns > 0 {
		metrics.SuccessRate = float64(metrics.SuccessCount) / float64(metrics.TotalRuns) * 100
		metrics.FailureRate = float64(metrics.FailureCount) / float64(metrics.TotalRuns) * 100
	}

	// Average duration of completed runs in 24h window (computed from started_at to completed_at)
	var avgResult struct {
		AvgDuration float64
	}
	r.db.WithContext(ctx).Model(&domain.WorkflowRun{}).
		Select("AVG((julianday(completed_at) - julianday(started_at)) * 86400000) as avg_duration").
		Where("completed_at >= ? AND started_at IS NOT NULL AND completed_at IS NOT NULL", cutoff).
		Scan(&avgResult)
	metrics.AvgDuration = avgResult.AvgDuration

	return metrics, nil
}
