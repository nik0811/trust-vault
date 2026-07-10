# TrustVault Frontend Specification

> Backend API base: `http://localhost:8080/api/v1/`
> Auth: JWT Bearer token in Authorization header

## Tech Stack

- **Framework:** Next.js 14+ (App Router)
- **UI Library:** shadcn/ui + Tailwind CSS
- **Charts:** Recharts
- **State:** Zustand (lightweight)
- **API Client:** TanStack Query (React Query)
- **Forms:** React Hook Form + Zod validation
- **Tables:** TanStack Table (sorting, filtering, pagination)
- **Auth:** JWT stored in httpOnly cookies
- **Theme:** Dark mode default (enterprise), light mode toggle

## Design Principles

- Clean, minimal, enterprise-grade (think Linear, Vercel Dashboard)
- Generous whitespace, no visual clutter
- Consistent spacing (8px grid system)
- **Page container padding:** ALL pages must use `px-6 py-6` (24px) padding on the main content area. No edge-to-edge content.
- **Card wrapping:** Full-width visualizations (graphs, maps, charts) must be wrapped in Card components with internal padding
- Left sidebar navigation (collapsible)
- Breadcrumbs on every page
- Toast notifications for actions
- Loading skeletons (not spinners)
- Mobile-responsive but desktop-first
- Color coding for severity: Red (critical), Orange (high), Yellow (medium), Blue (info), Green (healthy)

---

## Layout Structure

```
┌──────────────────────────────────────────────────┐
│  Top Bar: Logo | Search | Notifications | User   │
├────────┬─────────────────────────────────────────┤
│        │  Breadcrumb: Home > Data Sources > ...  │
│  Side  │─────────────────────────────────────────│
│  Nav   │                                         │
│        │          Main Content Area              │
│  - Dashboard                                     │
│  - Data Map                                      │
│  - Data Sources                                  │
│  - Classification                                │
│  - Labels                                        │
│  - Governance                                    │
│  - AI Gate                                       │
│  - Lineage                                       │
│  - Quality                                       │
│  - ROT Data                                      │
│  - Privacy                                       │
│  - Advisor                                       │
│  - Audit                                         │
│  - Integrations                                  │
│  - Settings                                      │
│        │                                         │
├────────┴─────────────────────────────────────────┤
│  Status Bar: System Health | Last Sync | Version │
└──────────────────────────────────────────────────┘
```

---

## Pages & Components

### 1. Authentication Pages

#### Login (`/login`)
- Email + password form
- "Remember me" checkbox
- MFA input (conditional, 6-digit code)
- Link to forgot password
- Clean centered card layout, dark background

#### Forgot Password (`/forgot-password`)
- Email input, submit button
- Success message with "check your email"

---

### 2. Dashboard (`/dashboard`)

The main overview page. Shows health at a glance.

**Sections:**
- **Header stats (4 cards):**
  - Total Data Sources (connected/total)
  - Records Classified (with % change vs last week)
  - Active Policies (with violations count)
  - Compliance Score (percentage gauge)

- **Classification Activity Chart:**
  - Area chart showing records classified per day (last 30 days)
  - Toggle: by sensitivity level (PII, PHI, PCI, Public)

- **Recent Alerts (table, 5 rows):**
  - Severity icon | Alert message | Source | Time | Action button

- **Data Quality Overview (mini cards):**
  - Completeness: 95% | Accuracy: 87% | Freshness: 92%

- **AI Gate Usage:**
  - Queries today: 12,450
  - Blocked: 23 (with reason breakdown)
  - Avg latency: 120ms

**API calls:**
```
GET /api/v1/analytics/summary
GET /api/v1/observability/health
GET /api/v1/notifications/events?limit=5
GET /api/v1/quality/trends
```

---

### 3. Data Sources (`/data-sources`)

#### List View (`/data-sources`)
- Table with columns: Name | Type | Status | Last Scan | Records | Health
- Type icons (PostgreSQL, MySQL, S3, Snowflake, BigQuery, File)
- Status badges: Connected (green), Scanning (blue), Error (red), Disconnected (gray)
- Actions: Scan Now, Edit, Delete
- "Add Data Source" button (top right)
- Filter by type, status
- Search by name

#### Add Data Source (`/data-sources/new`)
- Step wizard (3 steps):
  1. Select type (card grid with icons: PostgreSQL, MySQL, S3, Snowflake, BigQuery, Files, Custom)
  2. Connection details form (dynamic per type: host, port, database, credentials)
  3. Test connection + confirm

#### Data Source Detail (`/data-sources/[id]`)
- Connection info card
- Scan history table (date, duration, records found, status)
- Datasets discovered (table: name, schema, records, classifications)
- "Scan Now" button
- Health metrics (last 7 days uptime chart)

