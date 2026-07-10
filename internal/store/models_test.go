package store

import (
	"testing"
	"time"
)

func TestJSONScan(t *testing.T) {
	var j JSON

	// Test nil value
	err := j.Scan(nil)
	if err != nil {
		t.Errorf("Scan(nil) error = %v", err)
	}
	if j != nil {
		t.Errorf("Scan(nil) should result in nil JSON")
	}

	// Test byte slice
	j = nil
	err = j.Scan([]byte(`{"key":"value"}`))
	if err != nil {
		t.Errorf("Scan(bytes) error = %v", err)
	}
	if string(j) != `{"key":"value"}` {
		t.Errorf("Scan(bytes) = %s, want {\"key\":\"value\"}", string(j))
	}
}

func TestJSONValue(t *testing.T) {
	// Test empty JSON
	var j JSON
	val, err := j.Value()
	if err != nil {
		t.Errorf("Value() error = %v", err)
	}
	if val != nil {
		t.Errorf("Value() of empty JSON should be nil")
	}

	// Test non-empty JSON
	j = JSON(`{"key":"value"}`)
	val, err = j.Value()
	if err != nil {
		t.Errorf("Value() error = %v", err)
	}
	if string(val.([]byte)) != `{"key":"value"}` {
		t.Errorf("Value() = %s, want {\"key\":\"value\"}", string(val.([]byte)))
	}
}

func TestTenantModel(t *testing.T) {
	tenant := Tenant{
		ID:        "tenant-123",
		Name:      "Test Tenant",
		Slug:      "test-tenant",
		Status:    "active",
		Settings:  JSON(`{"theme":"dark"}`),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if tenant.ID != "tenant-123" {
		t.Errorf("Tenant.ID = %s, want tenant-123", tenant.ID)
	}
	if tenant.Status != "active" {
		t.Errorf("Tenant.Status = %s, want active", tenant.Status)
	}
}

func TestUserModel(t *testing.T) {
	user := User{
		ID:           "user-123",
		TenantID:     "tenant-456",
		Email:        "test@example.com",
		PasswordHash: "hashed",
		Name:         "Test User",
		Status:       "active",
		IsSuperAdmin: false,
		MFAEnabled:   false,
	}

	if user.Email != "test@example.com" {
		t.Errorf("User.Email = %s, want test@example.com", user.Email)
	}
	if user.IsSuperAdmin {
		t.Error("User.IsSuperAdmin should be false")
	}
}

func TestPolicyModel(t *testing.T) {
	policy := Policy{
		ID:          "policy-123",
		TenantID:    "tenant-456",
		Name:        "Block PII",
		Description: "Blocks PII from external LLMs",
		Type:        "access",
		Conditions:  JSON(`{"data_classification":["PII"]}`),
		Actions:     JSON(`{"action":"deny"}`),
		Regulations: JSON(`["GDPR","CCPA"]`),
		Active:      true,
		Priority:    1,
	}

	if policy.Type != "access" {
		t.Errorf("Policy.Type = %s, want access", policy.Type)
	}
	if !policy.Active {
		t.Error("Policy.Active should be true")
	}
}

func TestClassificationModel(t *testing.T) {
	classification := Classification{
		ID:         "class-123",
		TenantID:   "tenant-456",
		DatasetID:  "dataset-789",
		SourceID:   "source-abc",
		EntityType: "EMAIL",
		Value:      "test@example.com",
		Confidence: 0.95,
		Context:    JSON(`{"surrounding_text":"Contact us at"}`),
	}

	if classification.EntityType != "EMAIL" {
		t.Errorf("Classification.EntityType = %s, want EMAIL", classification.EntityType)
	}
	if classification.Confidence != 0.95 {
		t.Errorf("Classification.Confidence = %f, want 0.95", classification.Confidence)
	}
}

func TestLabelModel(t *testing.T) {
	label := Label{
		ID:           "label-123",
		TenantID:     "tenant-456",
		DatasetID:    "dataset-789",
		Label:        "CONFIDENTIAL",
		AutoAssigned: true,
		AssignedBy:   nil,
	}

	if label.Label != "CONFIDENTIAL" {
		t.Errorf("Label.Label = %s, want CONFIDENTIAL", label.Label)
	}
	if !label.AutoAssigned {
		t.Error("Label.AutoAssigned should be true")
	}
}

func TestDSARModel(t *testing.T) {
	deadline := time.Now().AddDate(0, 0, 30)
	dsar := DSAR{
		ID:        "dsar-123",
		TenantID:  "tenant-456",
		SubjectID: "subject-789",
		Type:      "access",
		Status:    "pending",
		Deadline:  deadline,
	}

	if dsar.Type != "access" {
		t.Errorf("DSAR.Type = %s, want access", dsar.Type)
	}
	if dsar.Status != "pending" {
		t.Errorf("DSAR.Status = %s, want pending", dsar.Status)
	}
}

func TestROTDataModel(t *testing.T) {
	rot := ROTData{
		ID:         "rot-123",
		TenantID:   "tenant-456",
		DatasetID:  "dataset-789",
		Category:   "obsolete",
		Score:      0.85,
		Reason:     "Not accessed in 6 months",
		SizeBytes:  1024 * 1024 * 100, // 100MB
		LastAccess: time.Now().AddDate(0, -6, 0),
	}

	if rot.Category != "obsolete" {
		t.Errorf("ROTData.Category = %s, want obsolete", rot.Category)
	}
	if rot.Score != 0.85 {
		t.Errorf("ROTData.Score = %f, want 0.85", rot.Score)
	}
}

func TestListOpts(t *testing.T) {
	opts := DefaultListOpts()

	if opts.Limit != 50 {
		t.Errorf("DefaultListOpts().Limit = %d, want 50", opts.Limit)
	}
	if opts.Offset != 0 {
		t.Errorf("DefaultListOpts().Offset = %d, want 0", opts.Offset)
	}
}
