package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flow-forger/flow-forger/backend/domain"
	"github.com/flow-forger/flow-forger/backend/repository"
	"github.com/flow-forger/flow-forger/backend/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WorkflowController struct {
	workflowRepo *repository.WorkflowRepository
	runRepo      *repository.RunRepository
	hub          *EventHub
	analyzer     usecase.ErrorAnalyzer
	validate     *validator.Validate
}

func NewWorkflowController(
	workflowRepo *repository.WorkflowRepository,
	runRepo *repository.RunRepository,
	hub *EventHub,
	analyzer usecase.ErrorAnalyzer,
) *WorkflowController {
	return &WorkflowController{
		workflowRepo: workflowRepo,
		runRepo:      runRepo,
		hub:          hub,
		analyzer:     analyzer,
		validate:     validator.New(),
	}
}

type CreateWorkflowRequest struct {
	ID         string `json:"id" validate:"required"`
	Name       string `json:"name" validate:"required"`
	Definition string `json:"definition" validate:"required"`
}

type UpdateWorkflowRequest struct {
	Name       string `json:"name" validate:"required"`
	Definition string `json:"definition" validate:"required"`
}

type RollbackWorkflowRequest struct {
	Version int `json:"version" validate:"required,min=1"`
}

func validateDefinition(defStr string) error {
	var def usecase.WorkflowDefinition
	if err := json.Unmarshal([]byte(defStr), &def); err != nil {
		return fmt.Errorf("invalid JSON definition: %w", err)
	}

	// Kahn's topological sort check
	if _, err := usecase.ValidateAndSort(def); err != nil {
		return fmt.Errorf("invalid workflow DAG: %w", err)
	}

	return nil
}

func (ctrl *WorkflowController) CreateWorkflow(c *fiber.Ctx) error {
	var req CreateWorkflowRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := ctrl.validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := validateDefinition(req.Definition); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	wf := &domain.Workflow{
		ID:         req.ID,
		Name:       req.Name,
		Definition: req.Definition,
	}

	if err := ctrl.workflowRepo.CreateWorkflow(c.UserContext(), wf); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(wf)
}

func (ctrl *WorkflowController) GetWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Workflow ID required"})
	}

	var versionPtr *int
	if versionStr := c.Query("version"); versionStr != "" {
		v, err := strconv.Atoi(versionStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid version parameter"})
		}
		versionPtr = &v
	}

	wf, err := ctrl.workflowRepo.GetWorkflow(c.UserContext(), id, versionPtr)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Workflow not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(wf)
}

func (ctrl *WorkflowController) ListWorkflows(c *fiber.Ctx) error {
	page := 1
	limit := 10

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	list, total, err := ctrl.workflowRepo.ListWorkflows(c.UserContext(), page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"workflows": list,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

func (ctrl *WorkflowController) UpdateWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Workflow ID required"})
	}

	var req UpdateWorkflowRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := ctrl.validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := validateDefinition(req.Definition); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	wf, err := ctrl.workflowRepo.UpdateWorkflow(c.UserContext(), id, req.Name, req.Definition)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Workflow not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(wf)
}

func (ctrl *WorkflowController) RollbackWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Workflow ID required"})
	}

	var req RollbackWorkflowRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := ctrl.validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	wf, err := ctrl.workflowRepo.RollbackWorkflow(c.UserContext(), id, req.Version)
	if err != nil {
		if err == gorm.ErrRecordNotFound || (err != nil && strings.Contains(err.Error(), "not found")) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Workflow or target version not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(wf)
}

