package pkg

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRequestIDMiddleware(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("generates request ID if not provided", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		requestID := w.Header().Get("X-Request-ID")
		if requestID == "" {
			t.Error("Expected X-Request-ID header to be set")
		}
	})

	t.Run("uses provided request ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", "custom-id-123")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		requestID := w.Header().Get("X-Request-ID")
		if requestID != "custom-id-123" {
			t.Errorf("Expected X-Request-ID = custom-id-123, got %s", requestID)
		}
	})
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	tests := []struct {
		header string
		want   string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"X-XSS-Protection", "1; mode=block"},
		{"Content-Security-Policy", "default-src 'self'"},
	}

	for _, tt := range tests {
		got := w.Header().Get(tt.header)
		if got != tt.want {
			t.Errorf("%s = %s, want %s", tt.header, got, tt.want)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(2, 2) // 2 requests per minute, burst of 2

	handler := rl.Middleware(func(r *http.Request) string {
		return "test-key"
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First two requests should succeed (burst)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Request 3: expected 429, got %d", w.Code)
	}
}

func TestIPWhitelistMiddleware(t *testing.T) {
	t.Run("allows all when env not set", func(t *testing.T) {
		os.Unsetenv("TEST_ALLOWED_IPS")
		handler := IPWhitelistMiddleware("TEST_ALLOWED_IPS")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 when no whitelist, got %d", w.Code)
		}
	})

	t.Run("blocks non-whitelisted IP", func(t *testing.T) {
		os.Setenv("TEST_ALLOWED_IPS", "10.0.0.1,10.0.0.2")
		defer os.Unsetenv("TEST_ALLOWED_IPS")

		handler := IPWhitelistMiddleware("TEST_ALLOWED_IPS")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403 for non-whitelisted IP, got %d", w.Code)
		}
	})

	t.Run("allows whitelisted IP", func(t *testing.T) {
		os.Setenv("TEST_ALLOWED_IPS", "10.0.0.1,192.168.1.100")
		defer os.Unsetenv("TEST_ALLOWED_IPS")

		handler := IPWhitelistMiddleware("TEST_ALLOWED_IPS")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for whitelisted IP, got %d", w.Code)
		}
	})
}

func TestGetCORSOrigins(t *testing.T) {
	t.Run("returns defaults when env not set", func(t *testing.T) {
		os.Unsetenv("CORS_ORIGINS")
		origins := GetCORSOrigins()
		if len(origins) != 3 {
			t.Errorf("Expected 3 default origins, got %d", len(origins))
		}
	})

	t.Run("parses env variable", func(t *testing.T) {
		os.Setenv("CORS_ORIGINS", "https://example.com,https://api.example.com")
		defer os.Unsetenv("CORS_ORIGINS")

		origins := GetCORSOrigins()
		if len(origins) != 2 {
			t.Errorf("Expected 2 origins, got %d", len(origins))
		}
		if origins[0] != "https://example.com" {
			t.Errorf("Expected first origin to be https://example.com, got %s", origins[0])
		}
	})
}
