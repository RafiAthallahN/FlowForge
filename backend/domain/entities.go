package domain

import (
	"time"
)

type Tenant struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	TenantID  string    `gorm:"not null" json:"tenant_id"` // Strictly satisfies spec "every single table must include a tenant_id column"
	Name      string    `gorm:"not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	TenantID     string    `gorm:"uniqueIndex:idx_tenant_email;not null;index" json:"tenant_id"`
	Tenant       Tenant    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Email        string    `gorm:"uniqueIndex:idx_tenant_email;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         string    `gorm:"not null" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Workflow struct {
	TenantID   string    `gorm:"primaryKey" json:"tenant_id"`
	Tenant     Tenant    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	ID         string    `gorm:"primaryKey" json:"id"`
	Version    int       `gorm:"primaryKey" json:"version"`
	Name       string    `gorm:"not null" json:"name"`
	Definition string    `gorm:"type:text;not null" json:"definition"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type WorkflowRun struct {
	ID              string     `gorm:"primaryKey" json:"id"`
	TenantID        string     `gorm:"not null;index:idx_runs_tenant_workflow_created,priority:1" json:"tenant_id"`
	Tenant          Tenant     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	WorkflowID      string     `gorm:"not null;index:idx_runs_tenant_workflow_created,priority:2" json:"workflow_id"`
	WorkflowVersion int        `gorm:"not null" json:"workflow_version"`
	Workflow        Workflow   `gorm:"foreignKey:TenantID,WorkflowID,WorkflowVersion;references:TenantID,ID,Version;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Status          string     `gorm:"not null" json:"status"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	ErrorMessage    *string    `json:"error_message"`
	Metadata        string     `json:"metadata"`
	CreatedAt       time.Time  `gorm:"index:idx_runs_tenant_workflow_created,priority:3" json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type ExecutionLog struct {
	ID            string      `gorm:"primaryKey" json:"id"`
	TenantID      string      `gorm:"not null;index" json:"tenant_id"`
	Tenant        Tenant      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	WorkflowRunID string      `gorm:"not null;index" json:"workflow_run_id"`
	WorkflowRun   WorkflowRun `gorm:"foreignKey:WorkflowRunID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	StepID        string      `gorm:"not null" json:"step_id"`
	Status        string      `gorm:"not null" json:"status"`
	RetryCount    int         `gorm:"not null;default:0" json:"retry_count"`
	DurationMS    int64       `gorm:"not null;default:0" json:"duration_ms"`
	LogOutput     string      `gorm:"type:text" json:"log_output"`
	FailureReason string      `gorm:"type:text" json:"failure_reason,omitempty"`
	SuggestedFix  string      `gorm:"type:text" json:"suggested_fix,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}
