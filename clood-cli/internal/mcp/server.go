// Package mcp provides an MCP (Model Context Protocol) server for clood.
// This enables AI agents to call clood tools via SSE streaming.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/analyze"
	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/inception"
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
	// CRITICAL: Preflight and gate tools - use these FIRST
	s.mcpServer.AddTool(s.preflightTool(), s.preflightHandler)
	s.mcpServer.AddTool(s.shouldSearchWebTool(), s.shouldSearchWebHandler)

	// Infrastructure tools
	s.mcpServer.AddTool(s.hostsTool(), s.hostsHandler)
	s.mcpServer.AddTool(s.modelsTool(), s.modelsHandler)
	s.mcpServer.AddTool(s.systemTool(), s.systemHandler)
	s.mcpServer.AddTool(s.healthTool(), s.healthHandler)

	// LOCAL DISCOVERY TOOLS (0 network, 0 LLM tokens)
	// These should be used BEFORE any network requests or LLM calls
	s.mcpServer.AddTool(s.grepTool(), s.grepHandler)
	s.mcpServer.AddTool(s.treeTool(), s.treeHandler)
	s.mcpServer.AddTool(s.symbolsTool(), s.symbolsHandler)
	s.mcpServer.AddTool(s.importsTool(), s.importsHandler)
	s.mcpServer.AddTool(s.contextTool(), s.contextHandler)
	s.mcpServer.AddTool(s.capabilitiesTool(), s.capabilitiesHandler)
	s.mcpServer.AddTool(s.analyzeTool(), s.analyzeHandler)

	// The main event: ask local models
	s.mcpServer.AddTool(s.askTool(), s.askHandler)

	// INCEPTION: LLM-to-LLM sub-queries
	s.mcpServer.AddTool(s.inceptionTool(), s.inceptionHandler)
}

// =============================================================================
// Tool Definitions
// =============================================================================

// =============================================================================
// CRITICAL TOOLS - Use these FIRST
// =============================================================================

func (s *Server) preflightTool() mcp.Tool {
	return mcp.NewTool("clood_preflight",
		mcp.WithDescription(`🚨 START EVERY SESSION WITH THIS TOOL 🚨

Call this BEFORE doing any work. Returns:
- Available local discovery tools (grep, tree, symbols, imports, context)
- Online Ollama hosts and models
- Recommended workflow for this session

This ensures you use local resources before making network requests.
Cost: ZERO network, ZERO tokens, instant.`),
	)
}

func (s *Server) shouldSearchWebTool() mcp.Tool {
	return mcp.NewTool("clood_should_search_web",
		mcp.WithDescription(`🛑 REQUIRED: Call this BEFORE any web search 🛑

Analyzes your query and tells you if local tools can answer it instead.
- If local tools can help → Returns which tool to use instead
- If web search is needed → Returns "approved" with reason

DO NOT skip this gate. It prevents unnecessary network requests.
Cost: ZERO network, ZERO tokens, instant.`),
		mcp.WithString("query", mcp.Required(), mcp.Description("The query you were about to search the web for")),
	)
}

// =============================================================================
// Infrastructure Tools
// =============================================================================

func (s *Server) hostsTool() mcp.Tool {
	return mcp.NewTool("clood_hosts",
		mcp.WithDescription(`Check Ollama host status. ALWAYS call this before clood_ask.

Returns online/offline status, latency, and available models for each host.
Use this to verify local LLM is available before querying.
Cost: Local network only (no internet), ZERO tokens.`),
	)
}

func (s *Server) modelsTool() mcp.Tool {
	return mcp.NewTool("clood_models",
		mcp.WithDescription(`List available models across all Ollama hosts.

Shows which hosts have each model. Use to pick the right model for your task.
Cost: Local network only (no internet), ZERO tokens.`),
		mcp.WithString("host", mcp.Description("Optional: filter to specific host")),
	)
}

func (s *Server) systemTool() mcp.Tool {
	return mcp.NewTool("clood_system",
		mcp.WithDescription(`Display hardware info and model recommendations.

Shows CPU, memory, GPU, VRAM, and which models will fit.
Use to understand local compute capacity.
Cost: ZERO network, ZERO tokens, instant.`),
	)
}

