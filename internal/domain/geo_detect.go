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

// RegionInfo holds both the specific cloud region identifier and the country.
type RegionInfo struct {
	Region  string
	Country string
}

// Standard broad region codes used for residency grouping.
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

// awsRegionInfo maps exact AWS region strings to RegionInfo.
var awsRegionInfo = map[string]RegionInfo{
	// AWS AP regions
	"ap-south-1":     {Region: "ap-south-1", Country: "India"},
	"ap-south-2":     {Region: "ap-south-2", Country: "India"},
	"ap-southeast-1": {Region: "ap-southeast-1", Country: "Singapore"},
	"ap-southeast-2": {Region: "ap-southeast-2", Country: "Australia"},
	"ap-southeast-3": {Region: "ap-southeast-3", Country: "Indonesia"},
	"ap-southeast-4": {Region: "ap-southeast-4", Country: "Australia"},
	"ap-northeast-1": {Region: "ap-northeast-1", Country: "Japan"},
	"ap-northeast-2": {Region: "ap-northeast-2", Country: "South Korea"},
	"ap-northeast-3": {Region: "ap-northeast-3", Country: "Japan"},
	"ap-east-1":      {Region: "ap-east-1", Country: "Hong Kong"},
	// AWS US regions
	"us-east-1":      {Region: "us-east-1", Country: "United States"},
	"us-east-2":      {Region: "us-east-2", Country: "United States"},
	"us-west-1":      {Region: "us-west-1", Country: "United States"},
	"us-west-2":      {Region: "us-west-2", Country: "United States"},
	// AWS EU regions
	"eu-west-1":      {Region: "eu-west-1", Country: "Ireland"},
	"eu-west-2":      {Region: "eu-west-2", Country: "United Kingdom"},
	"eu-west-3":      {Region: "eu-west-3", Country: "France"},
	"eu-central-1":   {Region: "eu-central-1", Country: "Germany"},
	"eu-central-2":   {Region: "eu-central-2", Country: "Switzerland"},
	"eu-north-1":     {Region: "eu-north-1", Country: "Sweden"},
	"eu-south-1":     {Region: "eu-south-1", Country: "Italy"},
	"eu-south-2":     {Region: "eu-south-2", Country: "Spain"},
	// AWS CA/SA
	"ca-central-1":   {Region: "ca-central-1", Country: "Canada"},
	"ca-west-1":      {Region: "ca-west-1", Country: "Canada"},
	"sa-east-1":      {Region: "sa-east-1", Country: "Brazil"},
	// AWS ME/AF
	"me-south-1":     {Region: "me-south-1", Country: "UAE"},
	"me-central-1":   {Region: "me-central-1", Country: "UAE"},
	"af-south-1":     {Region: "af-south-1", Country: "South Africa"},
	// AWS IL
	"il-central-1":   {Region: "il-central-1", Country: "Israel"},
}

// hostnameRule maps a regex to a RegionInfo for precise region+country resolution.
type hostnameRule struct {
	re   *regexp.Regexp
	info RegionInfo
}

// awsHostnameRules extracts the exact AWS region slug from hostnames like
// "db.ap-south-1.rds.amazonaws.com" and maps it to a RegionInfo.
// The regex captures the region slug in group 1.
var awsHostnameCapture = regexp.MustCompile(
	`(?i)\.(ap-south(?:east|west)?-\d+|ap-northeast-\d+|ap-east-\d+|` +
		`us-east-\d+|us-west-\d+|` +
		`eu-(?:west|central|north|south)-\d+|` +
		`ca-(?:central|west)-\d+|` +
		`sa-east-\d+|` +
		`me-(?:south|central)-\d+|` +
		`af-south-\d+|` +
		`il-central-\d+)\.`,
)