**API calls:**
```
GET /api/v1/datasources
POST /api/v1/datasources
GET /api/v1/datasources/{id}
POST /api/v1/datasources/{id}/scan
GET /api/v1/datasources/{id}/status
```

---

### 4. Classification (`/classification`)

#### Overview (`/classification`)
- **Stats bar:** Total classified | PII found | PHI found | PCI found | Unclassified
- **Classification distribution pie chart** (by category)
- **Recent classifications table:** Dataset | Source | Entities Found | Confidence | Date
- **Model status card:** Model version, accuracy, last updated

#### Dataset Classification Detail (`/classification/[dataset_id]`)
- Dataset metadata (source, schema, record count)
- **Entity breakdown table:** Entity Type | Count | Confidence Avg | Samples
- Click entity type → shows sample detections with highlighted text
- Reclassify button
- Override classification (for governance admins)

#### Rules (`/classification/rules`)
- List of classification rules (table)
- Create rule form:
  - IF condition (classification type, confidence threshold, source)
  - THEN action (label, restrict, alert, redact)
- Rule priority ordering (drag & drop)
- Enable/disable toggle per rule

#### Models (`/classification/models`)
- Installed models list (GLiNER edge, GLiNER base, custom)
- Model performance metrics (accuracy, F1, latency)
- Fine-tune trigger button (upload training data)
- Model version history

**API calls:**
```
GET /api/v1/classify/results/{dataset_id}
POST /api/v1/classify/text
POST /api/v1/classify/dataset
POST /api/v1/classify/rules
GET /api/v1/classify/models
```

---

### 5. Governance (`/governance`)

#### Policies List (`/governance/policies`)
- Table: Policy Name | Type | Status | Violations | Created | Actions
- Filter by: type (access, redaction, AI, retention), status (active, draft, disabled)
- "Create Policy" button

#### Create/Edit Policy (`/governance/policies/new`)
- Form with sections:
  - **Name & Description**
  - **Conditions** (visual builder):
    - Data classification = [dropdown multi-select]
    - Source = [dropdown]
    - Destination type = [dropdown: internal_llm, external_llm, export, api]
    - User role = [dropdown]
  - **Action:** Allow | Deny | Redact | Alert
  - **Redaction strategy** (if redact): Mask | Hash | Remove | Replace
  - **Regulations:** Tag with GDPR, CCPA, HIPAA, etc. (multi-select chips)
- Preview panel showing policy in JSON

#### Policy Evaluation Console (`/governance/evaluate`)
- Test a policy against sample data
- Input: paste text or select dataset
- Output: shows what would be allowed/denied/redacted
- Useful for policy debugging before activation

**API calls:**
```
GET /api/v1/governance/policies
POST /api/v1/governance/policies
POST /api/v1/governance/evaluate
```

---

### 6. AI Gate (`/ai-gate`)

#### Overview (`/ai-gate`)
- **Live stats:** Queries/min | Blocked/min | Avg latency | Active connections
- **Real-time activity feed** (last 20 queries with: user, query preview, decision, latency)
- **Decision breakdown chart:** Allowed vs Redacted vs Blocked (donut)

#### Query Explorer (`/ai-gate/queries`)
- Searchable table of all gate queries
- Columns: Timestamp | User | Query (truncated) | Decision | Redactions | Latency | LLM
- Click row → full detail view:
  - Original query
  - Context chunks retrieved (with redactions highlighted)
  - Final prompt sent to LLM
  - LLM response
  - Governance decisions applied (which policies triggered)
  - Full audit trail

#### Configuration (`/ai-gate/config`)
- LLM endpoint configuration (URL, API key, model name)
- Vector DB configuration (Qdrant URL, collection, or external)
- Default governance mode: Strict | Moderate | Permissive
- Rate limits per user/role
- Allowed entity types in context

**API calls:**
```
GET /api/v1/gate/stats
GET /api/v1/gate/queries
POST /api/v1/gate/query (for testing)
```

---

### 7. Data Quality (`/quality`)

#### Dashboard (`/quality`)
- **Overall quality score gauge** (0-100)
- **5 dimension cards:** Completeness | Accuracy | Consistency | Timeliness | Uniqueness
  - Each with score, trend arrow, mini sparkline
- **Worst performing datasets table** (sorted by quality score ascending)
- **Quality trends line chart** (last 30 days, all 5 dimensions)

#### Dataset Quality (`/quality/[dataset_id]`)
- Full quality report for single dataset
- Dimension breakdown with details
- Issues list (sortable): Type | Column | Severity | Count | Suggested Fix
- "Run Assessment" button
- Historical quality chart for this dataset

