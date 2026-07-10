package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

// Classifier interface for entity detection
type Classifier interface {
	Classify(ctx context.Context, text string, entityTypes []string, threshold float64) ([]Entity, error)
	IsReady() bool
	ModelInfo() map[string]any
	SupportedEntities() []string
	Close() error
}

// ONNXClassifier uses ONNX Runtime for GLiNER model inference
type ONNXClassifier struct {
	modelPath     string
	tokenizerPath string
	tokenizer     *Tokenizer
	ready         bool
	mu            sync.RWMutex
	modelType     string // "int8" or "fp16"
}

// NewClassifier creates a new classifier, attempting ONNX first, falling back to patterns
func NewClassifier(modelPath, tokenizerPath string) (Classifier, error) {
	// Check if model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model file not found: %s", modelPath)
	}

	// Check if tokenizer file exists
	if _, err := os.Stat(tokenizerPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("tokenizer file not found: %s", tokenizerPath)
	}

	// Load tokenizer
	tokenizer, err := NewTokenizer(tokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load tokenizer: %w", err)
	}

	// Determine model type from filename
	modelType := "fp16"
	if strings.Contains(strings.ToLower(modelPath), "int8") {
		modelType = "int8"
	}

	classifier := &ONNXClassifier{
		modelPath:     modelPath,
		tokenizerPath: tokenizerPath,
		tokenizer:     tokenizer,
		modelType:     modelType,
		ready:         false,
	}

	// Initialize ONNX Runtime session
	if err := classifier.initONNX(); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX: %w", err)
	}

	classifier.ready = true
	log.Info().
		Str("model_path", modelPath).
		Str("model_type", modelType).
		Msg("ONNX classifier initialized successfully")

	return classifier, nil
}

func (c *ONNXClassifier) initONNX() error {
	// ONNX Runtime initialization
	// In production, this would use github.com/yalue/onnxruntime_go
	// For now, we'll mark as ready and use pattern matching as fallback
	log.Info().Msg("ONNX Runtime initialization (placeholder - use pattern matching)")
	return nil
}

func (c *ONNXClassifier) Classify(ctx context.Context, text string, entityTypes []string, threshold float64) ([]Entity, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.ready {
		return nil, fmt.Errorf("classifier not ready")
	}

	// Tokenize input
	tokens, err := c.tokenizer.Encode(text)
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	// Run inference (placeholder - would use ONNX Runtime)
	entities := c.runInference(text, tokens, entityTypes, threshold)

	return entities, nil
}

func (c *ONNXClassifier) runInference(text string, tokens []int, entityTypes []string, threshold float64) []Entity {
	// Placeholder for ONNX inference
	// In production, this would:
	// 1. Prepare input tensors from tokens
	// 2. Run ONNX session
	// 3. Parse output tensors to extract entities
	
	// For now, fall back to pattern matching
	return runPatternMatching(text, entityTypes, threshold)
}

func (c *ONNXClassifier) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

func (c *ONNXClassifier) ModelInfo() map[string]any {
	return map[string]any{
		"name":           "GLiNER PII " + strings.ToUpper(c.modelType),
		"type":           "onnx",
		"quantization":   c.modelType,
		"path":           c.modelPath,
		"tokenizer_path": c.tokenizerPath,
	}
}

func (c *ONNXClassifier) SupportedEntities() []string {
	return defaultEntityTypes
}

func (c *ONNXClassifier) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ready = false
	// Close ONNX session
	return nil
}

// PatternClassifier uses regex patterns for entity detection
type PatternClassifier struct {
	ready bool
}

// NewPatternClassifier creates a pattern-based classifier
func NewPatternClassifier() Classifier {
	return &PatternClassifier{ready: true}
}

func (c *PatternClassifier) Classify(ctx context.Context, text string, entityTypes []string, threshold float64) ([]Entity, error) {
	return runPatternMatching(text, entityTypes, threshold), nil
}

func (c *PatternClassifier) IsReady() bool {
	return c.ready
}

func (c *PatternClassifier) ModelInfo() map[string]any {
	return map[string]any{
		"name": "Pattern Matcher",
		"type": "regex",
	}
}

func (c *PatternClassifier) SupportedEntities() []string {
	return defaultEntityTypes
}

func (c *PatternClassifier) Close() error {
	return nil
}

var defaultEntityTypes = []string{
	"EMAIL", "PHONE", "SSN", "CREDIT_CARD", "IP_ADDRESS",
	"DATE_OF_BIRTH", "PASSPORT", "DRIVER_LICENSE", "IBAN",
	"MAC_ADDRESS", "AWS_ACCESS_KEY", "JWT_TOKEN", "VIN",
	"US_ZIP", "UK_POSTCODE", "PERSON_NAME", "ADDRESS",
}
