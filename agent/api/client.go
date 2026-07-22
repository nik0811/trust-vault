package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/securelens/securelens-agent/config"
	"github.com/securelens/securelens-agent/scanner"
)

type Client struct {
	apiURL     string
	apiKey     string
	httpClient *http.Client
	agentID    string
}

type RegisterRequest struct {
	Hostname     string `json:"hostname"`
	IP           string `json:"ip,omitempty"`
	OS           string `json:"os"`
	AgentVersion string `json:"agent_version"`
}

type RegisterResponse struct {
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type ReportRequest struct {
	AgentID      string            `json:"agent_id"`
	ScanResult   *scanner.ScanResult `json:"scan_result"`
	Paths        []string          `json:"paths"`
	Hostname     string            `json:"hostname"`
}

type ReportResponse struct {
	Status    string `json:"status"`
	ReportID  string `json:"report_id,omitempty"`
	Message   string `json:"message,omitempty"`
	ViewURL   string `json:"view_url,omitempty"`
}

type StatusResponse struct {
	AgentID    string    `json:"agent_id"`
	Status     string    `json:"status"`
	LastScan   time.Time `json:"last_scan,omitempty"`
	TotalScans int       `json:"total_scans"`
	Findings   int       `json:"total_findings"`
}

func NewClient(cfg *config.Config) *Client {
	// Normalize API URL to include /api/v1
	apiURL := strings.TrimSuffix(cfg.APIURL, "/")
	if !strings.HasSuffix(apiURL, "/api/v1") {
		apiURL = apiURL + "/api/v1"
	}
	
	return &Client{
		apiURL:  apiURL,
		apiKey:  cfg.APIKey,
		agentID: cfg.AgentID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Register(hostname string) (*RegisterResponse, error) {
	req := RegisterRequest{
		Hostname:     hostname,
		OS:           fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		AgentVersion: "1.0.0",
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.apiURL+"/endpoints/agents/register", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SecureLens API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return &result, nil
}

func (c *Client) Report(result *scanner.ScanResult, paths []string) (*ReportResponse, error) {
	hostname, _ := os.Hostname()
	req := ReportRequest{
		AgentID:    c.agentID,
		ScanResult: result,
		Paths:      paths,
		Hostname:   hostname,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.apiURL+"/endpoints/agents/report", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to upload results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("report upload failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var reportResp ReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&reportResp); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return &reportResp, nil
}

func (c *Client) Status() (*StatusResponse, error) {
	httpReq, err := http.NewRequest("GET", c.apiURL+"/endpoints/agents/"+c.agentID+"/status", nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status check failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return &status, nil
}

func (c *Client) Heartbeat() error {
	httpReq, err := http.NewRequest("POST", c.apiURL+"/endpoints/agents/"+c.agentID+"/heartbeat", nil)
	if err != nil {
		return err
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed (HTTP %d)", resp.StatusCode)
	}

	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("X-Agent-ID", c.agentID)
	req.Header.Set("User-Agent", "SecureLens-Agent/1.0.0")
}
