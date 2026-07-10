package api

const openAPISpecJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "TrustVault API",
    "description": "TrustVault is an enterprise-grade Data & AI Trust Platform that provides classification, governance, and audit capabilities for data flowing to/from AI systems.",
    "version": "1.0.0",
    "contact": {
      "name": "TrustVault Support",
      "email": "support@trustvault.io"
    },
    "license": {
      "name": "Proprietary"
    }
  },
  "servers": [
    {
      "url": "http://localhost:8080",
      "description": "Local development server"
    }
  ],
  "tags": [
    {"name": "Authentication", "description": "User authentication and API key management"},
    {"name": "Users", "description": "User management"},
    {"name": "Roles", "description": "Role and permission management"},
    {"name": "Data Sources", "description": "Data source connection management"},
    {"name": "Policies", "description": "Governance policy management"},
    {"name": "Classification", "description": "Data classification and PII detection"},
    {"name": "AI Gate", "description": "AI query gateway with governance"},
    {"name": "Quality", "description": "Data quality assessment"},
    {"name": "Privacy", "description": "Privacy compliance (DSAR, PIA, RoPA)"},
    {"name": "Audit", "description": "Audit trail and compliance reporting"},
    {"name": "Observability", "description": "System health and metrics"},
    {"name": "AI Governance", "description": "AI model governance and eligibility"},
    {"name": "Notifications", "description": "Alerts and webhooks"},
    {"name": "Jobs", "description": "Scheduled job management"},
    {"name": "Remediation", "description": "Data remediation actions"},
    {"name": "Reports", "description": "Report generation and analytics"},
    {"name": "Labels", "description": "Sensitivity label management"},
    {"name": "Feedback", "description": "Classification feedback and corrections"},
    {"name": "Advisor", "description": "Compliance advisor and recommendations"},
    {"name": "ROT", "description": "Redundant, Obsolete, Trivial data detection"},
    {"name": "Integrations", "description": "External system integrations"},
    {"name": "Data Map", "description": "Data lineage and mapping"}
  ],
  "security": [{"BearerAuth": []}],
  "paths": {
    "/health": {
      "get": {
        "summary": "Health check",
        "tags": ["Observability"],
        "security": [],
        "responses": {
          "200": {"description": "Service is healthy", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/HealthResponse"}}}}
        }
      }
    },
    "/metrics": {
      "get": {
        "summary": "Prometheus metrics",
        "description": "Returns metrics in Prometheus exposition format",
        "tags": ["Observability"],
        "security": [],
        "responses": {
          "200": {"description": "Prometheus metrics", "content": {"text/plain": {"schema": {"type": "string"}}}}
        }
      }
    },
    "/version": {
      "get": {
        "summary": "API version information",
        "tags": ["Observability"],
        "security": [],
        "responses": {
          "200": {"description": "Version info", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/VersionInfo"}}}}
        }
      }
    },
    "/api/v1/auth/login": {
      "post": {
        "summary": "User login",
        "tags": ["Authentication"],
        "security": [],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/LoginRequest"}}}},
        "responses": {
          "200": {"description": "Login successful", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/LoginResponse"}}}},
          "401": {"description": "Invalid credentials", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Error"}}}}
        }
      }
    },
    "/api/v1/auth/logout": {
      "post": {
        "summary": "User logout",
        "tags": ["Authentication"],
        "responses": {
          "200": {"description": "Logout successful", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/StatusResponse"}}}}
        }
      }
    },
    "/api/v1/auth/refresh": {
      "post": {
        "summary": "Refresh access token",
        "tags": ["Authentication"],
        "security": [],
        "responses": {
          "200": {"description": "Token refreshed", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/LoginResponse"}}}}
        }
      }
    },
    "/api/v1/auth/api-keys": {
      "post": {
        "summary": "Create API key",
        "tags": ["Authentication"],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/CreateAPIKeyRequest"}}}},
        "responses": {
          "201": {"description": "API key created", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/APIKey"}}}}
        }
      }
    },
    "/api/v1/users": {
      "get": {
        "summary": "List users",
        "tags": ["Users"],
        "parameters": [
          {"$ref": "#/components/parameters/Limit"},
          {"$ref": "#/components/parameters/Offset"}
        ],
        "responses": {
          "200": {"description": "List of users", "content": {"application/json": {"schema": {"type": "array", "items": {"$ref": "#/components/schemas/User"}}}}}
        }
      },
      "post": {
        "summary": "Create user",
        "tags": ["Users"],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/CreateUserRequest"}}}},
        "responses": {
          "201": {"description": "User created", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/User"}}}}
        }
      }
    },
    "/api/v1/users/{id}": {
      "get": {
        "summary": "Get user by ID",
        "tags": ["Users"],
        "parameters": [{"$ref": "#/components/parameters/ID"}],
        "responses": {
          "200": {"description": "User details", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/User"}}}},
          "404": {"description": "User not found", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Error"}}}}
        }
      },
      "put": {
        "summary": "Update user",
        "tags": ["Users"],
        "parameters": [{"$ref": "#/components/parameters/ID"}],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/UpdateUserRequest"}}}},
        "responses": {
          "200": {"description": "User updated", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/User"}}}}
        }
      },
      "delete": {
        "summary": "Delete user",
        "tags": ["Users"],
        "parameters": [{"$ref": "#/components/parameters/ID"}],
        "responses": {
          "200": {"description": "User deleted", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/StatusResponse"}}}}
        }
      }
    },
    "/api/v1/datasources": {
      "get": {
        "summary": "List data sources",
        "tags": ["Data Sources"],
        "parameters": [{"$ref": "#/components/parameters/Limit"}, {"$ref": "#/components/parameters/Offset"}],
        "responses": {
          "200": {"description": "List of data sources", "content": {"application/json": {"schema": {"type": "array", "items": {"$ref": "#/components/schemas/DataSource"}}}}}
        }
      },
      "post": {
        "summary": "Create data source",
        "tags": ["Data Sources"],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/CreateDataSourceRequest"}}}},
        "responses": {
          "201": {"description": "Data source created", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/DataSource"}}}}
        }
      }
    },
    "/api/v1/datasources/{id}": {
      "get": {
        "summary": "Get data source by ID",
        "tags": ["Data Sources"],
        "parameters": [{"$ref": "#/components/parameters/ID"}],
        "responses": {
          "200": {"description": "Data source details", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/DataSource"}}}}
        }
      },
      "put": {
        "summary": "Update data source",
        "tags": ["Data Sources"],
        "parameters": [{"$ref": "#/components/parameters/ID"}],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/UpdateDataSourceRequest"}}}},
        "responses": {
          "200": {"description": "Data source updated", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/DataSource"}}}}
        }
      },
      "delete": {
        "summary": "Delete data source",
        "tags": ["Data Sources"],
        "parameters": [{"$ref": "#/components/parameters/ID"}],
        "responses": {
          "200": {"description": "Data source deleted", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/StatusResponse"}}}}
        }
      }
    },
    "/api/v1/datasources/{id}/scan": {
      "post": {
        "summary": "Trigger data source scan",
        "tags": ["Data Sources"],
        "parameters": [{"$ref": "#/components/parameters/ID"}],
        "responses": {
          "200": {"description": "Scan started", "content": {"application/json": {"schema": {"type": "object", "properties": {"status": {"type": "string"}, "job_id": {"type": "string"}}}}}}
        }
      }
    },
    "/api/v1/governance/policies": {
      "get": {
        "summary": "List governance policies",
        "tags": ["Policies"],
        "parameters": [{"$ref": "#/components/parameters/Limit"}, {"$ref": "#/components/parameters/Offset"}],
        "responses": {
          "200": {"description": "List of policies", "content": {"application/json": {"schema": {"type": "array", "items": {"$ref": "#/components/schemas/Policy"}}}}}
        }
      },
      "post": {
        "summary": "Create governance policy",
        "tags": ["Policies"],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/CreatePolicyRequest"}}}},
        "responses": {
          "201": {"description": "Policy created", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Policy"}}}}
        }
      }
    },
    "/api/v1/governance/evaluate": {
      "post": {
        "summary": "Evaluate data against policies",
        "tags": ["Policies"],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/EvaluateRequest"}}}},
        "responses": {
          "200": {"description": "Evaluation result", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/EvaluateResponse"}}}}
        }
      }
    },
    "/api/v1/classify/text": {
      "post": {
        "summary": "Classify text for PII and sensitive data",
        "tags": ["Classification"],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/ClassifyTextRequest"}}}},
        "responses": {
          "200": {"description": "Classification results", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/ClassifyResponse"}}}}
        }
      }
    },
    "/api/v1/classify/dataset": {
      "post": {
        "summary": "Classify entire dataset",
        "tags": ["Classification"],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/ClassifyDatasetRequest"}}}},
        "responses": {
          "200": {"description": "Classification job queued", "content": {"application/json": {"schema": {"type": "object", "properties": {"status": {"type": "string"}, "job_id": {"type": "string"}}}}}}
        }
      }
    },
    "/api/v1/gate/query": {
      "post": {
        "summary": "Query AI with governance",
        "description": "Send a query through the AI Gate with automatic context retrieval, policy evaluation, and response validation",
        "tags": ["AI Gate"],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/GateQueryRequest"}}}},
        "responses": {
          "200": {"description": "Query response", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/GateQueryResponse"}}}}
        }
      }
    },
    "/api/v1/gate/retrieve": {
      "post": {
        "summary": "Retrieve context chunks",
        "tags": ["AI Gate"],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/RetrieveRequest"}}}},
        "responses": {
          "200": {"description": "Retrieved chunks", "content": {"application/json": {"schema": {"type": "object", "properties": {"chunks": {"type": "array", "items": {"$ref": "#/components/schemas/ChunkResult"}}}}}}}
        }
      }
    },
    "/api/v1/quality/datasets/{id}": {
      "get": {
        "summary": "Get quality score for dataset",
        "tags": ["Quality"],
        "parameters": [{"$ref": "#/components/parameters/ID"}],
        "responses": {
          "200": {"description": "Quality score", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/QualityScore"}}}}
        }
      }
    },
    "/api/v1/quality/assess": {
      "post": {
        "summary": "Trigger quality assessment",
        "tags": ["Quality"],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"type": "object", "properties": {"dataset_id": {"type": "string"}}, "required": ["dataset_id"]}}}},
        "responses": {
          "200": {"description": "Assessment queued", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/StatusResponse"}}}}
        }
      }
    },
    "/api/v1/privacy/dsar": {
      "get": {
        "summary": "List DSARs",
        "tags": ["Privacy"],
        "parameters": [{"$ref": "#/components/parameters/Limit"}, {"$ref": "#/components/parameters/Offset"}],
        "responses": {
          "200": {"description": "List of DSARs", "content": {"application/json": {"schema": {"type": "array", "items": {"$ref": "#/components/schemas/DSAR"}}}}}
        }
      },
      "post": {
        "summary": "Create DSAR",
        "tags": ["Privacy"],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/CreateDSARRequest"}}}},
        "responses": {
          "201": {"description": "DSAR created", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/DSAR"}}}}
        }
      }
    },
    "/api/v1/audit/trail": {
      "get": {
        "summary": "Get audit trail",
        "tags": ["Audit"],
        "parameters": [{"$ref": "#/components/parameters/Limit"}, {"$ref": "#/components/parameters/Offset"}],
        "responses": {
          "200": {"description": "Audit log entries", "content": {"application/json": {"schema": {"type": "array", "items": {"$ref": "#/components/schemas/AuditLog"}}}}}
        }
      }
    },
    "/api/v1/observability/health": {
      "get": {
        "summary": "Get system health",
        "tags": ["Observability"],
        "responses": {
          "200": {"description": "System health status", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/SystemHealth"}}}}
        }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "BearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT",
        "description": "JWT Bearer token"
      },
      "APIKeyAuth": {
        "type": "apiKey",
        "in": "header",
        "name": "X-API-Key",
        "description": "API Key for service-to-service authentication"
      }
    },
    "parameters": {
      "ID": {"name": "id", "in": "path", "required": true, "schema": {"type": "string", "format": "uuid"}, "description": "Resource ID"},
      "Limit": {"name": "limit", "in": "query", "schema": {"type": "integer", "default": 50, "maximum": 100}, "description": "Maximum items to return"},
      "Offset": {"name": "offset", "in": "query", "schema": {"type": "integer", "default": 0}, "description": "Items to skip"}
    },
    "schemas": {
      "Error": {"type": "object", "properties": {"error": {"type": "string"}}},
      "StatusResponse": {"type": "object", "properties": {"status": {"type": "string"}}},
      "HealthResponse": {"type": "object", "properties": {"status": {"type": "string", "example": "ok"}}},
      "VersionInfo": {"type": "object", "properties": {"version": {"type": "string"}, "build_date": {"type": "string"}, "git_commit": {"type": "string"}, "go_version": {"type": "string"}, "environment": {"type": "string"}}},
      "LoginRequest": {"type": "object", "required": ["email", "password"], "properties": {"email": {"type": "string", "format": "email"}, "password": {"type": "string", "minLength": 8}}},
      "LoginResponse": {"type": "object", "properties": {"access_token": {"type": "string"}, "refresh_token": {"type": "string"}, "expires_in": {"type": "integer"}}},
      "CreateAPIKeyRequest": {"type": "object", "required": ["name"], "properties": {"name": {"type": "string"}, "permissions": {"type": "array", "items": {"type": "string"}}, "expires_in": {"type": "integer"}}},
      "APIKey": {"type": "object", "properties": {"id": {"type": "string"}, "name": {"type": "string"}, "expires_at": {"type": "string", "format": "date-time"}, "created_at": {"type": "string", "format": "date-time"}}},
      "User": {"type": "object", "properties": {"id": {"type": "string"}, "email": {"type": "string"}, "name": {"type": "string"}, "status": {"type": "string"}, "mfa_enabled": {"type": "boolean"}, "last_login_at": {"type": "string", "format": "date-time"}, "created_at": {"type": "string", "format": "date-time"}}},
      "CreateUserRequest": {"type": "object", "required": ["email", "password"], "properties": {"email": {"type": "string", "format": "email"}, "password": {"type": "string", "minLength": 8}, "name": {"type": "string"}, "role_id": {"type": "string"}}},
      "UpdateUserRequest": {"type": "object", "properties": {"name": {"type": "string"}, "status": {"type": "string"}}},
      "DataSource": {"type": "object", "properties": {"id": {"type": "string"}, "name": {"type": "string"}, "type": {"type": "string"}, "config": {"type": "object"}, "status": {"type": "string"}, "last_scan": {"type": "string", "format": "date-time"}, "created_at": {"type": "string", "format": "date-time"}}},
      "CreateDataSourceRequest": {"type": "object", "required": ["name", "type"], "properties": {"name": {"type": "string"}, "type": {"type": "string", "enum": ["postgresql", "mysql", "snowflake", "bigquery", "s3", "azure_blob", "gcs"]}, "config": {"type": "object"}}},
      "UpdateDataSourceRequest": {"type": "object", "properties": {"name": {"type": "string"}, "config": {"type": "object"}}},
      "Policy": {"type": "object", "properties": {"id": {"type": "string"}, "name": {"type": "string"}, "description": {"type": "string"}, "type": {"type": "string"}, "conditions": {"type": "object"}, "actions": {"type": "object"}, "regulations": {"type": "object"}, "active": {"type": "boolean"}, "priority": {"type": "integer"}, "created_at": {"type": "string", "format": "date-time"}}},
      "CreatePolicyRequest": {"type": "object", "required": ["name", "type"], "properties": {"name": {"type": "string"}, "description": {"type": "string"}, "type": {"type": "string", "enum": ["access", "redaction", "ai", "retention"]}, "conditions": {"type": "object"}, "actions": {"type": "object"}, "regulations": {"type": "array", "items": {"type": "string"}}, "priority": {"type": "integer"}}},
      "EvaluateRequest": {"type": "object", "required": ["data"], "properties": {"data": {"type": "string"}, "context": {"type": "object"}, "policy_ids": {"type": "array", "items": {"type": "string"}}}},
      "EvaluateResponse": {"type": "object", "properties": {"decision": {"type": "string", "enum": ["allow", "deny", "redact"]}, "redactions": {"type": "array", "items": {"$ref": "#/components/schemas/Redaction"}}, "violations": {"type": "array", "items": {"$ref": "#/components/schemas/PolicyViolation"}}, "applied_policies": {"type": "array", "items": {"type": "string"}}}},
      "Redaction": {"type": "object", "properties": {"start": {"type": "integer"}, "end": {"type": "integer"}, "type": {"type": "string"}, "masked": {"type": "string"}}},
      "PolicyViolation": {"type": "object", "properties": {"policy_id": {"type": "string"}, "policy_name": {"type": "string"}, "reason": {"type": "string"}}},
      "ClassifyTextRequest": {"type": "object", "required": ["text"], "properties": {"text": {"type": "string"}, "entity_types": {"type": "array", "items": {"type": "string"}}}},
      "ClassifyDatasetRequest": {"type": "object", "required": ["dataset_id"], "properties": {"dataset_id": {"type": "string"}, "async": {"type": "boolean"}}},
      "ClassifyResponse": {"type": "object", "properties": {"entities": {"type": "array", "items": {"$ref": "#/components/schemas/Entity"}}}},
      "Entity": {"type": "object", "properties": {"entity": {"type": "string"}, "value": {"type": "string"}, "confidence": {"type": "number"}, "start": {"type": "integer"}, "end": {"type": "integer"}}},
      "GateQueryRequest": {"type": "object", "required": ["query"], "properties": {"query": {"type": "string"}, "context": {"type": "object"}, "max_chunks": {"type": "integer", "default": 5}, "llm_endpoint": {"type": "string"}, "model": {"type": "string"}, "stream": {"type": "boolean"}}},
      "GateQueryResponse": {"type": "object", "properties": {"id": {"type": "string"}, "response": {"type": "string"}, "context": {"type": "array", "items": {"$ref": "#/components/schemas/ChunkResult"}}, "decision": {"type": "string"}, "redactions": {"type": "array", "items": {"$ref": "#/components/schemas/Redaction"}}, "latency_ms": {"type": "integer"}}},
      "RetrieveRequest": {"type": "object", "required": ["query"], "properties": {"query": {"type": "string"}, "max_chunks": {"type": "integer"}}},
      "ChunkResult": {"type": "object", "properties": {"id": {"type": "string"}, "content": {"type": "string"}, "source": {"type": "string"}, "score": {"type": "number"}, "metadata": {"type": "object"}}},
      "QualityScore": {"type": "object", "properties": {"id": {"type": "string"}, "dataset_id": {"type": "string"}, "overall": {"type": "number"}, "completeness": {"type": "number"}, "accuracy": {"type": "number"}, "consistency": {"type": "number"}, "timeliness": {"type": "number"}, "uniqueness": {"type": "number"}, "issues": {"type": "object"}, "created_at": {"type": "string", "format": "date-time"}}},
      "DSAR": {"type": "object", "properties": {"id": {"type": "string"}, "subject_id": {"type": "string"}, "type": {"type": "string", "enum": ["access", "delete", "rectify"]}, "status": {"type": "string"}, "deadline": {"type": "string", "format": "date-time"}, "results": {"type": "object"}, "completed_at": {"type": "string", "format": "date-time"}, "created_at": {"type": "string", "format": "date-time"}}},
      "CreateDSARRequest": {"type": "object", "required": ["subject_id", "type"], "properties": {"subject_id": {"type": "string"}, "type": {"type": "string", "enum": ["access", "delete", "rectify"]}}},
      "AuditLog": {"type": "object", "properties": {"id": {"type": "string"}, "user_id": {"type": "string"}, "action": {"type": "string"}, "resource": {"type": "string"}, "resource_id": {"type": "string"}, "details": {"type": "object"}, "ip": {"type": "string"}, "created_at": {"type": "string", "format": "date-time"}}},
      "SystemHealth": {"type": "object", "properties": {"status": {"type": "string"}, "checks": {"type": "object", "additionalProperties": {"type": "string"}}}}
    }
  }
}`
