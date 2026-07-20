package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// NotificationEvent represents an event that triggers notifications
type NotificationEvent struct {
	Type      string         `json:"type"`
	Severity  string         `json:"severity"`
	Title     string         `json:"title"`
	Message   string         `json:"message"`
	Resource  string         `json:"resource"`
	Details   map[string]any `json:"details"`
	TenantID  string         `json:"tenant_id"`
	Timestamp time.Time      `json:"timestamp"`
}

// IntegrationConfig holds parsed integration configuration
type IntegrationConfig struct {
	ID       string
	Type     string
	Name     string
	Config   map[string]any
	TenantID string
}

// NotificationResult holds the result of sending a notification
type NotificationResult struct {
	IntegrationID string
	Success       bool
	Error         string
	LatencyMs     int64
}

// SendNotification sends a notification through all active integrations for a tenant
func SendNotification(ctx context.Context, integrations []IntegrationConfig, event NotificationEvent) []NotificationResult {
	results := make([]NotificationResult, 0, len(integrations))

	for _, integration := range integrations {
		start := time.Now()
		var err error

		switch integration.Type {
		case "slack":
			err = sendSlackNotification(ctx, integration.Config, event)
		case "teams":
			err = sendTeamsNotification(ctx, integration.Config, event)
		case "email":
			err = sendEmailNotification(ctx, integration.Config, event)
		case "webhook", "rest_api":
			err = sendWebhookNotification(ctx, integration.Config, event)
		case "jira":
			err = sendJiraNotification(ctx, integration.Config, event)
		case "servicenow":
			err = sendServiceNowNotification(ctx, integration.Config, event)
		case "pagerduty":
			err = sendPagerDutyNotification(ctx, integration.Config, event)
		case "splunk", "siem":
			err = sendSplunkNotification(ctx, integration.Config, event)
		default:
			log.Debug().Str("type", integration.Type).Msg("Unsupported integration type for notifications")
			continue
		}

		result := NotificationResult{
			IntegrationID: integration.ID,
			Success:       err == nil,
			LatencyMs:     time.Since(start).Milliseconds(),
		}
		if err != nil {
			result.Error = err.Error()
			log.Warn().Err(err).Str("integration", integration.Name).Str("type", integration.Type).Msg("Failed to send notification")
		}
		results = append(results, result)
	}

	return results
}

func sendSlackNotification(ctx context.Context, config map[string]any, event NotificationEvent) error {
	webhookURL, _ := config["webhook_url"].(string)
	if webhookURL == "" {
		return fmt.Errorf("missing webhook_url")
	}

	color := "#36a64f"
	switch event.Severity {
	case "critical", "high":
		color = "#dc3545"
	case "medium", "warning":
		color = "#ffc107"
	case "low", "info":
		color = "#17a2b8"
	}

	payload := map[string]any{
		"attachments": []map[string]any{
			{
				"color":  color,
				"title":  event.Title,
				"text":   event.Message,
				"footer": "SecureLens",
				"ts":     event.Timestamp.Unix(),
				"fields": []map[string]any{
					{"title": "Type", "value": event.Type, "short": true},
					{"title": "Severity", "value": event.Severity, "short": true},
				},
			},
		},
	}

	return postJSON(ctx, webhookURL, payload, nil)
}

func sendTeamsNotification(ctx context.Context, config map[string]any, event NotificationEvent) error {
	webhookURL, _ := config["webhook_url"].(string)
	if webhookURL == "" {
		return fmt.Errorf("missing webhook_url")
	}

	themeColor := "00FF00"
	switch event.Severity {
	case "critical", "high":
		themeColor = "FF0000"
	case "medium", "warning":
		themeColor = "FFA500"
	case "low", "info":
		themeColor = "0078D7"
	}

	payload := map[string]any{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"themeColor": themeColor,
		"summary":    event.Title,
		"sections": []map[string]any{
			{
				"activityTitle": event.Title,
				"facts": []map[string]any{
					{"name": "Type", "value": event.Type},
					{"name": "Severity", "value": event.Severity},
					{"name": "Resource", "value": event.Resource},
				},
				"text":     event.Message,
				"markdown": true,
			},
		},
	}

	return postJSON(ctx, webhookURL, payload, nil)
}

