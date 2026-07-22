package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// LLMProxy defines the interface for LLM operations
type LLMProxy interface {
	Chat(ctx context.Context, messages []LLMMessage, opts LLMOptions) (*LLMResponse, error)
	IsHealthy(ctx context.Context) bool
	Provider() string
}

// LLMMessage represents a chat message
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMOptions holds options for LLM requests
type LLMOptions struct {
	Model       string  `json:"model,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
}

// LLMResponse represents an LLM response
type LLMResponse struct {
	ID           string `json:"id"`
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason"`
	Usage        LLMUsage `json:"usage"`
}

// LLMUsage tracks token usage
type LLMUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMConfig holds configuration for LLM connections
type LLMConfig struct {
	Provider    string `json:"provider"` // openai, anthropic, azure, bedrock, custom
	Endpoint    string `json:"endpoint,omitempty"`
	APIKey      string `json:"api_key,omitempty"`
	Model       string `json:"model,omitempty"`
	AzureDeployment string `json:"azure_deployment,omitempty"`
	AWSRegion   string `json:"aws_region,omitempty"`
}

// NewLLMProxy creates an LLM proxy based on configuration
func NewLLMProxy(cfg LLMConfig) (LLMProxy, error) {
	switch strings.ToLower(cfg.Provider) {
	case "openai":
		return NewOpenAIProxy(cfg), nil
	case "anthropic":
		return NewAnthropicProxy(cfg), nil
	case "azure", "azure_openai":
		return NewAzureOpenAIProxy(cfg), nil
	case "bedrock", "aws_bedrock":
		return NewBedrockProxy(cfg), nil
	case "custom", "http":
		return NewCustomLLMProxy(cfg), nil
	case "ollama", "":
		return NewOllamaProxy(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}

// BaseLLMProxy provides common HTTP client functionality
type BaseLLMProxy struct {
	client   *http.Client
	endpoint string
	apiKey   string
	model    string
	provider string
}

func newBaseLLMProxy(endpoint, apiKey, model, provider string) BaseLLMProxy {
	return BaseLLMProxy{
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		endpoint: strings.TrimSuffix(endpoint, "/"),
		apiKey:   apiKey,
		model:    model,
		provider: provider,
	}
}

func (b *BaseLLMProxy) Provider() string {
	return b.provider
}

// OpenAIProxy implements LLMProxy for OpenAI
type OpenAIProxy struct {
	BaseLLMProxy
}

func NewOpenAIProxy(cfg LLMConfig) *OpenAIProxy {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	model := cfg.Model
	if model == "" {
		model = "gpt-4"
	}
	return &OpenAIProxy{
		BaseLLMProxy: newBaseLLMProxy(endpoint, apiKey, model, "openai"),
	}
}

func (o *OpenAIProxy) Chat(ctx context.Context, messages []LLMMessage, opts LLMOptions) (*LLMResponse, error) {
	model := opts.Model
	if model == "" {
		model = o.model
	}
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	reqBody := map[string]any{
		"model":      model,
		"messages":   messages,
		"max_tokens": maxTokens,
	}
	if opts.Temperature > 0 {
		reqBody["temperature"] = opts.Temperature
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID      string `json:"id"`
		Choices []struct {
			Message      LLMMessage `json:"message"`
			FinishReason string     `json:"finish_reason"`
		} `json:"choices"`
		Usage LLMUsage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	content := ""
	finishReason := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
		finishReason = result.Choices[0].FinishReason
	}

	return &LLMResponse{
		ID:           result.ID,
		Content:      content,
		FinishReason: finishReason,
		Usage:        result.Usage,
	}, nil
}

func (o *OpenAIProxy) IsHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", o.endpoint+"/models", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	resp, err := o.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// AnthropicProxy implements LLMProxy for Anthropic Claude
type AnthropicProxy struct {
	BaseLLMProxy
}

func NewAnthropicProxy(cfg LLMConfig) *AnthropicProxy {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.anthropic.com"
	}
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	model := cfg.Model
	if model == "" {
		model = "claude-3-sonnet-20240229"
	}
	return &AnthropicProxy{
		BaseLLMProxy: newBaseLLMProxy(endpoint, apiKey, model, "anthropic"),
	}
}

func (a *AnthropicProxy) Chat(ctx context.Context, messages []LLMMessage, opts LLMOptions) (*LLMResponse, error) {
	model := opts.Model
	if model == "" {
		model = a.model
	}
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	var systemPrompt string
	var chatMessages []map[string]string
	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}
		chatMessages = append(chatMessages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	reqBody := map[string]any{
		"model":      model,
		"messages":   chatMessages,
		"max_tokens": maxTokens,
	}
	if systemPrompt != "" {
		reqBody["system"] = systemPrompt
	}
	if opts.Temperature > 0 {
		reqBody["temperature"] = opts.Temperature
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", a.endpoint+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID      string `json:"id"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	content := ""
	for _, c := range result.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &LLMResponse{
		ID:           result.ID,
		Content:      content,
		FinishReason: result.StopReason,
		Usage: LLMUsage{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		},
	}, nil
}

func (a *AnthropicProxy) IsHealthy(ctx context.Context) bool {
	return a.apiKey != ""
}

// AzureOpenAIProxy implements LLMProxy for Azure OpenAI
type AzureOpenAIProxy struct {
	BaseLLMProxy
	deployment string
}

func NewAzureOpenAIProxy(cfg LLMConfig) *AzureOpenAIProxy {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")
	}
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("AZURE_OPENAI_API_KEY")
	}
	deployment := cfg.AzureDeployment
	if deployment == "" {
		deployment = os.Getenv("AZURE_OPENAI_DEPLOYMENT")
	}
	return &AzureOpenAIProxy{
		BaseLLMProxy: newBaseLLMProxy(endpoint, apiKey, cfg.Model, "azure"),
		deployment:   deployment,
	}
}

