// Package dev provides CORS middleware for Kubric HTTP servers.
// File: K-DEV-BLD-006_chi_cors_middleware.go
package dev

import (
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/cors"
)

// NewCORSMiddleware returns a Chi-compatible CORS middleware handler.
//
// Configuration is read from environment variables:
//   - DEV_MODE=true         — allow all origins (do not use in production)
//   - CORS_ALLOWED_ORIGINS  — comma-separated list of allowed origins
//     (e.g. "https://app.kubric.io,https://staging.kubric.io")
//     When empty and DEV_MODE is not set, defaults to the primary app domain.
func NewCORSMiddleware() func(http.Handler) http.Handler {
	devMode := strings.EqualFold(os.Getenv("DEV_MODE"), "true")
	originsEnv := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))

	var allowedOrigins []string
	switch {
	case devMode:
		allowedOrigins = []string{"*"}
	case originsEnv != "":
		for _, o := range strings.Split(originsEnv, ",") {
			if o = strings.TrimSpace(o); o != "" {
				allowedOrigins = append(allowedOrigins, o)
			}
		}
	default:
		allowedOrigins = []string{"https://app.kubric.io", "https://kubric.io"}
	}

	return cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-Request-ID",
			"X-Tenant-ID",
			"X-Correlation-ID",
			"X-Kubric-Agent-ID",
		},
		ExposedHeaders: []string{
			"X-Request-ID",
			"X-Correlation-ID",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		// Credentials must be false when AllowedOrigins contains "*".
		AllowCredentials: !devMode,
		MaxAge:           300, // 5 minutes — advise browsers to cache preflight
		OptionsPassthrough: false,
		Debug:              devMode,
	})
}

// AllowedOriginsList returns the parsed list of allowed origins for logging/inspection.
func AllowedOriginsList() []string {
	devMode := strings.EqualFold(os.Getenv("DEV_MODE"), "true")
	if devMode {
		return []string{"*"}
	}
	originsEnv := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if originsEnv == "" {
		return []string{"https://app.kubric.io", "https://kubric.io"}
	}
	var out []string
	for _, o := range strings.Split(originsEnv, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}
