package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dirtybirdnj/clood/internal/hosts"
	"gopkg.in/yaml.v3"
)

// Config represents the global clood configuration
type Config struct {
	Hosts    []*hosts.Host `yaml:"hosts"`
	Tiers    TierConfig    `yaml:"tiers"`
	Routing  RoutingConfig `yaml:"routing"`
	Defaults DefaultsConfig `yaml:"defaults"`
}

// TierConfig defines model tiers
type TierConfig struct {
	Fast     TierSettings `yaml:"fast"`
	Deep     TierSettings `yaml:"deep"`
	Analysis TierSettings `yaml:"analysis"` // Reasoning models for code review
	Writing  TierSettings `yaml:"writing"`  // Models for documentation/prose
}

// TierSettings contains settings for a tier
type TierSettings struct {
	Model    string `yaml:"model"`
	Fallback string `yaml:"fallback,omitempty"` // Fallback model if primary unavailable
}

// RoutingConfig defines routing behavior
type RoutingConfig struct {
	Strategy string `yaml:"strategy"` // "fastest", "round-robin", "least-loaded"
	Fallback bool   `yaml:"fallback"` // Try next host if first fails
}

// DefaultsConfig contains default settings
type DefaultsConfig struct {
	Stream  bool   `yaml:"stream"`
	Timeout string `yaml:"timeout"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Hosts: []*hosts.Host{
			{Name: "local-gpu", URL: "http://localhost:11434", Priority: 1, Enabled: true},
			{Name: "local-cpu", URL: "http://localhost:11435", Priority: 2, Enabled: true},
			{Name: "ubuntu25", URL: "http://192.168.4.63:11434", Priority: 1, Enabled: true},
			{Name: "mac-mini", URL: "http://192.168.4.41:11434", Priority: 2, Enabled: true},
		},
		Tiers: TierConfig{
			Fast:     TierSettings{Model: "qwen2.5-coder:3b"},
			Deep:     TierSettings{Model: "qwen2.5-coder:7b"},
			Analysis: TierSettings{Model: "deepseek-r1:14b", Fallback: "llama3.1:8b"},
			Writing:  TierSettings{Model: "llama3.1:8b", Fallback: "mistral:7b"},
		},
		Routing: RoutingConfig{
			Strategy: "fastest",
			Fallback: true,
		},
		Defaults: DefaultsConfig{
			Stream:  true,
			Timeout: "30s",
		},
	}
}

// ConfigDir returns the config directory path
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".clood"
	}
	return filepath.Join(home, ".config", "clood")
}

// ConfigPath returns the full path to the config file
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// Load reads the config from disk, falling back to defaults
func Load() (*Config, error) {
	path := ConfigPath()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return defaults
		return DefaultConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Fill in missing defaults
	if cfg.Tiers.Fast.Model == "" {
		cfg.Tiers.Fast.Model = "qwen2.5-coder:3b"
	}
	if cfg.Tiers.Deep.Model == "" {
		cfg.Tiers.Deep.Model = "qwen2.5-coder:7b"
	}
	if cfg.Tiers.Analysis.Model == "" {
		cfg.Tiers.Analysis.Model = "deepseek-r1:14b"
		cfg.Tiers.Analysis.Fallback = "llama3.1:8b"
	}
	if cfg.Tiers.Writing.Model == "" {
		cfg.Tiers.Writing.Model = "llama3.1:8b"
		cfg.Tiers.Writing.Fallback = "mistral:7b"
	}
	if cfg.Routing.Strategy == "" {
		cfg.Routing.Strategy = "fastest"
	}
	if cfg.Defaults.Timeout == "" {
		cfg.Defaults.Timeout = "30s"
	}

	// Enable hosts by default if not specified
	for _, h := range cfg.Hosts {
		if h.Priority == 0 {
			h.Priority = 10 // Default priority
		}
		// Enabled defaults to true (Go zero value is false, so we handle this specially)
		// The YAML `enabled: false` must be explicit
	}

	return &cfg, nil
}

// Save writes the config to disk
func Save(cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	path := ConfigPath()
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// GetTimeout returns the timeout as a duration
func (c *Config) GetTimeout() time.Duration {
	d, err := time.ParseDuration(c.Defaults.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

// Tier constants
const (
	TierFast     = 1
	TierDeep     = 2
	TierAnalysis = 3
	TierWriting  = 4
)

// GetTierModel returns the model for a given tier
func (c *Config) GetTierModel(tier int) string {
	switch tier {
	case TierFast:
		return c.Tiers.Fast.Model
	case TierDeep:
		return c.Tiers.Deep.Model
	case TierAnalysis:
		return c.Tiers.Analysis.Model
	case TierWriting:
		return c.Tiers.Writing.Model
	default:
		return c.Tiers.Deep.Model
	}
}

// GetTierFallback returns the fallback model for a given tier
func (c *Config) GetTierFallback(tier int) string {
	switch tier {
	case TierAnalysis:
		return c.Tiers.Analysis.Fallback
	case TierWriting:
		return c.Tiers.Writing.Fallback
	default:
		return ""
	}
}

// TierName returns a human-readable name for a tier
func TierName(tier int) string {
	switch tier {
	case TierFast:
		return "Fast"
	case TierDeep:
		return "Deep"
	case TierAnalysis:
		return "Analysis"
	case TierWriting:
		return "Writing"
	default:
		return "Unknown"
	}
}

// Init creates the config directory and default config if they don't exist
func Init() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	path := ConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := DefaultConfig()
		return Save(cfg)
	}

	return nil
}

// Exists returns true if the config file exists
func Exists() bool {
	_, err := os.Stat(ConfigPath())
	return err == nil
}

// WriteExampleConfig writes an example configuration file with comments
func WriteExampleConfig(path string) error {
	example := `# clood configuration
# Location: ~/.config/clood/config.yaml

# Ollama hosts to connect to
hosts:
  - name: local-gpu
    url: http://localhost:11434
    priority: 1      # Lower = higher priority
    enabled: true

  - name: local-cpu
    url: http://localhost:11435
    priority: 2
    enabled: true

  - name: ubuntu25
    url: http://192.168.4.63:11434
    priority: 1
    enabled: true

  - name: mac-mini
    url: http://192.168.4.41:11434
    priority: 2
    enabled: true

# Model tiers for query routing
tiers:
  fast:
    model: qwen2.5-coder:3b    # Quick responses
  deep:
    model: qwen2.5-coder:7b    # Complex reasoning

# Routing behavior
routing:
  strategy: fastest   # Options: fastest, round-robin, least-loaded
  fallback: true      # Try next host if first fails

# Default settings
defaults:
  stream: true        # Stream responses by default
  timeout: 30s        # Request timeout
`
	return os.WriteFile(path, []byte(example), 0644)
}