func (ctrl *WorkflowController) RunWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Workflow ID required"})
	}

	wf, err := ctrl.workflowRepo.GetWorkflow(c.UserContext(), id, nil)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Workflow not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var def usecase.WorkflowDefinition
	if err := json.Unmarshal([]byte(wf.Definition), &def); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Invalid workflow definition stored"})
	}

	tenantID, _ := c.UserContext().Value(domain.ContextKeyTenantID).(string)
	if tenantID == "" {
		tenantID, _ = c.UserContext().Value("tenant_id").(string)
	}

	runID := uuid.New().String()
	startedAt := time.Now()

	run := &domain.WorkflowRun{
		ID:              runID,
		TenantID:        tenantID,
		WorkflowID:      wf.ID,
		WorkflowVersion: wf.Version,
		Status:          string(usecase.StatusRunning),
		StartedAt:       &startedAt,
	}

	if err := ctrl.runRepo.CreateRun(c.UserContext(), run); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("failed to create run: %v", err)})
	}

	// Trigger async engine execution
	go func() {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "tenant_id", tenantID)
		ctx = context.WithValue(ctx, domain.ContextKeyTenantID, tenantID)

		eng := usecase.NewEngine(nil)
		eng.OnStepStatusChange = func(stepID string, status usecase.StepStatus, logLine string, durationMS int64) {
			// Publish to SSE EventHub
			ctrl.hub.Publish(tenantID, StepEvent{
				RunID:    runID,
				StepID:   stepID,
				Status:   string(status),
				LogLine:  logLine,
				Duration: durationMS,
			})
		}

		result, err := eng.Execute(ctx, def, 5*time.Minute)
		completedAt := time.Now()
		var errMsg *string
		var finalStatus string

		if err != nil {
			finalStatus = string(usecase.StatusFailed)
			msg := err.Error()
			errMsg = &msg
		} else {
			finalStatus = string(result.Status)
			if result.ErrorMessage != "" {
				errMsg = &result.ErrorMessage
			}

			// Backfill logs to guarantee database consistency for all steps
			for stepID, stepLog := range result.StepLogs {
				var failureReason string
				var suggestedFix string

				if stepLog.Status == usecase.StatusFailed {
					var stepType string
					var configPayload string
					for _, s := range def.Steps {
						if s.ID == stepID {
							stepType = string(s.Type)
							if configBytes, err := json.Marshal(s.Config); err == nil {
								configPayload = string(configBytes)
							} else {
								configPayload = fmt.Sprintf("%v", s.Config)
							}
							break
						}
					}

					analysis, err := ctrl.analyzer.AnalyzeFailure(ctx, stepID, stepType, stepLog.LogOutput, configPayload)
					if err == nil && analysis != nil {
						failureReason = analysis.Reason
						suggestedFix = analysis.SuggestedFix
					}
				}

				logID := uuid.New().String()
				_ = ctrl.runRepo.CreateLog(ctx, &domain.ExecutionLog{
					ID:            logID,
					TenantID:      tenantID,
					WorkflowRunID: runID,
					StepID:        stepID,
					Status:        string(stepLog.Status),
					RetryCount:    stepLog.RetryCount,
					DurationMS:    stepLog.DurationMS,
					LogOutput:     stepLog.LogOutput,
					FailureReason: failureReason,
					SuggestedFix:  suggestedFix,
				})

				// Publish the final SSE event for the step (includes AI failure diagnostics if applicable)
				ctrl.hub.Publish(tenantID, StepEvent{
					RunID:         runID,
					StepID:        stepID,
					Status:        string(stepLog.Status),
					LogLine:       stepLog.LogOutput,
					Duration:      stepLog.DurationMS,
					FailureReason: failureReason,
					SuggestedFix:  suggestedFix,
				})
			}
		}

		_ = ctrl.runRepo.UpdateRun(ctx, runID, tenantID, finalStatus, &completedAt, errMsg)

		// Send final SSE event for the overall workflow completion status
		ctrl.hub.Publish(tenantID, StepEvent{
			RunID:    runID,
			StepID:   "__workflow__",
			Status:   finalStatus,
			LogLine:  "Workflow run completed",
			Duration: time.Since(startedAt).Milliseconds(),
		})
	}()

	return c.Status(fiber.StatusAccepted).JSON(run)
}
