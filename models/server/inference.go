package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	ort "github.com/yalue/onnxruntime_go"
)

// GLiNERSession wraps the ONNX Runtime session for GLiNER inference
type GLiNERSession struct {
	session      *ort.DynamicAdvancedSession
	inputNames   []string
	outputNames  []string
	maxWidth     int
	maxSeqLength int
}

// NewGLiNERSession creates a new ONNX session for GLiNER model
func NewGLiNERSession(modelPath string) (*GLiNERSession, error) {
	// Find and set the ONNX Runtime shared library path
	libPath := findONNXRuntimeLib()
	if libPath == "" {
		return nil, fmt.Errorf("ONNX Runtime shared library not found")
	}
	ort.SetSharedLibraryPath(libPath)

	// Initialize ONNX Runtime environment
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	// Create session options
	options, err := ort.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to create session options: %w", err)
	}
	defer options.Destroy()

	// Set number of threads based on CPU cores
	numThreads := runtime.NumCPU()
	if numThreads > 4 {
		numThreads = 4
	}
	if err := options.SetIntraOpNumThreads(numThreads); err != nil {
		return nil, fmt.Errorf("failed to set thread count: %w", err)
	}

	// GLiNER model input/output names (span-based model)
	inputNames := []string{"input_ids", "attention_mask", "words_mask", "text_lengths", "span_idx", "span_mask"}
	outputNames := []string{"logits"}

	// Create dynamic session
	session, err := ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create ONNX session: %w", err)
	}

	return &GLiNERSession{
		session:      session,
		inputNames:   inputNames,
		outputNames:  outputNames,
		maxWidth:     12,
		maxSeqLength: 512,
	}, nil
}

// findONNXRuntimeLib searches for the ONNX Runtime shared library
func findONNXRuntimeLib() string {
	var libNames []string
	switch runtime.GOOS {
	case "linux":
		libNames = []string{"libonnxruntime.so", "libonnxruntime.so.1"}
	case "darwin":
		libNames = []string{"libonnxruntime.dylib", "libonnxruntime.1.dylib"}
	case "windows":
		libNames = []string{"onnxruntime.dll"}
	}

	searchPaths := []string{
		"/usr/lib",
		"/usr/local/lib",
		"/opt/onnxruntime/lib",
		"/models/lib",
		"./lib",
		os.Getenv("ONNXRUNTIME_LIB_PATH"),
	}

	if ldPath := os.Getenv("LD_LIBRARY_PATH"); ldPath != "" {
		searchPaths = append(searchPaths, strings.Split(ldPath, ":")...)
	}

	for _, dir := range searchPaths {
		if dir == "" {
			continue
		}
		for _, name := range libNames {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	return ""
}

// RunInference performs GLiNER inference on tokenized input
func (s *GLiNERSession) RunInference(
	inputIDs []int64,
	attentionMask []int64,
	wordsMask []int64,
	textLengths []int64,
	spanIdx []int64,
	spanMask []bool,
	batchSize int,
	seqLen int,
	numSpans int,
) ([]float32, []int64, error) {
	// Create input tensors
	inputIDsTensor, err := ort.NewTensor(ort.NewShape(int64(batchSize), int64(seqLen)), inputIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	attentionMaskTensor, err := ort.NewTensor(ort.NewShape(int64(batchSize), int64(seqLen)), attentionMask)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	defer attentionMaskTensor.Destroy()

	wordsMaskTensor, err := ort.NewTensor(ort.NewShape(int64(batchSize), int64(seqLen)), wordsMask)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create words_mask tensor: %w", err)
	}
	defer wordsMaskTensor.Destroy()

	textLengthsTensor, err := ort.NewTensor(ort.NewShape(int64(batchSize), 1), textLengths)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create text_lengths tensor: %w", err)
	}
	defer textLengthsTensor.Destroy()

	spanIdxTensor, err := ort.NewTensor(ort.NewShape(int64(batchSize), int64(numSpans), 2), spanIdx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create span_idx tensor: %w", err)
	}
	defer spanIdxTensor.Destroy()

	// span_mask needs to be bool type - use CustomDataTensor
	spanMaskBytes := make([]byte, len(spanMask))
	for i, v := range spanMask {
		if v {
			spanMaskBytes[i] = 1
		} else {
			spanMaskBytes[i] = 0
		}
	}
	spanMaskTensor, err := ort.NewCustomDataTensor(ort.NewShape(int64(batchSize), int64(numSpans)), spanMaskBytes, ort.TensorElementDataTypeBool)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create span_mask tensor: %w", err)
	}
	defer spanMaskTensor.Destroy()

	// Create output tensor
	outputTensor, err := ort.NewEmptyTensor[float32](ort.NewShape(int64(batchSize), int64(numSpans), int64(len(defaultEntityTypes))))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create output tensor: %w", err)
	}
	defer outputTensor.Destroy()

	// Run inference
	inputs := []ort.Value{inputIDsTensor, attentionMaskTensor, wordsMaskTensor, textLengthsTensor, spanIdxTensor, spanMaskTensor}
	outputs := []ort.Value{outputTensor}

	if err := s.session.Run(inputs, outputs); err != nil {
		return nil, nil, fmt.Errorf("inference failed: %w", err)
	}

	outputData := outputTensor.GetData()
	outputShape := outputTensor.GetShape()

	shapeInt64 := make([]int64, len(outputShape))
	for i, v := range outputShape {
		shapeInt64[i] = v
	}

	return outputData, shapeInt64, nil
}