func (s *Server) healthTool() mcp.Tool {
	return mcp.NewTool("clood_health",
		mcp.WithDescription(`Full health check of all clood services.

Checks hosts, models, config, and tier assignments.
Use when things aren't working or at session start.
Cost: Local network only (no internet), ZERO tokens.`),
	)
}

func (s *Server) askTool() mcp.Tool {
	return mcp.NewTool("clood_ask",
		mcp.WithDescription(`Query LOCAL Ollama LLM. Use INSTEAD of cloud LLM APIs.

⚠️  BEFORE calling this: Run clood_hosts to verify a host is online.

Routes to best available local model. Use for:
- Code generation and analysis
- Explaining code patterns
- Best practices questions

Cost: Local LLM tokens only, ZERO cloud API calls, ZERO internet.`),
		mcp.WithString("prompt", mcp.Required(), mcp.Description("The prompt to send to the model")),
		mcp.WithString("model", mcp.Description("Specific model to use (default: routes to best available)")),
		mcp.WithString("host", mcp.Description("Specific host to use (default: fastest responding)")),
		mcp.WithBoolean("dialogue", mcp.Description("If true, model will ask clarifying questions before implementing")),
	)
}

func (s *Server) inceptionTool() mcp.Tool {
	return mcp.NewTool("clood_inception",
		mcp.WithDescription(`🌀 INCEPTION: Query an expert LLM model mid-stream.

Use this when you need specialized knowledge from a different model:
- science: Physics, chemistry, biology facts
- math: Calculations, proofs, formulas
- code: Code review, programming patterns
- creative: Brainstorming, writing

Example: You're writing simulation code and need orbital velocity.
Call: clood_inception expert="science" query="What is ISS orbital velocity?"
Response: "7.66 km/s at 408km altitude"
Continue your work with the expert knowledge.

This is ONE-LEVEL deep - the expert cannot call other experts.
Cost: Local LLM tokens only, ZERO cloud API.`),
		mcp.WithString("query", mcp.Required(), mcp.Description("The question for the expert model")),
		mcp.WithString("expert", mcp.Required(), mcp.Description("Expert type: science, math, code, creative, or model name")),
	)
}

// =============================================================================
// LOCAL DISCOVERY TOOLS (0 network, 0 LLM tokens)
// Use these BEFORE making any network requests or LLM calls
// =============================================================================

func (s *Server) grepTool() mcp.Tool {
	return mcp.NewTool("clood_grep",
		mcp.WithDescription(`🔍 USE THIS INSTEAD OF WEB SEARCH for codebase questions.

Replaces these web searches:
- "where is X in this codebase" → clood_grep "X" --files_only
- "what files contain Y" → clood_grep "Y"
- "how does Z work in this project" → clood_grep "Z"

Cost: ZERO network, ZERO tokens, instant.
ALWAYS use this before considering WebSearch for code-related queries.`),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Regex pattern to search for")),
		mcp.WithString("path", mcp.Description("Directory to search in (default: current directory)")),
		mcp.WithBoolean("files_only", mcp.Description("Only return file names, not matching lines")),
		mcp.WithBoolean("ignore_case", mcp.Description("Case insensitive search")),
		mcp.WithString("type", mcp.Description("Filter by file type: go, py, js, ts, rs, etc.")),
	)
}

func (s *Server) treeTool() mcp.Tool {
	return mcp.NewTool("clood_tree",
		mcp.WithDescription(`🌳 USE THIS INSTEAD OF WEB SEARCH for project structure.

Replaces these web searches:
- "project structure"
- "what directories exist"
- "codebase layout"

Respects .gitignore. Shows clean directory tree.
Cost: ZERO network, ZERO tokens, instant.`),
		mcp.WithString("path", mcp.Description("Directory to show (default: current directory)")),
		mcp.WithNumber("depth", mcp.Description("Maximum depth to traverse (default: 3)")),
	)
}

