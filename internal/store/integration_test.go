package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

var testDB *DB

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://politica:politica_dev_pass@localhost:5432/securelens_test?sslmode=disable"
	}

	var err error
	testDB, err = NewDB(dbURL)
	if err != nil {
		panic("Failed to connect to test database: " + err.Error())
	}

	// Run migrations
	RunMigrations(testDB, "up", 0)

	code := m.Run()

	// Cleanup
	testDB.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	testDB.Close()

	os.Exit(code)
}

func createTestTenant(t *testing.T) string {
	t.Helper()
	tenantID := uuid.New().String()
	slug := "test-" + tenantID[:8]
	_, err := testDB.Exec(`INSERT INTO tenants (id, name, slug, status) VALUES ($1, $2, $3, 'active')`,
		tenantID, "Test Tenant", slug)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return tenantID
}

func TestRepository_CRUD(t *testing.T) {
	tenantID := createTestTenant(t)
	ctx := context.Background()

	repo := NewRepo[DataSource](testDB, "datasources")

	// Create
	ds := &DataSource{
		TenantID: tenantID,
		Name:     "Test Source",
		Type:     "postgres",
		Status:   "pending",
	}
	err := repo.Create(ctx, ds)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if ds.ID == "" {
		t.Error("Expected ID to be set")
	}

	// FindByID
	found, err := repo.FindByID(ctx, tenantID, ds.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found == nil {
		t.Fatal("Expected to find datasource")
	}
	if found.Name != "Test Source" {
		t.Errorf("Name = %s, want Test Source", found.Name)
	}

	// List
	list, err := repo.List(ctx, tenantID, ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) == 0 {
		t.Error("Expected at least one datasource")
	}

	// Update
	ds.Name = "Updated Source"
	err = repo.Update(ctx, ds)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	updated, _ := repo.FindByID(ctx, tenantID, ds.ID)
	if updated.Name != "Updated Source" {
		t.Errorf("Name = %s, want Updated Source", updated.Name)
	}

	// Delete
	err = repo.Delete(ctx, tenantID, ds.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify delete
	deleted, _ := repo.FindByID(ctx, tenantID, ds.ID)
	if deleted != nil {
		t.Error("Expected datasource to be deleted")
	}
}

func TestRepository_MultiTenantIsolation(t *testing.T) {
	tenant1ID := createTestTenant(t)
	tenant2ID := createTestTenant(t)
	ctx := context.Background()

	repo := NewRepo[DataSource](testDB, "datasources")

	// Create in tenant 1
	ds1 := &DataSource{
		TenantID: tenant1ID,
		Name:     "Tenant1 Source",
		Type:     "postgres",
		Status:   "active",
	}
	repo.Create(ctx, ds1)

	// Create in tenant 2
	ds2 := &DataSource{
		TenantID: tenant2ID,
		Name:     "Tenant2 Source",
		Type:     "mysql",
		Status:   "active",
	}
	repo.Create(ctx, ds2)

	// Tenant 1 should only see their datasource
	list1, _ := repo.List(ctx, tenant1ID, ListOpts{Limit: 100})
	for _, ds := range list1 {
		if ds.Name == "Tenant2 Source" {
			t.Error("Tenant 1 should not see Tenant 2's datasource")
		}
	}

	// Tenant 2 should only see their datasource
	list2, _ := repo.List(ctx, tenant2ID, ListOpts{Limit: 100})
	for _, ds := range list2 {
		if ds.Name == "Tenant1 Source" {
			t.Error("Tenant 2 should not see Tenant 1's datasource")
		}
	}

	// Cross-tenant access should fail
	crossAccess, _ := repo.FindByID(ctx, tenant2ID, ds1.ID)
	if crossAccess != nil {
		t.Error("Cross-tenant access should return nil")
	}
}

func TestRepository_Pagination(t *testing.T) {
	tenantID := createTestTenant(t)
	ctx := context.Background()

	repo := NewRepo[DataSource](testDB, "datasources")

	// Create 5 datasources
	for i := 0; i < 5; i++ {
		ds := &DataSource{
			TenantID: tenantID,
			Name:     "Source " + string(rune('A'+i)),
			Type:     "postgres",
			Status:   "active",
		}
		repo.Create(ctx, ds)
	}

	// Get first page
	page1, _ := repo.List(ctx, tenantID, ListOpts{Limit: 2, Offset: 0})
	if len(page1) != 2 {
		t.Errorf("Page 1 length = %d, want 2", len(page1))
	}

	// Get second page
	page2, _ := repo.List(ctx, tenantID, ListOpts{Limit: 2, Offset: 2})
	if len(page2) != 2 {
		t.Errorf("Page 2 length = %d, want 2", len(page2))
	}

	// Get third page
	page3, _ := repo.List(ctx, tenantID, ListOpts{Limit: 2, Offset: 4})
	if len(page3) != 1 {
		t.Errorf("Page 3 length = %d, want 1", len(page3))
	}
}

func TestRepository_AllModels(t *testing.T) {
	tenantID := createTestTenant(t)
	ctx := context.Background()

	// Test User repository
	t.Run("User", func(t *testing.T) {
		repo := NewRepo[User](testDB, "users")
		user := &User{
			TenantID:     tenantID,
			Email:        "test-" + uuid.New().String()[:8] + "@example.com",
			PasswordHash: "hash",
			Name:         "Test User",
			Status:       "active",
		}
		if err := repo.Create(ctx, user); err != nil {
			t.Fatalf("Create user failed: %v", err)
		}
		if user.ID == "" {
			t.Error("Expected user ID")
		}
	})

	// Test Role repository
	t.Run("Role", func(t *testing.T) {
		repo := NewRepo[Role](testDB, "roles")
		role := &Role{
			TenantID:    tenantID,
			Name:        "test-role-" + uuid.New().String()[:8],
			Description: "Test Role",
			Permissions: JSON(`["read", "write"]`),
		}
		if err := repo.Create(ctx, role); err != nil {
			t.Fatalf("Create role failed: %v", err)
		}
	})

	// Test Policy repository
	t.Run("Policy", func(t *testing.T) {
		repo := NewRepo[Policy](testDB, "policies")
		policy := &Policy{
			TenantID:    tenantID,
			Name:        "Test Policy",
			Type:        "access",
			Description: "Test",
		}
		if err := repo.Create(ctx, policy); err != nil {
			t.Fatalf("Create policy failed: %v", err)
		}
	})

	// Test Job repository
	t.Run("Job", func(t *testing.T) {
		repo := NewRepo[Job](testDB, "jobs")
		job := &Job{
			TenantID: tenantID,
			Name:     "Test Job",
			Type:     "scan",
			Schedule: "0 0 * * *",
			Status:   "scheduled",
		}
		if err := repo.Create(ctx, job); err != nil {
			t.Fatalf("Create job failed: %v", err)
		}
	})

	// Test Notification repository
	t.Run("Notification", func(t *testing.T) {
		repo := NewRepo[Notification](testDB, "notifications")
		notif := &Notification{
			TenantID: tenantID,
			Type:     "alert",
			Severity: "high",
			Title:    "Test Alert",
			Message:  "Test message",
		}
		if err := repo.Create(ctx, notif); err != nil {
			t.Fatalf("Create notification failed: %v", err)
		}
	})

	// Test Webhook repository
	t.Run("Webhook", func(t *testing.T) {
		repo := NewRepo[Webhook](testDB, "webhooks")
		webhook := &Webhook{
			TenantID: tenantID,
			URL:      "https://example.com/webhook",
			Active:   true,
		}
		if err := repo.Create(ctx, webhook); err != nil {
			t.Fatalf("Create webhook failed: %v", err)
		}
	})

	// Test Label repository
	t.Run("Label", func(t *testing.T) {
		repo := NewRepo[Label](testDB, "labels")
		label := &Label{
			TenantID:     tenantID,
			DatasetID:    "test-dataset",
			Label:        "CONFIDENTIAL",
			AutoAssigned: true,
		}
		if err := repo.Create(ctx, label); err != nil {
			t.Fatalf("Create label failed: %v", err)
		}
	})

	// Test Integration repository
	t.Run("Integration", func(t *testing.T) {
		repo := NewRepo[Integration](testDB, "integrations")
		integration := &Integration{
			TenantID: tenantID,
			Name:     "Test Integration",
			Type:     "dlp",
			Provider: "test",
			Status:   "active",
		}
		if err := repo.Create(ctx, integration); err != nil {
			t.Fatalf("Create integration failed: %v", err)
		}
	})

	// Test QualityScore repository
	t.Run("QualityScore", func(t *testing.T) {
		repo := NewRepo[QualityScore](testDB, "quality_scores")
		score := &QualityScore{
			TenantID:     tenantID,
			DatasetID:    "test-dataset",
			Overall:      0.85,
			Completeness: 0.9,
			Accuracy:     0.8,
			Consistency:  0.85,
			Timeliness:   0.9,
			Uniqueness:   0.8,
		}
		if err := repo.Create(ctx, score); err != nil {
			t.Fatalf("Create quality score failed: %v", err)
		}
	})

	// Test DSAR repository
	t.Run("DSAR", func(t *testing.T) {
		repo := NewRepo[DSAR](testDB, "dsars")
		dsar := &DSAR{
			TenantID:  tenantID,
			SubjectID: "user-123",
			Type:      "access",
			Status:    "pending",
			Deadline:  time.Now().AddDate(0, 0, 30),
		}
		if err := repo.Create(ctx, dsar); err != nil {
			t.Fatalf("Create DSAR failed: %v", err)
		}
	})

	// Test ROTData repository
	t.Run("ROTData", func(t *testing.T) {
		repo := NewRepo[ROTData](testDB, "rot_data")
		rot := &ROTData{
			TenantID:   tenantID,
			DatasetID:  "test-dataset",
			Category:   "obsolete",
			Score:      0.8,
			Reason:     "Not accessed in 2 years",
			SizeBytes:  1024000,
			LastAccess: time.Now().AddDate(-2, 0, 0),
		}
		if err := repo.Create(ctx, rot); err != nil {
			t.Fatalf("Create ROT data failed: %v", err)
		}
	})
}
