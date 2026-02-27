// Package dev provides JWT authentication middleware for Kubric HTTP servers.
// File: K-DEV-BLD-007_chi_jwt_auth.go
package dev

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// claimsKey is an unexported type used as a context key for JWT claims,
// preventing collisions with other context values.
type claimsKey struct{}

// Claims represents the Kubric JWT payload.
type Claims struct {
	TenantID    string   `json:"tenant_id"`
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Groups      []string `json:"groups"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// HasGroup returns true when the claims contain the given group name.
func (c *Claims) HasGroup(group string) bool {
	for _, g := range c.Groups {
		if g == group {
			return true
		}
	}
	return false
}

// HasPermission returns true when the claims contain the given permission string.
func (c *Claims) HasPermission(perm string) bool {
	for _, p := range c.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// JWTAuthMiddleware validates Bearer tokens using the HMAC secret in JWT_SECRET.
//
// On a valid token the Claims struct is stored in the request context under
// claimsKey{} and the X-Tenant-ID header is set so downstream handlers can
// read the tenant without re-parsing the token.
//
// On failure the handler writes a JSON 401 response and stops the chain.
func JWTAuthMiddleware(next http.Handler) http.Handler {
	secret := []byte(os.Getenv("JWT_SECRET"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSONError(w, http.StatusUnauthorized, "missing or malformed bearer token")
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return secret, nil
		}, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}))

		if err != nil || !token.Valid {
			writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		// Propagate tenant context for downstream handlers and RLS.
		if claims.TenantID != "" {
			w.Header().Set("X-Tenant-ID", claims.TenantID)
		}

		ctx := context.WithValue(r.Context(), claimsKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClaims extracts the validated Claims from a request context.
// Returns nil when no claims are present (e.g. unauthenticated handler).
func GetClaims(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey{}).(*Claims)
	return c
}

// RequireGroup returns middleware that gates access to a specific JWT group.
// Must be chained after JWTAuthMiddleware.
func RequireGroup(group string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil || !claims.HasGroup(group) {
				writeJSONError(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission returns middleware that gates access to a specific permission string.
// Must be chained after JWTAuthMiddleware.
func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil || !claims.HasPermission(perm) {
				writeJSONError(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// writeJSONError writes a JSON error body with the specified status code.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
