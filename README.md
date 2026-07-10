# TrustVault - Data & AI Trust Platform

TrustVault is an enterprise-grade Data & AI Trust Platform that acts as an active governance gateway between data sources and AI systems. It classifies, governs, and audits all data flowing to and from AI (RAG pipelines + LLMs).

## Features

- **AI Gate**: Proxy between applications and LLMs with real-time governance
- **Classification Engine**: GLiNER ONNX model for 60+ PII types at 4M chars/sec
- **Governance Engine**: Policy-based access control, redaction, and compliance
- **Multi-tenant**: Full tenant isolation with RBAC
- **Data Quality**: 5-dimension quality scoring (completeness, accuracy, consistency, timeliness, uniqueness)
- **Privacy Compliance**: DSAR automation, PIA, RoPA, consent management
- **AI Governance**: EU AI Act compliance, model cards, training data lineage
- **Sensitivity Labels**: Auto-assign Public/Internal/Confidential/Restricted
- **ROT Detection**: Identify Redundant, Obsolete, Trivial data
- **Outbound Integrations**: Push to DLP, privacy platforms, catalogs
- **Enterprise Monitoring**: Prometheus metrics, structured logging, audit trails

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Make

### Run Locally

```bash
# Start infrastructure
make docker-up

# Run migrations
make migrate-up

# Start the server
make dev
```

The API will be available at `http://localhost:8080`.

### API Documentation

- **Swagger UI**: `http://localhost:8080/api/docs` - Interactive API documentation
- **OpenAPI Spec**: `http://localhost:8080/api/openapi.json` - OpenAPI 3.0 specification

### Superadmin Setup

TrustVault uses an invite-only registration system. The superadmin is created from environment variables on first startup:

```bash
# Set in docker-compose.local.yml or environment
SUPERADMIN_EMAIL=admin@yourcompany.com
SUPERADMIN_PASSWORD=your-secure-password
SUPERADMIN_NAME=Super Admin  # optional
```

**Important**: Change the default placeholder credentials in `docker-compose.local.yml` before running in any environment.

### Internal Admin Port

- **Internal Admin API**: `8099` (requires superadmin token)

## Monitoring & Observability

### Prometheus Metrics

TrustVault exposes Prometheus metrics at `/metrics` for comprehensive monitoring.

#### Available Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `trustvault_http_requests_total` | Counter | Total HTTP requests by method, endpoint, status |
| `trustvault_http_request_duration_seconds` | Histogram | HTTP request latency |
| `trustvault_http_active_connections` | Gauge | Current active connections |
| `trustvault_db_connections_open` | Gauge | Open database connections |
| `trustvault_db_connections_in_use` | Gauge | Database connections in use |
| `trustvault_db_connections_idle` | Gauge | Idle database connections |
| `trustvault_db_query_duration_seconds` | Histogram | Database query latency |
| `trustvault_kafka_messages_produced_total` | Counter | Kafka messages produced by topic |
| `trustvault_kafka_messages_consumed_total` | Counter | Kafka messages consumed |
| `trustvault_kafka_consumer_lag` | Gauge | Kafka consumer lag |
| `trustvault_classification_jobs_total` | Counter | Classification jobs by status |
| `trustvault_classification_duration_seconds` | Histogram | Classification job duration |
| `trustvault_classification_entities_found_total` | Counter | Entities found by type |
| `trustvault_gate_queries_total` | Counter | AI Gate queries by decision |
| `trustvault_gate_query_duration_seconds` | Histogram | AI Gate query latency |
| `trustvault_audit_logs_total` | Counter | Audit log entries by action |
| `trustvault_errors_total` | Counter | Errors by type and component |

#### Prometheus Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'trustvault'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
    scrape_interval: 15s
```

#### Grafana Dashboard

Import the TrustVault dashboard or create panels with these queries:

```promql
# Request rate
rate(trustvault_http_requests_total[5m])

# Error rate
sum(rate(trustvault_http_requests_total{status=~"5.."}[5m])) / sum(rate(trustvault_http_requests_total[5m]))

# P99 latency
histogram_quantile(0.99, rate(trustvault_http_request_duration_seconds_bucket[5m]))

# Classification throughput
rate(trustvault_classification_jobs_total{status="success"}[5m])