func sendEmailNotification(ctx context.Context, config map[string]any, event NotificationEvent) error {
	host, _ := config["smtp_host"].(string)
	port := 587
	if p, ok := config["smtp_port"].(float64); ok {
		port = int(p)
	}
	user, _ := config["smtp_user"].(string)
	password, _ := config["smtp_password"].(string)
	from, _ := config["from_address"].(string)
	toStr, _ := config["to_addresses"].(string)

	if host == "" || from == "" || toStr == "" {
		return fmt.Errorf("missing required email configuration")
	}

	to := strings.Split(toStr, ",")
	for i := range to {
		to[i] = strings.TrimSpace(to[i])
	}

	subject := fmt.Sprintf("[SecureLens] %s - %s", strings.ToUpper(event.Severity), event.Title)
	body := fmt.Sprintf(`SecureLens Notification

Type: %s
Severity: %s
Resource: %s

%s

---
This is an automated message from SecureLens.
`, event.Type, event.Severity, event.Resource, event.Message)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, strings.Join(to, ", "), subject, body)

	addr := fmt.Sprintf("%s:%d", host, port)

	var auth smtp.Auth
	if user != "" && password != "" {
		auth = smtp.PlainAuth("", user, password, host)
	}

	return smtp.SendMail(addr, auth, from, to, []byte(msg))
}

func sendWebhookNotification(ctx context.Context, config map[string]any, event NotificationEvent) error {
	url, _ := config["url"].(string)
	if url == "" {
		url, _ = config["webhook_url"].(string)
	}
	if url == "" {
		return fmt.Errorf("missing url")
	}

	method, _ := config["method"].(string)
	if method == "" {
		method = "POST"
	}

	headers := map[string]string{"Content-Type": "application/json"}

	authType, _ := config["auth_type"].(string)
	token, _ := config["token"].(string)
	switch authType {
	case "bearer":
		if token != "" {
			headers["Authorization"] = "Bearer " + token
		}
	case "basic":
		if token != "" {
			headers["Authorization"] = "Basic " + token
		}
	}

	if customHeaders, ok := config["headers"].(string); ok && customHeaders != "" {
		var h map[string]string
		if json.Unmarshal([]byte(customHeaders), &h) == nil {
			for k, v := range h {
				headers[k] = v
			}
		}
	}

	payload := map[string]any{
		"event":     event.Type,
		"severity":  event.Severity,
		"title":     event.Title,
		"message":   event.Message,
		"resource":  event.Resource,
		"details":   event.Details,
		"timestamp": event.Timestamp.Format(time.RFC3339),
	}

	return postJSONWithMethod(ctx, method, url, payload, headers)
}

func sendJiraNotification(ctx context.Context, config map[string]any, event NotificationEvent) error {
	baseURL, _ := config["url"].(string)
	email, _ := config["email"].(string)
	apiToken, _ := config["api_token"].(string)
	projectKey, _ := config["project_key"].(string)
	issueType, _ := config["issue_type"].(string)

	if baseURL == "" || email == "" || apiToken == "" || projectKey == "" {
		return fmt.Errorf("missing required Jira configuration")
	}
	if issueType == "" {
		issueType = "Task"
	}

	priority := "Medium"
	switch event.Severity {
	case "critical":
		priority = "Highest"
	case "high":
		priority = "High"
	case "low":
		priority = "Low"
	}

	payload := map[string]any{
		"fields": map[string]any{
			"project":     map[string]string{"key": projectKey},
			"summary":     fmt.Sprintf("[%s] %s", strings.ToUpper(event.Severity), event.Title),
			"description": fmt.Sprintf("*Type:* %s\n*Severity:* %s\n*Resource:* %s\n\n%s", event.Type, event.Severity, event.Resource, event.Message),
			"issuetype":   map[string]string{"name": issueType},
			"priority":    map[string]string{"name": priority},
		},
	}

	url := strings.TrimSuffix(baseURL, "/") + "/rest/api/3/issue"
	return postJSONWithBasicAuth(ctx, url, payload, email, apiToken)
}

