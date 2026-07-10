package pkg

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

// AuditConfig holds audit logging configuration
type AuditConfig struct {
	RetentionDays int  // Days to retain audit logs (0 = forever)
	Enabled       bool // Whether audit logging is enabled
}

// DefaultAuditConfig returns default audit configuration
func DefaultAuditConfig() AuditConfig {
	retentionDays := 90 // Default 90 days
	if v := os.Getenv("AUDIT_LOG_RETENTION_DAYS"); v != "" {
		if days, err := strconv.Atoi(v); err == nil && days > 0 {
			retentionDays = days
		}
	}

	enabled := true
	if v := os.Getenv("AUDIT_LOG_ENABLED"); v == "false" {
		enabled = false
	}

	return AuditConfig{
		RetentionDays: retentionDays,
		Enabled:       enabled,
	}
}

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID          string                 `json:"id" db:"id"`
	TenantID    string                 `json:"tenant_id" db:"tenant_id"`
	UserID      string                 `json:"user_id" db:"user_id"`
	Action      string                 `json:"action" db:"action"`
	Resource    string                 `json:"resource" db:"resource"`
	ResourceID  string                 `json:"resource_id" db:"resource_id"`
	Method      string                 `json:"method" db:"method"`
	Endpoint    string                 `json:"endpoint" db:"endpoint"`
	IP          string                 `json:"ip" db:"ip"`
	UserAgent   string                 `json:"user_agent" db:"user_agent"`
	RequestID   string                 `json:"request_id" db:"request_id"`
	StatusCode  int                    `json:"status_code" db:"status_code"`
	DurationMs  int64                  `json:"duration_ms" db:"duration_ms"`
	BeforeState json.RawMessage        `json:"before_state,omitempty" db:"before_state"`
	AfterState  json.RawMessage        `json:"after_state,omitempty" db:"after_state"`
	Details     map[string]interface{} `json:"details,omitempty" db:"details"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

// AuditLogger provides audit logging functionality
type AuditLogger struct {
	config AuditConfig
	writer AuditWriter
}

// AuditWriter interface for writing audit logs
type AuditWriter interface {
	Write(ctx context.Context, entry AuditEntry) error
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(writer AuditWriter) *AuditLogger {
	return &AuditLogger{
		config: DefaultAuditConfig(),
		writer: writer,
	}
}

// Log writes an audit entry
func (a *AuditLogger) Log(ctx context.Context, entry AuditEntry) {
	if !a.config.Enabled {
		return
	}

	// Enrich entry with context
	if entry.TenantID == "" {
		entry.TenantID = TenantFromCtx(ctx)
	}
	if entry.UserID == "" {
		entry.UserID = UserFromCtx(ctx)
	}
	if entry.RequestID == "" {
		entry.RequestID = middleware.GetReqID(ctx)
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	// Record metrics
	RecordAuditLog(entry.Action, entry.Resource)

	// Write to storage
	if err := a.writer.Write(ctx, entry); err != nil {
		log.Error().Err(err).
			Str("action", entry.Action).
			Str("resource", entry.Resource).
			Msg("Failed to write audit log")
	}
}

// LogAction is a convenience method for logging simple actions
func (a *AuditLogger) LogAction(ctx context.Context, action, resource, resourceID string) {
	a.Log(ctx, AuditEntry{
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
	})
}

// LogChange logs a change with before/after state
func (a *AuditLogger) LogChange(ctx context.Context, action, resource, resourceID string, before, after interface{}) {
	var beforeJSON, afterJSON json.RawMessage
	if before != nil {
		beforeJSON, _ = json.Marshal(before)
	}
	if after != nil {
		afterJSON, _ = json.Marshal(after)
	}

	a.Log(ctx, AuditEntry{
		Action:      action,
		Resource:    resource,
		ResourceID:  resourceID,
		BeforeState: beforeJSON,
		AfterState:  afterJSON,
	})
}

// AuditMiddleware creates middleware that logs all write operations
func AuditMiddleware(logger *AuditLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only audit write operations
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			// Determine action from method
			action := methodToAction(r.Method)
			resource := extractResource(r.URL.Path)

			logger.Log(r.Context(), AuditEntry{
				Action:     action,
				Resource:   resource,
				Method:     r.Method,
				Endpoint:   r.URL.Path,
				IP:         getRealIP(r),
				UserAgent:  r.UserAgent(),
				StatusCode: ww.Status(),
				DurationMs: time.Since(start).Milliseconds(),
			})
		})
	}
}

// methodToAction converts HTTP method to audit action
func methodToAction(method string) string {
	switch method {
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return method
	}
}

// extractResource extracts resource name from URL path
func extractResource(path string) string {
	// Simple extraction - in production, use proper path parsing
	// /api/v1/users/123 -> users
	// /api/v1/datasources/456/scan -> datasources
	parts := splitPath(path)
	if len(parts) >= 3 {
		return parts[2] // Skip "api" and "v1"
	}
	return path
}

// splitPath splits a URL path into parts
func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// getRealIP extracts the real client IP from request
func getRealIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// CleanupOldAuditLogs removes audit logs older than retention period
func CleanupOldAuditLogs(ctx context.Context, db interface{ ExecContext(context.Context, string, ...interface{}) (interface{}, error) }, retentionDays int) error {
	if retentionDays <= 0 {
		return nil // No cleanup if retention is disabled
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	_, err := db.ExecContext(ctx,
		"DELETE FROM audit_logs WHERE created_at < $1",
		cutoff)
	return err
}
