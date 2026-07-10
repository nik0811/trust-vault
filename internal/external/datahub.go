package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type DataHub struct {
	baseURL string
	client  *http.Client
}

func NewDataHub(baseURL string) *DataHub {
	if baseURL == "" {
		baseURL = envOr("DATAHUB_URL", "http://datahub-gms:8080")
	}
	return &DataHub{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type GraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

func (d *DataHub) GraphQL(ctx context.Context, query string, variables map[string]any, result any) error {
	body, _ := json.Marshal(GraphQLRequest{Query: query, Variables: variables})
	req, err := http.NewRequestWithContext(ctx, "POST", d.baseURL+"/api/graphql", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var gqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return err
	}
	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}
	if result != nil {
		return json.Unmarshal(gqlResp.Data, result)
	}
	return nil
}

func (d *DataHub) GetDataset(ctx context.Context, urn string) (map[string]any, error) {
	query := `query getDataset($urn: String!) {
		dataset(urn: $urn) {
			urn
			name
			platform { name }
			properties { description }
		}
	}`
	var result map[string]any
	err := d.GraphQL(ctx, query, map[string]any{"urn": urn}, &result)
	return result, err
}

func (d *DataHub) EmitLineage(ctx context.Context, event map[string]any) error {
	body, _ := json.Marshal(event)
	req, err := http.NewRequestWithContext(ctx, "POST", d.baseURL+"/openapi/openlineage/api/v1/lineage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("lineage emit failed: %d", resp.StatusCode)
	}
	return nil
}