#### Thresholds (`/quality/thresholds`)
- Set alert thresholds per dimension
- Form: dimension, minimum score, alert severity

**API calls:**
```
GET /api/v1/quality/datasets/{id}
GET /api/v1/quality/datasets/{id}/issues
GET /api/v1/quality/trends
POST /api/v1/quality/thresholds
```

---

### 8. Privacy & Compliance (`/privacy`)

#### Overview (`/privacy`)
- Compliance posture cards: GDPR | CCPA | HIPAA | DPDP (each with status percentage)
- Active DSARs count with deadline warnings
- Retention violations count
- Recent consent changes

#### DSAR Management (`/privacy/dsar`)
- Table of all DSARs: Subject | Type | Status | Deadline | Submitted | Assigned
- Status badges: New | In Progress | Completed | Overdue
- Create DSAR form (subject identifier, type: access/delete/rectify)
- DSAR detail view: progress steps, data found, download package button

#### Privacy Impact Assessments (`/privacy/pia`)
- List of PIAs by dataset
- Auto-generated risk scores
- PIA detail: risk matrix, recommendations, linked policies

#### Records of Processing (`/privacy/ropa`)
- Table: Activity | Legal Basis | Data Categories | Retention | Cross-border
- Add/edit processing activity form

#### Consent Management (`/privacy/consent`)
- Search by subject
- Consent status per purpose
- Record/withdraw consent actions

#### Retention (`/privacy/retention`)
- Retention policies list
- Violations table (data past retention)
- Execute retention actions (archive/delete)

**API calls:**
```
GET /api/v1/privacy/dsar
POST /api/v1/privacy/dsar
GET /api/v1/privacy/pia/{dataset_id}
GET /api/v1/privacy/ropa
GET /api/v1/privacy/retention/violations
```

---

### 9. Audit Trail (`/audit`)

#### Audit Log (`/audit`)
- Full-text searchable event log
- Filters: date range, user, action type, resource, tenant
- Columns: Timestamp | User | Action | Resource | Details | IP
- Export to CSV/PDF
- Immutable -- no delete buttons

#### AI Usage Audit (`/audit/ai-usage`)
- What data went to AI, when, for whom
- Columns: Timestamp | Dataset | User | LLM | Purpose | Governance Decision
- Filter by dataset, user, time range

#### Compliance Reports (`/audit/reports`)
- Generate report button (select type: GDPR, CCPA, full audit, AI usage)
- List of generated reports with download links
- Schedule recurring reports

**API calls:**
```
GET /api/v1/audit/trail
GET /api/v1/audit/datasets/{id}/ai-usage
GET /api/v1/audit/compliance-report
POST /api/v1/reports/generate
```

---

### 10. Observability (`/observability`)

#### System Health (`/observability`)
- Service status cards (gateway, workers, classifier, ingestion, Kafka, DB)
- Green/Red indicators per service
- Kafka consumer lag chart
- Processing throughput (records/sec) real-time chart
- Error rate chart

#### Data Source Health (`/observability/sources`)
- Per-source health: last update, volume trend, schema changes
- Alert on stale sources

#### Alerts (`/observability/alerts`)
- Active alerts table
- Alert history
- Configure alert rules (threshold, severity, channel)

**API calls:**
```
GET /api/v1/observability/health
GET /api/v1/observability/metrics
GET /api/v1/observability/alerts
```

---

### 11. AI Governance (`/ai-governance`)

#### Policies (`/ai-governance/policies`)
- AI-specific policies (what data can go to AI)
- Create policy: eligible classifications, allowed models, training vs inference

#### Eligibility Checker (`/ai-governance/check`)
- Select dataset → shows AI eligibility status
- Reasons for ineligibility (which policy blocks it)

#### Model Registry (`/ai-governance/models`)
- Registered AI models
- Data lineage per model (what trained it)
- Auto-generated model cards

**API calls:**
```
GET /api/v1/ai-governance/policies
POST /api/v1/ai-governance/evaluate
GET /api/v1/ai-governance/lineage/{model_id}
```

---

### 12. Documents (`/documents`)

#### Upload & Process (`/documents`)
- Drag-and-drop upload zone (PDF, XLSX, CSV, images, DOCX)
- Processing queue with status per file
- Extracted content preview (text + detected entities highlighted)

#### Review Queue (`/documents/review`)
- Low-confidence classifications needing human verification
- Side-by-side: original document (rendered) + detected entities
- Approve/reject/reclassify actions

**API calls:**
```
POST /api/v1/documents/extract
POST /api/v1/documents/classify
GET /api/v1/documents/review-queue
```

---

### 13. Notifications (`/notifications`)

- Notification center (bell icon in top bar → dropdown)
- Full notifications page with all history
- Mark read/unread
- Configure notification preferences per user