func (az *AzureOpenAIProxy) Chat(ctx context.Context, messages []LLMMessage, opts LLMOptions) (*LLMResponse, error) {
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	reqBody := map[string]any{
		"messages":   messages,
		"max_tokens": maxTokens,
	}
	if opts.Temperature > 0 {
		reqBody["temperature"] = opts.Temperature
	}

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=2024-02-15-preview",
		az.endpoint, az.deployment)

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", az.apiKey)

	resp, err := az.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Azure OpenAI API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID      string `json:"id"`
		Choices []struct {
			Message      LLMMessage `json:"message"`
			FinishReason string     `json:"finish_reason"`
		} `json:"choices"`
		Usage LLMUsage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	content := ""
	finishReason := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
		finishReason = result.Choices[0].FinishReason
	}

	return &LLMResponse{
		ID:           result.ID,
		Content:      content,
		FinishReason: finishReason,
		Usage:        result.Usage,
	}, nil
}

func (az *AzureOpenAIProxy) IsHealthy(ctx context.Context) bool {
	return az.endpoint != "" && az.apiKey != "" && az.deployment != ""
}

// BedrockProxy implements LLMProxy for AWS Bedrock
type BedrockProxy struct {
	client *bedrockruntime.Client
	model  string
}

func NewBedrockProxy(cfg LLMConfig) *BedrockProxy {
	region := cfg.AWSRegion
	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1"
		}
	}

	model := cfg.Model
	if model == "" {
		model = "anthropic.claude-3-sonnet-20240229-v1:0"
	}

	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	var awsCfg aws.Config
	var err error

	if accessKey != "" && secretKey != "" {
		awsCfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		)
	} else {
		awsCfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithRegion(region),
		)
	}

	if err != nil {
		return &BedrockProxy{model: model}
	}

	return &BedrockProxy{
		client: bedrockruntime.NewFromConfig(awsCfg),
		model:  model,
	}
}

