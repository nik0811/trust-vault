package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"github.com/securelens/securelens/internal/external"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

type Server struct {
	router         *chi.Mux
	internalRouter *chi.Mux
	db             *store.DB
	kafka          *external.Kafka
	datahub        *external.DataHub
	qdrant         *external.Qdrant
	httpServer     *http.Server
	internalServer *http.Server
	startTime      *time.Time
	classifier     any // GLiNER classifier service reference

	// Repositories
	tenants             *store.Repository[store.Tenant]
	users               *store.Repository[store.User]
	roles               *store.Repository[store.Role]
	datasources         *store.Repository[store.DataSource]
	policies            *store.Repository[store.Policy]
	classifications     *store.Repository[store.Classification]
	auditLogs           *store.Repository[store.AuditLog]
	gateQueries         *store.Repository[store.GateQuery]
	qualityScores       *store.Repository[store.QualityScore]
	dsars               *store.Repository[store.DSAR]
	jobs                *store.Repository[store.Job]
	notifications       *store.Repository[store.Notification]
	webhooks            *store.Repository[store.Webhook]
	labels              *store.Repository[store.Label]
	feedback            *store.Repository[store.Feedback]
	integrations        *store.Repository[store.Integration]
	rotData             *store.Repository[store.ROTData]
	remediationActions  *store.Repository[store.RemediationAction]
	reports             *store.Repository[store.Report]
	labelRules          *store.Repository[store.LabelRule]
	ropa                *store.Repository[store.RoPA]
	playbooks           *store.Repository[store.Playbook]
	modelLineage        *store.Repository[store.ModelLineage]
	integrationLogs     *store.Repository[store.IntegrationLog]
	dataFlows           *store.Repository[store.DataFlow]
	duplicateGroups     *store.Repository[store.DuplicateGroup]
	reviewQueue         *store.Repository[store.ReviewQueueItem]
	retentionPolicies   *store.Repository[store.RetentionPolicy]
	retentionViolations *store.Repository[store.RetentionViolation]
	scanLogs            *store.Repository[store.ScanLog]
	classificationRules      *store.Repository[store.ClassificationRule]
	dpias                    *store.Repository[store.DPIA]
	consentRecords           *store.Repository[store.ConsentRecord]
	criticalDataElements     *store.Repository[store.CriticalDataElement]
	dataProfiles             *store.Repository[store.DataProfile]
	documentClassifications  *store.Repository[store.DocumentClassification]
	endpointAgents           *store.Repository[store.EndpointAgent]
	residencyRules           *store.Repository[store.ResidencyRule]
	endpointScans            *store.Repository[store.EndpointScan]
	consentPreferences       *store.Repository[store.ConsentPreference]
	complianceAssessments    *store.Repository[store.ComplianceAssessment]
}

