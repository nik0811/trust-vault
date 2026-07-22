package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	APIKey    string   `yaml:"api_key"`
	APIURL    string   `yaml:"api_url"`
	AgentID   string   `yaml:"agent_id,omitempty"`
	Hostname  string   `yaml:"hostname,omitempty"`
	Exclude   []string `yaml:"exclude,omitempty"`
	Interval  string   `yaml:"interval,omitempty"`
	LastScan  string   `yaml:"last_scan,omitempty"`
}

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".securelens", "config.yaml")
}

func EnsureConfigDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".securelens")
	return os.MkdirAll(dir, 0700)
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found at %s - run 'securelens-agent init' first", path)
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Save(path string) error {
	if path == "" {
		path = DefaultConfigPath()
	}
	if err := EnsureConfigDir(); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func Exists(path string) bool {
	if path == "" {
		path = DefaultConfigPath()
	}
	_, err := os.Stat(path)
	return err == nil
}
