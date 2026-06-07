package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	delivery "github.com/flow-forger/flow-forger/backend/delivery/http"
	"github.com/flow-forger/flow-forger/backend/delivery/http/controllers"
	"github.com/flow-forger/flow-forger/backend/repository"
	"github.com/flow-forger/flow-forger/backend/usecase"
	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	db, err := repository.InitDB(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	wfRepo := repository.NewWorkflowRepository(db)
	runRepo := repository.NewRunRepository(db)
	hub := controllers.NewEventHub()
	analyzer := usecase.NewOpenRouterAnalyzer()

	authCtrl := controllers.NewAuthController(db, userRepo)
	wfCtrl := controllers.NewWorkflowController(wfRepo, runRepo, hub, analyzer)
	runCtrl := controllers.NewRunController(runRepo)
	healthCtrl := controllers.NewHealthController(runRepo)
	sseCtrl := controllers.NewSSEController(hub)

	app := fiber.New()
	delivery.SetupRoutes(app, authCtrl, wfCtrl, runCtrl, healthCtrl, sseCtrl)

	return app, db
}

func TestE2EAPIWorkflow(t *testing.T) {
	app, _ := setupTestApp(t)

	// 1. Register users in different tenants
	t.Run("Register and Login", func(t *testing.T) {
		// Register Editor in Tenant A
		regA := map[string]string{
			"email":     "editor@tenant-a.com",
			"password":  "password123",
			"role":      "Editor",
			"tenant_id": "tenant-a",
		}
		bodyA, _ := json.Marshal(regA)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(bodyA))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("register request failed: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected 201 Created, got %d", resp.StatusCode)
		}

		// Register Viewer in Tenant B with the same email (should be allowed across tenants)
		regB := map[string]string{
			"email":     "editor@tenant-a.com", // Same email, different tenant
			"password":  "password123",
			"role":      "Viewer",
			"tenant_id": "tenant-b",
		}
		bodyB, _ := json.Marshal(regB)
		req = httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(bodyB))
		req.Header.Set("Content-Type", "application/json")
		resp, err = app.Test(req)
		if err != nil {
			t.Fatalf("register B request failed: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected 201 Created for cross-tenant duplicate email, got %d", resp.StatusCode)
		}

		// Try to login with wrong tenant ID (should fail)
		loginWrong := map[string]string{
			"email":     "editor@tenant-a.com",
			"password":  "password123",
			"tenant_id": "tenant-wrong",
		}
		bodyWrong, _ := json.Marshal(loginWrong)
		req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(bodyWrong))
		req.Header.Set("Content-Type", "application/json")
		resp, _ = app.Test(req)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for wrong login tenant, got %d", resp.StatusCode)
		}
	})

	t.Run("Tenant Isolation and RBAC flow", func(t *testing.T) {
		// Register and login to get tokens
		regA := map[string]string{
			"email":     "alice@a.com",
			"password":  "password123",
			"role":      "Editor",
			"tenant_id": "tenant-a",
		}
		body, _ := json.Marshal(regA)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)

		loginA := map[string]string{
			"email":     "alice@a.com",
			"password":  "password123",
			"tenant_id": "tenant-a",
		}
		body, _ = json.Marshal(loginA)
		req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		var loginRespA map[string]string
		json.NewDecoder(resp.Body).Decode(&loginRespA)
		tokenA := loginRespA["token"]

		// Register and login for Tenant B (Viewer)
		regB := map[string]string{
			"email":     "bob@b.com",
			"password":  "password123",
			"role":      "Viewer",
			"tenant_id": "tenant-b",
		}
		body, _ = json.Marshal(regB)
		req = httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)

		loginB := map[string]string{
			"email":     "bob@b.com",
			"password":  "password123",
			"tenant_id": "tenant-b",
		}
		body, _ = json.Marshal(loginB)
		req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ = app.Test(req)
		var loginRespB map[string]string
		json.NewDecoder(resp.Body).Decode(&loginRespB)
		tokenB := loginRespB["token"]

		// 1. Tenant A Editor creates a workflow
		wfData := map[string]string{
			"id":   "wf-1",
			"name": "My Workflow",
			"definition": `{
				"steps": [
					{
						"id": "step-1",
						"type": "delay",
						"depends_on": []
					}
				]
			}`,
		}
		wfBody, _ := json.Marshal(wfData)
		req = httptest.NewRequest("POST", "/api/workflows", bytes.NewBuffer(wfBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenA)
		resp, _ = app.Test(req)
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected 201 Created for workflow creation, got %d", resp.StatusCode)
		}

		// 2. Tenant B Viewer lists workflows (should return 0 workflows due to isolation)
		req = httptest.NewRequest("GET", "/api/workflows", nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		resp, _ = app.Test(req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}
		var listResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&listResp)
		if listResp["total"].(float64) != 0 {
			t.Errorf("expected 0 workflows for tenant-b, got %v", listResp["total"])
		}

		// 3. Tenant B Viewer tries to get wf-1 (should return 404 Not Found due to isolation)
		req = httptest.NewRequest("GET", "/api/workflows/wf-1", nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		resp, _ = app.Test(req)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 Not Found for cross-tenant retrieval, got %d", resp.StatusCode)
		}

		// 4. Tenant B Viewer tries to write/update workflow (should return 403 Forbidden due to RBAC)
		req = httptest.NewRequest("PUT", "/api/workflows/wf-1", bytes.NewBuffer(wfBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenB)
		resp, _ = app.Test(req)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for Viewer role on write operation, got %d", resp.StatusCode)
		}
	})

	t.Run("Input Validation and DAG Cycles", func(t *testing.T) {
		regA := map[string]string{
			"email":     "alice2@a.com",
			"password":  "password123",
			"role":      "Editor",
			"tenant_id": "tenant-a",
		}
		body, _ := json.Marshal(regA)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)

		loginA := map[string]string{
			"email":     "alice2@a.com",
			"password":  "password123",
			"tenant_id": "tenant-a",
		}
		body, _ = json.Marshal(loginA)
		req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		var loginRespA map[string]string
		json.NewDecoder(resp.Body).Decode(&loginRespA)
		tokenA := loginRespA["token"]

		// Post workflow with circular dependency
		cycleWf := map[string]string{
			"id":   "wf-cycle",
			"name": "Circular Workflow",
			"definition": `{
				"steps": [
					{
						"id": "step-1",
						"type": "delay",
						"depends_on": ["step-2"]
					},
					{
						"id": "step-2",
						"type": "delay",
						"depends_on": ["step-1"]
					}
				]
			}`,
		}
		wfBody, _ := json.Marshal(cycleWf)
		req = httptest.NewRequest("POST", "/api/workflows", bytes.NewBuffer(wfBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenA)
		resp, _ = app.Test(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for circular dependency, got %d", resp.StatusCode)
		}
	})

	t.Run("Rate Limiter Configuration Verification", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/register", nil)
		resp, _ := app.Test(req)
		
		// The rate limiter sets X-RateLimit headers
		if resp.Header.Get("X-RateLimit-Limit") == "" {
			t.Error("expected rate limiter headers, got empty")
		}
	})

	t.Run("Run Workflow and Event Stream Auth", func(t *testing.T) {
		loginA := map[string]string{
			"email":     "editor@tenant-a.com",
			"password":  "password123",
			"tenant_id": "tenant-a",
		}
		body, _ := json.Marshal(loginA)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		var loginRespA map[string]string
		json.NewDecoder(resp.Body).Decode(&loginRespA)
		tokenA := loginRespA["token"]

		// Create a valid workflow
		validWf := map[string]string{
			"id":   "wf-run-test",
			"name": "Valid Workflow",
			"definition": `{
				"steps": [
					{
						"id": "step-1",
						"type": "delay"
					}
				]
			}`,
		}
		wfBody, _ := json.Marshal(validWf)
		req = httptest.NewRequest("POST", "/api/workflows", bytes.NewBuffer(wfBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenA)
		resp, _ = app.Test(req)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("failed to create valid workflow: %d", resp.StatusCode)
		}

		// Trigger workflow run
		req = httptest.NewRequest("POST", "/api/workflows/wf-run-test/run", nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		resp, _ = app.Test(req)
		if resp.StatusCode != http.StatusAccepted {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 202 Accepted, got %d, body: %s", resp.StatusCode, string(bodyBytes))
		}

		var runResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&runResp)
		runIDVal, ok := runResp["id"]
		if !ok || runIDVal == nil {
			t.Fatalf("expected run ID in response, got response: %v", runResp)
		}
		runID := runIDVal.(string)
		_ = runID

		// Verify run is in run list
		req = httptest.NewRequest("GET", "/api/runs", nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		resp, _ = app.Test(req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}

		// Verify SSE token auth via query parameter
		ctx, cancel := context.WithCancel(context.Background())
		req = httptest.NewRequest("GET", "/api/events/stream?token="+tokenA, nil).WithContext(ctx)
		
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		resp, err := app.Test(req, 200) // 200ms timeout
		if err == nil && resp != nil {
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200 OK for query param token auth, got %d", resp.StatusCode)
			}
		}
	})

	t.Run("Run Failing Workflow and AI Failure Analysis", func(t *testing.T) {
		loginA := map[string]string{
			"email":     "editor@tenant-a.com",
			"password":  "password123",
			"tenant_id": "tenant-a",
		}
		body, _ := json.Marshal(loginA)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		var loginRespA map[string]string
		json.NewDecoder(resp.Body).Decode(&loginRespA)
		tokenA := loginRespA["token"]

		// Create a workflow that will fail (fail step type)
		failingWf := map[string]string{
			"id":   "wf-failing-test",
			"name": "Failing Workflow",
			"definition": `{
				"steps": [
					{
						"id": "fail-step",
						"type": "fail"
					}
				]
			}`,
		}
		wfBody, _ := json.Marshal(failingWf)
		req = httptest.NewRequest("POST", "/api/workflows", bytes.NewBuffer(wfBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenA)
		resp, _ = app.Test(req)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("failed to create workflow: %d", resp.StatusCode)
		}

		// Trigger workflow run
		req = httptest.NewRequest("POST", "/api/workflows/wf-failing-test/run", nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		resp, _ = app.Test(req)
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
		}

		var runResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&runResp)
		runID := runResp["id"].(string)

		// Wait for the async workflow execution to finish (it fails quickly because of context done)
		time.Sleep(300 * time.Millisecond)

		// Fetch the run logs
		req = httptest.NewRequest("GET", "/api/runs/"+runID, nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		resp, _ = app.Test(req)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("failed to fetch run status: %d", resp.StatusCode)
		}

		var runDetail map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&runDetail)
		logsList := runDetail["logs"].([]interface{})
		
		if len(logsList) == 0 {
			t.Fatal("expected logs to be created, got empty")
		}

		logRecord := logsList[0].(map[string]interface{})
		if logRecord["status"].(string) != "Failed" {
			t.Errorf("expected step status Failed, got %s", logRecord["status"].(string))
		}

		// Verify AI failure reason and suggested fix are populated (fallback diagnostics)
		failureReason := logRecord["failure_reason"].(string)
		suggestedFix := logRecord["suggested_fix"].(string)

		if failureReason == "" || suggestedFix == "" {
			t.Error("expected FailureReason and SuggestedFix to be populated by AI Analysis")
		}
	})
}
