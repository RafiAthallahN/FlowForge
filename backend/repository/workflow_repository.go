package repository

import (
	"context"
	"fmt"

	"github.com/flow-forger/flow-forger/backend/domain"
	"gorm.io/gorm"
)

type WorkflowRepository struct {
	db *gorm.DB
}

func NewWorkflowRepository(db *gorm.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

func getContextTenantID(ctx context.Context) string {
	if val, ok := ctx.Value(domain.ContextKeyTenantID).(string); ok {
		return val
	}
	if val, ok := ctx.Value("tenant_id").(string); ok {
		return val
	}
	return ""
}

func (r *WorkflowRepository) ListWorkflows(ctx context.Context, page, limit int) ([]domain.Workflow, int64, error) {
	var list []domain.Workflow
	var total int64

	// Count unique workflows for the current tenant
	err := r.db.WithContext(ctx).Model(&domain.Workflow{}).
		Distinct("id").Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	// Query the latest version of each workflow for the current tenant.
	// Since tenant isolation plugin is active, all subqueries and queries
	// are automatically filtered by tenant_id = tenantID.
	subQuery := r.db.WithContext(ctx).
		Select("id, MAX(version) as max_ver").
		Model(&domain.Workflow{}).
		Group("id")

	err = r.db.WithContext(ctx).Table("workflows").
		Joins("JOIN (?) as latest ON workflows.id = latest.id AND workflows.version = latest.max_ver", subQuery).
		Limit(limit).Offset(offset).
		Order("workflows.created_at DESC").
		Scan(&list).Error

	return list, total, err
}

func (r *WorkflowRepository) GetWorkflow(ctx context.Context, id string, version *int) (*domain.Workflow, error) {
	var wf domain.Workflow
	query := r.db.WithContext(ctx).Where("id = ?", id)
	if version != nil {
		query = query.Where("version = ?", *version)
	} else {
		query = query.Order("version DESC")
	}

	if err := query.First(&wf).Error; err != nil {
		return nil, err
	}
	return &wf, nil
}

func (r *WorkflowRepository) CreateWorkflow(ctx context.Context, wf *domain.Workflow) error {
	wf.Version = 1
	tenantID := getContextTenantID(ctx)
	if tenantID != "" {
		wf.TenantID = tenantID
	}
	return r.db.WithContext(ctx).Create(wf).Error
}

func (r *WorkflowRepository) UpdateWorkflow(ctx context.Context, id string, name string, definition string) (*domain.Workflow, error) {
	latest, err := r.GetWorkflow(ctx, id, nil)
	if err != nil {
		return nil, err
	}

	tenantID := getContextTenantID(ctx)
	newWf := domain.Workflow{
		ID:         id,
		TenantID:   tenantID,
		Version:    latest.Version + 1,
		Name:       name,
		Definition: definition,
	}

	if err := r.db.WithContext(ctx).Create(&newWf).Error; err != nil {
		return nil, err
	}
	return &newWf, nil
}

func (r *WorkflowRepository) RollbackWorkflow(ctx context.Context, id string, targetVersion int) (*domain.Workflow, error) {
	target, err := r.GetWorkflow(ctx, id, &targetVersion)
	if err != nil {
		return nil, fmt.Errorf("target version not found: %w", err)
	}

	latest, err := r.GetWorkflow(ctx, id, nil)
	if err != nil {
		return nil, err
	}

	tenantID := getContextTenantID(ctx)
	rolled := domain.Workflow{
		ID:         id,
		TenantID:   tenantID,
		Version:    latest.Version + 1,
		Name:       target.Name,
		Definition: target.Definition,
	}

	if err := r.db.WithContext(ctx).Create(&rolled).Error; err != nil {
		return nil, err
	}
	return &rolled, nil
}
