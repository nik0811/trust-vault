package main

import "os"

// Config holds the classifier service configuration
type Config struct {
	Port          int
	ModelPath     string
	TokenizerPath string
	LogLevel      string
	
	// Model settings
	MaxSequenceLength int
	DefaultThreshold  float64
	BatchSize         int
	
	// Entity types to detect
	EnabledEntities []string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Port:              8085,
		ModelPath:         "/models/gliner-pii-edge-int8.onnx",
		TokenizerPath:     "/models/tokenizer.json",
		LogLevel:          "info",
		MaxSequenceLength: 512,
		DefaultThreshold:  0.5,
		BatchSize:         32,
		EnabledEntities:   defaultEntityTypes,
	}
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() *Config {
	cfg := DefaultConfig()

	if v := os.Getenv("PORT"); v != "" {
		// Parse port
	}
	if v := os.Getenv("MODEL_PATH"); v != "" {
		cfg.ModelPath = v
	}
	if v := os.Getenv("TOKENIZER_PATH"); v != "" {
		cfg.TokenizerPath = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	return cfg
}

// ModelConfig holds GLiNER model-specific configuration
type ModelConfig struct {
	// Input configuration
	MaxLength     int      `json:"max_length"`
	EntityTypes   []string `json:"entity_types"`
	
	// Output configuration
	NumLabels     int     `json:"num_labels"`
	Threshold     float64 `json:"threshold"`
	
	// Model architecture
	HiddenSize    int `json:"hidden_size"`
	NumLayers     int `json:"num_layers"`
	NumHeads      int `json:"num_heads"`
}

// DefaultModelConfig returns the default GLiNER model configuration
func DefaultModelConfig() *ModelConfig {
	return &ModelConfig{
		MaxLength:   512,
		EntityTypes: defaultEntityTypes,
		NumLabels:   len(defaultEntityTypes),
		Threshold:   0.5,
		HiddenSize:  768,
		NumLayers:   12,
		NumHeads:    12,
	}
}

// EntityTypeConfig defines configuration for a specific entity type
type EntityTypeConfig struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Threshold   float64 `json:"threshold"`
	Enabled     bool    `json:"enabled"`
	Category    string  `json:"category"` // PII, PHI, PCI, etc.
}

// DefaultEntityConfigs returns the default entity type configurations
func DefaultEntityConfigs() map[string]EntityTypeConfig {
	return map[string]EntityTypeConfig{
		"EMAIL": {
			Name:        "EMAIL",
			Description: "Email addresses",
			Threshold:   0.5,
			Enabled:     true,
			Category:    "PII",
		},
		"PHONE": {
			Name:        "PHONE",
			Description: "Phone numbers",
			Threshold:   0.5,
			Enabled:     true,
			Category:    "PII",
		},
		"SSN": {
			Name:        "SSN",
			Description: "Social Security Numbers",
			Threshold:   0.7,
			Enabled:     true,
			Category:    "PII",
		},
		"CREDIT_CARD": {
			Name:        "CREDIT_CARD",
			Description: "Credit card numbers",
			Threshold:   0.7,
			Enabled:     true,
			Category:    "PCI",
		},
		"IP_ADDRESS": {
			Name:        "IP_ADDRESS",
			Description: "IP addresses",
			Threshold:   0.5,
			Enabled:     true,
			Category:    "PII",
		},
		"DATE_OF_BIRTH": {
			Name:        "DATE_OF_BIRTH",
			Description: "Dates of birth",
			Threshold:   0.5,
			Enabled:     true,
			Category:    "PII",
		},
		"PASSPORT": {
			Name:        "PASSPORT",
			Description: "Passport numbers",
			Threshold:   0.6,
			Enabled:     true,
			Category:    "PII",
		},
		"DRIVER_LICENSE": {
			Name:        "DRIVER_LICENSE",
			Description: "Driver's license numbers",
			Threshold:   0.6,
			Enabled:     true,
			Category:    "PII",
		},
		"IBAN": {
			Name:        "IBAN",
			Description: "International Bank Account Numbers",
			Threshold:   0.7,
			Enabled:     true,
			Category:    "PCI",
		},
		"MAC_ADDRESS": {
			Name:        "MAC_ADDRESS",
			Description: "MAC addresses",
			Threshold:   0.5,
			Enabled:     true,
			Category:    "PII",
		},
		"AWS_ACCESS_KEY": {
			Name:        "AWS_ACCESS_KEY",
			Description: "AWS access keys",
			Threshold:   0.8,
			Enabled:     true,
			Category:    "SECRET",
		},
		"JWT_TOKEN": {
			Name:        "JWT_TOKEN",
			Description: "JWT tokens",
			Threshold:   0.8,
			Enabled:     true,
			Category:    "SECRET",
		},
		"VIN": {
			Name:        "VIN",
			Description: "Vehicle Identification Numbers",
			Threshold:   0.6,
			Enabled:     true,
			Category:    "PII",
		},
		"US_ZIP": {
			Name:        "US_ZIP",
			Description: "US ZIP codes",
			Threshold:   0.5,
			Enabled:     true,
			Category:    "PII",
		},
		"UK_POSTCODE": {
			Name:        "UK_POSTCODE",
			Description: "UK postcodes",
			Threshold:   0.5,
			Enabled:     true,
			Category:    "PII",
		},
		"PERSON_NAME": {
			Name:        "PERSON_NAME",
			Description: "Person names",
			Threshold:   0.6,
			Enabled:     true,
			Category:    "PII",
		},
		"ADDRESS": {
			Name:        "ADDRESS",
			Description: "Physical addresses",
			Threshold:   0.6,
			Enabled:     true,
			Category:    "PII",
		},
		"MEDICAL_RECORD": {
			Name:        "MEDICAL_RECORD",
			Description: "Medical record numbers",
			Threshold:   0.7,
			Enabled:     true,
			Category:    "PHI",
		},
		"HEALTH_INSURANCE_ID": {
			Name:        "HEALTH_INSURANCE_ID",
			Description: "Health insurance IDs",
			Threshold:   0.7,
			Enabled:     true,
			Category:    "PHI",
		},
	}
}