// Close releases ONNX Runtime resources
func (s *GLiNERSession) Close() error {
	if s.session != nil {
		if err := s.session.Destroy(); err != nil {
			return err
		}
	}
	return ort.DestroyEnvironment()
}

// GLiNERPreprocessor handles input preparation for GLiNER model
type GLiNERPreprocessor struct {
	tokenizer     *Tokenizer
	entityTypes   []string
	maxWidth      int
	maxSeqLength  int
	clsTokenID    int
	sepTokenID    int
	padTokenID    int
	entityStartID int
	entityEndID   int
}

// NewGLiNERPreprocessor creates a preprocessor for GLiNER input
func NewGLiNERPreprocessor(tokenizer *Tokenizer, entityTypes []string) *GLiNERPreprocessor {
	return &GLiNERPreprocessor{
		tokenizer:     tokenizer,
		entityTypes:   entityTypes,
		maxWidth:      12,
		maxSeqLength:  512,
		clsTokenID:    1,
		sepTokenID:    2,
		padTokenID:    0,
		entityStartID: 250103,
		entityEndID:   250104,
	}
}

// PrepareInputs prepares all inputs for GLiNER inference
func (p *GLiNERPreprocessor) PrepareInputs(text string, entityTypes []string) (*GLiNERInputs, error) {
	if len(entityTypes) == 0 {
		entityTypes = p.entityTypes
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil, fmt.Errorf("empty text")
	}

	var inputIDs []int64
	var attentionMask []int64
	var wordsMask []int64
	var wordPositions []int

	// Add [CLS]
	inputIDs = append(inputIDs, int64(p.clsTokenID))
	attentionMask = append(attentionMask, 1)
	wordsMask = append(wordsMask, 0)

	// Add entity type labels with markers
	for _, entityType := range entityTypes {
		inputIDs = append(inputIDs, int64(p.entityStartID))
		attentionMask = append(attentionMask, 1)
		wordsMask = append(wordsMask, 0)

		labelTokens, _ := p.tokenizer.Encode(strings.ToLower(entityType))
		for _, tok := range labelTokens {
			inputIDs = append(inputIDs, int64(tok))
			attentionMask = append(attentionMask, 1)
			wordsMask = append(wordsMask, 0)
		}
	}

	inputIDs = append(inputIDs, int64(p.entityEndID))
	attentionMask = append(attentionMask, 1)
	wordsMask = append(wordsMask, 0)

	// Add text words
	for _, word := range words {
		wordPositions = append(wordPositions, len(inputIDs))

		wordTokens, _ := p.tokenizer.Encode(word)
		if len(wordTokens) == 0 {
			wordTokens = []int{p.tokenizer.unkTokenID}
		}

		for i, tok := range wordTokens {
			inputIDs = append(inputIDs, int64(tok))
			attentionMask = append(attentionMask, 1)
			if i == 0 {
				wordsMask = append(wordsMask, 1)
			} else {
				wordsMask = append(wordsMask, 0)
			}
		}
	}

	inputIDs = append(inputIDs, int64(p.sepTokenID))
	attentionMask = append(attentionMask, 1)
	wordsMask = append(wordsMask, 0)

	seqLen := len(inputIDs)
	numWords := len(words)

	if seqLen < p.maxSeqLength {
		padLen := p.maxSeqLength - seqLen
		for i := 0; i < padLen; i++ {
			inputIDs = append(inputIDs, int64(p.padTokenID))
			attentionMask = append(attentionMask, 0)
			wordsMask = append(wordsMask, 0)
		}
	} else if seqLen > p.maxSeqLength {
		inputIDs = inputIDs[:p.maxSeqLength]
		attentionMask = attentionMask[:p.maxSeqLength]
		wordsMask = wordsMask[:p.maxSeqLength]
		seqLen = p.maxSeqLength
	}

	spanIdx, spanMask := p.generateSpanIndices(numWords)

	return &GLiNERInputs{
		InputIDs:      inputIDs,
		AttentionMask: attentionMask,
		WordsMask:     wordsMask,
		TextLengths:   []int64{int64(numWords)},
		SpanIdx:       spanIdx,
		SpanMask:      spanMask,
		SeqLen:        seqLen,
		NumWords:      numWords,
		NumSpans:      len(spanMask),
		Words:         words,
		WordPositions: wordPositions,
		EntityTypes:   entityTypes,
	}, nil
}