---

### 14. Jobs (`/jobs`)

- Scheduled jobs list (cron expression, next run, last run, status)
- Job history with logs
- Create/edit/delete jobs
- Manual trigger button

**API calls:**
```
GET /api/v1/jobs
POST /api/v1/jobs
POST /api/v1/jobs/{id}/run-now
```

---

### 15. Settings (`/settings`)

#### Tenant Settings (`/settings/general`)
- Tenant name, logo upload
- Default governance mode
- Timezone, language

#### Users & Roles (`/settings/users`)
- User list table: Name | Email | Role | Status | Last Login
- Invite user form (email + role selection)
- Edit user roles
- Deactivate user

#### Roles (`/settings/roles`)
- Built-in roles (view only)
- Custom roles: create with permission checkboxes (grouped by resource)

#### API Keys (`/settings/api-keys`)
- List of active keys (masked)
- Create key: name, permissions, expiry
- Revoke key

#### Webhooks (`/settings/webhooks`)
- Registered webhooks list
- Create: URL, events to listen for, secret
- Test webhook button

#### Integrations (`/settings/integrations`)
- LLM endpoint config
- Vector DB config
- DataHub connection status

**API calls:**
```
GET /api/v1/users
POST /api/v1/users
GET /api/v1/roles
POST /api/v1/auth/api-keys
POST /api/v1/notifications/webhooks
```

---

### 16. Data Lineage (`/lineage`)

TrustVault owns the lineage visualization -- users never see DataHub directly.

**Layout note:** The lineage graph must respect the standard page container padding (`px-6 py-6` or 24px) like all other pages. The graph canvas should be contained within a card component with proper margins, not edge-to-edge.

#### Lineage Explorer (`/lineage`)
- **Page header:** "Data Lineage" title + subtitle (same spacing as other pages)
- **Graph container:** Wrapped in a Card component with `p-4` internal padding
- Interactive graph visualization (React Flow or D3)
- Nodes = datasets, jobs, AI models, LLM endpoints
- Edges = data flow direction with labels (scanned, classified, redacted, consumed)
- Click node → shows metadata, classification, policies applied
- Filter by: source, classification type, time range, AI usage
- Search: "Show me everything connected to this table"
- **Legend bar:** Below graph with node type indicators (Source, Transform, Output, AI) - should have `mt-4` margin

#### Dataset Lineage (`/lineage/[dataset_id]`)
- Focused view: upstream (where data came from) + downstream (where it went)
- Timeline slider: see lineage at any point in time
- Highlight path to AI: colored path showing how data reached an LLM
- Impact analysis panel: "If I delete this, what breaks?"

#### AI Provenance (`/lineage/ai-provenance`)
- Specialized view: trace data from source → classification → AI consumption
- For EU AI Act compliance: prove what data informed which AI decisions
- Exportable as compliance evidence (PDF/JSON)

**API calls:**
```
GET /api/v1/audit/lineage/{dataset_id}
GET /api/v1/audit/lineage/{dataset_id}/upstream
GET /api/v1/audit/lineage/{dataset_id}/downstream
GET /api/v1/audit/lineage/{dataset_id}/ai-path
```

---

### 17. Data Map (`/data-map`)

Visual answer to "Where is ALL your data?" — the foundation of governance.

#### Data Map Overview (`/data-map`)
- **Interactive graph view:** Sources as nodes, data flows as edges
- Color-coded by sensitivity label (green=Public, yellow=Internal, orange=Confidential, red=Restricted)
- Click source → expand to show datasets within
- Filter by: source type, sensitivity, geography, owner
- Search: find any dataset across the entire estate

#### Geographic View (`/data-map/geography`)
- World map showing data center locations
- Data volume per region (bubble size)
- Cross-border flow indicators (for GDPR transfer compliance)
- Click region → shows sources and datasets in that location

#### Coverage Dashboard (`/data-map/coverage`)
- **Classification coverage:** % of data estate that's been classified
- **Dark data:** Datasets that exist but aren't governed (ungoverned sources)
- **Shadow IT:** Data in unapproved locations
- Progress bars per source showing scan/classification completion

**API calls:**
```
GET /api/v1/datamap
GET /api/v1/datamap/sources
GET /api/v1/datamap/flows
GET /api/v1/datamap/coverage
GET /api/v1/datamap/geography
GET /api/v1/datamap/dark-data
```

---

### 18. Sensitivity Labels (`/labels`)

Microsoft Information Protection-style labels auto-assigned from classifications.

#### Labels Overview (`/labels`)
- **Distribution chart:** Pie/bar showing data by label (Public, Internal, Confidential, Restricted)
- **Recent label changes:** Table of datasets whose labels changed recently
- **Label coverage:** % of datasets with assigned labels

