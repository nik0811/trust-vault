package domain

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/securelens/securelens/internal/store"
)

// Standard region codes used throughout the platform
const (
	RegionEU     = "EU"
	RegionUK     = "UK"
	RegionUSEast = "US-EAST"
	RegionUSWest = "US-WEST"
	RegionAPAC   = "APAC"
	RegionCA     = "CA"
	RegionLATAM  = "LATAM"
	RegionMEA    = "MEA"
)

// hostnameRule maps a compiled regex to a region
type hostnameRule struct {
	re     *regexp.Regexp
	region string
}

var hostnameRules = []hostnameRule{
	// AWS RDS
	{re: regexp.MustCompile(`(?i)\.ap-southeast`), region: RegionAPAC},
	{re: regexp.MustCompile(`(?i)\.ap-northeast`), region: RegionAPAC},
	{re: regexp.MustCompile(`(?i)\.ap-south`), region: RegionAPAC},
	{re: regexp.MustCompile(`(?i)\.ap-east`), region: RegionAPAC},
	{re: regexp.MustCompile(`(?i)\.us-east`), region: RegionUSEast},
	{re: regexp.MustCompile(`(?i)\.us-west`), region: RegionUSWest},
	{re: regexp.MustCompile(`(?i)\.eu-`), region: RegionEU},
	{re: regexp.MustCompile(`(?i)\.ca-`), region: RegionCA},
	{re: regexp.MustCompile(`(?i)\.sa-`), region: RegionLATAM},
	{re: regexp.MustCompile(`(?i)\.me-`), region: RegionMEA},
	{re: regexp.MustCompile(`(?i)\.af-`), region: RegionMEA},
	// Azure SQL
	{re: regexp.MustCompile(`(?i)eastus.*\.database\.windows\.net`), region: RegionUSEast},
	{re: regexp.MustCompile(`(?i)westus.*\.database\.windows\.net`), region: RegionUSWest},
	{re: regexp.MustCompile(`(?i)westeurope.*\.database\.windows\.net`), region: RegionEU},
	{re: regexp.MustCompile(`(?i)northeurope.*\.database\.windows\.net`), region: RegionEU},
	{re: regexp.MustCompile(`(?i)uksouth.*\.database\.windows\.net`), region: RegionUK},
	{re: regexp.MustCompile(`(?i)ukwest.*\.database\.windows\.net`), region: RegionUK},
	{re: regexp.MustCompile(`(?i)uaenorth.*\.database\.windows\.net`), region: RegionMEA},
	{re: regexp.MustCompile(`(?i)southeastasia.*\.database\.windows\.net`), region: RegionAPAC},
	{re: regexp.MustCompile(`(?i)eastasia.*\.database\.windows\.net`), region: RegionAPAC},
	{re: regexp.MustCompile(`(?i)canadacentral.*\.database\.windows\.net`), region: RegionCA},
	{re: regexp.MustCompile(`(?i)brazilsouth.*\.database\.windows\.net`), region: RegionLATAM},
	// Azure Blob / generic
	{re: regexp.MustCompile(`(?i)eastus.*\.blob\.core\.windows\.net`), region: RegionUSEast},
	{re: regexp.MustCompile(`(?i)westeurope.*\.blob\.core\.windows\.net`), region: RegionEU},
	{re: regexp.MustCompile(`(?i)uksouth.*\.blob\.core\.windows\.net`), region: RegionUK},
	// GCP CloudSQL
	{re: regexp.MustCompile(`(?i)\.asia-.*\.cloudsql\.goog`), region: RegionAPAC},
	{re: regexp.MustCompile(`(?i)\.europe-.*\.cloudsql\.goog`), region: RegionEU},
	{re: regexp.MustCompile(`(?i)\.us-east.*\.cloudsql\.goog`), region: RegionUSEast},
	{re: regexp.MustCompile(`(?i)\.us-west.*\.cloudsql\.goog`), region: RegionUSWest},
	{re: regexp.MustCompile(`(?i)\.us-central.*\.cloudsql\.goog`), region: RegionUSEast},
	{re: regexp.MustCompile(`(?i)\.northamerica-.*\.cloudsql\.goog`), region: RegionCA},
	{re: regexp.MustCompile(`(?i)\.southamerica-.*\.cloudsql\.goog`), region: RegionLATAM},
	// Snowflake
	{re: regexp.MustCompile(`(?i)\.us-east-[0-9]+\.snowflakecomputing\.com`), region: RegionUSEast},
	{re: regexp.MustCompile(`(?i)\.us-west-[0-9]+\.snowflakecomputing\.com`), region: RegionUSWest},
	{re: regexp.MustCompile(`(?i)\.eu-[a-z].*\.snowflakecomputing\.com`), region: RegionEU},
	{re: regexp.MustCompile(`(?i)\.ap-[a-z].*\.snowflakecomputing\.com`), region: RegionAPAC},
	{re: regexp.MustCompile(`(?i)\.ca-[a-z].*\.snowflakecomputing\.com`), region: RegionCA},
	// Redshift
	{re: regexp.MustCompile(`(?i)\.us-east-[0-9]+\.redshift\.amazonaws\.com`), region: RegionUSEast},
	{re: regexp.MustCompile(`(?i)\.us-west-[0-9]+\.redshift\.amazonaws\.com`), region: RegionUSWest},
	{re: regexp.MustCompile(`(?i)\.eu-[a-z].*\.redshift\.amazonaws\.com`), region: RegionEU},
	{re: regexp.MustCompile(`(?i)\.ap-[a-z].*\.redshift\.amazonaws\.com`), region: RegionAPAC},
}

