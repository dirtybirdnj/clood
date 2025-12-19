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

	// LOCAL DISCOVERY TOOLS (0 network, 0 LLM tokens)
	// These should be used BEFORE any network requests or LLM calls
	s.mcpServer.AddTool(s.grepTool(), s.grepHandler)
	s.mcpServer.AddTool(s.treeTool(), s.treeHandler)
	s.mcpServer.AddTool(s.symbolsTool(), s.symbolsHandler)
	s.mcpServer.AddTool(s.importsTool(), s.importsHandler)
	s.mcpServer.AddTool(s.contextTool(), s.contextHandler)
	s.mcpServer.AddTool(s.capabilitiesTool(), s.capabilitiesHandler)

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
// LOCAL DISCOVERY TOOLS (0 network, 0 LLM tokens)
// Use these BEFORE making any network requests or LLM calls
// =============================================================================

func (s *Server) grepTool() mcp.Tool {
	return mcp.NewTool("clood_grep",
		mcp.WithDescription("Search codebase with regex. ZERO network calls, ZERO LLM tokens. Use this FIRST before web searches. Returns matching files and lines."),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Regex pattern to search for")),
		mcp.WithString("path", mcp.Description("Directory to search in (default: current directory)")),
		mcp.WithBoolean("files_only", mcp.Description("Only return file names, not matching lines")),
		mcp.WithBoolean("ignore_case", mcp.Description("Case insensitive search")),
		mcp.WithString("type", mcp.Description("Filter by file type: go, py, js, ts, rs, etc.")),
	)
}

func (s *Server) treeTool() mcp.Tool {
	return mcp.NewTool("clood_tree",
		mcp.WithDescription("Display directory tree structure. ZERO network calls, ZERO LLM tokens. Respects .gitignore. Use to understand project layout."),
		mcp.WithString("path", mcp.Description("Directory to show (default: current directory)")),
		mcp.WithNumber("depth", mcp.Description("Maximum depth to traverse (default: 3)")),
	)
}

func (s *Server) symbolsTool() mcp.Tool {
	return mcp.NewTool("clood_symbols",
		mcp.WithDescription("Extract code symbols (functions, types, classes). ZERO network calls, ZERO LLM tokens. Supports Go, Python, JS/TS."),
		mcp.WithString("path", mcp.Required(), mcp.Description("File or directory to analyze")),
		mcp.WithBoolean("exported_only", mcp.Description("Only show exported/public symbols")),
		mcp.WithString("kind", mcp.Description("Filter by kind: func, type, class, const, var")),
	)
}

func (s *Server) importsTool() mcp.Tool {
	return mcp.NewTool("clood_imports",
		mcp.WithDescription("Analyze file imports and dependencies. ZERO network calls, ZERO LLM tokens. Shows internal, external, and stdlib imports."),
		mcp.WithString("path", mcp.Required(), mcp.Description("File or directory to analyze")),
	)
}

func (s *Server) contextTool() mcp.Tool {
	return mcp.NewTool("clood_context",
		mcp.WithDescription("Generate LLM-optimized project context. ZERO network calls, ZERO LLM tokens. Includes README, structure, key files."),
		mcp.WithString("path", mcp.Description("Directory to analyze (default: current directory)")),
		mcp.WithNumber("max_tokens", mcp.Description("Target token count (default: 4000)")),
	)
}

func (s *Server) capabilitiesTool() mcp.Tool {
	return mcp.NewTool("clood_capabilities",
		mcp.WithDescription("List what clood can do locally vs what requires network. Use this to plan your approach before starting a task."),
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
