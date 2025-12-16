# clood API Reference

## Package: internal/ollama

### Types

```go
// Client communicates with Ollama API
type Client struct {
    BaseURL    string
    HTTPClient *http.Client
}

// GenerateRequest for /api/generate
type GenerateRequest struct {
    Model   string                 `json:"model"`
    Prompt  string                 `json:"prompt"`
    Stream  bool                   `json:"stream"`
    Options map[string]interface{} `json:"options,omitempty"`
}

// GenerateResponse from /api/generate
type GenerateResponse struct {
    Model              string    `json:"model"`
    Response           string    `json:"response"`
    Done               bool      `json:"done"`
    TotalDuration      int64     `json:"total_duration,omitempty"`
    EvalCount          int       `json:"eval_count,omitempty"`
    EvalDuration       int64     `json:"eval_duration,omitempty"`
}

// Model from /api/tags
type Model struct {
    Name       string    `json:"name"`
    Size       int64     `json:"size"`
    ModifiedAt time.Time `json:"modified_at"`
    Details    struct {
        ParameterSize     string `json:"parameter_size"`
        QuantizationLevel string `json:"quantization_level"`
    } `json:"details"`
}

// BenchmarkResult from Benchmark()
type BenchmarkResult struct {
    Model             string
    TotalDuration     time.Duration
    GeneratedTokens   int
    GenerateTokPerSec float64
}
```

### Functions

```go
// NewClient creates a client for the given Ollama endpoint
func NewClient(baseURL string, timeout time.Duration) *Client

// Generate sends a prompt and returns the full response
func (c *Client) Generate(model, prompt string) (*GenerateResponse, error)

// GenerateStream sends a prompt and streams chunks via callback
func (c *Client) GenerateStream(model, prompt string, callback func(chunk GenerateResponse)) (*GenerateResponse, error)

// ListModels returns all models on this instance
func (c *Client) ListModels() ([]Model, error)

// Ping checks if the server is reachable
func (c *Client) Ping() (time.Duration, error)

// Benchmark runs a simple performance test
func (c *Client) Benchmark(model, prompt string) (*BenchmarkResult, error)
```

---

## Package: internal/hosts

### Types

```go
// Host represents an Ollama endpoint
type Host struct {
    Name     string `yaml:"name"`
    URL      string `yaml:"url"`
    Priority int    `yaml:"priority"`
    Enabled  bool   `yaml:"enabled"`
}

// HostStatus contains current status
type HostStatus struct {
    Host      *Host
    Online    bool
    Latency   time.Duration
    Version   string
    Models    []ollama.Model
    Error     error
}

// Manager handles multiple hosts
type Manager struct {
    hosts   []*Host
    clients map[string]*ollama.Client
    status  map[string]*HostStatus
}
```

### Functions

```go
// NewManager creates a host manager
func NewManager() *Manager

// AddHost adds a host to the manager
func (m *Manager) AddHost(host *Host)

// CheckAllHosts checks all hosts concurrently
func (m *Manager) CheckAllHosts() []*HostStatus

// GetBestHost returns the best available host
func (m *Manager) GetBestHost() *HostStatus

// GetHostWithModel returns best host that has a model
func (m *Manager) GetHostWithModel(modelName string) *HostStatus

// GetAllModels returns all models across hosts
func (m *Manager) GetAllModels() map[string][]string
```

---

## Package: internal/system

### Types

```go
// HardwareInfo contains detected hardware
type HardwareInfo struct {
    Hostname    string
    OS          string
    Arch        string
    CPUModel    string
    CPUCores    int
    MemoryGB    float64
    GPU         *GPUInfo
    DiskFreeGB  float64
    OllamaVRAM  float64
}

// GPUInfo contains GPU details
type GPUInfo struct {
    Name  string
    VRAM  float64
    Type  string  // "apple", "nvidia", "amd", "none"
    Cores int
}
```

### Functions

```go
// DetectHardware gathers local hardware info
func DetectHardware() (*HardwareInfo, error)

// ModelFits returns true if model fits in VRAM
func (h *HardwareInfo) ModelFits(sizeB float64) bool

// RecommendedModels returns suitable models for hardware
func (h *HardwareInfo) RecommendedModels() []string

// Summary returns human-readable summary
func (h *HardwareInfo) Summary() string
```

---

## Package: internal/router

### Constants

```go
const (
    TierFast = 1  // Simple queries
    TierDeep = 2  // Complex queries
)
```

### Types

```go
// RouteResult contains routing decision
type RouteResult struct {
    Tier       int
    Confidence float64
    Model      string
    Host       *hosts.HostStatus
    Client     *ollama.Client
}

// Router handles query routing
type Router struct {
    config  *config.Config
    manager *hosts.Manager
}
```

### Functions

```go
// ClassifyQuery determines tier for a query
func ClassifyQuery(query string) int

// ClassifyWithConfidence returns tier and confidence
func ClassifyWithConfidence(query string) (int, float64)

// NewRouter creates a router with config
func NewRouter(cfg *config.Config) *Router

// Route determines best host/model for query
func (r *Router) Route(query string, forceTier int, forceModel string) (*RouteResult, error)
```

---

## Package: internal/config

### Types

```go
// Config is the global configuration
type Config struct {
    Hosts    []*hosts.Host `yaml:"hosts"`
    Tiers    TierConfig    `yaml:"tiers"`
    Routing  RoutingConfig `yaml:"routing"`
    Defaults DefaultsConfig `yaml:"defaults"`
}

type TierConfig struct {
    Fast TierSettings `yaml:"fast"`
    Deep TierSettings `yaml:"deep"`
}

type TierSettings struct {
    Model string `yaml:"model"`
}
```

### Functions

```go
// Load reads config from disk or returns defaults
func Load() (*Config, error)

// Save writes config to disk
func Save(cfg *Config) error

// ConfigPath returns path to config file
func ConfigPath() string

// GetTierModel returns model for a tier
func (c *Config) GetTierModel(tier int) string
```

---

## Package: internal/tui

### Styles

```go
var (
    ColorPrimary   = lipgloss.Color("#00D4FF")  // Electric blue
    ColorSecondary = lipgloss.Color("#B8C5D0")  // Cloud gray
    ColorAccent    = lipgloss.Color("#FFD700")  // Lightning gold
    ColorSuccess   = lipgloss.Color("#00FF88")
    ColorWarning   = lipgloss.Color("#FF8C00")
    ColorError     = lipgloss.Color("#FF4444")
)

var (
    LogoStyle     lipgloss.Style  // For banner
    HeaderStyle   lipgloss.Style  // For section headers
    SuccessStyle  lipgloss.Style  // For success messages
    ErrorStyle    lipgloss.Style  // For error messages
    MutedStyle    lipgloss.Style  // For secondary text
    BoxStyle      lipgloss.Style  // For content boxes
)
```

### Functions

```go
// RenderBanner returns the styled ASCII banner
func RenderBanner() string

// RenderHeader returns a styled section header
func RenderHeader(title string) string

// RenderTier returns styled tier indicator
func RenderTier(tier int) string
```