func NewServer(db *store.DB, kafka *external.Kafka) *Server {
	now := time.Now()
	s := &Server{
		router:         chi.NewRouter(),
		internalRouter: chi.NewRouter(),
		db:             db,
		kafka:          kafka,
		datahub:        external.NewDataHub(""),
		qdrant:         external.NewQdrant("", ""),
		startTime:      &now,
		classifier:     true, // Classifier is available via HTTP service

		tenants:             store.NewRepo[store.Tenant](db, "tenants"),
		users:               store.NewRepo[store.User](db, "users"),
		roles:               store.NewRepo[store.Role](db, "roles"),
		datasources:         store.NewRepo[store.DataSource](db, "datasources"),
		policies:            store.NewRepo[store.Policy](db, "policies"),
		classifications:     store.NewRepo[store.Classification](db, "classifications"),
		auditLogs:           store.NewRepo[store.AuditLog](db, "audit_logs"),
		gateQueries:         store.NewRepo[store.GateQuery](db, "gate_queries"),
		qualityScores:       store.NewRepo[store.QualityScore](db, "quality_scores"),
		dsars:               store.NewRepo[store.DSAR](db, "dsars"),
		jobs:                store.NewRepo[store.Job](db, "jobs"),
		notifications:       store.NewRepo[store.Notification](db, "notifications"),
		webhooks:            store.NewRepo[store.Webhook](db, "webhooks"),
		labels:              store.NewRepo[store.Label](db, "labels"),
		feedback:            store.NewRepo[store.Feedback](db, "feedback"),
		integrations:        store.NewRepo[store.Integration](db, "integrations"),
		rotData:             store.NewRepo[store.ROTData](db, "rot_data"),
		remediationActions:  store.NewRepo[store.RemediationAction](db, "remediation_actions"),
		reports:             store.NewRepo[store.Report](db, "reports"),
		labelRules:          store.NewRepo[store.LabelRule](db, "label_rules"),
		ropa:                store.NewRepo[store.RoPA](db, "ropa"),
		playbooks:           store.NewRepo[store.Playbook](db, "playbooks"),
		modelLineage:        store.NewRepo[store.ModelLineage](db, "model_lineage"),
		integrationLogs:     store.NewRepo[store.IntegrationLog](db, "integration_logs"),
		dataFlows:           store.NewRepo[store.DataFlow](db, "data_flows"),
		duplicateGroups:     store.NewRepo[store.DuplicateGroup](db, "duplicate_groups"),
		reviewQueue:         store.NewRepo[store.ReviewQueueItem](db, "review_queue"),
		retentionPolicies:   store.NewRepo[store.RetentionPolicy](db, "retention_policies"),
		retentionViolations: store.NewRepo[store.RetentionViolation](db, "retention_violations"),
		scanLogs:            store.NewRepo[store.ScanLog](db, "scan_logs"),
		classificationRules:      store.NewRepo[store.ClassificationRule](db, "classification_rules"),
		dpias:                    store.NewRepo[store.DPIA](db, "dpias"),
		consentRecords:           store.NewRepo[store.ConsentRecord](db, "consent_records"),
		criticalDataElements:     store.NewRepo[store.CriticalDataElement](db, "critical_data_elements"),
		dataProfiles:             store.NewRepo[store.DataProfile](db, "data_profiles"),
		documentClassifications:  store.NewRepo[store.DocumentClassification](db, "document_classifications"),
		endpointAgents:           store.NewRepo[store.EndpointAgent](db, "endpoint_agents"),
		residencyRules:           store.NewRepo[store.ResidencyRule](db, "residency_rules"),
		endpointScans:            store.NewRepo[store.EndpointScan](db, "endpoint_scans"),
		consentPreferences:       store.NewRepo[store.ConsentPreference](db, "consent_preferences"),
		complianceAssessments:    store.NewRepo[store.ComplianceAssessment](db, "compliance_assessments"),
	}

	s.setupRoutes()
	s.setupInternalRoutes()
	return s
}

