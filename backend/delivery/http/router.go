package http

import (
	"time"

	"github.com/flow-forger/flow-forger/backend/delivery/http/controllers"
	"github.com/flow-forger/flow-forger/backend/delivery/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func SetupRoutes(
	app *fiber.App,
	authCtrl *controllers.AuthController,
	wfCtrl *controllers.WorkflowController,
	runCtrl *controllers.RunController,
	healthCtrl *controllers.HealthController,
	sseCtrl *controllers.SSEController,
) {
	// CORS configuration
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173,http://127.0.0.1:5173,http://localhost:82,http://127.0.0.1:82,http://localhost,http://127.0.0.1",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// Global Rate Limiter: max 100 requests per 1 minute
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
	}))

	api := app.Group("/api")

	// Authentication endpoints
	auth := api.Group("/auth")
	auth.Post("/register", authCtrl.Register)
	auth.Post("/login", authCtrl.Login)

	// Workflows endpoints (Requires valid JWT)
	workflows := api.Group("/workflows", middleware.AuthenticateJWT())

	// Read operations: accessible by Viewer, Editor, Admin
	workflows.Get("/", middleware.RequireRoles("Admin", "Editor", "Viewer"), wfCtrl.ListWorkflows)
	workflows.Get("/:id", middleware.RequireRoles("Admin", "Editor", "Viewer"), wfCtrl.GetWorkflow)

	// Write operations: accessible by Editor and Admin only
	workflows.Post("/", middleware.RequireRoles("Admin", "Editor"), wfCtrl.CreateWorkflow)
	workflows.Put("/:id", middleware.RequireRoles("Admin", "Editor"), wfCtrl.UpdateWorkflow)
	workflows.Post("/:id/rollback", middleware.RequireRoles("Admin", "Editor"), wfCtrl.RollbackWorkflow)
	workflows.Post("/:id/run", middleware.RequireRoles("Admin", "Editor"), wfCtrl.RunWorkflow)

	// Run history endpoints (Requires valid JWT)
	runs := api.Group("/runs", middleware.AuthenticateJWT())
	runs.Get("/", middleware.RequireRoles("Admin", "Editor", "Viewer"), runCtrl.ListRuns)
	runs.Get("/:id", middleware.RequireRoles("Admin", "Editor", "Viewer"), runCtrl.GetRun)

	// Health metrics endpoint (Requires valid JWT)
	health := api.Group("/health", middleware.AuthenticateJWT())
	health.Get("/metrics", middleware.RequireRoles("Admin", "Editor", "Viewer"), healthCtrl.GetMetrics)

	// SSE event stream endpoint (Requires valid JWT)
	sse := api.Group("/events", middleware.AuthenticateJWT())
	sse.Get("/stream", sseCtrl.Stream)
}

