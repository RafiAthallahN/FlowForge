package controllers

import (
	"github.com/flow-forger/flow-forger/backend/repository"
	"github.com/gofiber/fiber/v2"
)

type HealthController struct {
	runRepo *repository.RunRepository
}

func NewHealthController(runRepo *repository.RunRepository) *HealthController {
	return &HealthController{runRepo: runRepo}
}

func (ctrl *HealthController) GetMetrics(c *fiber.Ctx) error {
	metrics, err := ctrl.runRepo.GetHealthMetrics(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(metrics)
}