func (s *Server) symbolsTool() mcp.Tool {
	return mcp.NewTool("clood_symbols",
		mcp.WithDescription(`📦 USE THIS INSTEAD OF WEB SEARCH for function/type lookups.

Replaces these web searches:
- "what functions are in file.go"
- "function signature for Foo"
- "what types does this package define"

Extracts functions, types, classes from Go, Python, JS/TS.
Cost: ZERO network, ZERO tokens, instant.`),
		mcp.WithString("path", mcp.Required(), mcp.Description("File or directory to analyze")),
		mcp.WithBoolean("exported_only", mcp.Description("Only show exported/public symbols")),
		mcp.WithString("kind", mcp.Description("Filter by kind: func, type, class, const, var")),
	)
}

func (s *Server) importsTool() mcp.Tool {
	return mcp.NewTool("clood_imports",
		mcp.WithDescription(`📎 USE THIS INSTEAD OF WEB SEARCH for dependency questions.

Replaces these web searches:
- "what does this file import"
- "what dependencies does X use"
- "what packages are used here"

Shows internal, external, and stdlib imports.
Cost: ZERO network, ZERO tokens, instant.`),
		mcp.WithString("path", mcp.Required(), mcp.Description("File or directory to analyze")),
	)
}

func (s *Server) contextTool() mcp.Tool {
	return mcp.NewTool("clood_context",
		mcp.WithDescription(`📋 Generate LLM-ready project summary.

Creates a condensed context including:
- README content
- Project structure
- Key files

Use to quickly understand a project without reading every file.
Cost: ZERO network, ZERO tokens, instant.`),
		mcp.WithString("path", mcp.Description("Directory to analyze (default: current directory)")),
		mcp.WithNumber("max_tokens", mcp.Description("Target token count (default: 4000)")),
	)
}

func (s *Server) capabilitiesTool() mcp.Tool {
	return mcp.NewTool("clood_capabilities",
		mcp.WithDescription(`📊 List what clood can do locally vs what requires network.

Shows:
- Available local discovery tools
- Available Ollama tools
- Whether Ollama is online

Use to plan your approach: local tools first, network last.
Cost: ZERO network, ZERO tokens, instant.`),
	)
}

func (s *Server) analyzeTool() mcp.Tool {
	return mcp.NewTool("clood_analyze",
		mcp.WithDescription(`🔬 Run static analysis on Go codebase (like "clood bcbc").

Returns pre-computed analysis including:
- Build status (pass/fail)
- Go vet issues
- TODO/FIXME items
- Recent commits and hot files
- Symbol counts (funcs, types, methods)

Use this to quickly understand codebase health before making changes.
Cost: ZERO network, ZERO tokens (runs go build/vet locally).`),
		mcp.WithString("path", mcp.Description("Directory to analyze (default: current directory)")),
		mcp.WithBoolean("run_tests", mcp.Description("Also run tests (slower)")),
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

func (s *Server) inceptionHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	expert, ok := args["expert"].(string)
	if !ok || expert == "" {
		return mcp.NewToolResultError("expert is required (science, math, code, creative, or model name)"), nil
	}

	// Create inception handler
	handler := inception.NewHandler()

	// Build sub-query
	subQuery := inception.SubQuery{
		Model: expert,
		Query: query,
	}

	// Execute synchronously
	result := handler.ExecuteSubQuery(ctx, subQuery)

	if result.Error != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Inception failed: %v", result.Error)), nil
	}

	// Format response
	response := fmt.Sprintf("🌀 INCEPTION RESPONSE [%s → %s]\n"+
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
		"Query: %s\n"+
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
		"%s\n"+
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
		"Duration: %.2fs",
		expert, handler.Registry[expert],
		query,
		result.Response,
		result.Duration.Seconds())

	return mcp.NewToolResultText(response), nil
}

// =============================================================================
// LOCAL DISCOVERY HANDLERS (0 network, 0 LLM tokens)
// =============================================================================

