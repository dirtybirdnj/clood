package hosts

import (
	"sort"
	"sync"
	"time"

	"github.com/dirtybirdnj/clood/internal/ollama"
)

// BackendType identifies the API format a host uses
type BackendType string

const (
	BackendOllama BackendType = "ollama" // Default: Ollama API format
	BackendOpenAI BackendType = "openai" // OpenAI-compatible (llama.cpp, vLLM, etc.)
)

// Host represents an LLM endpoint
type Host struct {
	Name     string      `yaml:"name"`
	URL      string      `yaml:"url"`
	Priority int         `yaml:"priority"` // Lower = higher priority
	Enabled  bool        `yaml:"enabled"`
	Backend  BackendType `yaml:"backend,omitempty"` // ollama (default) or openai
	Models   []string    `yaml:"models,omitempty"`  // Static model list (for openai backends)
	APIKey   string      `yaml:"api_key,omitempty"` // Optional API key
}

// HostStatus contains the current status of a host
type HostStatus struct {
	Host      *Host
	Online    bool
	Latency   time.Duration
	Version   string
	Models    []ollama.Model
	Error     error
	CheckedAt time.Time
}

// Manager handles multiple Ollama hosts
type Manager struct {
	hosts   []*Host
	clients map[string]*ollama.Client
	status  map[string]*HostStatus
	mu      sync.RWMutex
}

// NewManager creates a new host manager
func NewManager() *Manager {
	return &Manager{
		hosts:   make([]*Host, 0),
		clients: make(map[string]*ollama.Client),
		status:  make(map[string]*HostStatus),
	}
}

// DefaultHosts returns the default host configuration
func DefaultHosts() []*Host {
	return []*Host{
		// Ollama backends (auto-discover models)
		{Name: "local-gpu", URL: "http://localhost:11434", Priority: 1, Enabled: true, Backend: BackendOllama},
		{Name: "ubuntu25", URL: "http://192.168.4.64:11434", Priority: 1, Enabled: true, Backend: BackendOllama},
		{Name: "mac-mini", URL: "http://192.168.4.41:11434", Priority: 2, Enabled: true, Backend: BackendOllama},
		// Example llama.cpp backend (static model list required)
		// {Name: "ubuntu25-llamacpp", URL: "http://192.168.4.64:8080", Priority: 0, Enabled: true, Backend: BackendOpenAI, Models: []string{"qwen2.5-coder-7b"}},
	}
}

// GetBackend returns the backend type, defaulting to Ollama
func (h *Host) GetBackend() BackendType {
	if h.Backend == "" {
		return BackendOllama
	}
	return h.Backend
}

// AddHost adds a host to the manager
func (m *Manager) AddHost(host *Host) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hosts = append(m.hosts, host)
	m.clients[host.Name] = ollama.NewClient(host.URL, 120*time.Second)
}

// AddHosts adds multiple hosts
func (m *Manager) AddHosts(hosts []*Host) {
	for _, h := range hosts {
		m.AddHost(h)
	}
}

// GetHost returns a host by name
func (m *Manager) GetHost(name string) *Host {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, h := range m.hosts {
		if h.Name == name {
			return h
		}
	}
	return nil
}

// GetClient returns the Ollama client for a host
func (m *Manager) GetClient(name string) *ollama.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clients[name]
}

// GetAllHosts returns all hosts
func (m *Manager) GetAllHosts() []*Host {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Host, len(m.hosts))
	copy(result, m.hosts)
	return result
}

// CheckHost checks the status of a single host
func (m *Manager) CheckHost(host *Host) *HostStatus {
	client := m.GetClient(host.Name)
	if client == nil {
		return &HostStatus{
			Host:      host,
			Online:    false,
			Error:     nil,
			CheckedAt: time.Now(),
		}
	}

	status := &HostStatus{
		Host:      host,
		CheckedAt: time.Now(),
	}

	// Ping for latency
	latency, err := client.Ping()
	if err != nil {
		status.Online = false
		status.Error = err
		return status
	}

	status.Online = true
	status.Latency = latency

	// Get version
	if version, err := client.Version(); err == nil {
		status.Version = version
	}

	// Get models
	if models, err := client.ListModels(); err == nil {
		status.Models = models
	}

	// Update cached status
	m.mu.Lock()
	m.status[host.Name] = status
	m.mu.Unlock()

	return status
}

