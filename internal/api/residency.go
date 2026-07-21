package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/securelens/securelens/internal/domain"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

// residencyViolation is what we persist and return from the violations table.
type residencyViolation struct {
	ID               string   `db:"id"               json:"id"`
	TenantID         string   `db:"tenant_id"        json:"-"`
	RuleID           string   `db:"rule_id"          json:"rule_id"`
	DatasourceID     string   `db:"datasource_id"    json:"datasource_id"`
	DatasourceName   string   `db:"datasource_name"  json:"datasource_name"`
	DatasourceRegion string   `db:"datasource_region" json:"datasource_region"`
	RuleName         string   `db:"rule_name"        json:"rule_name"`
	Regulation       string   `db:"regulation"       json:"regulation"`
	AllowedRegions   []string `db:"-"                json:"allowed_regions"`
	Reason           string   `db:"-"                json:"reason"`
}

func (s *Server) createResidencyRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	var req struct {
		Name           string   `json:"name" validate:"required"`
		Regulation     string   `json:"regulation"`
		AllowedRegions []string `json:"allowed_regions"`
		DataTypes      []string `json:"data_types"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// Convert Go slices to PostgreSQL array format using pq.Array
	var ruleID string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO residency_rules (id, tenant_id, name, regulation, allowed_regions, data_types, active, created_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, true, NOW())
		 RETURNING id`,
		tenantID, req.Name, req.Regulation, pkg.PGArray(req.AllowedRegions), pkg.PGArray(req.DataTypes),
	).Scan(&ruleID)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	// Store as JSON for the response
	regionsJSON, _ := json.Marshal(req.AllowedRegions)
	typesJSON, _ := json.Marshal(req.DataTypes)

	rule := &store.ResidencyRule{
		ID:             ruleID,
		TenantID:       tenantID,
		Name:           req.Name,
		Regulation:     req.Regulation,
		AllowedRegions: store.JSON(regionsJSON),
		DataTypes:      store.JSON(typesJSON),
		Active:         true,
	}

	// Evaluate all existing datasources against the new rule and persist violations.
	datasources, err := s.datasources.List(ctx, tenantID, store.ListOpts{Limit: 10000})
	if err != nil {
		log.Warn().Err(err).Str("rule_id", rule.ID).Msg("residency: failed to list datasources for violation check")
	} else {
		s.persistViolationsForRule(ctx, tenantID, rule, req.AllowedRegions, datasources)
	}

	pkg.JSON(w, rule, http.StatusCreated)
}

