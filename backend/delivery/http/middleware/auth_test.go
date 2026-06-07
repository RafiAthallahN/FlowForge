package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flow-forger/flow-forger/backend/domain"
	"github.com/gofiber/fiber/v2"
)

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken("user-123", "tenant-456", "Admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken returned empty token")
	}
}

func TestAuthenticateJWT(t *testing.T) {
	app := fiber.New()

	app.Get("/protected", AuthenticateJWT(), func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		return c.JSON(fiber.Map{
			"tenant_id":        ctx.Value("tenant_id"),
			"user_id":          ctx.Value("user_id"),
			"user_role":        ctx.Value("user_role"),
			"type_safe_tenant": ctx.Value(domain.ContextKeyTenantID),
			"type_safe_user":   ctx.Value(domain.ContextKeyUserID),
			"type_safe_role":   ctx.Value(domain.ContextKeyUserRole),
		})
	})

	token, err := GenerateToken("user-123", "tenant-456", "Admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	t.Run("Missing Auth Header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Test request failed: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Invalid Header Prefix", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Test request failed: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Invalid Token Signature", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token-string")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Test request failed: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Valid Token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Test request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if body["user_id"] != "user-123" {
			t.Errorf("expected user_id user-123, got %v", body["user_id"])
		}
		if body["tenant_id"] != "tenant-456" {
			t.Errorf("expected tenant_id tenant-456, got %v", body["tenant_id"])
		}
		if body["user_role"] != "Admin" {
			t.Errorf("expected user_role Admin, got %v", body["user_role"])
		}
		if body["type_safe_user"] != "user-123" {
			t.Errorf("expected type_safe_user user-123, got %v", body["type_safe_user"])
		}
		if body["type_safe_tenant"] != "tenant-456" {
			t.Errorf("expected type_safe_tenant tenant-456, got %v", body["type_safe_tenant"])
		}
		if body["type_safe_role"] != "Admin" {
			t.Errorf("expected type_safe_role Admin, got %v", body["type_safe_role"])
		}
	})
}

func TestRequireRoles(t *testing.T) {
	app := fiber.New()

	app.Get("/admin", AuthenticateJWT(), RequireRoles("Admin"), func(c *fiber.Ctx) error {
		return c.SendString("admin access")
	})

	app.Get("/editor-or-admin", AuthenticateJWT(), RequireRoles("Admin", "Editor"), func(c *fiber.Ctx) error {
		return c.SendString("editor or admin access")
	})

	app.Get("/no-auth-require-roles", RequireRoles("Admin"), func(c *fiber.Ctx) error {
		return c.SendString("no auth")
	})

	tokenAdmin, _ := GenerateToken("u-admin", "t-1", "Admin")
	tokenEditor, _ := GenerateToken("u-editor", "t-1", "Editor")
	tokenViewer, _ := GenerateToken("u-viewer", "t-1", "Viewer")

	t.Run("Role Allowed (Admin on /admin)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Test request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Role Forbidden (Viewer on /admin)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+tokenViewer)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Test request failed: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", resp.StatusCode)
		}
	})

	t.Run("Role Allowed Multi (Editor on /editor-or-admin)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/editor-or-admin", nil)
		req.Header.Set("Authorization", "Bearer "+tokenEditor)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Test request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Role Allowed Multi (Admin on /editor-or-admin)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/editor-or-admin", nil)
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Test request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Role Forbidden Multi (Viewer on /editor-or-admin)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/editor-or-admin", nil)
		req.Header.Set("Authorization", "Bearer "+tokenViewer)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Test request failed: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", resp.StatusCode)
		}
	})

	t.Run("No Auth context (missing roles value in context)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/no-auth-require-roles", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Test request failed: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
	})
}
