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

// DatasetColumn represents a column in a dataset schema
type DatasetColumn struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Nullable    bool   `json:"nullable"`
}

// GetDatasetSchema fetches the schema (columns) for a dataset from DataHub
func (d *DataHub) GetDatasetSchema(ctx context.Context, urn string) ([]DatasetColumn, error) {
	query := `query getDatasetSchema($urn: String!) {
		dataset(urn: $urn) {
			schemaMetadata {
				fields {
					fieldPath
					nativeDataType
					description
					nullable
				}
			}
		}
	}`
	
	var result struct {
		Dataset struct {
			SchemaMetadata struct {
				Fields []struct {
					FieldPath      string `json:"fieldPath"`
					NativeDataType string `json:"nativeDataType"`
					Description    string `json:"description"`
					Nullable       bool   `json:"nullable"`
				} `json:"fields"`
			} `json:"schemaMetadata"`
		} `json:"dataset"`
	}
	
	err := d.GraphQL(ctx, query, map[string]any{"urn": urn}, &result)
	if err != nil {
		return nil, err
	}
	
	columns := make([]DatasetColumn, 0, len(result.Dataset.SchemaMetadata.Fields))
	for _, f := range result.Dataset.SchemaMetadata.Fields {
		columns = append(columns, DatasetColumn{
			Name:        f.FieldPath,
			Type:        f.NativeDataType,
			Description: f.Description,
			Nullable:    f.Nullable,
		})
	}
	
	return columns, nil
}

// SearchDatasets searches for datasets in DataHub by platform/database
func (d *DataHub) SearchDatasets(ctx context.Context, platform, database string) ([]string, error) {
	query := `query searchDatasets($input: SearchInput!) {
		search(input: $input) {
			searchResults {
				entity {
					urn
				}
			}
		}
	}`
	
	searchQuery := fmt.Sprintf("platform:%s", platform)
	if database != "" {
		searchQuery += fmt.Sprintf(" AND database:%s", database)
	}
	
	var result struct {
		Search struct {
			SearchResults []struct {
				Entity struct {
					URN string `json:"urn"`
				} `json:"entity"`
			} `json:"searchResults"`
		} `json:"search"`
	}
	
	err := d.GraphQL(ctx, query, map[string]any{
		"input": map[string]any{
			"type":  "DATASET",
			"query": searchQuery,
			"start": 0,
			"count": 100,
		},
	}, &result)
	if err != nil {
		return nil, err
	}
	
	urns := make([]string, 0, len(result.Search.SearchResults))
	for _, r := range result.Search.SearchResults {
		urns = append(urns, r.Entity.URN)
	}
	
	return urns, nil
}
