package api

import (
	"net/http"
	"time"

	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

func (s *Server) getDashboardStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var totalDatasources, activeDatasources int
	s.db.GetContext(ctx, &totalDatasources, "SELECT COUNT(*) FROM datasources WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &activeDatasources, "SELECT COUNT(*) FROM datasources WHERE tenant_id = $1 AND status = 'active'", tenantID)

	var totalClassifications int
	s.db.GetContext(ctx, &totalClassifications, "SELECT COUNT(*) FROM classifications WHERE tenant_id = $1", tenantID)

	var totalPolicies, activePolicies int
	s.db.GetContext(ctx, &totalPolicies, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &activePolicies, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true", tenantID)

	var totalQueries, blockedQueries int
	var avgLatency float64
	s.db.GetContext(ctx, &totalQueries, "SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &blockedQueries, "SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1 AND decision = 'deny'", tenantID)
	s.db.GetContext(ctx, &avgLatency, "SELECT COALESCE(AVG(latency_ms), 0) FROM gate_queries WHERE tenant_id = $1", tenantID)

	var piiCount int
	s.db.GetContext(ctx, &piiCount,
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1 AND entity_type IN ('PII', 'SSN', 'EMAIL', 'PHONE', 'CREDIT_CARD')", tenantID)

	var pendingDSARs int
	s.db.GetContext(ctx, &pendingDSARs, "SELECT COUNT(*) FROM dsars WHERE tenant_id = $1 AND status = 'pending'", tenantID)

	var unreadNotifications int
	s.db.GetContext(ctx, &unreadNotifications, "SELECT COUNT(*) FROM notifications WHERE tenant_id = $1 AND read = false", tenantID)

	pkg.JSON(w, map[string]any{
		"datasources": map[string]any{
			"total":  totalDatasources,
			"active": activeDatasources,
		},
		"classifications": map[string]any{
			"total":     totalClassifications,
			"pii_count": piiCount,
		},
		"policies": map[string]any{
			"total":  totalPolicies,
			"active": activePolicies,
		},
		"ai_gate": map[string]any{
			"total_queries":   totalQueries,
			"blocked_queries": blockedQueries,
			"avg_latency_ms":  avgLatency,
		},
		"privacy": map[string]any{
			"pending_dsars": pendingDSARs,
		},
		"notifications": map[string]any{
			"unread": unreadNotifications,
		},
		"timestamp": time.Now(),
	})
}

func (s *Server) getDashboardRecentActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, _ := pkg.ParseListOpts(r)
	if limit > 50 {
		limit = 50
	}

	type Activity struct {
		ID        string    `json:"id"`
		Type      string    `json:"type"`
		Action    string    `json:"action"`
		Resource  string    `json:"resource"`
		Details   string    `json:"details"`
		Timestamp time.Time `json:"timestamp"`
	}

	var activities []Activity

	// Get recent audit logs
	var auditLogs []store.AuditLog
	s.db.SelectContext(ctx, &auditLogs,
		`SELECT id, action, resource, COALESCE(resource_id::text, '') as resource_id, created_at 
		 FROM audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`,
		tenantID, limit)

	for _, log := range auditLogs {
		activities = append(activities, Activity{
			ID:        log.ID,
			Type:      "audit",
			Action:    log.Action,
			Resource:  log.Resource,
			Details:   log.ResourceID,
			Timestamp: log.CreatedAt,
		})
	}

	// Get recent scans
	var scanLogs []store.ScanLog
	s.db.SelectContext(ctx, &scanLogs,
		`SELECT id, datasource_id, status, message, created_at 
		 FROM scan_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`,
		tenantID, limit/2)

	for _, scan := range scanLogs {
		activities = append(activities, Activity{
			ID:        scan.ID,
			Type:      "scan",
			Action:    scan.Status,
			Resource:  "datasource",
			Details:   scan.Message,
			Timestamp: scan.CreatedAt,
		})
	}

	// Get recent gate queries
	type GateQueryRow struct {
		ID        string    `db:"id"`
		Decision  string    `db:"decision"`
		CreatedAt time.Time `db:"created_at"`
	}
	var gateQueries []GateQueryRow
	s.db.SelectContext(ctx, &gateQueries,
		`SELECT id, COALESCE(decision, 'allow') as decision, created_at 
		 FROM gate_queries WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`,
		tenantID, limit/2)

	for _, gq := range gateQueries {
		activities = append(activities, Activity{
			ID:        gq.ID,
			Type:      "ai_gate",
			Action:    gq.Decision,
			Resource:  "query",
			Details:   "",
			Timestamp: gq.CreatedAt,
		})
	}

	// Sort by timestamp descending and limit
	if len(activities) > limit {
		activities = activities[:limit]
	}

	pkg.JSON(w, map[string]any{
		"activities": activities,
		"total":      len(activities),
	})
}

func (s *Server) getDashboardCharts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	// Classifications by type
	type TypeCount struct {
		EntityType string `db:"entity_type" json:"type"`
		Count      int    `db:"count" json:"count"`
	}
	var classificationsByType []TypeCount
	s.db.SelectContext(ctx, &classificationsByType,
		`SELECT entity_type, COUNT(*) as count 
		 FROM classifications WHERE tenant_id = $1 
		 GROUP BY entity_type ORDER BY count DESC LIMIT 10`,
		tenantID)

	// Queries over time (last 7 days)
	type DayCount struct {
		Date  string `db:"date" json:"date"`
		Count int    `db:"count" json:"count"`
	}
	var queriesOverTime []DayCount
	s.db.SelectContext(ctx, &queriesOverTime,
		`SELECT DATE(created_at)::text as date, COUNT(*) as count 
		 FROM gate_queries WHERE tenant_id = $1 AND created_at > NOW() - INTERVAL '7 days'
		 GROUP BY DATE(created_at) ORDER BY date`,
		tenantID)

	// Classifications over time (last 7 days)
	var classificationsOverTime []DayCount
	s.db.SelectContext(ctx, &classificationsOverTime,
		`SELECT DATE(created_at)::text as date, COUNT(*) as count 
		 FROM classifications WHERE tenant_id = $1 AND created_at > NOW() - INTERVAL '7 days'
		 GROUP BY DATE(created_at) ORDER BY date`,
		tenantID)

	// Datasources by type
	var datasourcesByType []TypeCount
	s.db.SelectContext(ctx, &datasourcesByType,
		`SELECT type as entity_type, COUNT(*) as count 
		 FROM datasources WHERE tenant_id = $1 
		 GROUP BY type ORDER BY count DESC`,
		tenantID)

	// Sensitivity label distribution
	type LabelCount struct {
		Label string `db:"label" json:"label"`
		Count int    `db:"count" json:"count"`
	}
	var labelDistribution []LabelCount
	s.db.SelectContext(ctx, &labelDistribution,
		`SELECT COALESCE(label, 'UNCLASSIFIED') as label, COUNT(*) as count 
		 FROM labels WHERE tenant_id = $1 
		 GROUP BY label ORDER BY count DESC`,
		tenantID)

	pkg.JSON(w, map[string]any{
		"classifications_by_type": classificationsByType,
		"queries_over_time":       queriesOverTime,
		"classifications_over_time": classificationsOverTime,
		"datasources_by_type":     datasourcesByType,
		"label_distribution":      labelDistribution,
	})
}
