package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Agent represents a configured agent role
type Agent struct {
	Name        string  `yaml:"-"` // Set from map key
	Description string  `yaml:"description,omitempty"`
	Model       string  `yaml:"model,omitempty"`
	Host        string  `yaml:"host,omitempty"`
	System      string  `yaml:"system,omitempty"`
	Temperature float64 `yaml:"temperature,omitempty"`
	MaxTokens   int     `yaml:"max_tokens,omitempty"`
	Timeout     string  `yaml:"timeout,omitempty"`
}

// Defaults contains default settings for all agents
type Defaults struct {
	Timeout   string `yaml:"timeout,omitempty"`
	MaxTokens int    `yaml:"max_tokens,omitempty"`
}

// AgentConfig is the root config structure
type AgentConfig struct {
	Agents   map[string]*Agent `yaml:"agents"`
	Defaults Defaults          `yaml:"defaults,omitempty"`
}

// ParsedTimeout returns the timeout as a duration
func (a *Agent) ParsedTimeout() time.Duration {
	if a.Timeout == "" {
		return 120 * time.Second // default
	}
	d, err := time.ParseDuration(a.Timeout)
	if err != nil {
		return 120 * time.Second
	}
	return d
}

// GetEffectiveTemperature returns temperature or default
func (a *Agent) GetEffectiveTemperature() float64 {
	if a.Temperature > 0 {
		return a.Temperature
	}
	return 0.7 // sensible default
}

// LoadConfig loads agent configuration from project or global location
func LoadConfig() (*AgentConfig, error) {
	// Search order:
	// 1. .clood/agents.yaml (project-level)
	// 2. ~/.config/clood/agents.yaml (global)

	// Try project-level first
	projectPath := ".clood/agents.yaml"
	if cfg, err := loadFromPath(projectPath); err == nil {
		return cfg, nil
	}

	// Try global config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}

	globalPath := filepath.Join(homeDir, ".config", "clood", "agents.yaml")
	if cfg, err := loadFromPath(globalPath); err == nil {
		return cfg, nil
	}

	// Return empty config (no agents defined)
	return &AgentConfig{
		Agents: make(map[string]*Agent),
	}, nil
}

// LoadConfigWithFallback loads config and returns defaults if none found
func LoadConfigWithFallback() *AgentConfig {
	cfg, _ := LoadConfig()
	if cfg == nil {
		cfg = &AgentConfig{
			Agents: make(map[string]*Agent),
		}
	}

	// Add built-in default agents if none defined
	if len(cfg.Agents) == 0 {
		cfg.Agents = defaultAgents()
	}

	// Set names from map keys
	for name, agent := range cfg.Agents {
		agent.Name = name
	}

	return cfg
}

// loadFromPath attempts to load config from a specific path
func loadFromPath(path string) (*AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg AgentConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	// Set names from map keys
	for name, agent := range cfg.Agents {
		agent.Name = name
	}

	return &cfg, nil
}

// GetAgent retrieves an agent by name
func (c *AgentConfig) GetAgent(name string) *Agent {
	if agent, ok := c.Agents[name]; ok {
		return agent
	}
	return nil
}

// ListAgents returns all agent names
func (c *AgentConfig) ListAgents() []string {
	names := make([]string, 0, len(c.Agents))
	for name := range c.Agents {
		names = append(names, name)
	}
	return names
}

// defaultAgents returns built-in default agent configurations
func defaultAgents() map[string]*Agent {
	return map[string]*Agent{
		"reviewer": {
			Name:        "reviewer",
			Description: "Code review specialist",
			System: `You are a code reviewer. Analyze code for:
- Bugs and edge cases
- Security vulnerabilities
- Performance issues
- Style and readability
Return findings as structured bullet points.`,
			Temperature: 0.3,
		},
		"coder": {
			Name:        "coder",
			Description: "Code generation specialist",
			System: `You are a coding assistant. Write clean, well-documented code.
Follow existing patterns in the codebase.
Include error handling.`,
			Temperature: 0.7,
		},
		"documenter": {
			Name:        "documenter",
			Description: "Documentation writer",
			System: `You write clear, concise documentation.
Use examples where helpful.
Match the project's documentation style.`,
			Temperature: 0.5,
		},
		"analyst": {
			Name:        "analyst",
			Description: "Code analysis and reasoning",
			System: `You analyze code structure and architecture.
Explain how components interact.
Identify potential improvements.`,
			Temperature: 0.4,
		},
	}
}

// ConfigPaths returns the search paths for agent configs
func ConfigPaths() []string {
	paths := []string{".clood/agents.yaml"}

	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".config", "clood", "agents.yaml"))
	}

	return paths
}

// ConfigExists checks if any config file exists
func ConfigExists() (string, bool) {
	for _, path := range ConfigPaths() {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}
