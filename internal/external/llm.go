package external

import (
	"bufio"
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
)

type LLMProvider string

const (
	LLMProviderOllama  LLMProvider = "ollama"
	LLMProviderBedrock LLMProvider = "bedrock"
)

type LLM struct {
	endpoint       string
	apiKey         string
	model          string
	provider       LLMProvider
	client         *http.Client
	bedrockClient  *bedrockruntime.Client
}

func NewLLM(endpoint, apiKey, model string) *LLM {
	provider := LLMProvider(os.Getenv("LLM_PROVIDER"))
	if provider == "" {
		provider = LLMProviderOllama
	}

	llm := &LLM{
		endpoint: endpoint,
		apiKey:   apiKey,
		model:    model,
		provider: provider,
		client:   &http.Client{Timeout: 120 * time.Second},
	}

	if provider == LLMProviderBedrock {
		llm.initBedrockClient()
	}

	return llm
}

func (l *LLM) initBedrockClient() {
	region := os.Getenv("AWS_REGION_NAME")
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}
	if region == "" {
		region = "us-east-1"
	}

	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	var cfg aws.Config
	var err error

	if accessKey != "" && secretKey != "" {
		cfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithRegion(region),
		)
	}

	if err != nil {
		return
	}

	l.bedrockClient = bedrockruntime.NewFromConfig(cfg)
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type ChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type StreamChunk struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func (l *LLM) Chat(ctx context.Context, messages []ChatMessage) (*ChatResponse, error) {
	if l.provider == LLMProviderBedrock {
		return l.chatBedrock(ctx, messages)
	}
	return l.chatOllama(ctx, messages)
}

func (l *LLM) chatOllama(ctx context.Context, messages []ChatMessage) (*ChatResponse, error) {
	body, _ := json.Marshal(ChatRequest{
		Model:    l.model,
		Messages: messages,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", l.endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if l.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+l.apiKey)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM error %d: %s", resp.StatusCode, string(body))
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

type BedrockMessage struct {
	Role    string           `json:"role"`
	Content []BedrockContent `json:"content"`
}

type BedrockContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type BedrockRequest struct {
	AnthropicVersion string           `json:"anthropic_version"`
	MaxTokens        int              `json:"max_tokens"`
	System           string           `json:"system,omitempty"`
	Messages         []BedrockMessage `json:"messages"`
}

type BedrockResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
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

func (l *LLM) chatBedrock(ctx context.Context, messages []ChatMessage) (*ChatResponse, error) {
	if l.bedrockClient == nil {
		return nil, fmt.Errorf("bedrock client not initialized")
	}

	var systemPrompt string
	var bedrockMessages []BedrockMessage

	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}
		bedrockMessages = append(bedrockMessages, BedrockMessage{
			Role: msg.Role,
			Content: []BedrockContent{
				{Type: "text", Text: msg.Content},
			},
		})
	}

	bedrockReq := BedrockRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        4096,
		System:           systemPrompt,
		Messages:         bedrockMessages,
	}

	body, err := json.Marshal(bedrockReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bedrock request: %w", err)
	}

	modelID := l.model
	if strings.HasPrefix(modelID, "bedrock/") {
		modelID = strings.TrimPrefix(modelID, "bedrock/")
	}

	output, err := l.bedrockClient.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return nil, fmt.Errorf("bedrock invoke error: %w", err)
	}

	var bedrockResp BedrockResponse
	if err := json.Unmarshal(output.Body, &bedrockResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bedrock response: %w", err)
	}

	var responseText string
	for _, content := range bedrockResp.Content {
		if content.Type == "text" {
			responseText += content.Text
		}
	}

	return &ChatResponse{
		ID: bedrockResp.ID,
		Choices: []struct {
			Message      ChatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		}{
			{
				Message:      ChatMessage{Role: "assistant", Content: responseText},
				FinishReason: bedrockResp.StopReason,
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     bedrockResp.Usage.InputTokens,
			CompletionTokens: bedrockResp.Usage.OutputTokens,
			TotalTokens:      bedrockResp.Usage.InputTokens + bedrockResp.Usage.OutputTokens,
		},
	}, nil
}

func (l *LLM) ChatStream(ctx context.Context, messages []ChatMessage, onChunk func(string)) error {
	body, _ := json.Marshal(ChatRequest{
		Model:    l.model,
		Messages: messages,
		Stream:   true,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", l.endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if l.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+l.apiKey)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("LLM error %d: %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 6 || line[:6] != "data: " {
			continue
		}
		data := line[6:]
		if data == "[DONE]" {
			break
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onChunk(chunk.Choices[0].Delta.Content)
		}
	}
	return scanner.Err()
}
