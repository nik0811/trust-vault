package pkg

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogConfig holds logging configuration
type LogConfig struct {
	SampleRate     float64 // 0.0 to 1.0, percentage of requests to log for high-volume endpoints
	HighVolumeURLs []string
}

var defaultLogConfig = LogConfig{
	SampleRate:     0.1, // Log 10% of high-volume requests
	HighVolumeURLs: []string{"/health", "/metrics", "/api/v1/gate/query"},
}

// StructuredLogger is a middleware that logs requests with structured fields
func StructuredLogger(next http.Handler) http.Handler {
	return StructuredLoggerWithConfig(defaultLogConfig)(next)
}

// StructuredLoggerWithConfig returns a middleware with custom config
func StructuredLoggerWithConfig(config LogConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Increment active connections
			HTTPActiveConnections.Inc()
			defer HTTPActiveConnections.Dec()

			// Process request
			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			status := ww.Status()

			// Check if this is a high-volume endpoint that should be sampled
			shouldLog := true
			for _, url := range config.HighVolumeURLs {
				if r.URL.Path == url {
					// Simple sampling based on nanoseconds
					if float64(time.Now().UnixNano()%100)/100.0 > config.SampleRate {
						shouldLog = false
					}
					break
				}
			}

			// Record metrics (always)
			endpoint := normalizeEndpoint(r.URL.Path)
			RecordHTTPRequest(r.Method, endpoint, status, duration)

			// Log request (with sampling for high-volume)
			if shouldLog {
				logEvent := log.Info()
				if status >= 500 {
					logEvent = log.Error()
				} else if status >= 400 {
					logEvent = log.Warn()
				}

				logEvent.
					Str("request_id", middleware.GetReqID(r.Context())).
					Str("tenant_id", TenantFromCtx(r.Context())).
					Str("user_id", UserFromCtx(r.Context())).
					Str("method", r.Method).
					Str("endpoint", r.URL.Path).
					Str("remote_addr", r.RemoteAddr).
					Int("status_code", status).
					Int64("duration_ms", duration.Milliseconds()).
					Int("bytes_written", ww.BytesWritten()).
					Str("user_agent", r.UserAgent()).
					Msg("HTTP request")
			}
		})
	}
}

// normalizeEndpoint normalizes URL paths for metrics (replaces IDs with placeholders)
func normalizeEndpoint(path string) string {
	// This is a simplified version - in production, use regex or path matching
	return path
}

// RequestLogger returns a zerolog logger with request context
func RequestLogger(ctx context.Context) zerolog.Logger {
	return log.With().
		Str("request_id", RequestIDFromCtx(ctx)).
		Str("tenant_id", TenantFromCtx(ctx)).
		Str("user_id", UserFromCtx(ctx)).
		Logger()
}

// RequestIDFromCtx extracts request ID from context
func RequestIDFromCtx(ctx context.Context) string {
	if reqID := middleware.GetReqID(ctx); reqID != "" {
		return reqID
	}
	return ""
}

// LogWithContext logs a message with request context
func LogWithContext(ctx context.Context, level zerolog.Level, msg string) {
	logger := RequestLogger(ctx)
	logger.WithLevel(level).Msg(msg)
}

// LogErrorWithContext logs an error with request context
func LogErrorWithContext(ctx context.Context, err error, msg string) {
	logger := RequestLogger(ctx)
	logger.Error().Err(err).Msg(msg)
}

// LogInfoWithContext logs info with request context
func LogInfoWithContext(ctx context.Context, msg string, fields map[string]interface{}) {
	logger := RequestLogger(ctx)
	event := logger.Info()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}