func sendServiceNowNotification(ctx context.Context, config map[string]any, event NotificationEvent) error {
	instance, _ := config["instance"].(string)
	username, _ := config["username"].(string)
	password, _ := config["password"].(string)
	tableName, _ := config["table_name"].(string)

	if instance == "" || username == "" {
		return fmt.Errorf("missing required ServiceNow configuration")
	}
	if tableName == "" {
		tableName = "incident"
	}

	urgency := "2"
	impact := "2"
	switch event.Severity {
	case "critical":
		urgency = "1"
		impact = "1"
	case "high":
		urgency = "1"
		impact = "2"
	case "low":
		urgency = "3"
		impact = "3"
	}

	payload := map[string]any{
		"short_description": fmt.Sprintf("[SecureLens] %s", event.Title),
		"description":       fmt.Sprintf("Type: %s\nSeverity: %s\nResource: %s\n\n%s", event.Type, event.Severity, event.Resource, event.Message),
		"urgency":           urgency,
		"impact":            impact,
		"category":          "Security",
	}

	url := fmt.Sprintf("https://%s.service-now.com/api/now/table/%s", instance, tableName)
	return postJSONWithBasicAuth(ctx, url, payload, username, password)
}

func sendPagerDutyNotification(ctx context.Context, config map[string]any, event NotificationEvent) error {
	routingKey, _ := config["routing_key"].(string)
	if routingKey == "" {
		return fmt.Errorf("missing routing_key")
	}

	severity := "warning"
	switch event.Severity {
	case "critical":
		severity = "critical"
	case "high":
		severity = "error"
	case "low", "info":
		severity = "info"
	}

	payload := map[string]any{
		"routing_key":  routingKey,
		"event_action": "trigger",
		"dedup_key":    fmt.Sprintf("securelens-%s-%s-%d", event.Type, event.Resource, event.Timestamp.Unix()),
		"payload": map[string]any{
			"summary":   event.Title,
			"severity":  severity,
			"source":    "securelens",
			"timestamp": event.Timestamp.Format(time.RFC3339),
			"custom_details": map[string]any{
				"type":     event.Type,
				"message":  event.Message,
				"resource": event.Resource,
			},
		},
	}

	return postJSON(ctx, "https://events.pagerduty.com/v2/enqueue", payload, nil)
}

func sendSplunkNotification(ctx context.Context, config map[string]any, event NotificationEvent) error {
	url, _ := config["url"].(string)
	token, _ := config["token"].(string)
	index, _ := config["index"].(string)

	if url == "" || token == "" {
		return fmt.Errorf("missing url or token")
	}

	payload := map[string]any{
		"event": map[string]any{
			"type":      event.Type,
			"severity":  event.Severity,
			"title":     event.Title,
			"message":   event.Message,
			"resource":  event.Resource,
			"details":   event.Details,
			"timestamp": event.Timestamp.Format(time.RFC3339),
		},
		"sourcetype": "securelens",
		"time":       event.Timestamp.Unix(),
	}
	if index != "" {
		payload["index"] = index
	}

	headers := map[string]string{
		"Authorization": "Splunk " + token,
		"Content-Type":  "application/json",
	}

	return postJSON(ctx, url, payload, headers)
}

func postJSON(ctx context.Context, url string, payload any, headers map[string]string) error {
	return postJSONWithMethod(ctx, "POST", url, payload, headers)
}

func postJSONWithMethod(ctx context.Context, method, url string, payload any, headers map[string]string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if headers == nil {
		headers = map[string]string{}
	}
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/json"
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return fmt.Errorf("request timed out")
		}
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d response", resp.StatusCode)
	}

	return nil
}

func postJSONWithBasicAuth(ctx context.Context, url string, payload any, username, password string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(username, password)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d response", resp.StatusCode)
	}

	return nil
}