// CheckAllHosts checks all hosts concurrently
func (m *Manager) CheckAllHosts() []*HostStatus {
	hosts := m.GetAllHosts()
	results := make([]*HostStatus, len(hosts))

	var wg sync.WaitGroup
	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, h *Host) {
			defer wg.Done()
			results[idx] = m.CheckHost(h)
		}(i, host)
	}
	wg.Wait()

	return results
}

// HostCheckResult is sent over the channel as each host completes checking
type HostCheckResult struct {
	Index  int
	Status *HostStatus
}

// CheckAllHostsStreaming checks all hosts concurrently and streams results
// as each host completes. Caller should read from the returned channel
// until it's closed. The total number of results will equal len(GetAllHosts()).
func (m *Manager) CheckAllHostsStreaming() (<-chan HostCheckResult, int) {
	hosts := m.GetAllHosts()
	results := make(chan HostCheckResult, len(hosts))

	go func() {
		var wg sync.WaitGroup
		for i, host := range hosts {
			wg.Add(1)
			go func(idx int, h *Host) {
				defer wg.Done()
				status := m.CheckHost(h)
				results <- HostCheckResult{Index: idx, Status: status}
			}(i, host)
		}
		wg.Wait()
		close(results)
	}()

	return results, len(hosts)
}

// GetOnlineHosts returns only online hosts, sorted by priority then latency
func (m *Manager) GetOnlineHosts() []*HostStatus {
	statuses := m.CheckAllHosts()

	var online []*HostStatus
	for _, s := range statuses {
		if s.Online && s.Host.Enabled {
			online = append(online, s)
		}
	}

	// Sort by priority (lower first), then latency
	sort.Slice(online, func(i, j int) bool {
		if online[i].Host.Priority != online[j].Host.Priority {
			return online[i].Host.Priority < online[j].Host.Priority
		}
		return online[i].Latency < online[j].Latency
	})

	return online
}

// GetBestHost returns the best available host (lowest priority, lowest latency)
func (m *Manager) GetBestHost() *HostStatus {
	online := m.GetOnlineHosts()
	if len(online) == 0 {
		return nil
	}
	return online[0]
}

// GetHostWithModel returns the best host that has the specified model
func (m *Manager) GetHostWithModel(modelName string) *HostStatus {
	online := m.GetOnlineHosts()

	for _, status := range online {
		for _, model := range status.Models {
			if model.Name == modelName {
				return status
			}
		}
	}

	return nil
}

// GetAllModels returns all unique models across all hosts
func (m *Manager) GetAllModels() map[string][]string {
	statuses := m.CheckAllHosts()
	models := make(map[string][]string) // model name -> list of hosts

	for _, status := range statuses {
		if !status.Online {
			continue
		}
		for _, model := range status.Models {
			models[model.Name] = append(models[model.Name], status.Host.Name)
		}
	}

	return models
}

// FindModel searches for a model across all hosts
func (m *Manager) FindModel(modelName string) []*HostStatus {
	statuses := m.CheckAllHosts()
	var found []*HostStatus

	for _, status := range statuses {
		if !status.Online {
			continue
		}
		for _, model := range status.Models {
			if model.Name == modelName {
				found = append(found, status)
				break
			}
		}
	}

	// Sort by priority then latency
	sort.Slice(found, func(i, j int) bool {
		if found[i].Host.Priority != found[j].Host.Priority {
			return found[i].Host.Priority < found[j].Host.Priority
		}
		return found[i].Latency < found[j].Latency
	})

	return found
}

// GetCachedStatus returns the cached status for a host (may be stale)
func (m *Manager) GetCachedStatus(name string) *HostStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status[name]
}