// hostnameRules handles non-AWS cloud providers that don't encode an exact region slug.
var hostnameRules = []hostnameRule{
	// Azure SQL — map to nearest AWS-style region for consistency
	{re: regexp.MustCompile(`(?i)eastus[0-9]*\.database\.windows\.net`), info: RegionInfo{"us-east-1", "United States"}},
	{re: regexp.MustCompile(`(?i)westus[0-9]*\.database\.windows\.net`), info: RegionInfo{"us-west-2", "United States"}},
	{re: regexp.MustCompile(`(?i)westeurope\.database\.windows\.net`), info: RegionInfo{"eu-west-1", "Ireland"}},
	{re: regexp.MustCompile(`(?i)northeurope\.database\.windows\.net`), info: RegionInfo{"eu-west-1", "Ireland"}},
	{re: regexp.MustCompile(`(?i)uksouth\.database\.windows\.net`), info: RegionInfo{"eu-west-2", "United Kingdom"}},
	{re: regexp.MustCompile(`(?i)ukwest\.database\.windows\.net`), info: RegionInfo{"eu-west-2", "United Kingdom"}},
	{re: regexp.MustCompile(`(?i)uaenorth\.database\.windows\.net`), info: RegionInfo{"me-south-1", "UAE"}},
	{re: regexp.MustCompile(`(?i)southeastasia\.database\.windows\.net`), info: RegionInfo{"ap-southeast-1", "Singapore"}},
	{re: regexp.MustCompile(`(?i)eastasia\.database\.windows\.net`), info: RegionInfo{"ap-east-1", "Hong Kong"}},
	{re: regexp.MustCompile(`(?i)centralindia\.database\.windows\.net`), info: RegionInfo{"ap-south-1", "India"}},
	{re: regexp.MustCompile(`(?i)canadacentral\.database\.windows\.net`), info: RegionInfo{"ca-central-1", "Canada"}},
	{re: regexp.MustCompile(`(?i)brazilsouth\.database\.windows\.net`), info: RegionInfo{"sa-east-1", "Brazil"}},
	// Azure Blob
	{re: regexp.MustCompile(`(?i)eastus[0-9]*\.blob\.core\.windows\.net`), info: RegionInfo{"us-east-1", "United States"}},
	{re: regexp.MustCompile(`(?i)westeurope\.blob\.core\.windows\.net`), info: RegionInfo{"eu-west-1", "Ireland"}},
	{re: regexp.MustCompile(`(?i)uksouth\.blob\.core\.windows\.net`), info: RegionInfo{"eu-west-2", "United Kingdom"}},
	// GCP CloudSQL
	{re: regexp.MustCompile(`(?i)\.asia-southeast[0-9]*\.cloudsql\.goog`), info: RegionInfo{"ap-southeast-1", "Singapore"}},
	{re: regexp.MustCompile(`(?i)\.asia-northeast[0-9]*\.cloudsql\.goog`), info: RegionInfo{"ap-northeast-1", "Japan"}},
	{re: regexp.MustCompile(`(?i)\.asia-south[0-9]*\.cloudsql\.goog`), info: RegionInfo{"ap-south-1", "India"}},
	{re: regexp.MustCompile(`(?i)\.asia-east[0-9]*\.cloudsql\.goog`), info: RegionInfo{"ap-east-1", "Hong Kong"}},
	{re: regexp.MustCompile(`(?i)\.europe-west[0-9]*\.cloudsql\.goog`), info: RegionInfo{"eu-west-1", "Ireland"}},
	{re: regexp.MustCompile(`(?i)\.europe-central[0-9]*\.cloudsql\.goog`), info: RegionInfo{"eu-central-1", "Germany"}},
	{re: regexp.MustCompile(`(?i)\.europe-north[0-9]*\.cloudsql\.goog`), info: RegionInfo{"eu-north-1", "Sweden"}},
	{re: regexp.MustCompile(`(?i)\.us-east[0-9]*\.cloudsql\.goog`), info: RegionInfo{"us-east-1", "United States"}},
	{re: regexp.MustCompile(`(?i)\.us-west[0-9]*\.cloudsql\.goog`), info: RegionInfo{"us-west-2", "United States"}},
	{re: regexp.MustCompile(`(?i)\.us-central[0-9]*\.cloudsql\.goog`), info: RegionInfo{"us-east-1", "United States"}},
	{re: regexp.MustCompile(`(?i)\.northamerica-.*\.cloudsql\.goog`), info: RegionInfo{"ca-central-1", "Canada"}},
	{re: regexp.MustCompile(`(?i)\.southamerica-.*\.cloudsql\.goog`), info: RegionInfo{"sa-east-1", "Brazil"}},
	// Snowflake account URLs
	{re: regexp.MustCompile(`(?i)\.us-east-[0-9]+\.snowflakecomputing\.com`), info: RegionInfo{"us-east-1", "United States"}},
	{re: regexp.MustCompile(`(?i)\.us-west-[0-9]+\.snowflakecomputing\.com`), info: RegionInfo{"us-west-2", "United States"}},
	{re: regexp.MustCompile(`(?i)\.eu-west-[0-9]+\.snowflakecomputing\.com`), info: RegionInfo{"eu-west-1", "Ireland"}},
	{re: regexp.MustCompile(`(?i)\.eu-central-[0-9]+\.snowflakecomputing\.com`), info: RegionInfo{"eu-central-1", "Germany"}},
	{re: regexp.MustCompile(`(?i)\.ap-southeast-[0-9]+\.snowflakecomputing\.com`), info: RegionInfo{"ap-southeast-1", "Singapore"}},
	{re: regexp.MustCompile(`(?i)\.ap-south-[0-9]+\.snowflakecomputing\.com`), info: RegionInfo{"ap-south-1", "India"}},
	{re: regexp.MustCompile(`(?i)\.ap-northeast-[0-9]+\.snowflakecomputing\.com`), info: RegionInfo{"ap-northeast-1", "Japan"}},
	{re: regexp.MustCompile(`(?i)\.ca-central-[0-9]+\.snowflakecomputing\.com`), info: RegionInfo{"ca-central-1", "Canada"}},
	// Redshift
	{re: regexp.MustCompile(`(?i)\.us-east-[0-9]+\.redshift\.amazonaws\.com`), info: RegionInfo{"us-east-1", "United States"}},
	{re: regexp.MustCompile(`(?i)\.us-west-[0-9]+\.redshift\.amazonaws\.com`), info: RegionInfo{"us-west-2", "United States"}},
	{re: regexp.MustCompile(`(?i)\.eu-west-[0-9]+\.redshift\.amazonaws\.com`), info: RegionInfo{"eu-west-1", "Ireland"}},
	{re: regexp.MustCompile(`(?i)\.eu-central-[0-9]+\.redshift\.amazonaws\.com`), info: RegionInfo{"eu-central-1", "Germany"}},
	{re: regexp.MustCompile(`(?i)\.ap-south-[0-9]+\.redshift\.amazonaws\.com`), info: RegionInfo{"ap-south-1", "India"}},
	{re: regexp.MustCompile(`(?i)\.ap-southeast-[0-9]+\.redshift\.amazonaws\.com`), info: RegionInfo{"ap-southeast-1", "Singapore"}},
	{re: regexp.MustCompile(`(?i)\.ap-northeast-[0-9]+\.redshift\.amazonaws\.com`), info: RegionInfo{"ap-northeast-1", "Japan"}},
	{re: regexp.MustCompile(`(?i)\.me-south-[0-9]+\.redshift\.amazonaws\.com`), info: RegionInfo{"me-south-1", "UAE"}},
}