#### Label Rules (`/labels/rules`)
- Configure classification → label mapping
- Visual rule builder: "IF contains PII AND source = HR THEN label = CONFIDENTIAL"
- Priority ordering (drag & drop)
- Test rule against sample data

#### Dataset Labels (`/labels/datasets`)
- Searchable table of all datasets with their labels
- Bulk label override (with approval workflow)
- Label history per dataset

**API calls:**
```
GET /api/v1/labels/summary
GET /api/v1/labels/datasets/{id}
POST /api/v1/labels/assign
GET /api/v1/labels/rules
POST /api/v1/labels/rules
```

---

### 19. ROT Data (`/rot`)

Identify and remediate Redundant, Obsolete, and Trivial data.

#### ROT Dashboard (`/rot`)
- **ROT summary cards:** Total ROT volume, % of estate, estimated storage cost
- **ROT breakdown:** Pie chart (Redundant vs Obsolete vs Trivial)
- **Top ROT sources:** Which sources have the most ROT
- **Trend chart:** ROT volume over time (is it growing or shrinking?)

#### Duplicates (`/rot/duplicates`)
- Table of duplicate/near-duplicate datasets
- Side-by-side comparison view
- Actions: Keep one, archive others, merge

#### Obsolete Data (`/rot/obsolete`)
- Data past retention period
- Data not accessed in X months
- Stale data (source hasn't updated)
- Actions: Archive, delete, extend retention

#### Trivial Data (`/rot/trivial`)
- Temp files, logs, test data, empty records
- Low-value data flagged by classification
- Actions: Delete, exclude from governance

**API calls:**
```
GET /api/v1/rot/summary
GET /api/v1/rot/datasets
GET /api/v1/rot/duplicates
POST /api/v1/rot/scan
POST /api/v1/rot/remediate
```

---

### 20. Compliance Advisor (`/advisor`)

AI-powered compliance recommendations — tells you what to DO, not just what's wrong.

#### Recommendations (`/advisor`)
- **Prioritized action list:** Top 10 things to fix, sorted by risk impact
- Each recommendation shows: issue, affected datasets, regulation, suggested action
- "Fix it" button → navigates to relevant page or triggers remediation
- Dismiss/snooze options with reason

#### Gap Analysis (`/advisor/gaps`)
- **Compliance gaps by regulation:** GDPR, CCPA, HIPAA, DPDP tabs
- Each gap shows: requirement, current state, what's missing
- Progress bars showing compliance percentage per regulation

#### Defense Docket (`/advisor/defense-docket`)
- Generate auditor-ready evidence package
- Select regulations to include
- Select date range
- Preview contents before generation
- Download as PDF/ZIP

#### Playbooks (`/advisor/playbooks`)
- Step-by-step remediation guides
- Filter by issue type (retention, consent, access control, etc.)
- Checklist format with completion tracking

**API calls:**
```
GET /api/v1/advisor/recommendations
GET /api/v1/advisor/gaps
POST /api/v1/advisor/defense-docket
GET /api/v1/advisor/playbook/{issue_type}
GET /api/v1/advisor/risk-score
```

---

### 21. Feedback & Learning (`/feedback`)

User corrections that make classifications smarter over time.

#### Feedback Dashboard (`/feedback`)
- **Stats:** Total corrections, accuracy improvement %, knowledge cache size
- **Recent corrections:** Table of user-submitted corrections
- **Correction trends:** Are corrections increasing or decreasing? (model improving)

#### Submit Correction (`/feedback/correct`)
- Shown inline on classification results (thumbs up/down + correct label dropdown)
- Also accessible as standalone form for bulk corrections

#### Custom Entities (`/feedback/entities`)
- Define tenant-specific entity types
- Provide examples for the model to learn
- View custom entity detection accuracy

#### Knowledge Cache (`/feedback/cache`)
- View cached classifications (instant lookups)
- Clear cache entries if needed
- Cache hit rate metrics

**API calls:**
```
POST /api/v1/feedback/correction
POST /api/v1/feedback/confirmation
GET /api/v1/feedback/stats
POST /api/v1/feedback/custom-entity
GET /api/v1/feedback/knowledge-cache
```

---

### 22. Integrations (`/integrations`)

Push TrustVault data to external enterprise tools.

#### Integrations List (`/integrations`)
- Configured integrations with status (connected, error, disabled)
- "Add Integration" button

#### Add Integration (`/integrations/new`)
- Select type: DLP, Privacy Platform, Data Catalog, SIEM, Ticketing, Communication
- Select specific tool (Microsoft Purview, OneTrust, Splunk, Jira, Slack, etc.)
- Configure connection (API keys, URLs, auth)
- Select what to sync (classifications, labels, violations, alerts)
- Set sync frequency (real-time, hourly, daily)

#### Integration Detail (`/integrations/[id]`)
- Connection status and health
- Sync history (last sync, records synced, errors)
- Manual sync trigger
- Edit configuration
- View logs

**API calls:**
```
POST /api/v1/integrations
GET /api/v1/integrations
POST /api/v1/integrations/{id}/test
POST /api/v1/integrations/{id}/sync
GET /api/v1/integrations/{id}/logs
```

---

### 17. Super Admin Panel (`/admin` - internal only)

Only visible to super_admin users. Separate section.

- **Tenants:** List all tenants, create/suspend/delete
- **Platform health:** Cross-tenant metrics
- **Impersonate:** Select tenant + user → get their view
- **Global config:** Default models, platform-wide settings
- **Billing/Usage:** Per-tenant resource consumption

**API calls (internal port :8099):**
```
GET /internal/v1/tenants
POST /internal/v1/tenants
POST /internal/v1/tenants/{id}/impersonate
```

---

## Component Library Requirements

### Reusable Components Needed:

1. **DataTable** - sortable, filterable, paginated, with row actions
2. **StatCard** - number + label + trend arrow + sparkline
3. **StatusBadge** - colored pill (green/red/yellow/blue/gray)
4. **PolicyBuilder** - visual condition builder (IF/THEN blocks)
5. **EntityHighlighter** - text with colored entity spans highlighted
6. **TimelineView** - vertical timeline for audit trails
7. **GaugeChart** - circular percentage indicator
8. **ConfidenceBar** - horizontal bar with confidence percentage
9. **FileUploadZone** - drag-and-drop with format icons
10. **SearchWithFilters** - search bar + filter chips
11. **Breadcrumbs** - auto-generated from route
12. **EmptyState** - illustration + message + action button
13. **ConfirmModal** - destructive action confirmation
14. **ToastNotification** - success/error/warning/info

---

## API Integration Pattern

All pages should use this pattern:

```typescript
// TanStack Query for data fetching
const { data, isLoading, error } = useQuery({
  queryKey: ['datasources', filters],
  queryFn: () => api.get('/datasources', { params: filters }),
});

// Mutations for actions
const createMutation = useMutation({
  mutationFn: (data) => api.post('/datasources', data),
  onSuccess: () => {
    queryClient.invalidateQueries(['datasources']);
    toast.success('Data source created');
  },
});
```

**Auth header injected globally:**
```typescript
api.interceptors.request.use((config) => {
  const token = getAccessToken();
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});
```

**Tenant context:** Backend handles tenant scoping via JWT. Frontend doesn't need to pass tenant_id -- it's implicit from the auth token.

---

## Color System (2026 Modern - OKLCH-aware, Dark-First, Theme-Aware)

Design direction: **Cool Mineral** — slate base, ice-cyan accent, warm copper highlight.
Dark-first. Elevated neutrals (no pure white/black). Lumen glow effects on active elements.

### Theme-Aware Architecture

The entire color system is built on **CSS custom properties mapped to semantic tokens**, not hardcoded values. This enables:

1. **System-aware:** Auto-detects OS preference (`prefers-color-scheme`)
2. **User override:** User can force dark/light/system in settings
3. **Tenant branding:** Tenant admins can override primary/accent colors
4. **Scheduled themes:** Optional time-based switching (e.g., dark after 6pm)
5. **High contrast mode:** Respects `prefers-contrast: more`
6. **Reduced motion:** Disables lumen glow animations when `prefers-reduced-motion`

```typescript
// Theme context structure
type ThemeMode = 'system' | 'dark' | 'light' | 'high-contrast';

interface ThemeConfig {
  mode: ThemeMode;                    // User preference
  resolved: 'dark' | 'light';         // Actual applied theme
  brand: {                             // Tenant-customizable
    primary: string;
    accent: string;
    logo: string;
  };
  radius: 'none' | 'sm' | 'md' | 'lg'; // Border radius preference
  density: 'compact' | 'default' | 'comfortable'; // UI density
}
```

### CSS Token System (Tailwind + CSS Variables)

```css
/* Root tokens — resolved at runtime based on theme mode */
:root {
  /* System switches automatically based on prefers-color-scheme */
  --color-scheme: light dark;
}

[data-theme="dark"] {
  --bg:             oklch(12% 0.02 250);      /* #0C1222 equivalent */
  --bg-surface:     oklch(16% 0.025 250);     /* #141D2E */
  --bg-raised:      oklch(20% 0.03 250);      /* #1C2740 */
  --bg-hover:       oklch(22% 0.03 250);
  --border:         oklch(28% 0.03 250);      /* #2A3A52 */
  --border-subtle:  oklch(22% 0.02 250);

  --text:           oklch(92% 0.01 250);      /* #E8EDF5 */
  --text-secondary: oklch(65% 0.03 250);      /* #8899B0 */
  --text-muted:     oklch(45% 0.03 250);      /* #4A5B73 */

  --primary:        oklch(72% 0.15 195);      /* Cyan 400 */
  --primary-hover:  oklch(78% 0.14 195);      /* Cyan 300 - lumen */
  --primary-muted:  oklch(72% 0.15 195 / 0.12);

  --accent:         oklch(60% 0.12 45);       /* Copper */
  --highlight:      oklch(68% 0.15 290);      /* Violet */

  --success:        oklch(72% 0.17 165);      /* Emerald */
  --warning:        oklch(78% 0.15 85);       /* Amber */
  --danger:         oklch(68% 0.18 25);       /* Red */
  --info:           oklch(82% 0.1 195);       /* Cyan light */

  --glow:           oklch(72% 0.15 195 / 0.2);
  --focus-ring:     oklch(72% 0.15 195 / 0.4);
  --selection:      oklch(72% 0.15 195 / 0.1);
}

[data-theme="light"] {
  --bg:             oklch(97% 0.005 250);     /* #F5F7FA */
  --bg-surface:     oklch(94% 0.008 250);     /* #EBEEF3 */
  --bg-raised:      oklch(100% 0 0);          /* White for cards */
  --bg-hover:       oklch(92% 0.01 250);
  --border:         oklch(85% 0.01 250);      /* #D1D9E6 */
  --border-subtle:  oklch(90% 0.005 250);

  --text:           oklch(18% 0.02 250);      /* #1A2332 */
  --text-secondary: oklch(45% 0.02 250);      /* #5C6B7F */
  --text-muted:     oklch(62% 0.02 250);      /* #94A3B8 */

  --primary:        oklch(55% 0.18 195);      /* Darker cyan for light bg */
  --primary-hover:  oklch(48% 0.18 195);
  --primary-muted:  oklch(55% 0.18 195 / 0.08);

  --accent:         oklch(52% 0.14 45);
  --highlight:      oklch(55% 0.18 290);

  --success:        oklch(55% 0.2 165);
  --warning:        oklch(60% 0.18 85);
  --danger:         oklch(55% 0.22 25);
  --info:           oklch(55% 0.15 195);

  --glow:           oklch(55% 0.18 195 / 0.1);
  --focus-ring:     oklch(55% 0.18 195 / 0.3);
  --selection:      oklch(55% 0.18 195 / 0.06);
}

[data-theme="high-contrast"] {
  --bg:             oklch(5% 0 0);
  --text:           oklch(98% 0 0);
  --border:         oklch(60% 0 0);
  --primary:        oklch(80% 0.2 195);
  /* All other tokens boosted for 7:1+ contrast */
}
```

### Tailwind Config Integration

```typescript
// tailwind.config.ts
export default {
  theme: {
    extend: {
      colors: {
        bg: 'var(--bg)',
        surface: 'var(--bg-surface)',
        raised: 'var(--bg-raised)',
        border: 'var(--border)',
        primary: 'var(--primary)',
        accent: 'var(--accent)',
        success: 'var(--success)',
        warning: 'var(--warning)',
        danger: 'var(--danger)',
      },
      boxShadow: {
        glow: '0 0 20px var(--glow)',
        'glow-sm': '0 0 10px var(--glow)',
      },
    },
  },
};
```

### Theme Provider Component

```tsx
// Usage in app layout
<ThemeProvider
  defaultMode="system"          // Respect OS preference
  storageKey="trustvault-theme" // Persist in localStorage
  tenantBrand={tenant.brand}    // Override from tenant settings
>
  <App />
</ThemeProvider>
```

### Theme Switching UI

- Toggle in top-right user menu: System / Dark / Light icons
- Settings page: full theme customization (mode, density, radius)
- Tenant admin: upload logo, set primary/accent color (brand kit)
- Transition: smooth 200ms color transitions on theme switch

### Tenant Brand Override (White-Label Support)

Tenant admins can customize via Settings:
```json
{
  "brand": {
    "primary": "oklch(65% 0.2 280)",   // Custom purple
    "accent": "oklch(70% 0.18 140)",   // Custom green
    "logo_url": "/uploads/tenant-logo.svg",
    "favicon_url": "/uploads/favicon.ico",
    "app_name": "Acme DataGuard"       // Replaces "TrustVault" in UI
  }
}
```

This allows enterprises to deploy TrustVault as their own branded product.

### Resolved Color Values (Fallback Reference)

For elements that need hex fallbacks (emails, PDFs, external):

```
── Dark Mode Resolved ──────────────────────────────────────
Primary:        #06B6D4  (Cyan)
Primary Hover:  #22D3EE  (Cyan glow)
Accent:         #D97706  (Copper)
Highlight:      #A78BFA  (Violet)

Success:        #34D399
Warning:        #FBBF24
Danger:         #F87171
Info:           #67E8F9

Background:     #0C1222
Surface:        #141D2E
Raised:         #1C2740
Border:         #2A3A52
Text:           #E8EDF5
Text Secondary: #8899B0

── Light Mode Resolved ─────────────────────────────────────
Primary:        #0891B2
Primary Hover:  #0E7490
Accent:         #B45309
Highlight:      #7C3AED

Success:        #059669
Warning:        #D97706
Danger:         #DC2626
Info:           #0891B2

Background:     #F5F7FA
Surface:        #EBEEF3
Raised:         #FFFFFF
Border:         #D1D9E6
Text:           #1A2332
Text Secondary: #5C6B7F
```

**Usage rules:**
- Primary cyan for navigation highlights, active states, primary buttons
- Copper/amber reserved for high-priority CTAs (max 2 per page) and warnings
- Violet at 3-5% frequency only (tags, badges, chart accents)
- Never use hardcoded hex in components — always reference CSS variables
- Lumen glow (`box-shadow: var(--glow)`) on hover states in dark mode
- Status colors always paired with icon + text (never color alone)
- All themes must pass WCAG 2.1 AA (4.5:1 text, 3:1 non-text)
- Animations disabled when `prefers-reduced-motion: reduce`

---

## Route Map

```
/login
/forgot-password
/dashboard
/data-map
/data-map/geography
/data-map/coverage
/data-sources
/data-sources/new
/data-sources/[id]
/classification
/classification/[dataset_id]
/classification/rules
/classification/models
/labels
/labels/rules
/labels/datasets
/governance/policies
/governance/policies/new
/governance/policies/[id]
/governance/evaluate
/ai-gate
/ai-gate/queries
/ai-gate/queries/[id]
/ai-gate/config
/quality
/quality/[dataset_id]
/quality/thresholds
/rot
/rot/duplicates
/rot/obsolete
/rot/trivial
/privacy
/privacy/dsar
/privacy/dsar/[id]
/privacy/pia
/privacy/ropa
/privacy/consent
/privacy/retention
/advisor
/advisor/gaps
/advisor/defense-docket
/advisor/playbooks
/audit
/audit/ai-usage
/audit/reports
/observability
/observability/sources
/observability/alerts
/ai-governance
/ai-governance/policies
/ai-governance/check
/ai-governance/models
/lineage
/lineage/[dataset_id]
/lineage/ai-provenance
/feedback
/feedback/entities
/feedback/cache
/integrations
/integrations/new
/integrations/[id]
/documents
/documents/review
/notifications
/jobs
/settings/general
/settings/users
/settings/roles
/settings/api-keys
/settings/webhooks
/settings/integrations
/admin (super admin only)
```

---

## Key UX Flows

### Flow 1: Connect Data Source → Scan → View Classifications

1. User clicks "Add Data Source"
2. Selects type (PostgreSQL)
3. Fills connection form
4. Clicks "Test Connection" → green checkmark
5. Confirms → source appears in list
6. Clicks "Scan Now" → progress indicator
7. Scan completes → sees datasets discovered
8. Clicks dataset → sees classification results with highlighted entities

### Flow 2: Create Governance Policy → Test → Activate

1. User navigates to Governance → Create Policy
2. Builds conditions visually (IF PII detected AND destination = external_llm)
3. Sets action (Redact, mask strategy)
4. Tags with regulation (GDPR)
5. Clicks "Test Policy" → shows evaluation results on sample data
6. Satisfied → clicks "Activate"
7. Policy now enforced on all AI Gate queries

### Flow 3: AI Gate Query (Developer using SDK)

1. Developer configures LLM endpoint in Settings
2. Sends query via API: `POST /api/v1/gate/query`
3. TrustVault retrieves context, classifies, redacts PII
4. Response returned with clean context
5. In UI: admin sees the query in AI Gate → Query Explorer
6. Can drill into full audit trail of what was redacted and why

---

## Responsive Breakpoints

- Desktop: 1280px+ (full sidebar + content)
- Tablet: 768px-1279px (collapsed sidebar icons)
- Mobile: <768px (hamburger menu, stacked layout)

---

## Accessibility

- All interactive elements keyboard-navigable
- ARIA labels on icons and status indicators
- Color is never the only indicator (always paired with text/icon)
- Minimum contrast ratio 4.5:1
- Focus visible outlines on all interactive elements
