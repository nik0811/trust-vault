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
	modelPath      string
	tokenizerPath  string
	tokenizer      *Tokenizer
	session        *GLiNERSession
	preprocessor   *GLiNERPreprocessor
	postprocessor  *GLiNERPostprocessor
	ready          bool
	onnxAvailable  bool
	mu             sync.RWMutex
	modelType      string // "int8" or "fp16"
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
		onnxAvailable: false,
	}

	// Initialize ONNX Runtime session
	if err := classifier.initONNX(); err != nil {
		log.Warn().Err(err).Msg("ONNX Runtime initialization failed, will use pattern matching")
		classifier.onnxAvailable = false
	} else {
		classifier.onnxAvailable = true
	}

	classifier.ready = true
	log.Info().
		Str("model_path", modelPath).
		Str("model_type", modelType).
		Bool("onnx_available", classifier.onnxAvailable).
		Msg("Classifier initialized")

	return classifier, nil
}

func (c *ONNXClassifier) initONNX() error {
	// Create ONNX session
	session, err := NewGLiNERSession(c.modelPath)
	if err != nil {
		return fmt.Errorf("failed to create ONNX session: %w", err)
	}
	c.session = session

	// Create preprocessor with default entity types
	c.preprocessor = NewGLiNERPreprocessor(c.tokenizer, defaultEntityTypes)

	// Create postprocessor with default threshold
	c.postprocessor = NewGLiNERPostprocessor(0.5)

	log.Info().
		Str("model_path", c.modelPath).
		Msg("ONNX Runtime initialized successfully")

	return nil
}

func (c *ONNXClassifier) Classify(ctx context.Context, text string, entityTypes []string, threshold float64) ([]Entity, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.ready {
		return nil, fmt.Errorf("classifier not ready")
	}

	// Use entity types or defaults
	if len(entityTypes) == 0 {
		entityTypes = defaultEntityTypes
	}

	// If ONNX is not available, fall back to pattern matching
	if !c.onnxAvailable || c.session == nil {
		log.Debug().Msg("Using pattern matching (ONNX not available)")
		return runPatternMatching(text, entityTypes, threshold), nil
	}

	// Run ONNX inference
	entities, err := c.runONNXInference(text, entityTypes, threshold)
	if err != nil {
		log.Warn().Err(err).Msg("ONNX inference failed, falling back to pattern matching")
		return runPatternMatching(text, entityTypes, threshold), nil
	}

	// Combine ONNX results with pattern matching for high-confidence patterns
	patternEntities := runPatternMatching(text, entityTypes, threshold)
	entities = mergeEntities(entities, patternEntities)

	return entities, nil
}

func (c *ONNXClassifier) runONNXInference(text string, entityTypes []string, threshold float64) ([]Entity, error) {
	// Prepare inputs
	inputs, err := c.preprocessor.PrepareInputs(text, entityTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare inputs: %w", err)
	}

	// Calculate number of spans
	numSpans := len(inputs.SpanMask)

	// Run inference
	logits, shape, err := c.session.RunInference(
		inputs.InputIDs,
		inputs.AttentionMask,
		inputs.WordsMask,
		inputs.TextLengths,
		inputs.SpanIdx,
		inputs.SpanMask,
		1, // batch size
		len(inputs.InputIDs),
		numSpans,
	)
	if err != nil {
		return nil, fmt.Errorf("inference failed: %w", err)
	}

	// Update postprocessor threshold
	c.postprocessor.threshold = threshold

	// Decode output
	entities := c.postprocessor.DecodeOutput(logits, shape, inputs, text)

	return entities, nil
}

// mergeEntities combines ONNX and pattern-based entities, preferring higher confidence
func mergeEntities(onnxEntities, patternEntities []Entity) []Entity {
	// Create a map of ONNX entities by position
	onnxMap := make(map[string]Entity)
	for _, e := range onnxEntities {
		key := fmt.Sprintf("%d:%d", e.Start, e.End)
		onnxMap[key] = e
	}

	// Add pattern entities that don't overlap with ONNX entities
	for _, pe := range patternEntities {
		key := fmt.Sprintf("%d:%d", pe.Start, pe.End)
		if existing, ok := onnxMap[key]; ok {
			// Keep the one with higher confidence
			if pe.Confidence > existing.Confidence {
				onnxMap[key] = pe
			}
		} else {
			// Check for overlaps
			overlaps := false
			for _, oe := range onnxEntities {
				if pe.Start < oe.End && pe.End > oe.Start {
					overlaps = true
					break
				}
			}
			if !overlaps {
				onnxMap[key] = pe
			}
		}
	}

	// Convert map back to slice
	result := make([]Entity, 0, len(onnxMap))
	for _, e := range onnxMap {
		result = append(result, e)
	}

	return result
}

func (c *ONNXClassifier) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

func (c *ONNXClassifier) ModelInfo() map[string]any {
	inferenceMode := "pattern"
	if c.onnxAvailable {
		inferenceMode = "onnx"
	}
	return map[string]any{
		"name":           "GLiNER PII " + strings.ToUpper(c.modelType),
		"type":           "onnx",
		"quantization":   c.modelType,
		"path":           c.modelPath,
		"tokenizer_path": c.tokenizerPath,
		"inference_mode": inferenceMode,
		"onnx_available": c.onnxAvailable,
	}
}

func (c *ONNXClassifier) SupportedEntities() []string {
	return defaultEntityTypes
}

func (c *ONNXClassifier) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ready = false
	
	if c.session != nil {
		if err := c.session.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing ONNX session")
			return err
		}
	}
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