// azureLocationMap maps Azure location names → RegionInfo.
var azureLocationMap = map[string]RegionInfo{
	"eastus":              {"us-east-1", "United States"},
	"eastus2":             {"us-east-1", "United States"},
	"westus":              {"us-west-2", "United States"},
	"westus2":             {"us-west-2", "United States"},
	"centralus":           {"us-east-1", "United States"},
	"northcentralus":      {"us-east-1", "United States"},
	"southcentralus":      {"us-east-1", "United States"},
	"westcentralus":       {"us-west-2", "United States"},
	"westeurope":          {"eu-west-1", "Ireland"},
	"northeurope":         {"eu-west-1", "Ireland"},
	"germanywestcentral":  {"eu-central-1", "Germany"},
	"francecentral":       {"eu-west-3", "France"},
	"swedencentral":       {"eu-north-1", "Sweden"},
	"switzerlandnorth":    {"eu-central-2", "Switzerland"},
	"uksouth":             {"eu-west-2", "United Kingdom"},
	"ukwest":              {"eu-west-2", "United Kingdom"},
	"southeastasia":       {"ap-southeast-1", "Singapore"},
	"eastasia":            {"ap-east-1", "Hong Kong"},
	"australiaeast":       {"ap-southeast-2", "Australia"},
	"japaneast":           {"ap-northeast-1", "Japan"},
	"koreacentral":        {"ap-northeast-2", "South Korea"},
	"centralindia":        {"ap-south-1", "India"},
	"southindia":          {"ap-south-1", "India"},
	"westindia":           {"ap-south-1", "India"},
	"canadacentral":       {"ca-central-1", "Canada"},
	"canadaeast":          {"ca-central-1", "Canada"},
	"brazilsouth":         {"sa-east-1", "Brazil"},
	"uaenorth":            {"me-south-1", "UAE"},
	"uaecentral":          {"me-south-1", "UAE"},
	"southafricanorth":    {"af-south-1", "South Africa"},
}

