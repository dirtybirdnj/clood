// Package mcp provides an MCP (Model Context Protocol) server for clood.
// This enables AI agents to call clood tools via SSE streaming.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

	// The main event: ask local models
	s.mcpServer.AddTool(s.askTool(), s.askHandler)
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

func (s *Server) askTool() mcp.Tool {
	return mcp.NewTool("clood_ask",
		mcp.WithDescription("Send a prompt to local LLM via clood routing. Returns model response. Use for code generation, analysis, or any LLM task."),
		mcp.WithString("prompt", mcp.Required(), mcp.Description("The prompt to send to the model")),
		mcp.WithString("model", mcp.Description("Specific model to use (default: routes to best available)")),
		mcp.WithString("host", mcp.Description("Specific host to use (default: fastest responding)")),
		mcp.WithBoolean("dialogue", mcp.Description("If true, model will ask clarifying questions before implementing")),
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

// Dialogue system prompt for interactive coding
const dialogueSystemPrompt = `You are a helpful coding assistant in a dialogue with a developer.

RULES:
1. ALWAYS confirm understanding before implementing
2. ASK clarifying questions when requirements are ambiguous
3. OFFER next steps after completing a task
4. RESPOND to feedback and iterate

FORMAT your responses with clear sections:
- [UNDERSTANDING] - What you think is being asked
- [QUESTIONS] - Clarifying questions (if any)
- [IMPLEMENTATION] - Code or explanation
- [NEXT] - Suggested next steps

This is a CONVERSATION, not a one-shot request.`

func (s *Server) askHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return mcp.NewToolResultError("prompt is required"), nil
	}

	// Get model/host preferences
	modelPref, _ := args["model"].(string)
	hostPref, _ := args["host"].(string)
	dialogue, _ := args["dialogue"].(bool)

	// Add dialogue system prompt if requested
	if dialogue {
		prompt = dialogueSystemPrompt + "\n\nUser request:\n" + prompt
	}

	// Reload config for latest host info
	cfg, _ := config.Load()
	if cfg != nil {
		s.hostMgr = hosts.NewManager()
		s.hostMgr.AddHosts(cfg.Hosts)
	}

	// Find best host/model
	var targetHost *hosts.Host
	var targetModel string

	if hostPref != "" {
		targetHost = s.hostMgr.GetHost(hostPref)
		if targetHost == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Host not found: %s", hostPref)), nil
		}
	}

	if modelPref != "" {
		targetModel = modelPref
	} else {
		// Use fast tier default
		if cfg != nil && cfg.Tiers.Fast.Model != "" {
			targetModel = cfg.Tiers.Fast.Model
		} else {
			targetModel = "qwen2.5-coder:3b"
		}
	}

	// If no host specified, find first online host with the model
	if targetHost == nil {
		statuses := s.hostMgr.CheckAllHosts()
		for _, st := range statuses {
			if !st.Online {
				continue
			}
			for _, m := range st.Models {
				if m.Name == targetModel || strings.HasPrefix(m.Name, targetModel) {
					targetHost = st.Host
					break
				}
			}
			if targetHost != nil {
				break
			}
		}
	}

	if targetHost == nil {
		return mcp.NewToolResultError(fmt.Sprintf("No online host found with model: %s", targetModel)), nil
	}

	// Call Ollama
	response, err := callOllama(targetHost.URL, targetModel, prompt)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Ollama error: %v", err)), nil
	}

	// Return with metadata
	result := fmt.Sprintf("🐱 %s @ %s\n\n%s", targetModel, targetHost.Name, response)
	return mcp.NewToolResultText(result), nil
}

// callOllama sends a prompt to Ollama and returns the response
func callOllama(baseURL, model, prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(baseURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	return result.Response, nil
}