// awsRegionMap maps AWS region prefixes → standard regions
var awsRegionMap = map[string]string{
	"us-east":    RegionUSEast,
	"us-west":    RegionUSWest,
	"eu-":        RegionEU,
	"ap-":        RegionAPAC,
	"ca-":        RegionCA,
	"sa-":        RegionLATAM,
	"me-":        RegionMEA,
	"af-":        RegionMEA,
}

// azureLocationMap maps Azure location names → standard regions
var azureLocationMap = map[string]string{
	"eastus":           RegionUSEast,
	"eastus2":          RegionUSEast,
	"westus":           RegionUSWest,
	"westus2":          RegionUSWest,
	"centralus":        RegionUSEast,
	"northcentralus":   RegionUSEast,
	"southcentralus":   RegionUSEast,
	"westcentralus":    RegionUSWest,
	"westeurope":       RegionEU,
	"northeurope":      RegionEU,
	"germanywestcentral": RegionEU,
	"francecentral":    RegionEU,
	"swedencentral":    RegionEU,
	"switzerlandnorth": RegionEU,
	"uksouth":          RegionUK,
	"ukwest":           RegionUK,
	"southeastasia":    RegionAPAC,
	"eastasia":         RegionAPAC,
	"australiaeast":    RegionAPAC,
	"japaneast":        RegionAPAC,
	"koreacentral":     RegionAPAC,
	"centralindia":     RegionAPAC,
	"canadacentral":    RegionCA,
	"canadaeast":       RegionCA,
	"brazilsouth":      RegionLATAM,
	"uaenorth":         RegionMEA,
	"uaecentral":       RegionMEA,
	"southafricanorth": RegionMEA,
}

// countryToRegion maps ISO 2-letter country codes → standard regions
var countryToRegion = map[string]string{
	"US": RegionUSEast,
	"GB": RegionUK,
	"DE": RegionEU, "FR": RegionEU, "NL": RegionEU, "IT": RegionEU,
	"ES": RegionEU, "SE": RegionEU, "PL": RegionEU, "BE": RegionEU,
	"AT": RegionEU, "DK": RegionEU, "FI": RegionEU, "IE": RegionEU,
	"PT": RegionEU, "CZ": RegionEU, "RO": RegionEU, "HU": RegionEU,
	"SK": RegionEU, "BG": RegionEU, "HR": RegionEU, "LU": RegionEU,
	"CA": RegionCA,
	"AU": RegionAPAC, "JP": RegionAPAC, "SG": RegionAPAC,
	"IN": RegionAPAC, "CN": RegionAPAC, "KR": RegionAPAC,
	"HK": RegionAPAC, "TW": RegionAPAC, "NZ": RegionAPAC,
	"TH": RegionAPAC, "MY": RegionAPAC, "ID": RegionAPAC,
	"BR": RegionLATAM, "MX": RegionLATAM, "AR": RegionLATAM,
	"CO": RegionLATAM, "CL": RegionLATAM, "PE": RegionLATAM,
	"AE": RegionMEA, "SA": RegionMEA, "ZA": RegionMEA,
	"EG": RegionMEA, "NG": RegionMEA, "KE": RegionMEA,
	"IL": RegionMEA, "TR": RegionMEA, "PK": RegionMEA,
}

// usWestStates contains US state names that map to US-WEST
var usWestStates = []string{"california", "oregon", "washington", "nevada", "arizona"}

// mapCloudRegionToStandard converts a cloud provider region string to a standard region.
func mapCloudRegionToStandard(region string) string {
	r := strings.ToLower(strings.TrimSpace(region))
	if r == "" {
		return ""
	}
	// Azure location names (no hyphens for many)
	if v, ok := azureLocationMap[r]; ok {
		return v
	}
	// AWS/GCP style prefixes
	for prefix, std := range awsRegionMap {
		if strings.HasPrefix(r, prefix) {
			// US special case: California/Oregon/Washington → US-WEST
			if std == RegionUSEast {
				for _, wsuffix := range []string{"us-west", "us-central2"} {
					if strings.HasPrefix(r, wsuffix) {
						return RegionUSWest
					}
				}
			}
			return std
		}
	}
	return ""
}