func (p *GLiNERPreprocessor) generateSpanIndices(numWords int) ([]int64, []bool) {
	var spanIdx []int64
	var spanMask []bool

	for start := 0; start < numWords; start++ {
		for width := 1; width <= p.maxWidth; width++ {
			end := start + width
			if end <= numWords {
				spanIdx = append(spanIdx, int64(start), int64(end))
				spanMask = append(spanMask, true)
			} else {
				spanIdx = append(spanIdx, 0, 0)
				spanMask = append(spanMask, false)
			}
		}
	}

	return spanIdx, spanMask
}

// GLiNERInputs holds all prepared inputs for inference
type GLiNERInputs struct {
	InputIDs      []int64
	AttentionMask []int64
	WordsMask     []int64
	TextLengths   []int64
	SpanIdx       []int64
	SpanMask      []bool
	SeqLen        int
	NumWords      int
	NumSpans      int
	Words         []string
	WordPositions []int
	EntityTypes   []string
}

// GLiNERPostprocessor handles output decoding from GLiNER model
type GLiNERPostprocessor struct {
	threshold float64
}

// NewGLiNERPostprocessor creates a postprocessor for GLiNER output
func NewGLiNERPostprocessor(threshold float64) *GLiNERPostprocessor {
	return &GLiNERPostprocessor{threshold: threshold}
}

// DecodeOutput converts model output to entities
func (p *GLiNERPostprocessor) DecodeOutput(
	logits []float32,
	shape []int64,
	inputs *GLiNERInputs,
	originalText string,
) []Entity {
	if len(shape) < 2 {
		return nil
	}

	numSpans := int(shape[1])
	numClasses := len(inputs.EntityTypes)
	if len(shape) >= 3 {
		numClasses = int(shape[2])
	}

	var entities []Entity
	seen := make(map[string]bool)
	maxWidth := 12

	spanIdx := 0
	for wordIdx := 0; wordIdx < inputs.NumWords; wordIdx++ {
		for width := 1; width <= maxWidth; width++ {
			if spanIdx >= numSpans {
				break
			}

			endIdx := wordIdx + width
			if endIdx > inputs.NumWords {
				spanIdx++
				continue
			}

			for classIdx := 0; classIdx < numClasses && classIdx < len(inputs.EntityTypes); classIdx++ {
				logitIdx := spanIdx*numClasses + classIdx
				if logitIdx >= len(logits) {
					continue
				}

				score := sigmoid(logits[logitIdx])
				if float64(score) < p.threshold {
					continue
				}

				entityWords := inputs.Words[wordIdx:endIdx]
				entityText := strings.Join(entityWords, " ")
				entityType := inputs.EntityTypes[classIdx]

				start, end := findTextPosition(originalText, entityText, wordIdx)

				key := fmt.Sprintf("%s:%d:%d", entityType, start, end)
				if seen[key] {
					continue
				}
				seen[key] = true

				entities = append(entities, Entity{
					Type:       entityType,
					Value:      entityText,
					Start:      start,
					End:        end,
					Confidence: float64(score),
				})
			}
			spanIdx++
		}
	}

	return deduplicateEntities(entities)
}

func sigmoid(x float32) float32 {
	return float32(1.0 / (1.0 + math.Exp(-float64(x))))
}

func findTextPosition(originalText, entityText string, wordHint int) (int, int) {
	idx := strings.Index(originalText, entityText)
	if idx >= 0 {
		return idx, idx + len(entityText)
	}

	lowerOrig := strings.ToLower(originalText)
	lowerEntity := strings.ToLower(entityText)
	idx = strings.Index(lowerOrig, lowerEntity)
	if idx >= 0 {
		return idx, idx + len(entityText)
	}

	words := strings.Fields(originalText)
	pos := 0
	for i := 0; i < wordHint && i < len(words); i++ {
		pos += len(words[i]) + 1
	}
	return pos, pos + len(entityText)
}
