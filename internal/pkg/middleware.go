package pkg

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), middleware.RequestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Relaxed CSP for API docs page to allow Swagger UI
		if strings.HasPrefix(r.URL.Path, "/api/docs") || strings.HasPrefix(r.URL.Path, "/api/openapi") {
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'unsafe-inline' https://unpkg.com; "+
					"style-src 'self' 'unsafe-inline' https://unpkg.com; "+
					"img-src 'self' data: https://unpkg.com; "+
					"font-src 'self' https://unpkg.com; "+
					"connect-src 'self' https://unpkg.com")
		} else {
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
		}

		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// IPWhitelistMiddleware restricts access to allowed IPs
func IPWhitelistMiddleware(allowedIPsEnv string) func(http.Handler) http.Handler {
	allowedIPs := parseIPList(os.Getenv(allowedIPsEnv))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(allowedIPs) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			clientIP := getClientIP(r)
			if !isIPAllowed(clientIP, allowedIPs) {
				log.Warn().Str("ip", clientIP).Msg("IP not in whitelist")
				Error(w, ErrForbidden, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func parseIPList(ips string) []string {
	if ips == "" {
		return nil
	}
	var result []string
	for _, ip := range strings.Split(ips, ",") {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			result = append(result, ip)
		}
	}
	return result
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func isIPAllowed(clientIP string, allowedIPs []string) bool {
	for _, allowed := range allowedIPs {
		if strings.Contains(allowed, "/") {
			_, cidr, err := net.ParseCIDR(allowed)
			if err == nil && cidr.Contains(net.ParseIP(clientIP)) {
				return true
			}
		} else if clientIP == allowed {
			return true
		}
	}
	return false
}

// RateLimiter provides per-key rate limiting
type RateLimiter struct {
	limiters sync.Map
	rate     rate.Limit
	burst    int
	cleanup  time.Duration
}

func NewRateLimiter(requestsPerMinute int, burst int) *RateLimiter {
	rl := &RateLimiter{
		rate:    rate.Limit(float64(requestsPerMinute) / 60.0),
		burst:   burst,
		cleanup: 10 * time.Minute,
	}
	go rl.cleanupLoop()
	return rl
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	if entry, ok := rl.limiters.Load(key); ok {
		e := entry.(*limiterEntry)
		e.lastSeen = time.Now()
		return e.limiter
	}
	limiter := rate.NewLimiter(rl.rate, rl.burst)
	rl.limiters.Store(key, &limiterEntry{limiter: limiter, lastSeen: time.Now()})
	return limiter
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	for range ticker.C {
		rl.limiters.Range(func(key, value any) bool {
			if time.Since(value.(*limiterEntry).lastSeen) > rl.cleanup {
				rl.limiters.Delete(key)
			}
			return true
		})
	}
}

func (rl *RateLimiter) Middleware(keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if !rl.getLimiter(key).Allow() {
				w.Header().Set("Retry-After", "60")
				JSON(w, map[string]string{
					"error":      "rate limit exceeded",
					"request_id": RequestIDFromCtx(r.Context()),
				}, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitByIP returns a middleware that rate limits by client IP
func RateLimitByIP(requestsPerMinute int) func(http.Handler) http.Handler {
	rl := NewRateLimiter(requestsPerMinute, requestsPerMinute)
	return rl.Middleware(getClientIP)
}

// RateLimitByTenant returns a middleware that rate limits by tenant+IP
func RateLimitByTenant(requestsPerMinute int) func(http.Handler) http.Handler {
	rl := NewRateLimiter(requestsPerMinute, requestsPerMinute)
	return rl.Middleware(func(r *http.Request) string {
		tenantID := TenantFromCtx(r.Context())
		if tenantID == "" {
			return getClientIP(r)
		}
		return tenantID + ":" + getClientIP(r)
	})
}

// ErrorWithRequestID returns an error response with request ID
func ErrorWithRequestID(w http.ResponseWriter, r *http.Request, err error, status ...int) {
	code := http.StatusInternalServerError
	if len(status) > 0 {
		code = status[0]
	}
	requestID := RequestIDFromCtx(r.Context())
	log.Error().Err(err).Str("request_id", requestID).Int("status", code).Msg("API error")
	JSON(w, map[string]string{
		"error":      err.Error(),
		"request_id": requestID,
	}, code)
}

// GetCORSOrigins returns allowed origins from environment or defaults
func GetCORSOrigins() []string {
	origins := os.Getenv("CORS_ORIGINS")
	if origins == "" {
		return []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:3002"}
	}
	var result []string
	for _, o := range strings.Split(origins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			result = append(result, o)
		}
	}
	return result
}
