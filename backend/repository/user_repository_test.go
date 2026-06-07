package repository

import (
	"context"
	"testing"

	"github.com/flow-forger/flow-forger/backend/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserRepository(t *testing.T) {
	db, err := InitDB(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	repo := NewUserRepository(db)
	ctx := context.Background()

	tenantID := "tenant-test"
	// Create tenant first
	err = db.Create(&domain.Tenant{ID: tenantID, TenantID: tenantID, Name: "Test Tenant"}).Error
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	user := &domain.User{
		ID:           "user-1",
		TenantID:     tenantID,
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "Admin",
	}

	// Create user (must be scoped by context or created explicitly)
	ctxWithTenant := context.WithValue(ctx, domain.ContextKeyTenantID, tenantID)
	err = repo.CreateUser(ctxWithTenant, user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Test GetUserByEmail respecting tenant isolation
	t.Run("Get User By Email Success", func(t *testing.T) {
		fetched, err := repo.GetUserByEmail(ctxWithTenant, "test@example.com")
		if err != nil {
			t.Fatalf("failed to get user: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected user, got nil")
		}
		if fetched.ID != "user-1" {
			t.Errorf("expected user-1, got %s", fetched.ID)
		}
	})

	t.Run("Get User By Email Mismatch Tenant", func(t *testing.T) {
		ctxWithWrongTenant := context.WithValue(context.Background(), domain.ContextKeyTenantID, "wrong-tenant")
		fetched, err := repo.GetUserByEmail(ctxWithWrongTenant, "test@example.com")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err != gorm.ErrRecordNotFound {
			t.Errorf("expected ErrRecordNotFound, got %v", err)
		}
		if fetched != nil {
			t.Errorf("expected nil user, got %v", fetched)
		}
	})

	t.Run("Get User By Email Not Found", func(t *testing.T) {
		fetched, err := repo.GetUserByEmail(ctxWithTenant, "nonexistent@example.com")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err != gorm.ErrRecordNotFound {
			t.Errorf("expected ErrRecordNotFound, got %v", err)
		}
		if fetched != nil {
			t.Errorf("expected nil user, got %v", fetched)
		}
	})
}
