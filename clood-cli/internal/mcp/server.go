// Package mcp provides an MCP (Model Context Protocol) server for clood.
// This enables AI agents to call clood tools via SSE streaming.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/system"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server with clood-specific functionality
type Server struct {
	mcpServer *server.MCPServer
	config    *config.Config
	hostMgr   *hosts.Manager
}

// NewServer creates a new clood MCP server with all tools registered
func NewServer() (*Server, error) {
	// Load clood config
	cfg, err := config.Load()
	if err != nil {
		// Use empty config if not found
		cfg = &config.Config{}
	}

	// Create host manager
	hostMgr := hosts.NewManager()
	hostMgr.AddHosts(cfg.Hosts)

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"clood",
		"0.2.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	s := &Server{
		mcpServer: mcpServer,
		config:    cfg,
		hostMgr:   hostMgr,
	}

	// Register all tools
	s.registerTools()

	return s, nil
}

// MCPServer returns the underlying MCP server for transport setup
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}

// registerTools adds all clood commands as MCP tools
func (s *Server) registerTools() {
	// Infrastructure tools
	s.mcpServer.AddTool(s.hostsTool(), s.hostsHandler)
	s.mcpServer.AddTool(s.modelsTool(), s.modelsHandler)
	s.mcpServer.AddTool(s.systemTool(), s.systemHandler)
	s.mcpServer.AddTool(s.healthTool(), s.healthHandler)
}

// =============================================================================
// Tool Definitions
// =============================================================================

func (s *Server) hostsTool() mcp.Tool {
	return mcp.NewTool("clood_hosts",
		mcp.WithDescription("List and check status of all configured Ollama hosts. Returns online/offline status, latency, and available models for each host."),
	)
}

func (s *Server) modelsTool() mcp.Tool {
	return mcp.NewTool("clood_models",
		mcp.WithDescription("List all available models across all Ollama hosts. Shows which hosts have each model."),
		mcp.WithString("host", mcp.Description("Optional: filter to specific host")),
	)
}

func (s *Server) systemTool() mcp.Tool {
	return mcp.NewTool("clood_system",
		mcp.WithDescription("Display hardware information and model recommendations. Shows CPU, memory, GPU, and which models will fit."),
	)
}

func (s *Server) healthTool() mcp.Tool {
	return mcp.NewTool("clood_health",
		mcp.WithDescription("Full health check of all clood services. Checks hosts, models, and configuration."),
	)
}

// =============================================================================
// Tool Handlers
// =============================================================================

func (s *Server) hostsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Reload hosts from config
	cfg, _ := config.Load()
	if cfg != nil {
		s.hostMgr = hosts.NewManager()
		s.hostMgr.AddHosts(cfg.Hosts)
	}

	statuses := s.hostMgr.CheckAllHosts()

	// Build JSON response
	type hostJSON struct {
		Name    string   `json:"name"`
		URL     string   `json:"url"`
		Online  bool     `json:"online"`
		Latency int64    `json:"latency_ms,omitempty"`
		Version string   `json:"version,omitempty"`
		Models  []string `json:"models,omitempty"`
		Error   string   `json:"error,omitempty"`
	}

	var result []hostJSON
	for _, st := range statuses {
		h := hostJSON{
			Name:   st.Host.Name,
			URL:    st.Host.URL,
			Online: st.Online,
		}
		if st.Online {
			h.Latency = st.Latency.Milliseconds()
			h.Version = st.Version
			for _, m := range st.Models {
				h.Models = append(h.Models, m.Name)
			}
		}
		if st.Error != nil {
			h.Error = st.Error.Error()
		}
		result = append(result, h)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) modelsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Reload config
	cfg, _ := config.Load()
	if cfg != nil {
		s.hostMgr = hosts.NewManager()
		s.hostMgr.AddHosts(cfg.Hosts)
	}

	// Check for host filter
	args := req.GetArguments()
	hostFilter, _ := args["host"].(string)

	if hostFilter != "" {
		host := s.hostMgr.GetHost(hostFilter)
		if host == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Host not found: %s", hostFilter)), nil
		}
		status := s.hostMgr.CheckHost(host)
		if !status.Online {
			return mcp.NewToolResultError(fmt.Sprintf("Host offline: %s", hostFilter)), nil
		}
		var models []string
		for _, m := range status.Models {
			models = append(models, m.Name)
		}
		data, _ := json.MarshalIndent(models, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}

	// All models across all hosts
	allModels := s.hostMgr.GetAllModels()
	data, _ := json.MarshalIndent(allModels, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) systemHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hw, err := system.DetectHardware()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error detecting hardware: %v", err)), nil
	}

	data, _ := json.MarshalIndent(hw.JSON(), "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) healthHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Reload config
	cfg, err := config.Load()

	health := map[string]interface{}{
		"config_loaded": err == nil,
	}

	if cfg != nil {
		s.hostMgr = hosts.NewManager()
		s.hostMgr.AddHosts(cfg.Hosts)

		statuses := s.hostMgr.CheckAllHosts()
		online := 0
		total := len(statuses)
		var hostStatuses []map[string]interface{}

		for _, st := range statuses {
			hs := map[string]interface{}{
				"name":   st.Host.Name,
				"online": st.Online,
			}
			if st.Online {
				online++
				hs["latency_ms"] = st.Latency.Milliseconds()
				hs["model_count"] = len(st.Models)
			}
			if st.Error != nil {
				hs["error"] = st.Error.Error()
			}
			hostStatuses = append(hostStatuses, hs)
		}

		health["hosts"] = hostStatuses
		health["hosts_online"] = online
		health["hosts_total"] = total
		health["tiers"] = map[string]string{
			"fast": cfg.Tiers.Fast.Model,
			"deep": cfg.Tiers.Deep.Model,
		}
	}

	data, _ := json.MarshalIndent(health, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