func (s *Server) grepHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return mcp.NewToolResultError("pattern is required"), nil
	}

	searchPath := "."
	if p, ok := args["path"].(string); ok && p != "" {
		searchPath = p
	}

	filesOnly, _ := args["files_only"].(bool)
	ignoreCase, _ := args["ignore_case"].(bool)
	fileType, _ := args["type"].(string)

	// Build regex
	flags := ""
	if ignoreCase {
		flags = "(?i)"
	}
	re, err := regexp.Compile(flags + pattern)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid regex: %v", err)), nil
	}

	type match struct {
		File    string `json:"file"`
		Line    int    `json:"line,omitempty"`
		Content string `json:"content,omitempty"`
	}

	var matches []match
	filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Skip hidden and vendor dirs
		if strings.Contains(path, "/.") || strings.Contains(path, "/vendor/") ||
			strings.Contains(path, "/node_modules/") || strings.Contains(path, "/.git/") {
			return nil
		}

		// Filter by type if specified
		if fileType != "" {
			ext := strings.TrimPrefix(filepath.Ext(path), ".")
			if ext != fileType {
				return nil
			}
		}

		// Search file
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		fileHasMatch := false

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if re.MatchString(line) {
				if filesOnly {
					if !fileHasMatch {
						matches = append(matches, match{File: path})
						fileHasMatch = true
					}
				} else {
					matches = append(matches, match{
						File:    path,
						Line:    lineNum,
						Content: line,
					})
				}
			}
		}
		return nil
	})

	data, _ := json.MarshalIndent(matches, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) treeHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	maxDepth := 3
	if d, ok := args["depth"].(float64); ok {
		maxDepth = int(d)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Directory: %s\n\n", path))

	var walkTree func(string, string, int) error
	walkTree = func(dir, prefix string, depth int) error {
		if depth >= maxDepth {
			return nil
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}

		// Filter entries
		var filtered []os.DirEntry
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			if name == "node_modules" || name == "vendor" || name == "__pycache__" {
				continue
			}
			filtered = append(filtered, e)
		}

		for i, entry := range filtered {
			isLast := i == len(filtered)-1
			connector := "├── "
			if isLast {
				connector = "└── "
			}

			name := entry.Name()
			if entry.IsDir() {
				name += "/"
			}
			sb.WriteString(prefix + connector + name + "\n")

			if entry.IsDir() {
				newPrefix := prefix + "│   "
				if isLast {
					newPrefix = prefix + "    "
				}
				walkTree(filepath.Join(dir, entry.Name()), newPrefix, depth+1)
			}
		}
		return nil
	}

	walkTree(path, "", 0)
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) symbolsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	path, ok := args["path"].(string)
	if !ok || path == "" {
		return mcp.NewToolResultError("path is required"), nil
	}

	exportedOnly, _ := args["exported_only"].(bool)
	kindFilter, _ := args["kind"].(string)

	type symbol struct {
		Name     string `json:"name"`
		Kind     string `json:"kind"`
		File     string `json:"file"`
		Line     int    `json:"line"`
		Exported bool   `json:"exported"`
	}

	var symbols []symbol

	// Patterns for different languages
	goFuncPattern := regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?(\w+)`)
	goTypePattern := regexp.MustCompile(`^type\s+(\w+)`)
	pyFuncPattern := regexp.MustCompile(`^def\s+(\w+)`)
	pyClassPattern := regexp.MustCompile(`^class\s+(\w+)`)

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if strings.Contains(p, "/.") || strings.Contains(p, "/vendor/") {
			return nil
		}

		ext := filepath.Ext(p)
		if ext != ".go" && ext != ".py" {
			return nil
		}

		file, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := strings.TrimSpace(scanner.Text())

			var name, kind string

			switch ext {
			case ".go":
				if m := goFuncPattern.FindStringSubmatch(line); m != nil {
					name, kind = m[1], "func"
				} else if m := goTypePattern.FindStringSubmatch(line); m != nil {
					name, kind = m[1], "type"
				}
			case ".py":
				if m := pyFuncPattern.FindStringSubmatch(line); m != nil {
					name, kind = m[1], "func"
				} else if m := pyClassPattern.FindStringSubmatch(line); m != nil {
					name, kind = m[1], "class"
				}
			}

			if name != "" {
				exported := false
				if ext == ".go" {
					exported = name[0] >= 'A' && name[0] <= 'Z'
				} else {
					exported = !strings.HasPrefix(name, "_")
				}

				if exportedOnly && !exported {
					continue
				}
				if kindFilter != "" && kind != kindFilter {
					continue
				}

				symbols = append(symbols, symbol{
					Name:     name,
					Kind:     kind,
					File:     p,
					Line:     lineNum,
					Exported: exported,
				})
			}
		}
		return nil
	})

	data, _ := json.MarshalIndent(symbols, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) importsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	path, ok := args["path"].(string)
	if !ok || path == "" {
		return mcp.NewToolResultError("path is required"), nil
	}

	var results []importInfoMCP

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if filepath.Ext(p) != ".go" {
			return nil
		}

		if strings.Contains(p, "/vendor/") {
			return nil
		}

		content, err := os.ReadFile(p)
		if err != nil {
			return nil
		}

		// Find import block
		importPattern := regexp.MustCompile(`import\s*\(\s*([\s\S]*?)\s*\)`)
		singleImport := regexp.MustCompile(`import\s+"([^"]+)"`)

		imp := importInfoMCP{File: p}

		if m := importPattern.FindStringSubmatch(string(content)); m != nil {
			lines := strings.Split(m[1], "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				line = strings.Trim(line, `"`)
				if line == "" || strings.HasPrefix(line, "//") {
					continue
				}
				// Remove alias if present
				parts := strings.Fields(line)
				if len(parts) > 1 {
					line = strings.Trim(parts[len(parts)-1], `"`)
				}

				categorizeImport(line, &imp)
			}
		} else if m := singleImport.FindAllStringSubmatch(string(content), -1); m != nil {
			for _, match := range m {
				categorizeImport(match[1], &imp)
			}
		}

		if len(imp.Internal)+len(imp.External)+len(imp.Stdlib) > 0 {
			results = append(results, imp)
		}
		return nil
	})

	data, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// importInfoMCP is used by the imports handler