// persistViolationsForRule inserts a violation row for every datasource whose region
// is not covered by the rule's allowed regions. Existing rows are left untouched (ON CONFLICT DO NOTHING).
func (s *Server) persistViolationsForRule(
	ctx context.Context,
	tenantID string,
	rule *store.ResidencyRule,
	allowedRegions []string,
	datasources []store.DataSource,
) {
	allowed := make(map[string]bool, len(allowedRegions))
	for _, r := range allowedRegions {
		allowed[r] = true
	}
	regionsJSON, _ := json.Marshal(allowedRegions)

	for _, ds := range datasources {
		region := ""
		if ds.Region != nil {
			region = *ds.Region
		}
		if allowed[region] {
			continue
		}
		id := uuid.New().String()
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO residency_violations
				(id, tenant_id, rule_id, datasource_id, datasource_name, datasource_region, rule_name, regulation, allowed_regions, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
			 ON CONFLICT (rule_id, datasource_id) DO NOTHING`,
			id, tenantID, rule.ID, ds.ID, ds.Name, region, rule.Name, rule.Regulation,
			store.JSON(regionsJSON),
		)
		if err != nil {
			log.Warn().Err(err).Str("datasource_id", ds.ID).Str("rule_id", rule.ID).Msg("residency: failed to insert violation")
		}
	}
}

func (s *Server) listResidencyRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)
	rules, _ := s.residencyRules.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	if rules == nil {
		rules = []store.ResidencyRule{}
	}
	pkg.JSON(w, map[string]any{"rules": rules, "total": len(rules)})
}

func (s *Server) deleteResidencyRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")
	rule, err := s.residencyRules.FindByID(ctx, tenantID, id)
	if err != nil || rule == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	if err := s.residencyRules.Delete(ctx, tenantID, id); err != nil {
		pkg.Error(w, err)
		return
	}
	// Cascade: remove all violations belonging to this rule.
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM residency_violations WHERE tenant_id = $1 AND rule_id = $2`,
		tenantID, id,
	); err != nil {
		log.Warn().Err(err).Str("rule_id", id).Msg("residency: failed to delete violations for rule")
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getResidencyViolations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	type row struct {
		ID               string `db:"id"`
		RuleID           string `db:"rule_id"`
		DatasourceID     string `db:"datasource_id"`
		DatasourceName   string `db:"datasource_name"`
		DatasourceRegion string `db:"datasource_region"`
		RuleName         string `db:"rule_name"`
		Regulation       string `db:"regulation"`
		AllowedRegions   []byte `db:"allowed_regions"`
	}
	var rows []row
	err := s.db.SelectContext(ctx, &rows,
		`SELECT id, rule_id, datasource_id, datasource_name, datasource_region,
		        rule_name, regulation, allowed_regions
		 FROM residency_violations
		 WHERE ($1::text = '' OR tenant_id = $1)
		 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		// Table may not exist yet in this deployment; fall back to inline evaluation.
		s.getResidencyViolationsInline(w, r)
		return
	}

	type Violation struct {
		ID               string   `json:"id"`
		DatasourceID     string   `json:"datasource_id"`
		DatasourceName   string   `json:"datasource_name"`
		DatasourceRegion string   `json:"datasource_region"`
		Region           string   `json:"region"` // alias kept for frontend compatibility
		RuleID           string   `json:"rule_id"`
		RuleName         string   `json:"rule_name"`
		Regulation       string   `json:"regulation"`
		AllowedRegions   []string `json:"allowed_regions"`
		Reason           string   `json:"reason"`
	}

	violations := make([]Violation, 0, len(rows))
	for _, rw := range rows {
		var ar []string
		_ = json.Unmarshal(rw.AllowedRegions, &ar)
		region := rw.DatasourceRegion
		if region == "" {
			region = "untagged"
		}
		reason := fmt.Sprintf("region %q is not in allowed regions for rule %q (%s)", region, rw.RuleName, rw.Regulation)
		violations = append(violations, Violation{
			ID:               rw.ID,
			DatasourceID:     rw.DatasourceID,
			DatasourceName:   rw.DatasourceName,
			DatasourceRegion: region,
			Region:           region,
			RuleID:           rw.RuleID,
			RuleName:         rw.RuleName,
			Regulation:       rw.Regulation,
			AllowedRegions:   ar,
			Reason:           reason,
		})
	}
	pkg.JSON(w, map[string]any{"violations": violations, "total": len(violations)})
}

// getResidencyViolationsInline is the legacy inline-evaluation fallback used when
// the residency_violations table is not yet available.
func (s *Server) getResidencyViolationsInline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	rules, _ := s.residencyRules.List(ctx, tenantID, store.ListOpts{Limit: 500})
	datasources, _ := s.datasources.List(ctx, tenantID, store.ListOpts{Limit: 1000})

	type Violation struct {
		DatasourceID     string   `json:"datasource_id"`
		DatasourceName   string   `json:"datasource_name"`
		DatasourceRegion string   `json:"datasource_region"`
		Region           string   `json:"region"`
		RuleID           string   `json:"rule_id"`
		RuleName         string   `json:"rule_name"`
		Regulation       string   `json:"regulation"`
		AllowedRegions   []string `json:"allowed_regions"`
		Reason           string   `json:"reason"`
	}

	violations := []Violation{}
	for _, ds := range datasources {
		region := ""
		if ds.Region != nil {
			region = *ds.Region
		}
		if region == "" {
			if len(rules) > 0 {
				violations = append(violations, Violation{
					DatasourceID:     ds.ID,
					DatasourceName:   ds.Name,
					DatasourceRegion: "untagged",
					Region:           "untagged",
					Reason:           "datasource has no region tag",
				})
			}
			continue
		}
		for _, rule := range rules {
			if !rule.Active {
				continue
			}
			var allowedRegions []string
			if err := json.Unmarshal(rule.AllowedRegions, &allowedRegions); err != nil {
				continue
			}
			inAllowed := false
			for _, ar := range allowedRegions {
				if ar == region {
					inAllowed = true
					break
				}
			}
			if !inAllowed {
				violations = append(violations, Violation{
					DatasourceID:     ds.ID,
					DatasourceName:   ds.Name,
					DatasourceRegion: region,
					Region:           region,
					RuleID:           rule.ID,
					RuleName:         rule.Name,
					Regulation:       rule.Regulation,
					AllowedRegions:   allowedRegions,
					Reason:           fmt.Sprintf("region %q not allowed by rule %q (%s)", region, rule.Name, rule.Regulation),
				})
			}
		}
	}
	pkg.JSON(w, map[string]any{"violations": violations, "total": len(violations)})
}

func (s *Server) tagDatasourceRegion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")
	ds, err := s.datasources.FindByID(ctx, tenantID, id)
	if err != nil || ds == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	var req struct {
		Region  string `json:"region" validate:"required"`
		Country string `json:"country"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}
	ds.Region = &req.Region
	ds.Country = &req.Country
	if err := s.datasources.Update(ctx, ds); err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, map[string]any{"datasource_id": id, "region": req.Region, "country": req.Country})
}

// detectRegionsHandler runs auto-detection on all datasources that have no region set.
// POST /residency/detect
func (s *Server) detectRegionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	sources, err := s.datasources.List(ctx, tenantID, store.ListOpts{Limit: 1000})
	if err != nil {
		pkg.Error(w, err)
		return
	}

	type result struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Region   string `json:"region"`
		Detected bool   `json:"detected"`
	}
	var results []result
	for i := range sources {
		ds := &sources[i]
		existing := ""
		if ds.Region != nil {
			existing = *ds.Region
		}
		if existing != "" {
			results = append(results, result{ID: ds.ID, Name: ds.Name, Region: existing, Detected: false})
			continue
		}
		detected := domain.DetectRegionInfo(ctx, ds)
		if detected.Region != "" {
			ds.Region = &detected.Region
			if detected.Country != "" {
				ds.Country = &detected.Country
			}
			if err := s.datasources.Update(ctx, ds); err == nil {
				log.Info().Str("datasource_id", ds.ID).Str("region", detected.Region).Str("country", detected.Country).Msg("bulk geo_detect: region set")
				results = append(results, result{ID: ds.ID, Name: ds.Name, Region: detected.Region, Detected: true})
			} else {
				results = append(results, result{ID: ds.ID, Name: ds.Name, Region: "", Detected: false})
			}
		} else {
			results = append(results, result{ID: ds.ID, Name: ds.Name, Region: "", Detected: false})
		}
	}
	if results == nil {
		results = []result{}
	}
	pkg.JSON(w, map[string]any{"results": results, "total": len(results)})
}

