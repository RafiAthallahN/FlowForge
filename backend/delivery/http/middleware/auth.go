package middleware

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/flow-forger/flow-forger/backend/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return []byte("super-secret-key")
	}
	return []byte(secret)
}

type Claims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(userID string, tenantID string, role string) (string, error) {
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

func AuthenticateJWT() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		var tokenStr string

		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// Check query param for SSE / EventSource support
			tokenStr = c.Query("token")
			if tokenStr == "" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or invalid authorization header"})
			}
		} else {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return getJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		// Inject to context using both type-safe and string keys for maximum compatibility
		ctx := context.WithValue(c.UserContext(), domain.ContextKeyTenantID, claims.TenantID)
		ctx = context.WithValue(ctx, domain.ContextKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, domain.ContextKeyUserRole, claims.Role)
		ctx = context.WithValue(ctx, "tenant_id", claims.TenantID)
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "user_role", claims.Role)
		c.SetUserContext(ctx)

		return c.Next()
	}
}

func RequireRoles(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleVal := c.UserContext().Value(domain.ContextKeyUserRole)
		if roleVal == nil {
			roleVal = c.UserContext().Value("user_role")
		}
		if roleVal == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		userRole := roleVal.(string)
		for _, r := range allowedRoles {
			if r == userRole {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden - Insufficient permissions"})
	}
}
