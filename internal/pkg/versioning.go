package pkg

import (
	"net/http"
	"os"
)

const (
	// APIVersion is the current API version
	APIVersion = "1.0.0"
	// HeaderAPIVersion is the response header for API version
	HeaderAPIVersion = "X-API-Version"
	// HeaderDeprecationWarning is the response header for deprecation warnings
	HeaderDeprecationWarning = "X-Deprecation-Warning"
	// HeaderRequestID is the response header for request ID
	HeaderRequestID = "X-Request-ID"
)

// DeprecatedEndpoint represents a deprecated API endpoint
type DeprecatedEndpoint struct {
	Path        string
	Method      string
	Message     string
	SunsetDate  string
	Replacement string
}

// DeprecatedEndpoints lists all deprecated endpoints
var DeprecatedEndpoints = []DeprecatedEndpoint{
	// Add deprecated endpoints here as needed
	// Example:
	// {Path: "/api/v1/old-endpoint", Method: "GET", Message: "Use /api/v1/new-endpoint instead", SunsetDate: "2025-01-01", Replacement: "/api/v1/new-endpoint"},
}

// APIVersionMiddleware adds API version headers to all responses
func APIVersionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add API version header
		w.Header().Set(HeaderAPIVersion, APIVersion)

		// Check for deprecated endpoints
		for _, dep := range DeprecatedEndpoints {
			if r.URL.Path == dep.Path && (dep.Method == "" || r.Method == dep.Method) {
				warning := dep.Message
				if dep.SunsetDate != "" {
					warning += " (sunset: " + dep.SunsetDate + ")"
				}
				if dep.Replacement != "" {
					warning += " Use " + dep.Replacement + " instead."
				}
				w.Header().Set(HeaderDeprecationWarning, warning)
				break
			}
		}

		next.ServeHTTP(w, r)
	})
}

// GetAPIVersion returns the current API version
func GetAPIVersion() string {
	if v := os.Getenv("API_VERSION"); v != "" {
		return v
	}
	return APIVersion
}

// VersionInfo returns version information for the API
type VersionInfo struct {
	Version     string `json:"version"`
	BuildDate   string `json:"build_date,omitempty"`
	GitCommit   string `json:"git_commit,omitempty"`
	GoVersion   string `json:"go_version,omitempty"`
	Environment string `json:"environment,omitempty"`
}

// GetVersionInfo returns the current version information
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:     GetAPIVersion(),
		BuildDate:   os.Getenv("BUILD_DATE"),
		GitCommit:   os.Getenv("GIT_COMMIT"),
		GoVersion:   os.Getenv("GO_VERSION"),
		Environment: os.Getenv("ENVIRONMENT"),
	}
}