// getResidencyStats returns real aggregate counts for the residency dashboard.
// GET /residency/stats
func (s *Server) getResidencyStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	datasources, _ := s.datasources.List(ctx, tenantID, store.ListOpts{Limit: 10000})
	rules, _ := s.residencyRules.List(ctx, tenantID, store.ListOpts{Limit: 10000})

	// Read violation count from DB table for consistency with /violations endpoint.
	var violationCount int
	err := s.db.GetContext(ctx, &violationCount,
		`SELECT COUNT(*) FROM residency_violations WHERE ($1::text = '' OR tenant_id = $1)`,
		tenantID,
	)
	if err != nil {
		// Table not yet created: fall back to inline count.
		violationCount = s.countViolationsInline(rules, datasources)
	}

	untagged := 0
	for _, ds := range datasources {
		if ds.Region == nil || *ds.Region == "" {
			untagged++
		}
	}

	pkg.JSON(w, map[string]any{
		"total_datasources": len(datasources),
		"violations":        violationCount,
		"residency_rules":   len(rules),
		"untagged_sources":  untagged,
	})
}

// countViolationsInline is the fallback used when residency_violations table is absent.
func (s *Server) countViolationsInline(rules []store.ResidencyRule, datasources []store.DataSource) int {
	count := 0
	for _, ds := range datasources {
		region := ""
		if ds.Region != nil {
			region = *ds.Region
		}
		if region == "" {
			if len(rules) > 0 {
				count++
			}
			continue
		}
		for _, rule := range rules {
			if !rule.Active {
				continue
			}
			var allowedRegions []string
			if err := json.Unmarshal(rule.AllowedRegions, &allowedRegions); err != nil {
				continue
			}
			inAllowed := false
			for _, ar := range allowedRegions {
				if ar == region {
					inAllowed = true
					break
				}
			}
			if !inAllowed {
				count++
			}
		}
	}
	return count
}

