package middleware

import (
	"net/http"
)

// RequireRole returns a middleware that enforces the caller has the given role
// (matched against the JWT groups claim stored by JWTAuth).
//
// Kubric standard roles:
//
//	kubric:admin    — full platform access (tenant CRUD, all services)
//	kubric:analyst  — read + triage access (VDR, KIC, KAI endpoints)
//	kubric:agent    — agent heartbeat + event publish (NOC /agents)
//	kubric:readonly — GET only across all services
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			groups := UserGroups(r.Context())
			for _, g := range groups {
				if g == role || g == "kubric:admin" {
					next.ServeHTTP(w, r)
					return
				}
			}
			jwtError(w, "insufficient role: "+role+" required", http.StatusForbidden)
		})
	}
}

// RequireAnyRole returns a middleware that passes if the caller has ANY of the
// listed roles.
func RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			groups := UserGroups(r.Context())
			for _, g := range groups {
				if g == "kubric:admin" {
					next.ServeHTTP(w, r)
					return
				}
				for _, role := range roles {
					if g == role {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
			jwtError(w, "insufficient role", http.StatusForbidden)
		})
	}
}
