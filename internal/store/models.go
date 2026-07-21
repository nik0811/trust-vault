package store

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"time"
)

type JSON json.RawMessage

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	// Remove null bytes that PostgreSQL can't handle
	clean := bytes.ReplaceAll([]byte(j), []byte{0x00}, []byte{})
	return clean, nil
}

func (j *JSON) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		// Make a copy to avoid buffer reuse issues
		cp := make([]byte, len(v))
		copy(cp, v)
		*j = cp
	case string:
		*j = []byte(v)
	default:
		*j = nil
	}
	return nil
}

func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	// Validate JSON before returning
	var tmp any
	if err := json.Unmarshal(j, &tmp); err != nil {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return nil
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// Tenant represents a customer organization
type Tenant struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name" validate:"required"`
	Slug      string    `db:"slug" json:"slug" validate:"required"`
	Status    string    `db:"status" json:"status"`
	Settings  JSON      `db:"settings" json:"settings"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// User represents a platform user
type User struct {
	ID           string     `db:"id" json:"id"`
	TenantID     string     `db:"tenant_id" json:"-"`
	Email        string     `db:"email" json:"email" validate:"required,email"`
	PasswordHash string     `db:"password_hash" json:"-"`
	Name         string     `db:"name" json:"name"`
	Status       string     `db:"status" json:"status"`
	IsSuperAdmin bool       `db:"is_super_admin" json:"is_super_admin"`
	MFAEnabled   bool       `db:"mfa_enabled" json:"mfa_enabled"`
	LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

// Role defines a set of permissions
type Role struct {
	ID          string    `db:"id" json:"id"`
	TenantID    string    `db:"tenant_id" json:"-"`
	Name        string    `db:"name" json:"name" validate:"required"`
	Description string    `db:"description" json:"description"`
	IsSystem    bool      `db:"is_system" json:"is_system"`
	Permissions JSON      `db:"permissions" json:"permissions"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// UserRole links users to roles
type UserRole struct {
	UserID     string    `db:"user_id" json:"user_id"`
	RoleID     string    `db:"role_id" json:"role_id"`
	AssignedAt time.Time `db:"assigned_at" json:"assigned_at"`
}

// APIKey for service-to-service auth
type APIKey struct {
	ID          string    `db:"id" json:"id"`
	TenantID    string    `db:"tenant_id" json:"-"`
	UserID      string    `db:"user_id" json:"user_id"`
	KeyHash     string    `db:"key_hash" json:"-"`
	Name        string    `db:"name" json:"name"`
	Permissions JSON      `db:"permissions" json:"permissions"`
	ExpiresAt   time.Time `db:"expires_at" json:"expires_at"`
	LastUsedAt  time.Time `db:"last_used_at" json:"last_used_at"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// DataSource represents a connected data system
type DataSource struct {
	ID               string     `db:"id" json:"id"`
	TenantID         string     `db:"tenant_id" json:"-"`
	Name             string     `db:"name" json:"name" validate:"required"`
	Type             string     `db:"type" json:"type" validate:"required"`
	Config           JSON       `db:"config" json:"config"`
	Status           string     `db:"status" json:"status"`
	LastScan         *time.Time `db:"last_scan" json:"last_scan,omitempty"`
	SensitivityLabel *string    `db:"sensitivity_label" json:"sensitivity_label,omitempty"`
	Region           *string    `db:"region" json:"region,omitempty"`
	Country          *string    `db:"country" json:"country,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
}

// Policy defines governance rules
type Policy struct {
	ID          string    `db:"id" json:"id"`
	TenantID    string    `db:"tenant_id" json:"-"`
	Name        string    `db:"name" json:"name" validate:"required"`
	Description string    `db:"description" json:"description"`
	Type        string    `db:"type" json:"type" validate:"required,oneof=access redaction ai retention"`
	Conditions  JSON      `db:"conditions" json:"conditions"`
	Actions     JSON      `db:"actions" json:"actions"`
	Regulations JSON      `db:"regulations" json:"regulations"`
	Active      bool      `db:"active" json:"active"`
	Priority    int       `db:"priority" json:"priority"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// Classification stores detected entities
type Classification struct {
	ID                   string    `db:"id" json:"id"`
	TenantID             string    `db:"tenant_id" json:"-"`
	DatasetID            string    `db:"dataset_id" json:"dataset_id"`
	SourceID             *string   `db:"source_id" json:"source_id,omitempty"`
	EntityType           string    `db:"entity_type" json:"entity_type"`
	Value                string    `db:"value" json:"value"`
	Confidence           float64   `db:"confidence" json:"confidence"`
	Context              JSON      `db:"context" json:"context"`
	LabelID              *string   `db:"label_id" json:"label_id,omitempty"`
	RuleID               *string   `db:"rule_id" json:"rule_id,omitempty"`
	ClassificationSource *string   `db:"classification_source" json:"classification_source,omitempty"`
	ValueSample          *string   `db:"value_sample" json:"value_sample,omitempty"`
	CreatedAt            time.Time `db:"created_at" json:"created_at"`
}

// AuditLog records all actions
type AuditLog struct {
	ID        string    `db:"id" json:"id"`
	TenantID  string    `db:"tenant_id" json:"-"`
	UserID    string    `db:"user_id" json:"user_id"`
	Action    string    `db:"action" json:"action"`
	Resource  string    `db:"resource" json:"resource"`
	ResourceID string   `db:"resource_id" json:"resource_id"`
	Details   JSON      `db:"details" json:"details"`
	IP        string    `db:"ip" json:"ip"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// GateQuery records AI Gate interactions
type GateQuery struct {
	ID            string    `db:"id" json:"id"`
	TenantID      string    `db:"tenant_id" json:"-"`
	UserID        string    `db:"user_id" json:"user_id"`
	Query         string    `db:"query" json:"query"`
	Context       JSON      `db:"context" json:"context"`
	Response      string    `db:"response" json:"response"`
	Decision      string    `db:"decision" json:"decision"`
	Redactions    JSON      `db:"redactions" json:"redactions"`
	LatencyMs     int       `db:"latency_ms" json:"latency_ms"`
	LLMEndpoint   string    `db:"llm_endpoint" json:"llm_endpoint"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// QualityScore stores data quality assessments
type QualityScore struct {
	ID           string    `db:"id" json:"id"`
	TenantID     string    `db:"tenant_id" json:"-"`
	DatasetID    string    `db:"dataset_id" json:"dataset_id"`
	Overall      float64   `db:"overall" json:"overall"`
	Completeness float64   `db:"completeness" json:"completeness"`
	Accuracy     float64   `db:"accuracy" json:"accuracy"`
	Consistency  float64   `db:"consistency" json:"consistency"`
	Timeliness   float64   `db:"timeliness" json:"timeliness"`
	Uniqueness   float64   `db:"uniqueness" json:"uniqueness"`
	Issues       JSON      `db:"issues" json:"issues"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// DSAR represents a data subject access request
type DSAR struct {
	ID          string     `db:"id" json:"id"`
	TenantID    string     `db:"tenant_id" json:"-"`
	SubjectID   string     `db:"subject_id" json:"subject_id"`
	Type        string     `db:"type" json:"type" validate:"oneof=access delete rectify"`
	Status      string     `db:"status" json:"status"`
	Deadline    time.Time  `db:"deadline" json:"deadline"`
	Results     JSON       `db:"results" json:"results"`
	AssignedTo  *string    `db:"assigned_to" json:"assigned_to,omitempty"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

// Job represents a scheduled task
type Job struct {
	ID        string     `db:"id" json:"id"`
	TenantID  string     `db:"tenant_id" json:"-"`
	Name      string     `db:"name" json:"name"`
	Type      string     `db:"type" json:"type"`
	Schedule  string     `db:"schedule" json:"schedule"`
	Config    JSON       `db:"config" json:"config"`
	Status    string     `db:"status" json:"status"`
	LastRun   *time.Time `db:"last_run" json:"last_run,omitempty"`
	NextRun   *time.Time `db:"next_run" json:"next_run,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}

// JobExecution tracks individual job runs
type JobExecution struct {
	ID          string     `db:"id" json:"id"`
	TenantID    string     `db:"tenant_id" json:"-"`
	JobID       string     `db:"job_id" json:"job_id"`
	Status      string     `db:"status" json:"status"`
	StartedAt   *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	Duration    *int       `db:"duration_ms" json:"duration_ms,omitempty"`
	Result      JSON       `db:"result" json:"result,omitempty"`
	Error       *string    `db:"error" json:"error,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

// Notification stores alerts and events
type Notification struct {
	ID        string    `db:"id" json:"id"`
	TenantID  string    `db:"tenant_id" json:"-"`
	Type      string    `db:"type" json:"type"`
	Severity  string    `db:"severity" json:"severity"`
	Title     string    `db:"title" json:"title"`
	Message   string    `db:"message" json:"message"`
	Resource  string    `db:"resource" json:"resource"`
	Read      bool      `db:"read" json:"read"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Webhook stores configured webhook endpoints
type Webhook struct {
	ID        string    `db:"id" json:"id"`
	TenantID  string    `db:"tenant_id" json:"-"`
	URL       string    `db:"url" json:"url" validate:"required,url"`
	Events    JSON      `db:"events" json:"events"`
	Secret    string    `db:"secret" json:"-"`
	Active    bool      `db:"active" json:"active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Label represents sensitivity labels
type Label struct {
	ID        string    `db:"id" json:"id"`
	TenantID  string    `db:"tenant_id" json:"-"`
	DatasetID string    `db:"dataset_id" json:"dataset_id"`
	Label     string    `db:"label" json:"label" validate:"oneof=PUBLIC INTERNAL CONFIDENTIAL HIGHLY_CONFIDENTIAL RESTRICTED"`
	AutoAssigned bool   `db:"auto_assigned" json:"auto_assigned"`
	AssignedBy *string  `db:"assigned_by" json:"assigned_by,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Feedback stores user corrections for self-learning
type Feedback struct {
	ID               string    `db:"id" json:"id"`
	TenantID         string    `db:"tenant_id" json:"-"`
	ClassificationID *string   `db:"classification_id" json:"classification_id,omitempty"`
	Type             string    `db:"type" json:"type" validate:"oneof=correction confirmation false_positive false_negative general"`
	OriginalLabel    string    `db:"original_label" json:"original_label"`
	CorrectedLabel   string    `db:"corrected_label" json:"corrected_label"`
	UserID           string    `db:"user_id" json:"user_id"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

// CustomEntity stores tenant-defined entity types for the classifier
type CustomEntity struct {
	ID          string    `db:"id" json:"id"`
	TenantID    string    `db:"tenant_id" json:"-"`
	Name        string    `db:"name" json:"name" validate:"required"`
	Pattern     string    `db:"pattern" json:"pattern" validate:"required"`
	Description string    `db:"description" json:"description"`
	Detections  int       `db:"detections" json:"detections"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// KnowledgeCacheEntry records a correction that teaches the classifier
type KnowledgeCacheEntry struct {
	ID         string    `db:"id" json:"id"`
	TenantID   string    `db:"tenant_id" json:"-"`
	EntityType string    `db:"entity_type" json:"entity_type"`
	Pattern    string    `db:"pattern" json:"pattern"`
	Correction string    `db:"correction" json:"correction"`
	HitCount   int       `db:"hit_count" json:"hit_count"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

// Integration stores outbound integration configs.
// Supported types: slack, teams, email, webhook, jira, servicenow, pagerduty,
// dlp, siem, splunk, sentinel, catalog, collibra, alation, onetrust, privacyops,
// rest_api, custom, privacy_platform, ticketing, communication
type Integration struct {
	ID        string     `db:"id" json:"id"`
	TenantID  string     `db:"tenant_id" json:"-"`
	Name      string     `db:"name" json:"name" validate:"required"`
	Type      string     `db:"type" json:"type" validate:"required"`
	Provider  string     `db:"provider" json:"provider"`
	Config    JSON       `db:"config" json:"config"`
	SyncFreq  string     `db:"sync_freq" json:"sync_freq"`
	Status    string     `db:"status" json:"status"`
	LastSync  *time.Time `db:"last_sync" json:"last_sync"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}

// ROTData stores ROT analysis results
type ROTData struct {
	ID         string     `db:"id" json:"id"`
	TenantID   string     `db:"tenant_id" json:"-"`
	DatasetID  string     `db:"dataset_id" json:"dataset_id"`
	Category   string     `db:"category" json:"category" validate:"oneof=redundant obsolete trivial"`
	Score      float64    `db:"score" json:"score"`
	Reason     string     `db:"reason" json:"reason"`
	SizeBytes  int64      `db:"size_bytes" json:"size_bytes"`
	LastAccess *time.Time `db:"last_access" json:"last_access"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// RemediationAction stores remediation tasks
type RemediationAction struct {
	ID         string     `db:"id" json:"id"`
	TenantID   string     `db:"tenant_id" json:"-"`
	Type       string     `db:"type" json:"type" validate:"required,oneof=redact encrypt delete quarantine label archive deduplicate flag"`
	ActionType string     `db:"action_type" json:"action_type"`
	DatasetID  string     `db:"dataset_id" json:"dataset_id"`
	Reason     string     `db:"reason" json:"reason"`
	Status     string     `db:"status" json:"status"`
	ApprovedBy *string    `db:"approved_by" json:"approved_by,omitempty"`
	ExecutedAt *time.Time `db:"executed_at" json:"executed_at,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at" json:"updated_at"`
}

// Report stores generated reports
type Report struct {
	ID        string     `db:"id" json:"id"`
	TenantID  string     `db:"tenant_id" json:"-"`
	Type      string     `db:"type" json:"type"`
	Status    string     `db:"status" json:"status"`
	DateFrom  *time.Time `db:"date_from" json:"date_from,omitempty"`
	DateTo    *time.Time `db:"date_to" json:"date_to,omitempty"`
	FilePath  string     `db:"file_path" json:"file_path"`
	Metadata  JSON       `db:"metadata" json:"metadata"`
	Content   JSON       `db:"content" json:"content,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}

// LabelRule defines auto-labeling rules
type LabelRule struct {
	ID             string    `db:"id" json:"id"`
	TenantID       string    `db:"tenant_id" json:"-"`
	Classification string    `db:"classification" json:"classification"`
	Label          string    `db:"label" json:"label"`
	Priority       int       `db:"priority" json:"priority"`
	Active         bool      `db:"active" json:"active"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// RoPA represents Records of Processing Activities
type RoPA struct {
	ID               string    `db:"id" json:"id"`
	TenantID         string    `db:"tenant_id" json:"-"`
	Name             string    `db:"name" json:"name"`
	Purpose          string    `db:"purpose" json:"purpose"`
	LegalBasis       string    `db:"legal_basis" json:"legal_basis"`
	DataCategories   JSON      `db:"data_categories" json:"data_categories"`
	Recipients       JSON      `db:"recipients" json:"recipients"`
	RetentionPeriod  string    `db:"retention_period" json:"retention_period"`
	SecurityMeasures string    `db:"security_measures" json:"security_measures"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

// Playbook stores remediation playbooks
type Playbook struct {
	ID        string    `db:"id" json:"id"`
	TenantID  string    `db:"tenant_id" json:"-"`
	IssueType string    `db:"issue_type" json:"issue_type"`
	Name      string    `db:"name" json:"name"`
	Steps     JSON      `db:"steps" json:"steps"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ModelLineage tracks AI model data lineage
type ModelLineage struct {
	ID        string    `db:"id" json:"id"`
	TenantID  string    `db:"tenant_id" json:"-"`
	ModelID   string    `db:"model_id" json:"model_id"`
	DatasetID string    `db:"dataset_id" json:"dataset_id"`
	UsageType string    `db:"usage_type" json:"usage_type"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// IntegrationLog stores integration sync logs
type IntegrationLog struct {
	ID            string    `db:"id" json:"id"`
	TenantID      string    `db:"tenant_id" json:"-"`
	IntegrationID string    `db:"integration_id" json:"integration_id"`
	Level         string    `db:"level" json:"level"`
	Message       string    `db:"message" json:"message"`
	Details       JSON      `db:"details" json:"details"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// DataFlow represents data lineage between datasets
type DataFlow struct {
	ID              string    `db:"id" json:"id"`
	TenantID        string    `db:"tenant_id" json:"-"`
	SourceDatasetID string    `db:"source_dataset_id" json:"source_dataset_id"`
	TargetDatasetID string    `db:"target_dataset_id" json:"target_dataset_id"`
	FlowType        string    `db:"flow_type" json:"flow_type"`
	Metadata        JSON      `db:"metadata" json:"metadata"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

// DuplicateGroup stores duplicate detection results
type DuplicateGroup struct {
	ID             string    `db:"id" json:"id"`
	TenantID       string    `db:"tenant_id" json:"-"`
	Hash           string    `db:"hash" json:"hash"`
	DatasetIDs     JSON      `db:"dataset_ids" json:"dataset_ids"`
	TotalSizeBytes int64     `db:"total_size_bytes" json:"total_size_bytes"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

// ReviewQueueItem represents a document pending review
type ReviewQueueItem struct {
	ID                    string     `db:"id" json:"id"`
	TenantID              string     `db:"tenant_id" json:"-"`
	DocumentID            string     `db:"document_id" json:"document_id"`
	DocumentName          string     `db:"document_name" json:"document_name"`
	Status                string     `db:"status" json:"status"`
	ClassificationResults JSON       `db:"classification_results" json:"classification_results"`
	AssignedTo            *string    `db:"assigned_to" json:"assigned_to,omitempty"`
	CreatedAt             time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time  `db:"updated_at" json:"updated_at"`
}

// RetentionPolicy defines data retention rules
type RetentionPolicy struct {
	ID             string    `db:"id" json:"id"`
	TenantID       string    `db:"tenant_id" json:"-"`
	Name           string    `db:"name" json:"name"`
	Classification string    `db:"classification" json:"classification"`
	RetentionDays  int       `db:"retention_days" json:"retention_days"`
	Action         string    `db:"action" json:"action"`
	Active         bool      `db:"active" json:"active"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// RetentionViolation tracks retention policy violations
type RetentionViolation struct {
	ID            string    `db:"id" json:"id"`
	TenantID      string    `db:"tenant_id" json:"-"`
	DatasetID     string    `db:"dataset_id" json:"dataset_id"`
	PolicyID      *string   `db:"policy_id" json:"policy_id,omitempty"`
	ViolationType string    `db:"violation_type" json:"violation_type"`
	DaysOverdue   int       `db:"days_overdue" json:"days_overdue"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// ClassificationModel represents available classification models
type ClassificationModel struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Size      string    `db:"size" json:"size"`
	Accuracy  float64   `db:"accuracy" json:"accuracy"`
	Speed     string    `db:"speed" json:"speed"`
	IsDefault bool      `db:"is_default" json:"default"`
	Active    bool      `db:"active" json:"active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Invitation represents a user invitation for registration
type Invitation struct {
	ID         string     `db:"id" json:"id"`
	TenantID   string     `db:"tenant_id" json:"tenant_id"`
	Email      string     `db:"email" json:"email" validate:"required,email"`
	Role       string     `db:"role" json:"role" validate:"required"`
	Token      string     `db:"token" json:"-"`
	InvitedBy  *string    `db:"invited_by" json:"invited_by,omitempty"`
	ExpiresAt  time.Time  `db:"expires_at" json:"expires_at"`
	AcceptedAt *time.Time `db:"accepted_at" json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// ScanLog stores scan history for datasources
type ScanLog struct {
	ID                 string     `db:"id" json:"id"`
	TenantID           string     `db:"tenant_id" json:"-"`
	DatasourceID       string     `db:"datasource_id" json:"datasource_id"`
	Status             string     `db:"status" json:"status"`
	StartedAt          time.Time  `db:"started_at" json:"started_at"`
	CompletedAt        *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	Message            string     `db:"message" json:"message"`
	Logs               JSON       `db:"logs" json:"logs"`
	DatasetsDiscovered int        `db:"datasets_discovered" json:"datasets_discovered"`
	CreatedAt          time.Time  `db:"created_at" json:"created_at"`
}

// EndpointAgent represents a registered endpoint device/server
type EndpointAgent struct {
	ID           string     `db:"id" json:"id"`
	TenantID     string     `db:"tenant_id" json:"-"`
	Hostname     string     `db:"hostname" json:"hostname" validate:"required"`
	IP           string     `db:"ip" json:"ip"`
	OS           string     `db:"os" json:"os"`
	AgentVersion string     `db:"agent_version" json:"agent_version"`
	Status       string     `db:"status" json:"status"`
	LastSeenAt   *time.Time `db:"last_seen_at" json:"last_seen_at,omitempty"`
	LastScanAt   *time.Time `db:"last_scan_at" json:"last_scan_at,omitempty"`
	ScanResults  JSON       `db:"scan_results" json:"scan_results"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}

// ResidencyRule defines geographic data residency requirements
type ResidencyRule struct {
	ID             string    `db:"id" json:"id"`
	TenantID       string    `db:"tenant_id" json:"-"`
	Name           string    `db:"name" json:"name" validate:"required"`
	Regulation     string    `db:"regulation" json:"regulation"`
	AllowedRegions JSON      `db:"allowed_regions" json:"allowed_regions"`
	DataTypes      JSON      `db:"data_types" json:"data_types"`
	Active         bool      `db:"active" json:"active"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

// ConsentWidgetConfig holds the tenant's consent banner configuration
type ConsentWidgetConfig struct {
	ID              string    `db:"id" json:"id"`
	TenantID        string    `db:"tenant_id" json:"-"`
	PrimaryColor    string    `db:"primary_color" json:"primary_color"`
	BackgroundColor string    `db:"background_color" json:"background_color"`
	TextColor       string    `db:"text_color" json:"text_color"`
	BannerTitle     string    `db:"banner_title" json:"banner_title"`
	BannerText      string    `db:"banner_text" json:"banner_text"`
	AcceptLabel     string    `db:"accept_label" json:"accept_label"`
	RejectLabel     string    `db:"reject_label" json:"reject_label"`
	Purposes        JSON      `db:"purposes" json:"purposes"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

// DPIA (Data Protection Impact Assessment)
type DPIA struct {
	ID                 string    `db:"id" json:"id"`
	TenantID           string    `db:"tenant_id" json:"-"`
	Name               string    `db:"name" json:"name" validate:"required"`
	Description        string    `db:"description" json:"description"`
	DataTypes          JSON      `db:"data_types" json:"data_types"`
	ProcessingPurpose  string    `db:"processing_purpose" json:"processing_purpose"`
	RiskLevel          string    `db:"risk_level" json:"risk_level"`
	Status             string    `db:"status" json:"status"`
	Steps              JSON      `db:"steps" json:"steps"`
	DPOConsulted       bool      `db:"dpo_consulted" json:"dpo_consulted"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time `db:"updated_at" json:"updated_at"`
}

// ConsentRecord tracks individual consent events
type ConsentRecord struct {
	ID        string    `db:"id" json:"id"`
	TenantID  string    `db:"tenant_id" json:"-"`
	SubjectID string    `db:"subject_id" json:"subject_id" validate:"required"`
	Purpose   string    `db:"purpose" json:"purpose" validate:"required"`
	Status    string    `db:"status" json:"status"`
	IP        string    `db:"ip" json:"ip"`
	Source    string    `db:"source" json:"source"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// CriticalDataElement designates a column as a Critical Data Element
type CriticalDataElement struct {
	ID                 string         `db:"id" json:"id"`
	TenantID           string         `db:"tenant_id" json:"-"`
	DatasourceID       sql.NullString `db:"datasource_id" json:"datasource_id"`
	ColumnName         string         `db:"column_name" json:"column_name" validate:"required"`
	TableName          string         `db:"table_name" json:"table_name" validate:"required"`
	BusinessDefinition string         `db:"business_definition" json:"business_definition"`
	DataOwner          string         `db:"data_owner" json:"data_owner"`
	Criticality        string         `db:"criticality" json:"criticality"`
	QualityScore       float64        `db:"quality_score" json:"quality_score"`
	CreatedAt          time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at" json:"updated_at"`
}

// DataProfile stores auto-profiling results for a datasource
type DataProfile struct {
	ID           string    `db:"id" json:"id"`
	TenantID     string    `db:"tenant_id" json:"-"`
	DatasourceID string    `db:"datasource_id" json:"datasource_id"`
	ProfileData  JSON      `db:"profile_data" json:"profile_data"`
	Status       string    `db:"status" json:"status"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// DocumentClassification links a document to its classification results
type DocumentClassification struct {
	ID           string    `db:"id" json:"id"`
	TenantID     string    `db:"tenant_id" json:"-"`
	DocumentID   string    `db:"document_id" json:"document_id"`
	DocumentName string    `db:"document_name" json:"document_name"`
	EntityTypes  JSON      `db:"entity_types" json:"entity_types"`
	Findings     JSON      `db:"findings" json:"findings"`
	Governed     bool      `db:"governed" json:"governed"`
	LabelApplied string    `db:"label_applied" json:"label_applied"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// ClassificationRule defines rules for classification overrides and patterns
type ClassificationRule struct {
	ID            string    `db:"id" json:"id"`
	TenantID      string    `db:"tenant_id" json:"-"`
	Name          string    `db:"name" json:"name" validate:"required"`
	Type          string    `db:"type" json:"type" validate:"required,oneof=override pattern whitelist threshold"`
	ColumnPattern string    `db:"column_pattern" json:"column_pattern"`
	ValuePattern  string    `db:"value_pattern" json:"value_pattern"`
	EntityType    string    `db:"entity_type" json:"entity_type"`
	Confidence    float64   `db:"confidence" json:"confidence"`
	Priority      int       `db:"priority" json:"priority"`
	Active        bool      `db:"active" json:"active"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// EndpointScan represents a registered API endpoint being scanned for PII exposure
type EndpointScan struct {
	ID         string     `db:"id" json:"id"`
	TenantID   string     `db:"tenant_id" json:"-"`
	Name       string     `db:"name" json:"name" validate:"required"`
	URL        string     `db:"url" json:"url" validate:"required"`
	Method     string     `db:"method" json:"method"`
	Headers    JSON       `db:"headers" json:"headers"`
	AuthType   string     `db:"auth_type" json:"auth_type"`
	AuthConfig JSON       `db:"auth_config" json:"auth_config"`
	Status     string     `db:"status" json:"status"`
	LastScan   *time.Time `db:"last_scan" json:"last_scan"`
	Findings   JSON       `db:"findings" json:"findings"`
	RiskLevel  string     `db:"risk_level" json:"risk_level"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at" json:"updated_at"`
}

// ConsentPreference stores per-subject consent preferences persistently
type ConsentPreference struct {
	ID          string    `db:"id" json:"id"`
	TenantID    string    `db:"tenant_id" json:"-"`
	SubjectID   string    `db:"subject_id" json:"subject_id" validate:"required"`
	Preferences JSON      `db:"preferences" json:"preferences"`
	IP          string    `db:"ip" json:"ip"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// ComplianceAssessment stores the result of a compliance assessment run
type ComplianceAssessment struct {
	ID                    string    `db:"id" json:"id"`
	TenantID              string    `db:"tenant_id" json:"-"`
	AssessedBy            string    `db:"assessed_by" json:"assessed_by"`
	ComplianceScore       float64   `db:"compliance_score" json:"compliance_score"`
	TotalFindings         int       `db:"total_findings" json:"total_findings"`
	CriticalFindings      int       `db:"critical_findings" json:"critical_findings"`
	HighFindings          int       `db:"high_findings" json:"high_findings"`
	MediumFindings        int       `db:"medium_findings" json:"medium_findings"`
	LowFindings           int       `db:"low_findings" json:"low_findings"`
	TotalEvidence         int       `db:"total_evidence" json:"total_evidence"`
	DataSourcesChecked    int       `db:"data_sources_checked" json:"data_sources_checked"`
	ClassificationsChecked int      `db:"classifications_checked" json:"classifications_checked"`
	PoliciesEvaluated     int       `db:"policies_evaluated" json:"policies_evaluated"`
	RegulationsCovered    JSON      `db:"regulations_covered" json:"regulations_covered"`
	Summary               JSON      `db:"summary" json:"summary"`
	CreatedAt             time.Time `db:"created_at" json:"created_at"`
}
