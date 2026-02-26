// Package middleware provides Chi-compatible HTTP middleware for Kubric services.
package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/managekube-hue/Kubric-UiDR/internal/schema"
)

// tenantKey is the unexported context key for the validated tenant ID.
type tenantKey struct{}

// TenantID extracts the validated tenant ID from ctx.
// Returns an empty string if no tenant was set (e.g. health-probe routes).
func TenantID(ctx context.Context) string {
	v, _ := ctx.Value(tenantKey{}).(string)
	return v
}

// TenantContext is a Chi middleware that reads the tenant ID from the
// X-Kubric-Tenant-Id request header. When the header is absent it tries
// the Chi URL parameter named "tenantID".
//
// Behaviour:
//   - Header/param present and VALID   → store in ctx, call next handler.
//   - Header/param present but INVALID → 401 with JSON error body.
//   - Header/param absent              → pass through unchanged (health probes, etc.).
//
// Handlers that require a tenant must call middleware.TenantID(r.Context())
// and reject the request themselves if the value is empty.
func TenantContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Kubric-Tenant-Id")
		if tenantID == "" {
			// Fallback: chi URL parameter (e.g. /tenants/{tenantID})
			tenantID = chi.URLParam(r, "tenantID")
		}

		if tenantID != "" {
			if err := schema.ValidateTenantID(tenantID); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"invalid tenant_id: must be lowercase alphanumeric/hyphen, 2-63 chars"}`))
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), tenantKey{}, tenantID))
		}

		next.ServeHTTP(w, r)
	})
}
