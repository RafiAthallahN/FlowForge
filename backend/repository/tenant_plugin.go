package repository

import (
	"context"

	"github.com/flow-forger/flow-forger/backend/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TenantIsolationPlugin struct{}

func (p *TenantIsolationPlugin) Name() string {
	return "tenant_isolation"
}

func getTenantID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if val, ok := ctx.Value(domain.ContextKeyTenantID).(string); ok && val != "" {
		return val, true
	}
	if val, ok := ctx.Value("tenant_id").(string); ok && val != "" {
		return val, true
	}
	return "", false
}

func (p *TenantIsolationPlugin) Initialize(db *gorm.DB) error {
	// Intercept Create operations (handles insertion & upsert isolation)
	err := db.Callback().Create().Before("gorm:create").Register("tenant_isolation_create", func(d *gorm.DB) {
		tenantID, ok := getTenantID(d.Statement.Context)
		if ok && tenantID != "" {
			// 1. Force the tenant_id on the model being created
			d.Statement.SetColumn("tenant_id", tenantID)

			// 2. Prevent ON CONFLICT (Save/Upsert) from hijacking other tenants' records
			if c, ok := d.Statement.Clauses["ON CONFLICT"]; ok {
				if onConflict, ok := c.Expression.(clause.OnConflict); ok {
					// Add WHERE table.tenant_id = 'tenant_id' to the DO UPDATE clause
					depExpr := clause.Eq{
						Column: clause.Column{Table: d.Statement.Table, Name: "tenant_id"},
						Value:  tenantID,
					}
					onConflict.Where.Exprs = append(onConflict.Where.Exprs, depExpr)
					c.Expression = onConflict
					d.Statement.Clauses["ON CONFLICT"] = c
				}
			}
		}
	})
	if err != nil {
		return err
	}

	// Intercept Query operations
	err = db.Callback().Query().Before("gorm:query").Register("tenant_isolation_query", func(d *gorm.DB) {
		tenantID, ok := getTenantID(d.Statement.Context)
		if ok && tenantID != "" {
			d.Where("tenant_id = ?", tenantID)
		}
	})
	if err != nil {
		return err
	}

	// Intercept Update operations
	err = db.Callback().Update().Before("gorm:update").Register("tenant_isolation_update", func(d *gorm.DB) {
		tenantID, ok := getTenantID(d.Statement.Context)
		if ok && tenantID != "" {
			d.Where("tenant_id = ?", tenantID)
		}
	})
	if err != nil {
		return err
	}

	// Intercept Delete operations
	err = db.Callback().Delete().Before("gorm:delete").Register("tenant_isolation_delete", func(d *gorm.DB) {
		tenantID, ok := getTenantID(d.Statement.Context)
		if ok && tenantID != "" {
			d.Where("tenant_id = ?", tenantID)
		}
	})
	return err
}