type importInfoMCP struct {
	File     string   `json:"file"`
	Internal []string `json:"internal,omitempty"`
	External []string `json:"external,omitempty"`
	Stdlib   []string `json:"stdlib,omitempty"`
}

func categorizeImport(imp string, info *importInfoMCP) {
	if strings.Contains(imp, ".") {
		if strings.HasPrefix(imp, "github.com/dirtybirdnj/clood") {
			info.Internal = append(info.Internal, imp)
		} else {
			info.External = append(info.External, imp)
		}
	} else {
		info.Stdlib = append(info.Stdlib, imp)
	}
}

func (s *Server) contextHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	maxTokens := 4000
	if t, ok := args["max_tokens"].(float64); ok {
		maxTokens = int(t)
	}

	absPath, _ := filepath.Abs(path)
	projectName := filepath.Base(absPath)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Project: %s\n\n", projectName))

	// Count files
	fileCount := 0
	dirCount := 0
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if !strings.HasPrefix(info.Name(), ".") {
				dirCount++
			}
		} else {
			fileCount++
		}
		return nil
	})

	sb.WriteString(fmt.Sprintf("**Files:** %d files, %d directories\n\n", fileCount, dirCount))

	// Include README if present
	readmeNames := []string{"README.md", "README", "readme.md"}
	for _, name := range readmeNames {
		content, err := os.ReadFile(filepath.Join(path, name))
		if err == nil {
			sb.WriteString("## README\n\n")
			readmeContent := string(content)
			maxChars := maxTokens * 2
			if len(readmeContent) > maxChars {
				readmeContent = readmeContent[:maxChars] + "\n...(truncated)"
			}
			sb.WriteString(readmeContent)
			sb.WriteString("\n\n")
			break
		}
	}

	// Key files
	sb.WriteString("## Key Files\n\n")
	keyFiles := []string{"main.go", "go.mod", "package.json", "Cargo.toml", "Makefile", "Dockerfile"}
	for _, kf := range keyFiles {
		if _, err := os.Stat(filepath.Join(path, kf)); err == nil {
			sb.WriteString(fmt.Sprintf("- `%s`\n", kf))
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) capabilitiesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check if Ollama is available
	ollamaAvailable := false
	cfg, _ := config.Load()
	if cfg != nil {
		mgr := hosts.NewManager()
		mgr.AddHosts(cfg.Hosts)
		statuses := mgr.CheckAllHosts()
		for _, st := range statuses {
			if st.Online {
				ollamaAvailable = true
				break
			}
		}
	}

	capabilities := map[string]interface{}{
		"local_tools": []string{
			"clood_grep - Search codebase with regex (0 network, 0 tokens)",
			"clood_tree - Directory structure (0 network, 0 tokens)",
			"clood_symbols - Extract code symbols (0 network, 0 tokens)",
			"clood_imports - Dependency analysis (0 network, 0 tokens)",
			"clood_context - Project summary (0 network, 0 tokens)",
			"clood_system - Hardware detection (0 network, 0 tokens)",
			"clood_analyze - Static analysis for Go projects (0 network, 0 tokens)",
		},
		"local_ollama_tools": []string{
			"clood_ask - Query local LLM",
			"clood_hosts - Check Ollama hosts",
			"clood_models - List available models",
			"clood_health - System health check",
		},
		"ollama_available": ollamaAvailable,
		"recommendation":   "Use local_tools FIRST before any network requests. Use local_ollama_tools before cloud APIs.",
	}

	data, _ := json.MarshalIndent(capabilities, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) analyzeHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	runTests := false
	if rt, ok := args["run_tests"].(bool); ok {
		runTests = rt
	}

	// Run static analysis
	analysis, err := analyze.RunAnalysis(path, runTests)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Analysis failed: %v", err)), nil
	}

	// Return formatted for Claude consumption
	return mcp.NewToolResultText(analysis.FormatForClaude()), nil
}