func (b *BedrockProxy) Chat(ctx context.Context, messages []LLMMessage, opts LLMOptions) (*LLMResponse, error) {
	if b.client == nil {
		return nil, fmt.Errorf("bedrock client not initialized")
	}

	model := opts.Model
	if model == "" {
		model = b.model
	}

	var converseMessages []types.Message
	var systemPrompts []types.SystemContentBlock

	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompts = append(systemPrompts, &types.SystemContentBlockMemberText{
				Value: msg.Content,
			})
			continue
		}

		role := types.ConversationRoleUser
		if msg.Role == "assistant" {
			role = types.ConversationRoleAssistant
		}

		converseMessages = append(converseMessages, types.Message{
			Role: role,
			Content: []types.ContentBlock{
				&types.ContentBlockMemberText{Value: msg.Content},
			},
		})
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	output, err := b.client.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId:  aws.String(model),
		Messages: converseMessages,
		System:   systemPrompts,
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens: aws.Int32(int32(maxTokens)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("bedrock converse error: %w", err)
	}

	var responseText string
	if output.Output != nil {
		if msgOutput, ok := output.Output.(*types.ConverseOutputMemberMessage); ok {
			for _, content := range msgOutput.Value.Content {
				if textContent, ok := content.(*types.ContentBlockMemberText); ok {
					responseText += textContent.Value
				}
			}
		}
	}

	var inputTokens, outputTokens int
	if output.Usage != nil {
		inputTokens = int(aws.ToInt32(output.Usage.InputTokens))
		outputTokens = int(aws.ToInt32(output.Usage.OutputTokens))
	}

	return &LLMResponse{
		ID:           "bedrock-" + model,
		Content:      responseText,
		FinishReason: string(output.StopReason),
		Usage: LLMUsage{
			PromptTokens:     inputTokens,
			CompletionTokens: outputTokens,
			TotalTokens:      inputTokens + outputTokens,
		},
	}, nil
}

func (b *BedrockProxy) IsHealthy(ctx context.Context) bool {
	return b.client != nil
}

func (b *BedrockProxy) Provider() string {
	return "bedrock"
}

// OllamaProxy implements LLMProxy for Ollama (local LLMs)
type OllamaProxy struct {
	BaseLLMProxy
}

func NewOllamaProxy(cfg LLMConfig) *OllamaProxy {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = os.Getenv("LLM_ENDPOINT")
		if endpoint == "" {
			endpoint = "http://localhost:11434/v1"
		}
	}
	model := cfg.Model
	if model == "" {
		model = os.Getenv("LLM_MODEL")
		if model == "" {
			model = "llama3.2"
		}
	}
	return &OllamaProxy{
		BaseLLMProxy: newBaseLLMProxy(endpoint, cfg.APIKey, model, "ollama"),
	}
}

func (o *OllamaProxy) Chat(ctx context.Context, messages []LLMMessage, opts LLMOptions) (*LLMResponse, error) {
	model := opts.Model
	if model == "" {
		model = o.model
	}

	reqBody := map[string]any{
		"model":    model,
		"messages": messages,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID      string `json:"id"`
		Choices []struct {
			Message      LLMMessage `json:"message"`
			FinishReason string     `json:"finish_reason"`
		} `json:"choices"`
		Usage LLMUsage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	content := ""
	finishReason := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
		finishReason = result.Choices[0].FinishReason
	}

	return &LLMResponse{
		ID:           result.ID,
		Content:      content,
		FinishReason: finishReason,
		Usage:        result.Usage,
	}, nil
}

func (o *OllamaProxy) IsHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", strings.TrimSuffix(o.endpoint, "/v1")+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// CustomLLMProxy implements LLMProxy for custom HTTP endpoints
type CustomLLMProxy struct {
	BaseLLMProxy
}

func NewCustomLLMProxy(cfg LLMConfig) *CustomLLMProxy {
	return &CustomLLMProxy{
		BaseLLMProxy: newBaseLLMProxy(cfg.Endpoint, cfg.APIKey, cfg.Model, "custom"),
	}
}

func (c *CustomLLMProxy) Chat(ctx context.Context, messages []LLMMessage, opts LLMOptions) (*LLMResponse, error) {
	model := opts.Model
	if model == "" {
		model = c.model
	}

	reqBody := map[string]any{
		"model":    model,
		"messages": messages,
	}
	if opts.MaxTokens > 0 {
		reqBody["max_tokens"] = opts.MaxTokens
	}
	if opts.Temperature > 0 {
		reqBody["temperature"] = opts.Temperature
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Custom LLM API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID      string `json:"id"`
		Content string `json:"content"`
		Choices []struct {
			Message      LLMMessage `json:"message"`
			FinishReason string     `json:"finish_reason"`
		} `json:"choices"`
		Usage LLMUsage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	content := result.Content
	finishReason := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
		finishReason = result.Choices[0].FinishReason
	}

	return &LLMResponse{
		ID:           result.ID,
		Content:      content,
		FinishReason: finishReason,
		Usage:        result.Usage,
	}, nil
}

func (c *CustomLLMProxy) IsHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint+"/health", nil)
	if err != nil {
		return false
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