func (s *Server) setupRoutes() {
	r := s.router

	// Security middleware (order matters)
	r.Use(pkg.RequestIDMiddleware)
	r.Use(pkg.SecurityHeadersMiddleware)
	r.Use(pkg.APIVersionMiddleware)
	r.Use(middleware.RealIP)
	r.Use(pkg.StructuredLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   pkg.GetCORSOrigins(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With", "X-Request-ID", "X-API-Key"},
		ExposedHeaders:   []string{"Link", "X-Request-ID", "X-API-Version", "X-Deprecation-Warning"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public endpoints (no auth/rate limiting)
	r.Get("/health", s.healthCheck)
	r.Get("/health/live", s.healthLive)
	r.Get("/health/ready", s.healthReady)
	r.Get("/metrics", promhttp.Handler().ServeHTTP)
	r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		pkg.JSON(w, pkg.GetVersionInfo())
	})

	// Service-to-service callbacks (no auth, internal network only)
	r.Post("/api/v1/datasources/callback", s.scanCallback)
	r.Post("/api/v1/datasources/progress", s.scanProgress)
	r.Post("/api/v1/classification/callback", s.classificationCallback)
	r.Post("/api/v1/classification/progress", s.classificationProgress)

	// API Documentation
	r.Get("/api/docs", s.swaggerUI)
	r.Get("/api/openapi.json", s.openAPISpec)

	// Agent downloads (public, no auth required)
	r.Get("/api/v1/downloads/agent", s.listAgentDownloads)
	r.Get("/api/v1/downloads/agent/{platform}", s.downloadAgent)

	r.Route("/api/v1", func(r chi.Router) {
		// Auth endpoints - rate limiting (60/min per IP for login attempts)
		r.Group(func(r chi.Router) {
			r.Use(pkg.RateLimitByIP(60))
			r.Post("/auth/login", s.login)
			r.Post("/auth/refresh", s.refreshToken)
			r.Get("/invitations/verify/{token}", s.verifyInvitation)
			r.Post("/auth/register", s.registerWithInvitation)
			
			// SSO public endpoints (no auth required)
			r.Get("/auth/sso/providers", s.listTenantSSOProviders)
			r.Get("/auth/sso/oidc/{provider_id}", s.ssoOIDCInitiate)
			r.Get("/auth/sso/oidc/callback", s.ssoOIDCCallback)
			r.Get("/auth/sso/saml/{provider_id}", s.ssoSAMLInitiate)
			r.Post("/auth/sso/saml/acs", s.ssoSAMLACS)
			r.Get("/auth/sso/saml/metadata/{tenant_id}", s.ssoSAMLMetadata)
		})

		// Protected routes with general rate limiting (1000/min per tenant+IP)
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)
			r.Use(s.tenantMiddleware)
			r.Use(pkg.RateLimitByTenant(1000))

			// Auth
			r.Post("/auth/logout", s.logout)
			r.Get("/auth/api-keys", s.listAPIKeys)
			r.Post("/auth/api-keys", s.createAPIKey)
			r.Delete("/auth/api-keys/{id}", s.revokeAPIKey)

			// Users
			r.Route("/users", func(r chi.Router) {
				r.Use(s.rbacMiddleware("users:read"))
				r.Get("/", s.listUsers)
				r.Post("/", s.createUser)
				r.Get("/{id}", s.getUser)
				r.Put("/{id}", s.updateUser)
				r.Delete("/{id}", s.deleteUser)
			})

			// Invitations
			r.Route("/invitations", func(r chi.Router) {
				r.Use(s.rbacMiddleware("users:read"))
				r.Get("/", s.listInvitations)
				r.Post("/", s.createInvitation)
				r.Delete("/{id}", s.cancelInvitation)
				r.Post("/{id}/resend", s.resendInvitation)
			})

			// SSO Provider Management (admin only)
			r.Route("/admin/sso/providers", func(r chi.Router) {
				r.Use(s.rbacMiddleware("settings:read"))
				r.Get("/", s.listSSOProviders)
				r.Post("/", s.createSSOProvider)
				r.Get("/{id}", s.getSSOProvider)
				r.Put("/{id}", s.updateSSOProvider)
				r.Delete("/{id}", s.deleteSSOProvider)
			})

			// Roles
			r.Route("/roles", func(r chi.Router) {
				r.Use(s.rbacMiddleware("roles:read"))
				r.Get("/", s.listRoles)
				r.Post("/", s.createRole)
				r.Put("/{id}", s.updateRole)
			})

			// Data Sources
			r.Route("/datasources", func(r chi.Router) {
				r.Use(s.rbacMiddleware("datasources:read"))
				r.Get("/", s.listDataSources)
				r.Post("/", s.createDataSource)
				r.Get("/{id}", s.getDataSource)
				r.Put("/{id}", s.updateDataSource)
				r.Delete("/{id}", s.deleteDataSource)
				r.Post("/{id}/scan", s.triggerScan)
				r.Get("/{id}/status", s.getScanStatus)
				r.Get("/{id}/logs", s.listScanLogs)
				r.Get("/{id}/classification-stats", s.getDataSourceClassificationStats)
			})

			// Policies (alias to governance/policies)
			r.Route("/policies", func(r chi.Router) {
				r.Use(s.rbacMiddleware("policies:read"))
				r.Get("/", s.listPolicies)
				r.Post("/", s.createPolicy)
				r.Get("/{id}", s.getPolicy)
				r.Put("/{id}", s.updatePolicy)
				r.Delete("/{id}", s.deletePolicy)
			})

			// Policies under governance
			r.Route("/governance/policies", func(r chi.Router) {
				r.Use(s.rbacMiddleware("policies:read"))
				r.Get("/", s.listPolicies)
				r.Post("/", s.createPolicy)
				r.Get("/{id}", s.getPolicy)
				r.Put("/{id}", s.updatePolicy)
				r.Delete("/{id}", s.deletePolicy)
			})
			r.Post("/governance/evaluate", s.evaluatePolicy)
			r.Get("/governance/stats", s.getGovernanceStats)

			// Classification
			r.Route("/classify", func(r chi.Router) {
				r.Post("/text", s.classifyText)
				r.Post("/dataset", s.classifyDataset)
				r.Get("/stats", s.getClassifyStats)
				r.Get("/results/{dataset_id}", s.getClassificationResults)
				r.Get("/rules", s.listClassificationRules)
				r.Post("/rules", s.createClassificationRule)
				r.Get("/rules/{id}", s.getClassificationRule)
				r.Put("/rules/{id}", s.updateClassificationRule)
				r.Delete("/rules/{id}", s.deleteClassificationRule)
				r.Get("/models", s.listModels)
				// Dataset-specific classification endpoints
				r.Get("/datasets/{id}", s.getDatasetClassification)
				r.Get("/datasets/{id}/columns", s.getDatasetColumns)
				r.Post("/datasets/{id}/reclassify", s.reclassifyDataset)
			})

			// AI Gate
			r.Route("/gate", func(r chi.Router) {
				r.Use(s.rbacMiddleware("gate:query"))
				r.Post("/query", s.gateQuery)
				r.Post("/embed", s.gateEmbed)
				r.Post("/retrieve", s.gateRetrieve)
				r.Post("/validate", s.gateValidate)
				r.Get("/stats", s.gateStats)
				r.Get("/audit", s.gateAudit)
				r.Get("/queries", s.listGateQueries)
				r.Get("/queries/{id}", s.getGateQuery)
			})

			// Quality
			r.Route("/quality", func(r chi.Router) {
				r.Get("/", s.getQualityOverview)
				r.Get("/dimensions", s.getQualityDimensions)
				r.Get("/datasets/{id}", s.getQualityScore)
				r.Get("/datasets/{id}/issues", s.getQualityIssues)
				r.Post("/assess", s.assessQuality)
				r.Get("/trends", s.getQualityTrends)
				r.Put("/threshold", s.setQualityThresholds)
				r.Post("/thresholds", s.setQualityThresholds)
				r.Post("/profile/{datasource_id}", s.autoProfileDataSource)
				r.Get("/profile/{datasource_id}", s.getDataProfile)
				r.Get("/summary", s.getQualitySummary)
			})

			// Critical Data Elements
			r.Route("/cde", func(r chi.Router) {
				r.Post("/", s.createCDE)
				r.Get("/", s.listCDEs)
				r.Delete("/{id}", s.deleteCDE)
			})

			// Privacy
			r.Route("/privacy", func(r chi.Router) {
				r.Get("/", s.getPrivacyOverview)
				r.Post("/dsar", s.createDSAR)
				r.Get("/dsar", s.listDSARs)
				r.Get("/dsar/{id}", s.getDSAR)
				r.Put("/dsar/{id}", s.updateDSAR)
				r.Delete("/dsar/{id}", s.deleteDSAR)
				r.Get("/dsar/{id}/package", s.getDSARPackage)
				r.Post("/dsar/{id}/execute", s.executeDSAR)
				r.Post("/pia", s.generatePIA)
				r.Get("/pia", s.listPIAs)
				r.Get("/pia/{dataset_id}", s.getPIA)
				r.Get("/ropa", s.listRoPA)
				r.Post("/ropa", s.createRoPA)
				r.Post("/consent", s.recordConsent)
				r.Delete("/consent/{subject_id}", s.withdrawConsent)
				r.Get("/retention/violations", s.getRetentionViolations)
				r.Post("/retention/policies", s.setRetentionPolicy)
				// DPIA workflow
				r.Post("/dpia", s.createDPIA)
				r.Get("/dpia", s.listDPIAs)
				r.Get("/dpia/{id}", s.getDPIA)
				r.Put("/dpia/{id}/step/{step}", s.updateDPIAStep)
				r.Put("/dpia/{id}/status", s.updateDPIAStatus)
				// Enhanced consent
				r.Post("/consent/record", s.recordConsentV2)
				r.Get("/consent/records", s.listConsentRecords)
				r.Get("/consent/stats", s.getConsentStats)
				r.Post("/consent/withdraw/{subject_id}", s.withdrawConsentV2)
			})

			// Audit
			r.Route("/audit", func(r chi.Router) {
				r.Use(s.rbacMiddleware("audit:read"))
				r.Get("/trail", s.getAuditTrail)
				r.Get("/datasets/{id}/ai-usage", s.getAIUsage)
				r.Get("/compliance-report", s.getComplianceReport)
				r.Get("/lineage/{dataset_id}", s.getLineage)
			})

			// Compliance
			r.Route("/compliance", func(r chi.Router) {
				r.Get("/recommendations", s.getRecommendations)
				r.Get("/advisor", s.getRecommendations)
				r.Get("/gaps", s.getComplianceGaps)
				r.Get("/report", s.getComplianceReport)
				r.Get("/risk-score", s.getRiskScore)
				r.Post("/assess", s.runComplianceAssessment)
				r.Get("/frameworks", s.listComplianceFrameworks)
			})

			// Observability
			r.Route("/observability", func(r chi.Router) {
				r.Get("/health", s.getSystemHealth)
				r.Get("/sources/{id}/health", s.getSourceHealth)
				r.Get("/metrics", s.getMetrics)
				r.Get("/alerts", s.getAlerts)
				r.Post("/alerts/rules", s.createAlertRule)
				r.Get("/summary", s.getObservabilitySummary)
			})

			// AI Governance
			r.Route("/ai-governance", func(r chi.Router) {
				r.Get("/policies", s.listAIGovPolicies)
				r.Post("/policies", s.createAIGovPolicy)
				r.Post("/evaluate", s.evaluateAIEligibility)
				r.Get("/eligible/{dataset_id}", s.getAIEligibility)
				r.Get("/lineage/{model_id}", s.getModelLineage)
				r.Post("/model-card", s.generateModelCard)
			})

			// Notifications
			r.Route("/notifications", func(r chi.Router) {
				r.Get("/", s.listNotifications)
				r.Put("/{id}/read", s.markNotificationRead)
				r.Put("/read-all", s.markAllNotificationsRead)
				r.Delete("/{id}", s.deleteNotification)
				r.Post("/webhooks", s.createWebhook)
				r.Get("/webhooks", s.listWebhooks)
				r.Delete("/webhooks/{id}", s.deleteWebhook)
				r.Get("/events", s.streamEvents)
			})

			// Jobs
			r.Route("/jobs", func(r chi.Router) {
				r.Get("/", s.listJobs)
				r.Post("/", s.createJob)
				r.Get("/{id}", s.getJob)
				r.Put("/{id}", s.updateJob)
				r.Delete("/{id}", s.deleteJob)
				r.Post("/{id}/run-now", s.runJobNow)
				r.Get("/{id}/history", s.getJobHistory)
			})

			// Remediation
			r.Route("/remediation", func(r chi.Router) {
				r.Get("/", s.listRemediationActions)
				r.Get("/actions", s.listRemediationActions)
				r.Post("/actions", s.createRemediationAction)
				r.Get("/actions/{id}/logs", s.getRemediationLogs)
				r.Post("/actions/{id}/execute", s.executeRemediationAction)
				r.Post("/actions/{id}/approve", s.approveRemediation)
				r.Get("/history", s.getRemediationHistory)
			})

			// Reports
			r.Route("/reports", func(r chi.Router) {
				r.Post("/", s.generateReport)
				r.Post("/generate", s.generateReport)
				r.Get("/", s.listReports)
				r.Get("/{id}", s.downloadReport)
			})
			r.Get("/analytics/summary", s.getAnalyticsSummary)
			r.Get("/analytics/trends", s.getAnalyticsTrends)

			// Labels
			r.Route("/labels", func(r chi.Router) {
				r.Get("/datasets/{id}", s.getDatasetLabel)
				r.Post("/assign", s.assignLabel)
				r.Get("/rules", s.getLabelRules)
				r.Post("/rules", s.createLabelRule)
				r.Put("/rules/{id}", s.updateLabelRule)
				r.Delete("/rules/{id}", s.deleteLabelRule)
				r.Get("/summary", s.getLabelSummary)
			})

			// Sensitivity rules alias (maps to labels/rules)
			r.Route("/sensitivity", func(r chi.Router) {
				r.Get("/rules", s.getLabelRules)
				r.Post("/rules", s.createLabelRule)
				r.Put("/rules/{id}", s.updateLabelRule)
				r.Delete("/rules/{id}", s.deleteLabelRule)
			})

			// AI Gate alias (maps to /gate)
			r.Route("/ai-gate", func(r chi.Router) {
				r.Get("/stats", s.gateStats)
				r.Get("/queries", s.listGateQueries)
				r.Get("/queries/{id}", s.getGateQuery)
				r.Get("/audit", s.gateAudit)
				r.Post("/query", s.gateQuery)
				r.Post("/embed", s.gateEmbed)
				r.Post("/retrieve", s.gateRetrieve)
				r.Post("/validate", s.gateValidate)
			})

			// Lineage top-level endpoint
			r.Get("/lineage", s.getLineageSummary)

			// Feedback
			r.Route("/feedback", func(r chi.Router) {
				r.Post("/", s.submitFeedback)
				r.Post("/correction", s.submitCorrection)
				r.Post("/confirmation", s.submitConfirmation)
				r.Get("/stats", s.getFeedbackStats)
				r.Get("/corrections", s.listCorrections)
				r.Get("/trend", s.getCorrectionTrend)
				r.Post("/custom-entity", s.createCustomEntity)
				r.Get("/custom-entities", s.listCustomEntities)
				r.Get("/knowledge-cache", s.getKnowledgeCache)
			})

			// Advisor
			r.Route("/advisor", func(r chi.Router) {
				r.Get("/", s.getAdvisorOverview)
				r.Get("/recommendations", s.getRecommendations)
				r.Get("/gaps", s.getComplianceGaps)
				r.Get("/defense-docket", s.getDefenseDocket)
				r.Post("/defense-docket", s.generateDefenseDocket)
				r.Get("/playbooks", s.listPlaybooks)
				r.Get("/playbook/{issue_type}", s.getPlaybook)
				r.Get("/risk-score", s.getRiskScore)
				r.Post("/assess", s.runComplianceAssessment)
				r.Get("/assessment/logs", s.listAssessmentLogs)
			})

			// ROT
			r.Route("/rot", func(r chi.Router) {
				r.Get("/summary", s.getROTSummary)
				r.Get("/datasets", s.getROTDatasets)
				r.Get("/items", s.listROTItems)
				r.Get("/duplicates", s.getDuplicates)
				r.Post("/scan", s.triggerROTScan)
				r.Post("/analyze", s.triggerROTScan)
				r.Get("/scan/{id}/status", s.getROTScanStatus)
				r.Post("/remediate", s.remediateROT)
			})

			// Integrations
			r.Route("/integrations", func(r chi.Router) {
				r.Get("/", s.listIntegrations)
				r.Post("/", s.createIntegration)
				r.Get("/{id}", s.getIntegration)
				r.Put("/{id}", s.updateIntegration)
				r.Delete("/{id}", s.deleteIntegration)
				r.Post("/{id}/test", s.testIntegration)
				r.Post("/{id}/sync", s.syncIntegration)
				r.Get("/{id}/logs", s.getIntegrationLogs)
			})

			// Data Map
			r.Route("/datamap", func(r chi.Router) {
				r.Get("/", s.getDataMap)
				r.Get("/sources", s.getDataMapSources)
				r.Get("/flows", s.getDataFlows)
				r.Get("/coverage", s.getCoverage)
				r.Get("/geography", s.getGeography)
				r.Get("/dark-data", s.getDarkData)
			})

			// Documents
			r.Route("/documents", func(r chi.Router) {
				r.Post("/extract", s.extractDocument)
				r.Post("/classify", s.classifyDocument)
				r.Get("/review-queue", s.getReviewQueue)
				r.Get("/{id}/classifications", s.getDocumentClassifications)
			})

			// Endpoint agents (legacy device-level scanning)
			r.Route("/endpoints/agents", func(r chi.Router) {
				r.Post("/register", s.registerEndpoint)
				r.Get("/", s.listEndpoints)
				r.Post("/report", s.reportAgentScan)
				r.Post("/{id}/scan", s.triggerEndpointScan)
				r.Post("/{id}/scan-results", s.receiveEndpointScanResults)
				r.Get("/{id}/results", s.getEndpointResults)
				r.Get("/{id}/status", s.getAgentStatus)
				r.Post("/{id}/heartbeat", s.agentHeartbeat)
			})

			// Endpoint URL scanning (API endpoint PII scanning)
			r.Route("/endpoints", func(r chi.Router) {
				r.Get("/", s.listEndpointScans)
				r.Post("/", s.createEndpointScan)
				r.Get("/{id}", s.getEndpointScan)
				r.Put("/{id}", s.updateEndpointScan)
				r.Delete("/{id}", s.deleteEndpointScan)
				r.Post("/{id}/scan", s.runEndpointScan)
				r.Get("/{id}/findings", s.getEndpointScanFindings)
			})

			// Data residency
			r.Route("/residency", func(r chi.Router) {
				r.Post("/rules", s.createResidencyRule)
				r.Get("/rules", s.listResidencyRules)
				r.Delete("/rules/{id}", s.deleteResidencyRule)
				r.Get("/violations", s.getResidencyViolations)
				r.Post("/datasources/{id}/tag-region", s.tagDatasourceRegion)
				r.Post("/detect", s.detectRegionsHandler)
				r.Get("/stats", s.getResidencyStats)
				r.Get("/regions", s.getResidencyRegions)
			})

			// Consent widget config & preferences
			r.Route("/consent", func(r chi.Router) {
				r.Get("/widget-config", s.getConsentWidgetConfig)
				r.Put("/widget-config", s.updateConsentWidgetConfig)
				r.Get("/embed-code", s.getConsentEmbedCode)
				r.Get("/preferences/{subject_id}", s.getConsentPreferences)
				r.Put("/preferences/{subject_id}", s.updateConsentPreferences)
			})

			// Dashboard endpoints
			r.Route("/dashboard", func(r chi.Router) {
				r.Get("/stats", s.getDashboardStats)
				r.Get("/recent-activity", s.getDashboardRecentActivity)
				r.Get("/charts", s.getDashboardCharts)
			})

			// Route aliases for frontend compatibility
			r.Get("/classification/rules", s.listClassificationRules)
			r.Post("/classification/rules", s.createClassificationRule)
			r.Get("/classification/rules/{id}", s.getClassificationRule)
			r.Put("/classification/rules/{id}", s.updateClassificationRule)
			r.Delete("/classification/rules/{id}", s.deleteClassificationRule)

			r.Get("/sensitivity-labels", s.getLabelRules)
			r.Post("/sensitivity-labels", s.createLabelRule)
			r.Put("/sensitivity-labels/{id}", s.updateLabelRule)
			r.Delete("/sensitivity-labels/{id}", s.deleteLabelRule)

			r.Get("/data-quality", s.getQualityOverview)
			r.Get("/data-quality/dimensions", s.getQualityDimensions)
			r.Get("/data-quality/trends", s.getQualityTrends)

			r.Get("/audit-logs", s.getAuditTrail)

			r.Get("/scheduled-jobs", s.listJobs)
			r.Post("/scheduled-jobs", s.createJob)
			r.Get("/scheduled-jobs/{id}", s.getJob)
			r.Put("/scheduled-jobs/{id}", s.updateJob)
			r.Delete("/scheduled-jobs/{id}", s.deleteJob)

			r.Get("/api-keys", s.listAPIKeys)
			r.Post("/api-keys", s.createAPIKey)
			r.Delete("/api-keys/{id}", s.revokeAPIKey)

			r.Get("/ai-gate/history", s.listGateQueries)
		})
	})
}