# AI Gate decisions
sum by (decision) (rate(trustvault_gate_queries_total[5m]))
```

### Structured Logging

All logs include structured fields for easy parsing and correlation:

| Field | Description |
|-------|-------------|
| `timestamp` | ISO 8601 timestamp |
| `level` | Log level (debug, info, warn, error) |
| `request_id` | Unique request identifier |
| `tenant_id` | Tenant identifier |
| `user_id` | User identifier |
| `endpoint` | API endpoint path |
| `method` | HTTP method |
| `status_code` | HTTP response status |
| `duration_ms` | Request duration in milliseconds |
| `remote_addr` | Client IP address |

#### Log Sampling

High-volume endpoints (`/health`, `/metrics`, `/api/v1/gate/query`) are sampled at 10% by default to reduce log volume. Configure via environment:

```bash
LOG_SAMPLE_RATE=0.1  # 10% sampling for high-volume endpoints
```

### Audit Logging

All write operations are automatically logged to the `audit_logs` table with:

- **Who**: User ID, tenant ID
- **What**: Action, resource, resource ID
- **When**: Timestamp
- **Where**: IP address, user agent
- **Before/After**: State changes for updates

#### Audit Log Retention

Configure retention via environment variable:

```bash
AUDIT_LOG_RETENTION_DAYS=90  # Default: 90 days (0 = forever)
AUDIT_LOG_ENABLED=true       # Default: true
```

### API Versioning

All responses include versioning headers:

| Header | Description |
|--------|-------------|
| `X-API-Version` | Current API version (e.g., `1.0.0`) |
| `X-Deprecation-Warning` | Warning for deprecated endpoints |
| `X-Request-ID` | Unique request identifier for tracing |

### Health Checks

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Basic health check |
| `GET /health/live` | Kubernetes liveness probe |
| `GET /health/ready` | Kubernetes readiness probe (checks DB, Kafka) |
| `GET /version` | API version information |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    TrustVault Platform                       │
├─────────────────────────────────────────────────────────────┤
│  API Gateway (:8080)                                         │
│  ├── Auth (JWT + API Keys)                                   │
│  ├── Tenant Middleware                                       │
│  ├── RBAC Middleware                                         │
│  ├── Prometheus Metrics (/metrics)                          │
│  └── Swagger UI (/api/docs)                                 │
├─────────────────────────────────────────────────────────────┤
│  Core Services                                               │
│  ├── AI Gate (proxy/intercept LLM calls)                    │
│  ├── Classification (GLiNER ONNX)                           │
│  ├── Governance (policy evaluation)                          │
│  ├── Data Fetch (governed retrieval)                        │
│  └── Audit (full lineage tracking)                          │
├─────────────────────────────────────────────────────────────┤
│  Infrastructure                                              │
│  ├── PostgreSQL (app state)                                 │
│  ├── Kafka (event streaming)                                │
│  ├── Qdrant (vector DB)                                     │
│  └── DataHub (metadata backbone)                            │
└─────────────────────────────────────────────────────────────┘
```

## API Endpoints

### Authentication
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/refresh` - Refresh token
- `POST /api/v1/auth/api-keys` - Create API key

### Data Sources
- `GET /api/v1/datasources` - List sources
- `POST /api/v1/datasources` - Create source
- `POST /api/v1/datasources/{id}/scan` - Trigger scan

### AI Gate
- `POST /api/v1/gate/query` - Query with governance
- `POST /api/v1/gate/retrieve` - Retrieve governed context
- `POST /api/v1/gate/validate` - Validate LLM response

### Governance
- `GET /api/v1/governance/policies` - List policies
- `POST /api/v1/governance/policies` - Create policy
- `POST /api/v1/governance/evaluate` - Evaluate data against policies

### Classification
- `POST /api/v1/classify/text` - Classify text
- `POST /api/v1/classify/dataset` - Classify dataset
- `GET /api/v1/classify/results/{dataset_id}` - Get results

### Quality
- `GET /api/v1/quality/datasets/{id}` - Get quality score
- `POST /api/v1/quality/assess` - Trigger assessment

### Privacy
- `POST /api/v1/privacy/dsar` - Create DSAR
- `GET /api/v1/privacy/pia/{dataset_id}` - Get PIA

### Audit
- `GET /api/v1/audit/trail` - Query audit events
- `GET /api/v1/audit/lineage/{dataset_id}` - Get lineage

## Project Structure

```
trustvault/
├── cmd/
│   ├── server/          # Main server binary
│   └── migrate/         # Migration runner
├── internal/
│   ├── api/             # HTTP handlers
│   ├── domain/          # Business logic
│   ├── store/           # Database layer
│   ├── events/          # Event bus
│   ├── external/        # External integrations
│   └── pkg/             # Shared utilities
├── migrations/          # SQL migrations
├── docker-compose.yml
├── Makefile
└── go.mod
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://...` | PostgreSQL connection |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka brokers |
| `DATAHUB_URL` | `http://localhost:8081` | DataHub GMS URL |
| `QDRANT_URL` | `http://localhost:6333` | Qdrant URL |
| `JWT_SECRET` | `change-me` | JWT signing secret |
| `SUPERADMIN_EMAIL` | - | Superadmin email (required) |
| `SUPERADMIN_PASSWORD` | - | Superadmin password (required) |
| `SUPERADMIN_NAME` | `Super Admin` | Superadmin display name |
| `API_VERSION` | `1.0.0` | API version (shown in headers) |
| `AUDIT_LOG_RETENTION_DAYS` | `90` | Days to retain audit logs (0 = forever) |
| `AUDIT_LOG_ENABLED` | `true` | Enable/disable audit logging |
| `LOG_SAMPLE_RATE` | `0.1` | Sampling rate for high-volume endpoints |
| `ENVIRONMENT` | - | Environment name (dev/staging/prod) |
| `BUILD_DATE` | - | Build date (set at compile time) |
| `GIT_COMMIT` | - | Git commit hash (set at compile time) |

## Development

```bash
# Run tests
make test

# Run with coverage
make test-coverage

# Lint
make lint

# Build
make build
```

## License

Proprietary - All rights reserved.
