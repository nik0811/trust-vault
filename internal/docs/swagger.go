// Package docs TrustVault API Documentation
//
// TrustVault is an enterprise-grade Data & AI Trust Platform that provides
// classification, governance, and audit capabilities for data flowing to/from AI systems.
//
//	Schemes: http, https
//	Host: localhost:8080
//	BasePath: /api/v1
//	Version: 1.0.0
//	License: Proprietary
//	Contact: TrustVault Support<support@trustvault.io>
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Security:
//	- BearerAuth:
//
//	SecurityDefinitions:
//	  BearerAuth:
//	    type: apiKey
//	    name: Authorization
//	    in: header
//	    description: JWT Bearer token. Format "Bearer {token}"
//	  APIKeyAuth:
//	    type: apiKey
//	    name: X-API-Key
//	    in: header
//	    description: API Key for service-to-service authentication
//
// swagger:meta
package docs

import "github.com/trustvault/trustvault/internal/store"

// Generic error response
// swagger:response errorResponse
type errorResponse struct {
	// in: body
	Body struct {
		// Error message
		// example: unauthorized
		Error string `json:"error"`
	}
}

// Generic success response
// swagger:response successResponse
type successResponse struct {
	// in: body
	Body struct {
		// Status message
		// example: ok
		Status string `json:"status"`
	}
}

// Health check response
// swagger:response healthResponse
type healthResponse struct {
	// in: body
	Body struct {
		// Health status
		// example: ok
		Status string `json:"status"`
	}
}

// ============= Authentication =============

// Login request
// swagger:parameters login
type loginParams struct {
	// Login credentials
	// in: body
	// required: true
	Body struct {
		// User email
		// required: true
		// example: admin@example.com
		Email string `json:"email"`
		// User password
		// required: true
		// example: password123
		Password string `json:"password"`
	}
}

// Login response
// swagger:response loginResponse
type loginResponse struct {
	// in: body
	Body struct {
		// JWT access token
		// example: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
		AccessToken string `json:"access_token"`
		// JWT refresh token
		// example: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
		RefreshToken string `json:"refresh_token"`
		// Token expiration in seconds
		// example: 86400
		ExpiresIn int `json:"expires_in"`
	}
}

// ============= Users =============

// List users response
// swagger:response usersListResponse
type usersListResponse struct {
	// in: body
	Body []store.User
}

// User response
// swagger:response userResponse
type userResponse struct {
	// in: body
	Body store.User
}

// Create user request
// swagger:parameters createUser
type createUserParams struct {
	// User details
	// in: body
	// required: true
	Body struct {
		// User email
		// required: true
		// example: user@example.com
		Email string `json:"email"`
		// User password (min 8 characters)
		// required: true
		// example: securepassword123
		Password string `json:"password"`
		// User display name
		// example: John Doe
		Name string `json:"name"`
		// Role ID to assign
		// example: 550e8400-e29b-41d4-a716-446655440000
		RoleID string `json:"role_id"`
	}
}

// ============= Data Sources =============

// List data sources response
// swagger:response dataSourcesListResponse
type dataSourcesListResponse struct {
	// in: body
	Body []store.DataSource
}

// Data source response
// swagger:response dataSourceResponse
type dataSourceResponse struct {
	// in: body
	Body store.DataSource
}

// Create data source request
// swagger:parameters createDataSource
type createDataSourceParams struct {
	// Data source details
	// in: body
	// required: true
	Body struct {
		// Data source name
		// required: true
		// example: Production PostgreSQL
		Name string `json:"name"`
		// Data source type
		// required: true
		// enum: postgresql,mysql,snowflake,bigquery,s3,azure_blob,gcs
		// example: postgresql
		Type string `json:"type"`
		// Connection configuration (JSON)
		// example: {"host": "db.example.com", "port": 5432, "database": "prod"}
		Config map[string]interface{} `json:"config"`
	}
}

// ============= Policies =============

// List policies response
// swagger:response policiesListResponse
type policiesListResponse struct {
	// in: body
	Body []store.Policy
}

// Policy response
// swagger:response policyResponse
type policyResponse struct {
	// in: body
	Body store.Policy
}

// Create policy request
// swagger:parameters createPolicy
type createPolicyParams struct {
	// Policy details
	// in: body
	// required: true
	Body struct {
		// Policy name
		// required: true
		// example: PII Redaction Policy
		Name string `json:"name"`
		// Policy description
		// example: Automatically redact PII in AI responses
		Description string `json:"description"`
		// Policy type
		// required: true
		// enum: access,redaction,ai,retention
		// example: redaction
		Type string `json:"type"`
		// Policy conditions (JSON)
		Conditions map[string]interface{} `json:"conditions"`
		// Policy actions (JSON)
		Actions map[string]interface{} `json:"actions"`
		// Applicable regulations
		// example: ["GDPR", "CCPA"]
		Regulations []string `json:"regulations"`
		// Policy priority (higher = evaluated first)
		// example: 100
		Priority int `json:"priority"`
	}
}

// Evaluate policy request
// swagger:parameters evaluatePolicy
type evaluatePolicyParams struct {
	// Evaluation request
	// in: body
	// required: true
	Body struct {
		// Data to evaluate
		// required: true
		// example: John Doe's SSN is 123-45-6789
		Data string `json:"data"`
		// Additional context
		Context map[string]interface{} `json:"context"`
		// Specific policy IDs to evaluate (optional, evaluates all if empty)
		PolicyIDs []string `json:"policy_ids"`
	}
}