func (s *Server) setupInternalRoutes() {
	r := s.internalRouter

	// Security middleware for internal routes
	r.Use(pkg.RequestIDMiddleware)
	r.Use(pkg.SecurityHeadersMiddleware)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(pkg.IPWhitelistMiddleware("SUPERADMIN_ALLOWED_IPS"))
	r.Use(pkg.RateLimitByIP(5)) // Strict rate limiting for admin

	r.Route("/internal/v1", func(r chi.Router) {
		r.Use(s.superAdminMiddleware)

		r.Post("/tenants", s.createTenant)
		r.Get("/tenants", s.listTenants)
		r.Get("/tenants/{id}", s.getTenant)
		r.Put("/tenants/{id}/suspend", s.suspendTenant)
		r.Delete("/tenants/{id}", s.deleteTenant)
		r.Post("/tenants/{id}/impersonate", s.impersonateTenant)
	})
}

func (s *Server) Run(port string) {
	s.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: s.router,
	}
	log.Info().Str("port", port).Msg("Starting API server")
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server failed")
	}
}

func (s *Server) RunInternal(port string) {
	s.internalServer = &http.Server{
		Addr:    ":" + port,
		Handler: s.internalRouter,
	}
	log.Info().Str("port", port).Msg("Starting internal admin server")
	if err := s.internalServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Internal server failed")
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	var errs []error
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if s.internalServer != nil {
		if err := s.internalServer.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	pkg.JSON(w, map[string]string{"status": "ok"})
}

func (s *Server) healthLive(w http.ResponseWriter, r *http.Request) {
	pkg.JSON(w, map[string]string{"status": "alive"})
}

func (s *Server) healthReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := map[string]string{}
	allHealthy := true

	// Check database
	if err := s.db.PingContext(ctx); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
		allHealthy = false
	} else {
		checks["database"] = "healthy"
	}

	// Check Kafka
	if s.kafka != nil && s.kafka.IsHealthy(ctx) {
		checks["kafka"] = "healthy"
	} else {
		checks["kafka"] = "unhealthy"
		allHealthy = false
	}

	// Check Qdrant
	if s.qdrant != nil && s.qdrant.IsHealthy(ctx) {
		checks["qdrant"] = "healthy"
	} else {
		checks["qdrant"] = "unhealthy"
	}

	status := http.StatusOK
	if !allHealthy {
		status = http.StatusServiceUnavailable
	}

	pkg.JSON(w, map[string]any{
		"status": map[bool]string{true: "ready", false: "not_ready"}[allHealthy],
		"checks": checks,
	}, status)
}

