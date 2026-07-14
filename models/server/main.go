package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
	
	requestCount  uint64
	inferenceTime int64 // nanoseconds, atomic
)

func main() {
	port := flag.Int("port", 8085, "HTTP server port")
	modelPath := flag.String("model", "", "Path to ONNX model file (overrides MODEL_PATH env)")
	tokenizerPath := flag.String("tokenizer", "", "Path to tokenizer.json (overrides TOKENIZER_PATH env)")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Configure logging
	level, _ := zerolog.ParseLevel(*logLevel)
	zerolog.SetGlobalLevel(level)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// Get paths from flags or environment
	mPath := *modelPath
	if mPath == "" {
		mPath = os.Getenv("MODEL_PATH")
	}
	if mPath == "" {
		mPath = "/models/gliner-pii-edge-int8.onnx"
	}

	tPath := *tokenizerPath
	if tPath == "" {
		tPath = os.Getenv("TOKENIZER_PATH")
	}
	if tPath == "" {
		tPath = "/models/tokenizer.json"
	}

	log.Info().
		Str("version", version).
		Str("model_path", mPath).
		Str("tokenizer_path", tPath).
		Int("port", *port).
		Msg("Starting SecureLens Classifier Service")

	// Initialize the classifier
	classifier, err := NewClassifier(mPath, tPath)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load ONNX model, falling back to pattern-based classification")
		classifier = NewPatternClassifier()
	}

	// Create HTTP server
	mux := http.NewServeMux()
	
	// Health endpoints
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/health/ready", readyHandler(classifier))
	mux.HandleFunc("/health/live", liveHandler)
	
	// Classification endpoints
	mux.HandleFunc("/classify", classifyHandler(classifier))
	mux.HandleFunc("/classify/batch", batchClassifyHandler(classifier))
	
	// Info endpoints
	mux.HandleFunc("/info", infoHandler(classifier))
	mux.HandleFunc("/metrics", metricsHandler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info().Msg("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Server forced to shutdown")
		}

		if err := classifier.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing classifier")
		}

		close(done)
	}()

	log.Info().Msgf("Server listening on :%d", *port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server failed to start")
	}

	<-done
	log.Info().Msg("Server stopped")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		
		log.Debug().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", wrapped.statusCode).
			Dur("duration", duration).
			Str("remote", r.RemoteAddr).
			Msg("Request completed")
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func readyHandler(c Classifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if c.IsReady() {
			json.NewEncoder(w).Encode(map[string]any{
				"status": "ready",
				"model":  c.ModelInfo(),
			})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not_ready"})
		}
	}
}

func liveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

func infoHandler(c Classifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"version":       version,
			"build_time":    buildTime,
			"model":         c.ModelInfo(),
			"entity_types":  c.SupportedEntities(),
			"request_count": atomic.LoadUint64(&requestCount),
		})
	}
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	avgInferenceNs := atomic.LoadInt64(&inferenceTime)
	count := atomic.LoadUint64(&requestCount)
	
	fmt.Fprintf(w, "# HELP classifier_requests_total Total number of classification requests\n")
	fmt.Fprintf(w, "# TYPE classifier_requests_total counter\n")
	fmt.Fprintf(w, "classifier_requests_total %d\n", count)
	fmt.Fprintf(w, "# HELP classifier_inference_time_nanoseconds Average inference time in nanoseconds\n")
	fmt.Fprintf(w, "# TYPE classifier_inference_time_nanoseconds gauge\n")
	fmt.Fprintf(w, "classifier_inference_time_nanoseconds %d\n", avgInferenceNs)
}

// ClassifyRequest is the input for classification
type ClassifyRequest struct {
	Text        string   `json:"text"`
	TenantID    string   `json:"tenant_id,omitempty"`
	EntityTypes []string `json:"entity_types,omitempty"`
	Threshold   float64  `json:"threshold,omitempty"`
}

// ClassifyResponse is the output from classification
type ClassifyResponse struct {
	Entities      []Entity `json:"entities"`
	ProcessingMs  int64    `json:"processing_ms"`
	ModelUsed     string   `json:"model_used"`
	CharCount     int      `json:"char_count"`
}

// Entity represents a detected entity
type Entity struct {
	Type       string  `json:"type"`
	Value      string  `json:"value"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Confidence float64 `json:"confidence"`
}

func classifyHandler(c Classifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ClassifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON: " + err.Error()})
			return
		}

		if req.Text == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "text field is required"})
			return
		}

		if req.Threshold == 0 {
			req.Threshold = 0.5
		}

		atomic.AddUint64(&requestCount, 1)
		start := time.Now()

		entities, err := c.Classify(r.Context(), req.Text, req.EntityTypes, req.Threshold)
		if err != nil {
			log.Error().Err(err).Msg("Classification failed")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Classification failed: " + err.Error()})
			return
		}

		duration := time.Since(start)
		atomic.StoreInt64(&inferenceTime, duration.Nanoseconds())

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ClassifyResponse{
			Entities:     entities,
			ProcessingMs: duration.Milliseconds(),
			ModelUsed:    c.ModelInfo()["name"].(string),
			CharCount:    len(req.Text),
		})
	}
}

// BatchClassifyRequest is the input for batch classification
type BatchClassifyRequest struct {
	Items     []ClassifyRequest `json:"items"`
	TenantID  string            `json:"tenant_id,omitempty"`
	Threshold float64           `json:"threshold,omitempty"`
}

// BatchClassifyResponse is the output from batch classification
type BatchClassifyResponse struct {
	Results      []ClassifyResponse `json:"results"`
	TotalMs      int64              `json:"total_ms"`
	ItemCount    int                `json:"item_count"`
	TotalChars   int                `json:"total_chars"`
}

func batchClassifyHandler(c Classifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req BatchClassifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON: " + err.Error()})
			return
		}

		if len(req.Items) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "items array is required"})
			return
		}

		if req.Threshold == 0 {
			req.Threshold = 0.5
		}

		start := time.Now()
		results := make([]ClassifyResponse, len(req.Items))
		totalChars := 0

		for i, item := range req.Items {
			if item.Threshold == 0 {
				item.Threshold = req.Threshold
			}
			
			itemStart := time.Now()
			entities, err := c.Classify(r.Context(), item.Text, item.EntityTypes, item.Threshold)
			if err != nil {
				log.Error().Err(err).Int("index", i).Msg("Batch item classification failed")
				entities = []Entity{}
			}

			results[i] = ClassifyResponse{
				Entities:     entities,
				ProcessingMs: time.Since(itemStart).Milliseconds(),
				ModelUsed:    c.ModelInfo()["name"].(string),
				CharCount:    len(item.Text),
			}
			totalChars += len(item.Text)
		}

		atomic.AddUint64(&requestCount, uint64(len(req.Items)))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(BatchClassifyResponse{
			Results:    results,
			TotalMs:    time.Since(start).Milliseconds(),
			ItemCount:  len(req.Items),
			TotalChars: totalChars,
		})
	}
}