// Evaluate policy response
// swagger:response evaluatePolicyResponse
type evaluatePolicyResponse struct {
	// in: body
	Body struct {
		// Decision: allow, deny, or redact
		// example: redact
		Decision string `json:"decision"`
		// Redactions to apply
		Redactions []struct {
			Start  int    `json:"start"`
			End    int    `json:"end"`
			Type   string `json:"type"`
			Masked string `json:"masked"`
		} `json:"redactions"`
		// Policy violations found
		Violations []struct {
			PolicyID   string `json:"policy_id"`
			PolicyName string `json:"policy_name"`
			Reason     string `json:"reason"`
		} `json:"violations"`
		// IDs of policies that were applied
		AppliedPolicies []string `json:"applied_policies"`
	}
}

// ============= Classification =============

// Classify text request
// swagger:parameters classifyText
type classifyTextParams struct {
	// Classification request
	// in: body
	// required: true
	Body struct {
		// Text to classify
		// required: true
		// example: Contact John at john@example.com or call 555-123-4567
		Text string `json:"text"`
		// Entity types to detect (optional, detects all if empty)
		// example: ["EMAIL", "PHONE", "SSN"]
		EntityTypes []string `json:"entity_types"`
	}
}

// Classification response
// swagger:response classifyResponse
type classifyResponse struct {
	// in: body
	Body struct {
		// Detected entities
		Entities []struct {
			// Entity type
			// example: EMAIL
			Entity string `json:"entity"`
			// Detected value
			// example: john@example.com
			Value string `json:"value"`
			// Confidence score (0-1)
			// example: 0.95
			Confidence float64 `json:"confidence"`
			// Start position in text
			// example: 16
			Start int `json:"start"`
			// End position in text
			// example: 32
			End int `json:"end"`
		} `json:"entities"`
	}
}

// ============= AI Gate =============

// Gate query request
// swagger:parameters gateQuery
type gateQueryParams struct {
	// Query request
	// in: body
	// required: true
	Body struct {
		// User query
		// required: true
		// example: What are the sales figures for Q4?
		Query string `json:"query"`
		// Additional context
		Context map[string]interface{} `json:"context"`
		// Maximum chunks to retrieve
		// example: 5
		MaxChunks int `json:"max_chunks"`
		// LLM endpoint URL (optional)
		// example: http://localhost:11434/v1
		LLMEndpoint string `json:"llm_endpoint"`
		// Model name (optional)
		// example: llama2
		Model string `json:"model"`
		// Enable streaming response
		// example: false
		Stream bool `json:"stream"`
	}
}

// Gate query response
// swagger:response gateQueryResponse
type gateQueryResponse struct {
	// in: body
	Body struct {
		// Query ID
		// example: 550e8400-e29b-41d4-a716-446655440000
		ID string `json:"id"`
		// LLM response
		// example: The Q4 sales figures show a 15% increase...
		Response string `json:"response"`
		// Retrieved context chunks
		Context []struct {
			ID       string                 `json:"id"`
			Content  string                 `json:"content"`
			Source   string                 `json:"source"`
			Score    float32                `json:"score"`
			Metadata map[string]interface{} `json:"metadata"`
		} `json:"context"`
		// Governance decision
		// example: allow
		Decision string `json:"decision"`
		// Applied redactions
		Redactions []struct {
			Start  int    `json:"start"`
			End    int    `json:"end"`
			Type   string `json:"type"`
			Masked string `json:"masked"`
		} `json:"redactions"`
		// Query latency in milliseconds
		// example: 1250
		LatencyMs int `json:"latency_ms"`
	}
}

// ============= Quality =============

// Quality score response
// swagger:response qualityScoreResponse
type qualityScoreResponse struct {
	// in: body
	Body store.QualityScore
}

// ============= Privacy =============

// DSAR list response
// swagger:response dsarListResponse
type dsarListResponse struct {
	// in: body
	Body []store.DSAR
}

// DSAR response
// swagger:response dsarResponse
type dsarResponse struct {
	// in: body
	Body store.DSAR
}

// Create DSAR request
// swagger:parameters createDSAR
type createDSARParams struct {
	// DSAR details
	// in: body
	// required: true
	Body struct {
		// Data subject identifier
		// required: true
		// example: user@example.com
		SubjectID string `json:"subject_id"`
		// Request type
		// required: true
		// enum: access,delete,rectify
		// example: access
		Type string `json:"type"`
	}
}

// ============= Audit =============

// Audit trail response
// swagger:response auditTrailResponse
type auditTrailResponse struct {
	// in: body
	Body []store.AuditLog
}

// ============= Common Parameters =============

// Pagination parameters
// swagger:parameters listUsers listDataSources listPolicies listDSARs listJobs listNotifications
type paginationParams struct {
	// Maximum number of items to return
	// in: query
	// default: 50
	// maximum: 100
	Limit int `json:"limit"`
	// Number of items to skip
	// in: query
	// default: 0
	Offset int `json:"offset"`
}

// ID path parameter
// swagger:parameters getUser updateUser deleteUser getDataSource updateDataSource deleteDataSource getPolicy updatePolicy deletePolicy getDSAR
type idParam struct {
	// Resource ID
	// in: path
	// required: true
	ID string `json:"id"`
}
