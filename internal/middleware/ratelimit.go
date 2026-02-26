package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ipLimiter holds per-IP rate limiters.
type ipLimiter struct {
	mu       sync.Mutex
	limiters map[string]*entry
}

type entry struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

var global = &ipLimiter{limiters: make(map[string]*entry)}

func init() {
	// Evict stale IP entries every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			global.mu.Lock()
			for ip, e := range global.limiters {
				if time.Since(e.lastSeen) > 10*time.Minute {
					delete(global.limiters, ip)
				}
			}
			global.mu.Unlock()
		}
	}()
}

func (il *ipLimiter) get(ip string) *rate.Limiter {
	il.mu.Lock()
	defer il.mu.Unlock()
	e, ok := il.limiters[ip]
	if !ok {
		// 100 req/s sustained, burst up to 200
		e = &entry{lim: rate.NewLimiter(100, 200)}
		il.limiters[ip] = e
	}
	e.lastSeen = time.Now()
	return e.lim
}

// RateLimit is a per-IP token-bucket rate limiter middleware.
// Allows 100 req/s per IP with a burst of 200.
// Returns 429 Too Many Requests when the bucket is empty.
func RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		// Trust X-Real-IP from trusted reverse proxies
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			ip = realIP
		}
		if !global.get(ip).Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