// Middleware
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string
		
		// First check Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "" && strings.HasPrefix(auth, "Bearer ") {
			tokenStr = strings.TrimPrefix(auth, "Bearer ")
		}
		
		// Fallback to query parameter (for SSE/EventSource which can't set headers)
		if tokenStr == "" {
			tokenStr = r.URL.Query().Get("token")
		}
		
		if tokenStr == "" {
			pkg.Error(w, pkg.ErrUnauthorized, http.StatusUnauthorized)
			return
		}

		// Check if this is an API key (starts with sl_)
		if strings.HasPrefix(tokenStr, "sl_") {
			// Validate API key
			ctx := r.Context()
			var apiKey struct {
				ID        string  `db:"id"`
				TenantID  string  `db:"tenant_id"`
				UserID    string  `db:"user_id"`
				KeyHash   string  `db:"key_hash"`
				ExpiresAt *time.Time `db:"expires_at"`
			}
			
			// Find API key by prefix (first 10 chars)
			prefix := tokenStr[:10]
			err := s.db.GetContext(ctx, &apiKey,
				`SELECT id, tenant_id, user_id, key_hash, expires_at 
				 FROM api_keys WHERE key_prefix = $1`, prefix)
			
			if err != nil {
				pkg.Error(w, pkg.ErrUnauthorized, http.StatusUnauthorized)
				return
			}
			
			// Verify the full key hash
			if !pkg.CheckPassword(tokenStr, apiKey.KeyHash) {
				pkg.Error(w, pkg.ErrUnauthorized, http.StatusUnauthorized)
				return
			}
			
			// Check expiration
			if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
				pkg.Error(w, pkg.ErrUnauthorized, http.StatusUnauthorized)
				return
			}
			
			// Update last_used_at
			go func() {
				s.db.ExecContext(context.Background(),
					`UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, apiKey.ID)
			}()
			
			// Set context values
			ctx = context.WithValue(ctx, pkg.CtxUserID, apiKey.UserID)
			ctx = context.WithValue(ctx, pkg.CtxTenantID, apiKey.TenantID)
			ctx = context.WithValue(ctx, pkg.CtxPermissions, []string{"*"}) // API keys have full access
			ctx = context.WithValue(ctx, pkg.CtxIsSuperAdmin, false)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Otherwise validate as JWT
		claims, err := pkg.ValidateToken(tokenStr)
		if err != nil {
			pkg.Error(w, pkg.ErrUnauthorized, http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, pkg.CtxUserID, claims.UserID)
		ctx = context.WithValue(ctx, pkg.CtxTenantID, claims.TenantID)
		ctx = context.WithValue(ctx, pkg.CtxPermissions, claims.Permissions)
		ctx = context.WithValue(ctx, pkg.CtxIsSuperAdmin, claims.IsSuperAdmin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) tenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := pkg.TenantFromCtx(r.Context())
		if tenantID == "" && !pkg.IsSuperAdminFromCtx(r.Context()) {
			pkg.Error(w, pkg.ErrForbidden, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) rbacMiddleware(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Superadmin bypasses all RBAC checks
			if pkg.IsSuperAdminFromCtx(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}

			// For mutating operations, check for write permission
			requiredPerm := permission
			if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" || r.Method == "PATCH" {
				// Convert read permission to write permission
				requiredPerm = strings.Replace(permission, ":read", ":write", 1)
			}
			if !pkg.HasPermission(r.Context(), requiredPerm) {
				pkg.Error(w, pkg.ErrForbidden, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) superAdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			pkg.Error(w, pkg.ErrUnauthorized, http.StatusUnauthorized)
			return
		}

		claims, err := pkg.ValidateToken(strings.TrimPrefix(auth, "Bearer "))
		if err != nil || !claims.IsSuperAdmin {
			pkg.Error(w, pkg.ErrForbidden, http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), pkg.CtxIsSuperAdmin, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// swaggerUI serves the Swagger UI
func (s *Server) swaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(swaggerUIHTML))
}

// openAPISpec serves the OpenAPI specification
func (s *Server) openAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(openAPISpecJSON))
}

// versionInfo returns API version information
func (s *Server) versionInfo(w http.ResponseWriter, r *http.Request) {
	pkg.JSON(w, pkg.GetVersionInfo())
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>SecureLens API Documentation</title>
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
    .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      SwaggerUIBundle({
        url: "/api/openapi.json",
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`
