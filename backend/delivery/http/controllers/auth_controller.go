package controllers

import (
	"context"
	"strings"

	"github.com/flow-forger/flow-forger/backend/delivery/http/middleware"
	"github.com/flow-forger/flow-forger/backend/domain"
	"github.com/flow-forger/flow-forger/backend/repository"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthController struct {
	db       *gorm.DB
	userRepo *repository.UserRepository
	validate *validator.Validate
}

func NewAuthController(db *gorm.DB, userRepo *repository.UserRepository) *AuthController {
	return &AuthController{
		db:       db,
		userRepo: userRepo,
		validate: validator.New(),
	}
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Role     string `json:"role" validate:"required,oneof=Admin Editor Viewer"`
	TenantID string `json:"tenant_id" validate:"required"`
}

func (ctrl *AuthController) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := ctrl.validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Ensure tenant exists, if not create it
	var tenant domain.Tenant
	err := ctrl.db.First(&tenant, "id = ?", req.TenantID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Auto-provision tenant
			tenant = domain.Tenant{
				ID:       req.TenantID,
				TenantID: req.TenantID,
				Name:     "Tenant " + req.TenantID,
			}
			if createErr := ctrl.db.Create(&tenant).Error; createErr != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to auto-create tenant: " + createErr.Error()})
			}
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	// Hash password
	bytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		TenantID:     req.TenantID,
		Email:        req.Email,
		PasswordHash: string(bytes),
		Role:         req.Role,
	}

	// Create user with context containing TenantID to pass GORM plugin checks
	ctx := context.WithValue(c.UserContext(), domain.ContextKeyTenantID, req.TenantID)
	if err := ctrl.userRepo.CreateUser(ctx, user); err != nil {
		// Handle duplicate email unique key violation
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "User with this email already exists in this tenant"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":        user.ID,
		"tenant_id": user.TenantID,
		"email":     user.Email,
		"role":      user.Role,
	})
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	TenantID string `json:"tenant_id" validate:"required"`
}

func (ctrl *AuthController) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := ctrl.validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Pass TenantID via context for tenant-scoped email query
	ctx := context.WithValue(c.UserContext(), domain.ContextKeyTenantID, req.TenantID)
	user, err := ctrl.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email, password, or tenant ID"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email, password, or tenant ID"})
	}

	// Generate JWT Token
	token, err := middleware.GenerateToken(user.ID, user.TenantID, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.JSON(fiber.Map{
		"token": token,
	})
}
