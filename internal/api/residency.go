package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/securelens/securelens/internal/domain"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

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
	regionsJSON, _ := json.Marshal(req.AllowedRegions)
	typesJSON, _ := json.Marshal(req.DataTypes)
	rule := &store.ResidencyRule{
		TenantID:       tenantID,
		Name:           req.Name,
		Regulation:     req.Regulation,
		AllowedRegions: store.JSON(regionsJSON),
		DataTypes:      store.JSON(typesJSON),
		Active:         true,
	}
	if err := s.residencyRules.Create(ctx, rule); err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, rule, http.StatusCreated)
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
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getResidencyViolations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	rules, _ := s.residencyRules.List(ctx, tenantID, store.ListOpts{Limit: 500})
	datasources, _ := s.datasources.List(ctx, tenantID, store.ListOpts{Limit: 1000})

	type Violation struct {
		DatasourceID   string `json:"datasource_id"`
		DatasourceName string `json:"datasource_name"`
		Region         string `json:"region"`
		RuleID         string `json:"rule_id"`
		RuleName       string `json:"rule_name"`
		Regulation     string `json:"regulation"`
		Reason         string `json:"reason"`
	}

	violations := []Violation{}
	for _, ds := range datasources {
		if ds.Region == nil || *ds.Region == "" {
			if len(rules) > 0 {
				violations = append(violations, Violation{
					DatasourceID:   ds.ID,
					DatasourceName: ds.Name,
					Region:         "untagged",
					Reason:         "datasource has no region tag",
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
			allowed := false
			for _, ar := range allowedRegions {
				if ar == *ds.Region {
					allowed = true
					break
				}
			}
			if !allowed {
				violations = append(violations, Violation{
					DatasourceID:   ds.ID,
					DatasourceName: ds.Name,
					Region:         *ds.Region,
					RuleID:         rule.ID,
					RuleName:       rule.Name,
					Regulation:     rule.Regulation,
					Reason:         fmt.Sprintf("region %q not allowed by rule %q (%s)", *ds.Region, rule.Name, rule.Regulation),
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

	untagged := 0
	violationCount := 0
	for _, ds := range datasources {
		if ds.Region == nil || *ds.Region == "" {
			untagged++
			if len(rules) > 0 {
				violationCount++
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
			allowed := false
			for _, ar := range allowedRegions {
				if ar == *ds.Region {
					allowed = true
					break
				}
			}
			if !allowed {
				violationCount++
			}
		}
	}

	pkg.JSON(w, map[string]any{
		"total_datasources": len(datasources),
		"violations":        violationCount,
		"residency_rules":   len(rules),
		"untagged_sources":  untagged,
	})
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
		"regions":         regions,
		"untagged_names":  untaggedNames,
		"untagged_count":  len(untaggedNames),
	})
}

func (s *Server) getConsentWidgetConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	var cfg store.ConsentWidgetConfig
	err := s.db.GetContext(ctx, &cfg,
		`SELECT * FROM consent_widget_configs WHERE tenant_id = $1`, tenantID)
	if err != nil {
		// Return defaults if not configured
		pkg.JSON(w, store.ConsentWidgetConfig{
			TenantID:        tenantID,
			PrimaryColor:    "#6366f1",
			BackgroundColor: "#ffffff",
			TextColor:       "#111827",
			BannerTitle:     "We value your privacy",
			BannerText:      "We use cookies and similar technologies to improve your experience.",
			AcceptLabel:     "Accept All",
			RejectLabel:     "Reject Non-Essential",
			Purposes:        store.JSON([]byte(`[]`)),
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
