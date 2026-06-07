package controllers

import (
	"strconv"

	"github.com/flow-forger/flow-forger/backend/repository"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type RunController struct {
	runRepo *repository.RunRepository
}

func NewRunController(runRepo *repository.RunRepository) *RunController {
	return &RunController{runRepo: runRepo}
}

func (ctrl *RunController) ListRuns(c *fiber.Ctx) error {
	workflowID := c.Query("workflow_id")
	page := 1
	limit := 10

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	runs, total, err := ctrl.runRepo.ListRuns(c.UserContext(), workflowID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"runs":  runs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (ctrl *RunController) GetRun(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Run ID required"})
	}

	run, logs, err := ctrl.runRepo.GetRunWithLogs(c.UserContext(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Run not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"run":  run,
		"logs": logs,
	})
}
