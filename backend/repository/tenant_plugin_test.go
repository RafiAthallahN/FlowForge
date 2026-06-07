package repository

import (
	"context"
	"testing"

	"github.com/flow-forger/flow-forger/backend/domain"
	"gorm.io/driver/sqlite"
)

func TestTenantIsolationPlugin(t *testing.T) {
	db, err := InitDB(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	// Create two tenants
	t1 := "tenant-1"
	t2 := "tenant-2"
	db.Create(&domain.Tenant{ID: t1, TenantID: t1, Name: "Tenant One"})
	db.Create(&domain.Tenant{ID: t2, TenantID: t2, Name: "Tenant Two"})

	// Create user for tenant-1
	u1 := domain.User{
		ID:           "u1",
		TenantID:     t1,
		Email:        "u1@tenant1.com",
		PasswordHash: "hash",
		Role:         "admin",
	}
	if err := db.Create(&u1).Error; err != nil {
		t.Fatalf("failed to create user 1: %v", err)
	}

	// Create user for tenant-2
	u2 := domain.User{
		ID:           "u2",
		TenantID:     t2,
		Email:        "u2@tenant2.com",
		PasswordHash: "hash",
		Role:         "admin",
	}
	if err := db.Create(&u2).Error; err != nil {
		t.Fatalf("failed to create user 2: %v", err)
	}

	// 1. Query tests with tenant isolation context
	t.Run("Query isolated by tenant_id", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), domain.ContextKeyTenantID, t1)
		var users []domain.User
		if err := db.WithContext(ctx).Find(&users).Error; err != nil {
			t.Fatalf("failed to query users: %v", err)
		}
		if len(users) != 1 {
			t.Errorf("expected 1 user, got %d", len(users))
		}
		if len(users) > 0 && users[0].ID != "u1" {
			t.Errorf("expected user u1, got %s", users[0].ID)
		}
	})

	t.Run("Query without tenant isolation context", func(t *testing.T) {
		var users []domain.User
		if err := db.Find(&users).Error; err != nil {
			t.Fatalf("failed to query users: %v", err)
		}
		if len(users) != 2 {
			t.Errorf("expected 2 users, got %d", len(users))
		}
	})

	// 2. Update tests with tenant isolation context
	t.Run("Update isolated by tenant_id", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), domain.ContextKeyTenantID, t1)

		// Attempt to update u2's role using t1's context.
		// Since tenant isolation forces "tenant_id = 'tenant-1'", it should not update u2 (tenant-2).
		res := db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", "u2").Update("role", "member")
		if res.Error != nil {
			t.Fatalf("failed to update user: %v", res.Error)
		}
		if res.RowsAffected != 0 {
			t.Errorf("expected 0 rows affected when updating across tenant boundaries, got %d", res.RowsAffected)
		}

		// Verify u2 role is still admin
		var checkU2 domain.User
		if err := db.First(&checkU2, "id = ?", "u2").Error; err != nil {
			t.Fatalf("failed to fetch user: %v", err)
		}
		if checkU2.Role != "admin" {
			t.Errorf("expected u2 role to remain admin, got %s", checkU2.Role)
		}

		// Update u1's role with t1's context
		res = db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", "u1").Update("role", "member")
		if res.Error != nil {
			t.Fatalf("failed to update user: %v", res.Error)
		}
		if res.RowsAffected != 1 {
			t.Errorf("expected 1 row affected, got %d", res.RowsAffected)
		}

		var checkU1 domain.User
		if err := db.First(&checkU1, "id = ?", "u1").Error; err != nil {
			t.Fatalf("failed to fetch user: %v", err)
		}
		if checkU1.Role != "member" {
			t.Errorf("expected u1 role to be member, got %s", checkU1.Role)
		}
	})

	// 3. Delete tests with tenant isolation context
	t.Run("Delete isolated by tenant_id", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), domain.ContextKeyTenantID, t1)

		// Attempt to delete u2 using t1 context - should not affect u2
		res := db.WithContext(ctx).Where("id = ?", "u2").Delete(&domain.User{})
		if res.Error != nil {
			t.Fatalf("failed to delete user: %v", res.Error)
		}
		if res.RowsAffected != 0 {
			t.Errorf("expected 0 rows affected, got %d", res.RowsAffected)
		}

		// Verify u2 still exists
		var checkU2 domain.User
		if err := db.First(&checkU2, "id = ?", "u2").Error; err != nil {
			t.Fatalf("failed to fetch user: %v", err)
		}

		// Delete u1 with t1 context - should work
		res = db.WithContext(ctx).Where("id = ?", "u1").Delete(&domain.User{})
		if res.Error != nil {
			t.Fatalf("failed to delete user: %v", res.Error)
		}
		if res.RowsAffected != 1 {
			t.Errorf("expected 1 row affected, got %d", res.RowsAffected)
		}

		// Verify u1 is deleted
		var checkU1 domain.User
		err := db.First(&checkU1, "id = ?", "u1").Error
		if err == nil {
			t.Error("expected user u1 to be deleted, but found it")
		}
	})

	// 4. Create Isolation Enforcements
	t.Run("Create forces tenant_id from context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), domain.ContextKeyTenantID, t1)
		uMismatched := domain.User{
			ID:           "u-mismatched",
			TenantID:     t2, // Mismatched tenant
			Email:        "mismatched@test.com",
			PasswordHash: "hash",
			Role:         "viewer",
		}
		if err := db.WithContext(ctx).Create(&uMismatched).Error; err != nil {
			t.Fatalf("failed to create mismatched user: %v", err)
		}

		// Verify it was saved under tenant-1 (forced from context)
		var checkUser domain.User
		if err := db.First(&checkUser, "id = ?", "u-mismatched").Error; err != nil {
			t.Fatalf("failed to retrieve user: %v", err)
		}
		if checkUser.TenantID != t1 {
			t.Errorf("expected tenant_id to be forced to %s, got %s", t1, checkUser.TenantID)
		}
	})

	// 5. Upsert Hijacking Isolation
	t.Run("Upsert (db.Save) does not hijack cross-tenant records", func(t *testing.T) {
		ctx1 := context.WithValue(context.Background(), domain.ContextKeyTenantID, t1)
		uOriginal := domain.User{
			ID:           "u-hijack",
			TenantID:     t1,
			Email:        "original@hijack.com",
			PasswordHash: "hash",
			Role:         "editor",
		}
		if err := db.WithContext(ctx1).Create(&uOriginal).Error; err != nil {
			t.Fatalf("failed to create original user: %v", err)
		}

		// Tenant 2 attempts to save the same record ID, changing the tenant and role
		ctx2 := context.WithValue(context.Background(), domain.ContextKeyTenantID, t2)
		uAttempt := domain.User{
			ID:           "u-hijack",
			TenantID:     t2,
			Email:        "hijacked@hijack.com",
			PasswordHash: "hash",
			Role:         "admin",
		}

		// Attempting to Save should fail or not modify Tenant 1's record because the
		// ON CONFLICT DO UPDATE statement's WHERE condition checks that tenant_id = tenant-2.
		// Since it is tenant-1, no update happens and no hijack occurs.
		res := db.WithContext(ctx2).Save(&uAttempt)

		// In SQLite, GORM's Save with an ON CONFLICT check that fails to update will
		// either return 0 rows affected, or error.
		if res.Error != nil {
			// In some setups, it can return an error if a conflict is not updated, which is safe.
			t.Logf("Upsert error (safe/expected): %v", res.Error)
		} else if res.RowsAffected != 0 {
			t.Errorf("expected 0 rows affected on cross-tenant save attempt, got %d", res.RowsAffected)
		}

		// Fetch the user unscoped/scoped to Tenant 1 to ensure it wasn't modified or hijacked
		var verifyUser domain.User
		if err := db.WithContext(ctx1).First(&verifyUser, "id = ?", "u-hijack").Error; err != nil {
			t.Fatalf("failed to retrieve original user: %v", err)
		}
		if verifyUser.TenantID != t1 {
			t.Errorf("VULNERABLE: record was hijacked! tenant_id changed to %s", verifyUser.TenantID)
		}
		if verifyUser.Email != "original@hijack.com" {
			t.Errorf("VULNERABLE: email was hijacked and updated to %s", verifyUser.Email)
		}
	})
}
