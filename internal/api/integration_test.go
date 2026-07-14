package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/securelens/securelens/internal/events"
	"github.com/securelens/securelens/internal/external"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

var testServer *Server
var testDB *store.DB

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://politica:politica_dev_pass@localhost:5432/securelens_test?sslmode=disable"
	}

	var err error
	testDB, err = store.NewDB(dbURL)
	if err != nil {
		panic("Failed to connect to test database: " + err.Error())
	}

	// Run migrations
	store.RunMigrations(testDB, "up", 0)

	// Start event bus
	ctx := context.Background()
	events.Start(ctx)

	// Create test server
	kafka := external.NewKafka("localhost:9092")
	testServer = NewServer(testDB, kafka)

	pkg.SetJWTSecret("test-secret-key")

	code := m.Run()

	// Cleanup
	testDB.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	testDB.Close()

	os.Exit(code)
}

func setupTestData(t *testing.T) (tenantID, userID, token string) {
	t.Helper()

	// Create tenant with proper UUID
	tenantID = uuid.New().String()
	slug := "test-" + tenantID[:8]
	_, err := testDB.Exec(`INSERT INTO tenants (id, name, slug, status) VALUES ($1, $2, $3, 'active')`,
		tenantID, "Test Tenant", slug)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// Create user with proper UUID
	userID = uuid.New().String()
	hash, _ := pkg.HashPassword("password123")
	email := "test-" + userID[:8] + "@example.com"
	_, err = testDB.Exec(`INSERT INTO users (id, tenant_id, email, password_hash, name, status) VALUES ($1, $2, $3, $4, $5, 'active')`,
		userID, tenantID, email, hash, "Test User")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Generate token
	token, _ = pkg.GenerateToken(userID, tenantID, []string{"*"}, false)
	return
}

func makeRequest(t *testing.T, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	testServer.router.ServeHTTP(w, req)
	return w
}

func TestIntegration_HealthCheck(t *testing.T) {
	w := makeRequest(t, "GET", "/health", nil, "")

	if w.Code != http.StatusOK {
		t.Errorf("Health check failed: %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("Health status = %s, want ok", resp["status"])
	}
}