// getResidencyRegions returns the geographic distribution of datasources grouped by region.
// GET /residency/regions
func (s *Server) getResidencyRegions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	datasources, _ := s.datasources.List(ctx, tenantID, store.ListOpts{Limit: 10000})

	type RegionEntry struct {
		Region      string   `json:"region"`
		Country     string   `json:"country"`
		Count       int      `json:"count"`
		Datasources []string `json:"datasource_names"`
	}

	grouped := map[string]*RegionEntry{}
	untaggedNames := []string{}

	for _, ds := range datasources {
		if ds.Region == nil || *ds.Region == "" {
			untaggedNames = append(untaggedNames, ds.Name)
			continue
		}
		key := *ds.Region
		entry, ok := grouped[key]
		if !ok {
			country := ""
			if ds.Country != nil {
				country = *ds.Country
			}
			entry = &RegionEntry{Region: key, Country: country}
			grouped[key] = entry
		}
		entry.Count++
		entry.Datasources = append(entry.Datasources, ds.Name)
	}

	regions := make([]RegionEntry, 0, len(grouped))
	for _, v := range grouped {
		regions = append(regions, *v)
	}

	pkg.JSON(w, map[string]any{
		"regions":        regions,
		"untagged_names": untaggedNames,
		"untagged_count": len(untaggedNames),
	})
}

func (s *Server) getConsentWidgetConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	var cfg store.ConsentWidgetConfig
	err := s.db.GetContext(ctx, &cfg,
		`SELECT * FROM consent_widget_configs WHERE tenant_id = $1`, tenantID)
	if err != nil {
		// Return defaults with null ID when not configured (indicates no config exists)
		pkg.JSON(w, map[string]any{
			"id":               nil,
			"tenant_id":        tenantID,
			"primary_color":    "#6366f1",
			"background_color": "#ffffff",
			"text_color":       "#111827",
			"banner_title":     "We value your privacy",
			"banner_text":      "We use cookies and similar technologies to improve your experience.",
			"accept_label":     "Accept All",
			"reject_label":     "Reject Non-Essential",
			"purposes":         []any{},
			"created_at":       nil,
			"updated_at":       nil,
		})
		return
	}
	pkg.JSON(w, cfg)
}

func (s *Server) updateConsentWidgetConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	var req store.ConsentWidgetConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pkg.Error(w, pkg.ErrBadRequest, http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO consent_widget_configs (tenant_id, primary_color, background_color, text_color, banner_title, banner_text, accept_label, reject_label, purposes, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
		 ON CONFLICT (tenant_id) DO UPDATE SET
		   primary_color=EXCLUDED.primary_color, background_color=EXCLUDED.background_color,
		   text_color=EXCLUDED.text_color, banner_title=EXCLUDED.banner_title,
		   banner_text=EXCLUDED.banner_text, accept_label=EXCLUDED.accept_label,
		   reject_label=EXCLUDED.reject_label, purposes=EXCLUDED.purposes, updated_at=NOW()`,
		tenantID, req.PrimaryColor, req.BackgroundColor, req.TextColor,
		req.BannerTitle, req.BannerText, req.AcceptLabel, req.RejectLabel, req.Purposes)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, map[string]string{"status": "updated"})
}

func (s *Server) getConsentPreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	subjectID := chi.URLParam(r, "subject_id")

	var pref store.ConsentPreference
	err := s.db.GetContext(ctx, &pref,
		`SELECT * FROM consent_preferences WHERE tenant_id = $1 AND subject_id = $2`, tenantID, subjectID)
	if err != nil {
		// Return defaults
		pkg.JSON(w, map[string]any{
			"subject_id":  subjectID,
			"preferences": map[string]any{"analytics": false, "marketing": false, "necessary": true},
		})
		return
	}
	var prefs map[string]any
	json.Unmarshal(pref.Preferences, &prefs)
	pkg.JSON(w, map[string]any{"subject_id": subjectID, "preferences": prefs})
}

func (s *Server) updateConsentPreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	subjectID := chi.URLParam(r, "subject_id")
	var prefs map[string]any
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		pkg.Error(w, pkg.ErrBadRequest, http.StatusBadRequest)
		return
	}
	prefsJSON, _ := json.Marshal(prefs)
	ip := r.RemoteAddr
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO consent_preferences (tenant_id, subject_id, preferences, ip, updated_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (tenant_id, subject_id) DO UPDATE SET
		   preferences = EXCLUDED.preferences, ip = EXCLUDED.ip, updated_at = NOW()`,
		tenantID, subjectID, prefsJSON, ip)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, map[string]any{"subject_id": subjectID, "preferences": prefs, "status": "updated"})
}

func (s *Server) getConsentEmbedCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	apiBase := os.Getenv("API_BASE_URL")
	if apiBase == "" {
		apiBase = "https://api.securelens.ai"
	}
	embedCode := fmt.Sprintf(`<script src="%s/api/v1/consent/widget.js?tenant=%s"></script>`, apiBase, tenantID)
	pkg.JSON(w, map[string]string{"embed_code": embedCode, "tenant_id": tenantID})
}