// DetectRegion attempts to determine the geographic region of a datasource
// using config values, hostname patterns, and optional IP geolocation.
// Returns empty string if region cannot be determined.
func DetectRegion(ctx context.Context, ds *store.DataSource) string {
	var configMap map[string]any
	if len(ds.Config) > 0 {
		if err := json.Unmarshal(ds.Config, &configMap); err != nil {
			configMap = nil
		}
	}

	// 1. Explicit region/location in config
	for _, key := range []string{"region", "location", "cloud_region", "aws_region", "azure_region", "gcp_region"} {
		if v, ok := configMap[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				if mapped := mapCloudRegionToStandard(s); mapped != "" {
					log.Debug().Str("datasource_id", ds.ID).Str("region", mapped).Str("method", "config_key:"+key).Msg("geo_detect: region from config")
					return mapped
				}
			}
		}
	}

	// 2. Snowflake account identifier (contains region in the account string)
	if v, ok := configMap["account"]; ok {
		if account, ok := v.(string); ok && account != "" {
			// e.g. "xy12345.us-east-1" or "orgname-accountname.eu-west-1"
			parts := strings.Split(account, ".")
			for _, part := range parts[1:] {
				if mapped := mapCloudRegionToStandard(part); mapped != "" {
					log.Debug().Str("datasource_id", ds.ID).Str("region", mapped).Str("method", "snowflake_account").Msg("geo_detect: region from snowflake account")
					return mapped
				}
			}
		}
	}

	// 3. Hostname-based detection
	host := extractHost(ds, configMap)
	if host != "" {
		for _, rule := range hostnameRules {
			if rule.re.MatchString(host) {
				log.Debug().Str("datasource_id", ds.ID).Str("region", rule.region).Str("host", host).Str("method", "hostname").Msg("geo_detect: region from hostname")
				return rule.region
			}
		}
	}

	// 4. IP-based geolocation (best-effort, only for public IPs)
	if host != "" {
		if region := geolocateHost(ctx, host); region != "" {
			log.Debug().Str("datasource_id", ds.ID).Str("region", region).Str("host", host).Str("method", "ip_geo").Msg("geo_detect: region from IP geolocation")
			return region
		}
	}

	log.Debug().Str("datasource_id", ds.ID).Str("type", ds.Type).Msg("geo_detect: could not determine region")
	return ""
}

// extractHost returns the hostname from config or connection string
func extractHost(ds *store.DataSource, configMap map[string]any) string {
	// Direct host key
	for _, key := range []string{"host", "hostname", "server", "endpoint"} {
		if v, ok := configMap[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	// Connection string / URL
	for _, key := range []string{"connection_string", "url", "dsn", "connection_url", "jdbc_url"} {
		if v, ok := configMap[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				if h := hostFromURL(s); h != "" {
					return h
				}
			}
		}
	}
	return ""
}

// hostFromURL extracts the hostname portion from a URL/connection string
func hostFromURL(raw string) string {
	// Handle jdbc: prefix
	raw = strings.TrimPrefix(raw, "jdbc:")
	// Must contain "://"
	idx := strings.Index(raw, "://")
	if idx < 0 {
		return ""
	}
	rest := raw[idx+3:]
	// Strip user:pass@
	if at := strings.Index(rest, "@"); at >= 0 {
		rest = rest[at+1:]
	}
	// Strip /path and ?query
	for _, sep := range []byte{'/', '?', '#'} {
		if i := strings.IndexByte(rest, sep); i >= 0 {
			rest = rest[:i]
		}
	}
	// Strip :port
	host, _, err := net.SplitHostPort(rest)
	if err != nil {
		host = rest
	}
	return strings.TrimSpace(host)
}

// ipAPIResponse is the JSON structure from ip-api.com
type ipAPIResponse struct {
	Status      string `json:"status"`
	CountryCode string `json:"countryCode"`
	RegionName  string `json:"regionName"`
}

// geolocateHost resolves the host to an IP and queries ip-api.com
func geolocateHost(ctx context.Context, host string) string {
	// Skip private/local addresses
	ip := net.ParseIP(host)
	if ip == nil {
		// Resolve hostname
		ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		addrs, err := net.DefaultResolver.LookupHost(ctx2, host)
		if err != nil || len(addrs) == 0 {
			return ""
		}
		ip = net.ParseIP(addrs[0])
	}
	if ip == nil || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return ""
	}

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET",
		"http://ip-api.com/json/"+ip.String()+"?fields=status,countryCode,regionName", nil)
	if err != nil {
		return ""
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return ""
	}

	var result ipAPIResponse
	if err := json.Unmarshal(body, &result); err != nil || result.Status != "success" {
		return ""
	}

	region, ok := countryToRegion[result.CountryCode]
	if !ok {
		return ""
	}
	// US special case: West Coast states → US-WEST
	if result.CountryCode == "US" {
		regionLower := strings.ToLower(result.RegionName)
		for _, state := range usWestStates {
			if strings.Contains(regionLower, state) {
				return RegionUSWest
			}
		}
	}
	return region
}
