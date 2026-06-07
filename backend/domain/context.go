package domain

type ContextKey string

const (
	ContextKeyTenantID ContextKey = "tenant_id"
	ContextKeyUserID   ContextKey = "user_id"
	ContextKeyUserRole ContextKey = "user_role"
)
