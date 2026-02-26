// Package middleware provides Chi-compatible HTTP middleware for Kubric services.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// jwtUserKey / jwtGroupsKey are unexported context keys for JWT claims.
type jwtUserKey struct{}
type jwtGroupsKey struct{}

// KubricClaims extends standard JWT claims with Kubric-specific fields.
// Authentik emits tenant_id and groups in the token; agent_id is optional.
type KubricClaims struct {
	TenantID string   `json:"tenant_id"`
	Groups   []string `json:"groups"`
	jwt.RegisteredClaims
}

// UserID returns the validated user identifier from ctx (JWT sub claim).
func UserID(ctx context.Context) string {
	v, _ := ctx.Value(jwtUserKey{}).(string)
	return v
}

// UserGroups returns the JWT groups claim from ctx.
func UserGroups(ctx context.Context) []string {
	v, _ := ctx.Value(jwtGroupsKey{}).([]string)
	return v
}

// JWTAuth returns a middleware that validates Bearer tokens using HS256.
//
// Configuration via environment:
//
//	KUBRIC_JWT_SECRET   — HMAC signing secret (required in production)
//	KUBRIC_JWT_BYPASS   — set to "true" to skip validation in local dev
//	KUBRIC_JWT_ISSUER   — expected issuer (e.g. https://auth.kubric.io/application/o/kubric/)
//
// Token claims must include tenant_id matching the X-Kubric-Tenant-Id header
// (when both are present). Groups are stored in ctx for RBAC enforcement.
func JWTAuth() func(http.Handler) http.Handler {
	secret := []byte(os.Getenv("KUBRIC_JWT_SECRET"))
	issuer := os.Getenv("KUBRIC_JWT_ISSUER")
	bypass := os.Getenv("KUBRIC_JWT_BYPASS") == "true"

	keyFunc := func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Dev bypass — never enable in production
			if bypass {
				next.ServeHTTP(w, r)
				return
			}

			// Skip health probes — unauthenticated by design
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}

			// Extract Bearer token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				jwtError(w, "missing or malformed Authorization header", http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// Parse and validate
			var claims KubricClaims
			opts := []jwt.ParserOption{jwt.WithValidMethods([]string{"HS256"})}
			if issuer != "" {
				opts = append(opts, jwt.WithIssuer(issuer))
			}

			token, err := jwt.ParseWithClaims(tokenStr, &claims, keyFunc, opts...)
			if err != nil || !token.Valid {
				jwtError(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Validate tenant claim matches header (defence-in-depth)
			headerTenant := r.Header.Get("X-Kubric-Tenant-Id")
			if headerTenant != "" && claims.TenantID != "" && headerTenant != claims.TenantID {
				jwtError(w, "tenant_id mismatch between header and token", http.StatusForbidden)
				return
			}

			// Store claims in context for downstream handlers
			ctx := r.Context()
			sub, _ := claims.GetSubject()
			ctx = context.WithValue(ctx, jwtUserKey{}, sub)
			ctx = context.WithValue(ctx, jwtGroupsKey{}, claims.Groups)
			// If token has tenant_id but header doesn't, propagate it
			if claims.TenantID != "" && headerTenant == "" {
				ctx = context.WithValue(ctx, tenantKey{}, claims.TenantID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func jwtError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	body, _ := json.Marshal(map[string]string{"error": msg})
	_, _ = w.Write(body)
}