// =============================================================================
// CRITICAL TOOL HANDLERS - Preflight and Web Search Gate
// =============================================================================

func (s *Server) preflightHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get current working directory
	cwd, _ := os.Getwd()

	// Check Ollama status
	ollamaStatus := "OFFLINE"
	var onlineHosts []string
	var availableModels []string

	cfg, _ := config.Load()
	if cfg != nil {
		mgr := hosts.NewManager()
		mgr.AddHosts(cfg.Hosts)
		statuses := mgr.CheckAllHosts()
		for _, st := range statuses {
			if st.Online {
				ollamaStatus = "ONLINE"
				onlineHosts = append(onlineHosts, fmt.Sprintf("%s (%dms)", st.Host.Name, st.Latency.Milliseconds()))
				for _, m := range st.Models {
					availableModels = append(availableModels, m.Name)
				}
			}
		}
	}

	// Build preflight report
	var sb strings.Builder
	sb.WriteString("🚀 CLOOD PREFLIGHT CHECK\n")
	sb.WriteString("========================\n\n")

	sb.WriteString("📍 Working Directory: " + cwd + "\n\n")

	sb.WriteString("🔧 LOCAL DISCOVERY TOOLS (Use these FIRST - instant, 0 tokens):\n")
	sb.WriteString("   • clood_grep \"pattern\"     - Search codebase (replaces web search)\n")
	sb.WriteString("   • clood_tree               - Project structure\n")
	sb.WriteString("   • clood_symbols path/      - Extract functions/types\n")
	sb.WriteString("   • clood_imports path/      - Dependency analysis\n")
	sb.WriteString("   • clood_context            - Generate project summary\n\n")

	sb.WriteString("🤖 OLLAMA STATUS: " + ollamaStatus + "\n")
	if len(onlineHosts) > 0 {
		sb.WriteString("   Online hosts: " + strings.Join(onlineHosts, ", ") + "\n")
		if len(availableModels) > 5 {
			sb.WriteString("   Models: " + strings.Join(availableModels[:5], ", ") + " + " + fmt.Sprintf("%d more", len(availableModels)-5) + "\n")
		} else if len(availableModels) > 0 {
			sb.WriteString("   Models: " + strings.Join(availableModels, ", ") + "\n")
		}
		sb.WriteString("   → Use clood_ask for local LLM queries (no cloud API needed)\n")
	} else {
		sb.WriteString("   No Ollama hosts available. Cloud LLM may be needed.\n")
	}
	sb.WriteString("\n")

	sb.WriteString("⚠️  BEFORE WEB SEARCH:\n")
	sb.WriteString("   Call clood_should_search_web first - it will redirect to local tools if possible.\n\n")

	sb.WriteString("📋 RECOMMENDED WORKFLOW:\n")
	sb.WriteString("   1. clood_tree → Understand project structure\n")
	sb.WriteString("   2. clood_grep → Find relevant code\n")
	sb.WriteString("   3. clood_symbols → Know the API surface\n")
	sb.WriteString("   4. clood_ask → Query local LLM if needed\n")
	sb.WriteString("   5. WebSearch → ONLY if above tools can't help\n")

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) shouldSearchWebHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	queryLower := strings.ToLower(query)

	// Patterns that indicate codebase questions (should use local tools)
	codebasePatterns := []struct {
		patterns []string
		tool     string
		reason   string
	}{
		{
			patterns: []string{"where is", "find file", "which file", "what file", "locate"},
			tool:     "clood_grep",
			reason:   "Finding files in codebase",
		},
		{
			patterns: []string{"how does", "how do", "what does", "explain", "understand"},
			tool:     "clood_grep + clood_context",
			reason:   "Understanding code requires reading it first",
		},
		{
			patterns: []string{"project structure", "directory", "folder", "layout", "codebase structure"},
			tool:     "clood_tree",
			reason:   "Project structure is local",
		},
		{
			patterns: []string{"function", "method", "class", "type", "interface", "signature"},
			tool:     "clood_symbols",
			reason:   "Code symbols are extractable locally",
		},
		{
			patterns: []string{"import", "depend", "package", "module", "require"},
			tool:     "clood_imports",
			reason:   "Dependency analysis is local",
		},
		{
			patterns: []string{"in this codebase", "in this project", "in this repo", "in our code"},
			tool:     "clood_grep",
			reason:   "Codebase questions should use local search",
		},
	}

	// Check for codebase patterns
	for _, cp := range codebasePatterns {
		for _, pattern := range cp.patterns {
			if strings.Contains(queryLower, pattern) {
				result := map[string]interface{}{
					"verdict":     "USE_LOCAL_TOOL",
					"tool":        cp.tool,
					"reason":      cp.reason,
					"instruction": fmt.Sprintf("Instead of web search, use: %s", cp.tool),
					"original_query": query,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(data)), nil
			}
		}
	}

	// Patterns that suggest local LLM can help
	llmPatterns := []string{
		"best practice", "how to implement", "pattern for", "approach to",
		"should i", "recommend", "suggestion",
	}

	for _, pattern := range llmPatterns {
		if strings.Contains(queryLower, pattern) {
			// Check if Ollama is available
			ollamaOnline := false
			cfg, _ := config.Load()
			if cfg != nil {
				mgr := hosts.NewManager()
				mgr.AddHosts(cfg.Hosts)
				for _, st := range mgr.CheckAllHosts() {
					if st.Online {
						ollamaOnline = true
						break
					}
				}
			}

			if ollamaOnline {
				result := map[string]interface{}{
					"verdict":     "USE_LOCAL_LLM",
					"tool":        "clood_ask",
					"reason":      "General coding question - local LLM can help",
					"instruction": "Use clood_ask to query local Ollama instead of web search",
					"original_query": query,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(data)), nil
			}
		}
	}

	// Web search is approved for external information
	externalPatterns := []string{
		"latest", "current", "news", "update", "release", "version",
		"documentation", "docs", "api reference", "official",
		"github.com", "stackoverflow", "npm", "pypi", "crates.io",
	}

	reason := "Query appears to need external/current information"
	for _, pattern := range externalPatterns {
		if strings.Contains(queryLower, pattern) {
			reason = fmt.Sprintf("Query contains '%s' - likely needs external source", pattern)
			break
		}
	}

	result := map[string]interface{}{
		"verdict":     "WEB_SEARCH_APPROVED",
		"reason":      reason,
		"reminder":    "After web search, prefer clood_ask for follow-up questions if Ollama is online",
		"original_query": query,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
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