// countryCodeToCountryName maps ISO 2-letter country codes → full country name.
var countryCodeToCountryName = map[string]string{
	"US": "United States",
	"GB": "United Kingdom",
	"DE": "Germany",
	"FR": "France",
	"NL": "Netherlands",
	"IT": "Italy",
	"ES": "Spain",
	"SE": "Sweden",
	"IE": "Ireland",
	"CA": "Canada",
	"AU": "Australia",
	"JP": "Japan",
	"SG": "Singapore",
	"IN": "India",
	"CN": "China",
	"KR": "South Korea",
	"HK": "Hong Kong",
	"BR": "Brazil",
	"AE": "UAE",
	"SA": "Saudi Arabia",
	"ZA": "South Africa",
	"IL": "Israel",
}

// mapCloudRegionToInfo converts a cloud provider region string to a RegionInfo.
// It checks the exact awsRegionInfo map first, then the azureLocationMap.
func mapCloudRegionToInfo(region string) RegionInfo {
	r := strings.ToLower(strings.TrimSpace(region))
	if r == "" {
		return RegionInfo{}
	}
	// Exact AWS region match
	if info, ok := awsRegionInfo[r]; ok {
		return info
	}
	// Azure location name
	if info, ok := azureLocationMap[r]; ok {
		return info
	}
	return RegionInfo{}
}

// DetectRegionInfo attempts to determine the specific region and country of a datasource
// using config values, hostname patterns, and optional IP geolocation.
// Returns zero-value RegionInfo if region cannot be determined.
func DetectRegionInfo(ctx context.Context, ds *store.DataSource) RegionInfo {
	var configMap map[string]any
	if len(ds.Config) > 0 {
		if err := json.Unmarshal(ds.Config, &configMap); err != nil {
			configMap = nil
		}
	}

	// 1. Explicit region/location key in config
	for _, key := range []string{"region", "location", "cloud_region", "aws_region", "azure_region", "gcp_region"} {
		if v, ok := configMap[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				if info := mapCloudRegionToInfo(s); info.Region != "" {
					log.Debug().Str("datasource_id", ds.ID).Str("region", info.Region).Str("method", "config_key:"+key).Msg("geo_detect: region from config")
					return info
				}
			}
		}
	}

	// 2. Snowflake account identifier (encodes region, e.g. "xy12345.ap-south-1")
	if v, ok := configMap["account"]; ok {
		if account, ok := v.(string); ok && account != "" {
			for _, part := range strings.Split(account, ".")[1:] {
				if info := mapCloudRegionToInfo(part); info.Region != "" {
					log.Debug().Str("datasource_id", ds.ID).Str("region", info.Region).Str("method", "snowflake_account").Msg("geo_detect: region from snowflake account")
					return info
				}
			}
		}
	}

	// 3. Hostname-based detection
	host := extractHost(ds, configMap)
	if host != "" {
		// 3a. Try to capture an exact AWS region slug from the hostname
		if m := awsHostnameCapture.FindStringSubmatch(host); len(m) >= 2 {
			slug := strings.ToLower(m[1])
			if info, ok := awsRegionInfo[slug]; ok {
				log.Debug().Str("datasource_id", ds.ID).Str("region", info.Region).Str("host", host).Str("method", "hostname_aws_slug").Msg("geo_detect: region from hostname AWS slug")
				return info
			}
		}
		// 3b. Non-AWS cloud provider rules
		for _, rule := range hostnameRules {
			if rule.re.MatchString(host) {
				log.Debug().Str("datasource_id", ds.ID).Str("region", rule.info.Region).Str("host", host).Str("method", "hostname_rule").Msg("geo_detect: region from hostname rule")
				return rule.info
			}
		}
		// 3c. keyword hints in the hostname itself
		hostLower := strings.ToLower(host)
		switch {
		case strings.Contains(hostLower, "mumbai") || strings.Contains(hostLower, "ap-south"):
			return RegionInfo{"ap-south-1", "India"}
		case strings.Contains(hostLower, "dubai") || strings.Contains(hostLower, "uae") || strings.Contains(hostLower, "me-south"):
			return RegionInfo{"me-south-1", "UAE"}
		case strings.Contains(hostLower, "singapore") || strings.Contains(hostLower, "ap-southeast-1"):
			return RegionInfo{"ap-southeast-1", "Singapore"}
		case strings.Contains(hostLower, "tokyo") || strings.Contains(hostLower, "ap-northeast-1"):
			return RegionInfo{"ap-northeast-1", "Japan"}
		case strings.Contains(hostLower, "sydney") || strings.Contains(hostLower, "ap-southeast-2"):
			return RegionInfo{"ap-southeast-2", "Australia"}
		case strings.Contains(hostLower, "london") || strings.Contains(hostLower, "eu-west-2"):
			return RegionInfo{"eu-west-2", "United Kingdom"}
		case strings.Contains(hostLower, "frankfurt") || strings.Contains(hostLower, "eu-central-1"):
			return RegionInfo{"eu-central-1", "Germany"}
		case strings.Contains(hostLower, "ireland") || strings.Contains(hostLower, "eu-west-1"):
			return RegionInfo{"eu-west-1", "Ireland"}
		case strings.Contains(hostLower, "virginia") || strings.Contains(hostLower, "us-east-1"):
			return RegionInfo{"us-east-1", "United States"}
		case strings.Contains(hostLower, "oregon") || strings.Contains(hostLower, "us-west-2"):
			return RegionInfo{"us-west-2", "United States"}
		}
	}

	// 4. IP-based geolocation (best-effort, public IPs only)
	if host != "" {
		if info := geolocateHostInfo(ctx, host); info.Region != "" {
			log.Debug().Str("datasource_id", ds.ID).Str("region", info.Region).Str("host", host).Str("method", "ip_geo").Msg("geo_detect: region from IP geolocation")
			return info
		}
	}

	log.Debug().Str("datasource_id", ds.ID).Str("type", ds.Type).Msg("geo_detect: could not determine region")
	return RegionInfo{}
}

// DetectRegion is a backward-compatible wrapper that returns only the region string.
func DetectRegion(ctx context.Context, ds *store.DataSource) string {
	return DetectRegionInfo(ctx, ds).Region
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
	Country     string `json:"country"`
	RegionName  string `json:"regionName"`
}

// geolocateHostInfo resolves the host to an IP, queries ip-api.com, and returns RegionInfo.
func geolocateHostInfo(ctx context.Context, host string) RegionInfo {
	ip := net.ParseIP(host)
	if ip == nil {
		ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		addrs, err := net.DefaultResolver.LookupHost(ctx2, host)
		if err != nil || len(addrs) == 0 {
			return RegionInfo{}
		}
		ip = net.ParseIP(addrs[0])
	}
	if ip == nil || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return RegionInfo{}
	}

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET",
		"http://ip-api.com/json/"+ip.String()+"?fields=status,countryCode,country,regionName", nil)
	if err != nil {
		return RegionInfo{}
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return RegionInfo{}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return RegionInfo{}
	}

	var result ipAPIResponse
	if err := json.Unmarshal(body, &result); err != nil || result.Status != "success" {
		return RegionInfo{}
	}

	countryName := result.Country
	if name, ok := countryCodeToCountryName[result.CountryCode]; ok {
		countryName = name
	}

	// Use a broad fallback region slug derived from country code
	regionSlug := countryCodeToRegionSlug(result.CountryCode, result.RegionName)
	return RegionInfo{Region: regionSlug, Country: countryName}
}

// countryCodeToRegionSlug maps an ISO country code and region name to a region slug.
func countryCodeToRegionSlug(countryCode, regionName string) string {
	switch countryCode {
	case "US":
		regionLower := strings.ToLower(regionName)
		for _, state := range []string{"california", "oregon", "washington", "nevada", "arizona"} {
			if strings.Contains(regionLower, state) {
				return "us-west-2"
			}
		}
		return "us-east-1"
	case "GB":
		return "eu-west-2"
	case "IE":
		return "eu-west-1"
	case "DE":
		return "eu-central-1"
	case "FR":
		return "eu-west-3"
	case "SE":
		return "eu-north-1"
	case "CA":
		return "ca-central-1"
	case "BR":
		return "sa-east-1"
	case "SG":
		return "ap-southeast-1"
	case "AU":
		return "ap-southeast-2"
	case "JP":
		return "ap-northeast-1"
	case "KR":
		return "ap-northeast-2"
	case "IN":
		return "ap-south-1"
	case "HK":
		return "ap-east-1"
	case "AE":
		return "me-south-1"
	case "ZA":
		return "af-south-1"
	case "IL":
		return "il-central-1"
	default:
		return ""
	}
}
