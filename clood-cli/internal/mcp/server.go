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
	"github.com/dirtybirdnj/clood/internal/clipboard"
	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/git"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/sqlite"
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
	s.mcpServer.AddTool(s.systemTool(), s.systemHandler)

	// LOCAL DISCOVERY TOOLS (0 network, 0 LLM tokens)
	// These should be used BEFORE any network requests or LLM calls
	s.mcpServer.AddTool(s.grepTool(), s.grepHandler)
	s.mcpServer.AddTool(s.treeTool(), s.treeHandler)
	s.mcpServer.AddTool(s.symbolsTool(), s.symbolsHandler)
	s.mcpServer.AddTool(s.importsTool(), s.importsHandler)
	s.mcpServer.AddTool(s.contextTool(), s.contextHandler)
	s.mcpServer.AddTool(s.analyzeTool(), s.analyzeHandler)

	// The main event: ask local models
	s.mcpServer.AddTool(s.askTool(), s.askHandler)

	// GIT: Enhanced git operations
	s.mcpServer.AddTool(s.gitDiffTool(), s.gitDiffHandler)
	s.mcpServer.AddTool(s.gitLogTool(), s.gitLogHandler)
	s.mcpServer.AddTool(s.gitBranchesTool(), s.gitBranchesHandler)
	s.mcpServer.AddTool(s.gitCreatePRTool(), s.gitCreatePRHandler)

	// SQLITE: Database query tools
	s.mcpServer.AddTool(s.sqliteQueryTool(), s.sqliteQueryHandler)
	s.mcpServer.AddTool(s.sqliteSchemaTool(), s.sqliteSchemaHandler)

	// CLIPBOARD: System clipboard access
	s.mcpServer.AddTool(s.clipboardReadTool(), s.clipboardReadHandler)
	s.mcpServer.AddTool(s.clipboardWriteTool(), s.clipboardWriteHandler)

	// CATFIGHT: Single-host model comparison
	s.mcpServer.AddTool(s.catfightTool(), s.catfightHandler)
}

// =============================================================================
// Tool Definitions
// =============================================================================

// =============================================================================
// CRITICAL TOOLS - Use these FIRST
// =============================================================================

func (s *Server) preflightTool() mcp.Tool {
	return mcp.NewTool("clood_preflight",
		mcp.WithDescription(`START HERE. Call this first in every session.

Returns: available tools, online hosts, models, recommended workflow.

WHEN TO USE: At session start, before any other clood tool.
WHEN NOT TO USE: Never skip this.

Cost: ZERO (instant, no network, no tokens).`),
	)
}

func (s *Server) shouldSearchWebTool() mcp.Tool {
	return mcp.NewTool("clood_should_search_web",
		mcp.WithDescription(`GATE: Call before any WebSearch to check if local tools can answer instead.

Returns: "USE_LOCAL_TOOL" with redirect, or "WEB_SEARCH_APPROVED" if needed.

WHEN TO USE: Before every WebSearch call.
WHEN NOT TO USE: Never skip - prevents unnecessary network requests.

Example: query="where is auth handled" → redirects to clood_grep

Cost: ZERO (instant, no network, no tokens).`),
		mcp.WithString("query", mcp.Required(), mcp.Description("The query you were about to search the web for")),
	)
}

// =============================================================================
// Infrastructure Tools
// =============================================================================

func (s *Server) hostsTool() mcp.Tool {
	return mcp.NewTool("clood_hosts",
		mcp.WithDescription(`Check which Ollama hosts are online and what models they have.

Returns: host status, latency, available models per host.

WHEN TO USE: Before clood_ask, to verify LLM is available.
WHEN NOT TO USE: If preflight already showed hosts are online.

Cost: Local network only (no internet), ZERO tokens.`),
	)
}

func (s *Server) systemTool() mcp.Tool {
	return mcp.NewTool("clood_system",
		mcp.WithDescription(`Show hardware specs and model recommendations.

Returns: CPU, RAM, GPU, VRAM, recommended models for this hardware.

WHEN TO USE: Choosing which model to use, or diagnosing performance.
WHEN NOT TO USE: If you already know the hardware capabilities.

Cost: ZERO (instant, no network, no tokens).`),
	)
}

func (s *Server) askTool() mcp.Tool {
	return mcp.NewTool("clood_ask",
		mcp.WithDescription(`Query local Ollama LLM. Use instead of cloud APIs.

PREREQUISITE: Run clood_hosts first to verify a host is online.

WHEN TO USE: Code questions, explanations, generation - after exhausting grep/symbols/context.
WHEN NOT TO USE: For codebase questions (use grep/symbols first), external docs (use WebSearch).

ROLES (optional):
- reviewer: Find bugs, security issues, suggest improvements
- coder: Generate clean, efficient code
- analyst: Explain behavior, identify patterns
- documenter: Write documentation

Cost: Local LLM tokens only, ZERO cloud/internet.`),
		mcp.WithString("prompt", mcp.Required(), mcp.Description("The prompt to send to the model")),
		mcp.WithString("model", mcp.Description("Specific model (default: best available)")),
		mcp.WithString("host", mcp.Description("Specific host (default: fastest)")),
		mcp.WithString("role", mcp.Description("Role: reviewer, coder, analyst, documenter")),
		mcp.WithBoolean("dialogue", mcp.Description("If true, model asks clarifying questions first")),
	)
}

// =============================================================================
// LOCAL DISCOVERY TOOLS (0 network, 0 LLM tokens)
// Use these BEFORE making any network requests or LLM calls
// =============================================================================

func (s *Server) grepTool() mcp.Tool {
	return mcp.NewTool("clood_grep",
		mcp.WithDescription(`Search codebase with regex. USE THIS instead of WebSearch for code questions.

WHEN TO USE:
- "where is X" → grep "X" files_only=true
- "what files contain Y" → grep "Y"
- "how does Z work" → grep "Z" then read results

WHEN NOT TO USE: External docs, current events (use WebSearch after should_search_web).

Cost: ZERO (instant, no network, no tokens).`),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Regex pattern to search for")),
		mcp.WithString("path", mcp.Description("Directory to search (default: current)")),
		mcp.WithBoolean("files_only", mcp.Description("Return only file names, not lines")),
		mcp.WithBoolean("ignore_case", mcp.Description("Case insensitive")),
		mcp.WithString("type", mcp.Description("File type filter: go, py, js, ts, rs")),
	)
}

func (s *Server) treeTool() mcp.Tool {
	return mcp.NewTool("clood_tree",
		mcp.WithDescription(`Show directory structure. Respects .gitignore.

WHEN TO USE: Understanding project layout, finding where code lives.
WHEN NOT TO USE: If you already know the structure from preflight.

Cost: ZERO (instant, no network, no tokens).`),
		mcp.WithString("path", mcp.Description("Directory to show (default: current)")),
		mcp.WithNumber("depth", mcp.Description("Max depth (default: 3)")),
	)
}

func (s *Server) symbolsTool() mcp.Tool {
	return mcp.NewTool("clood_symbols",
		mcp.WithDescription(`Extract function/type definitions from code. Supports Go, Python, JS/TS.

WHEN TO USE: Finding function signatures, listing types in a package.
WHEN NOT TO USE: Finding usages (use grep instead).

Cost: ZERO (instant, no network, no tokens).`),
		mcp.WithString("path", mcp.Required(), mcp.Description("File or directory to analyze")),
		mcp.WithBoolean("exported_only", mcp.Description("Only exported/public symbols")),
		mcp.WithString("kind", mcp.Description("Filter: func, type, class, const, var")),
	)
}

func (s *Server) importsTool() mcp.Tool {
	return mcp.NewTool("clood_imports",
		mcp.WithDescription(`Analyze Go imports and dependencies.

Returns: internal, external, and stdlib imports categorized.

WHEN TO USE: Understanding what a file depends on, refactoring.
WHEN NOT TO USE: Non-Go files (not yet supported).

Cost: ZERO (instant, no network, no tokens).`),
		mcp.WithString("path", mcp.Required(), mcp.Description("File or directory to analyze")),
	)
}

func (s *Server) contextTool() mcp.Tool {
	return mcp.NewTool("clood_context",
		mcp.WithDescription(`Generate LLM-ready project summary (README, structure, key files).

WHEN TO USE: Getting oriented in a new project, before asking clood_ask about the codebase.
WHEN NOT TO USE: If you've already read the specific files you need.

Cost: ZERO (instant, no network, no tokens).`),
		mcp.WithString("path", mcp.Description("Directory to analyze (default: current)")),
		mcp.WithNumber("max_tokens", mcp.Description("Target token count (default: 4000)")),
	)
}

func (s *Server) analyzeTool() mcp.Tool {
	return mcp.NewTool("clood_analyze",
		mcp.WithDescription(`Run static analysis on Go codebase: build, vet, TODOs, recent commits.

WHEN TO USE: Before making changes, to understand codebase health.
WHEN NOT TO USE: Non-Go projects.

Returns: build status, vet issues, TODOs, hot files, symbol counts.

Cost: ZERO tokens (runs go build/vet locally, may take a few seconds).`),
		mcp.WithString("path", mcp.Description("Directory to analyze (default: current)")),
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

func (s *Server) systemHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hw, err := system.DetectHardware()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error detecting hardware: %v", err)), nil
	}

	data, _ := json.MarshalIndent(hw.JSON(), "", "  ")
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
	role, _ := args["role"].(string)
	dialogue, _ := args["dialogue"].(bool)

	// Build system prompt based on role (from delegate functionality)
	var systemPrompt string
	switch role {
	case "reviewer":
		systemPrompt = "You are a code review specialist. Analyze code for bugs, security issues, and improvements. Be concise and actionable."
	case "coder":
		systemPrompt = "You are a code generation specialist. Write clean, efficient code. Include comments where helpful."
	case "analyst":
		systemPrompt = "You are a code analysis specialist. Explain code behavior, identify patterns, and provide insights."
	case "documenter":
		systemPrompt = "You are a documentation specialist. Write clear, helpful documentation for code and APIs."
	}

	// Add dialogue system prompt if requested (appends to role prompt)
	if dialogue {
		if systemPrompt != "" {
			systemPrompt += "\n\n" + dialogueSystemPrompt
		} else {
			systemPrompt = dialogueSystemPrompt
		}
	}

	// Prepend system prompt to user prompt if set
	if systemPrompt != "" {
		prompt = systemPrompt + "\n\nUser request:\n" + prompt
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

// =============================================================================
// GIT TOOLS - Enhanced git operations
// =============================================================================

func (s *Server) gitDiffTool() mcp.Tool {
	return mcp.NewTool("clood_git_diff",
		mcp.WithDescription(`Show git diff for files, commits, or staged changes.

WHEN TO USE: Before committing, to review changes. When investigating what changed.
WHEN NOT TO USE: For commit history (use clood_git_log instead).

Options: specific file, compare against commit, staged only, summary stats.

Cost: ZERO (local git operation, instant).`),
		mcp.WithString("path", mcp.Description("Repository path (default: current directory)")),
		mcp.WithString("file", mcp.Description("Specific file to diff")),
		mcp.WithString("commit", mcp.Description("Compare against specific commit (e.g., HEAD~1)")),
		mcp.WithBoolean("staged", mcp.Description("Show only staged changes")),
		mcp.WithBoolean("stat", mcp.Description("Show summary stats instead of full diff")),
	)
}

func (s *Server) gitLogTool() mcp.Tool {
	return mcp.NewTool("clood_git_log",
		mcp.WithDescription(`Show commit history with filtering options.

WHEN TO USE: To understand recent changes, find who changed what, search commit messages.
WHEN NOT TO USE: For file contents (use clood_grep). For current changes (use clood_git_diff).

Filters: author, date range, message search, specific file.

Cost: ZERO (local git operation, instant).`),
		mcp.WithString("path", mcp.Description("Repository path (default: current directory)")),
		mcp.WithNumber("count", mcp.Description("Number of commits to show (default: 20)")),
		mcp.WithString("author", mcp.Description("Filter by author name/email")),
		mcp.WithString("since", mcp.Description("Show commits since date (e.g., '2024-01-01')")),
		mcp.WithString("grep", mcp.Description("Search in commit messages")),
		mcp.WithString("file", mcp.Description("Show only commits affecting this file")),
	)
}

func (s *Server) gitBranchesTool() mcp.Tool {
	return mcp.NewTool("clood_git_branches",
		mcp.WithDescription(`List git branches (local and optionally remote).

WHEN TO USE: To see available branches, check current branch, find feature branches.
WHEN NOT TO USE: For commit history (use clood_git_log).

Cost: ZERO (local git operation, instant).`),
		mcp.WithString("path", mcp.Description("Repository path (default: current directory)")),
		mcp.WithBoolean("remote", mcp.Description("Include remote branches")),
	)
}

func (s *Server) gitCreatePRTool() mcp.Tool {
	return mcp.NewTool("clood_git_create_pr",
		mcp.WithDescription(`Create a GitHub Pull Request from current branch.

WHEN TO USE: After committing and pushing changes, to create a PR for review.
WHEN NOT TO USE: Before pushing (push first). For local-only changes.

Requires: gh CLI installed and authenticated.

Cost: One GitHub API call.`),
		mcp.WithString("title", mcp.Required(), mcp.Description("PR title (short, descriptive)")),
		mcp.WithString("body", mcp.Description("PR description (markdown supported, explain what and why)")),
		mcp.WithString("base", mcp.Description("Base branch to merge into (default: main)")),
		mcp.WithString("head", mcp.Description("Head branch (default: current branch)")),
		mcp.WithBoolean("draft", mcp.Description("Create as draft PR (default: false)")),
		mcp.WithString("path", mcp.Description("Repository path (default: current directory)")),
	)
}

// Git handlers

func (s *Server) gitDiffHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	opts := git.DiffOptions{
		Path: path,
	}

	if file, ok := args["file"].(string); ok {
		opts.File = file
	}
	if commit, ok := args["commit"].(string); ok {
		opts.Commit = commit
	}
	if staged, ok := args["staged"].(bool); ok {
		opts.Staged = staged
	}
	if stat, ok := args["stat"].(bool); ok {
		opts.Stat = stat
	}

	diff, err := git.Diff(opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("git diff failed: %v", err)), nil
	}

	if diff == "" {
		return mcp.NewToolResultText("No changes detected"), nil
	}

	return mcp.NewToolResultText(diff), nil
}

func (s *Server) gitLogHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	opts := git.LogOptions{
		Path: path,
	}

	if count, ok := args["count"].(float64); ok {
		opts.Count = int(count)
	}
	if author, ok := args["author"].(string); ok {
		opts.Author = author
	}
	if since, ok := args["since"].(string); ok {
		opts.Since = since
	}
	if grep, ok := args["grep"].(string); ok {
		opts.Grep = grep
	}
	if file, ok := args["file"].(string); ok {
		opts.File = file
	}

	entries, err := git.Log(opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("git log failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(entries, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) gitBranchesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	includeRemote := false
	if remote, ok := args["remote"].(bool); ok {
		includeRemote = remote
	}

	branches, err := git.Branches(path, includeRemote)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("git branches failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(branches, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) gitCreatePRHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	title, ok := args["title"].(string)
	if !ok || title == "" {
		return mcp.NewToolResultError("title is required"), nil
	}

	opts := git.CreatePROptions{
		Title: title,
	}

	if path, ok := args["path"].(string); ok && path != "" {
		opts.Path = path
	}
	if body, ok := args["body"].(string); ok {
		opts.Body = body
	}
	if base, ok := args["base"].(string); ok && base != "" {
		opts.Base = base
	}
	if head, ok := args["head"].(string); ok && head != "" {
		opts.Head = head
	}
	if draft, ok := args["draft"].(bool); ok {
		opts.Draft = draft
	}

	result, err := git.CreatePR(opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("create PR failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// =============================================================================
// SQLITE TOOLS - Database query capabilities
// =============================================================================

func (s *Server) sqliteQueryTool() mcp.Tool {
	return mcp.NewTool("clood_sqlite_query",
		mcp.WithDescription(`Execute a SELECT query on a local SQLite database.

WHEN TO USE: Querying app data, browsing local databases, extracting structured data.
WHEN NOT TO USE: Modifying data (read-only: SELECT, PRAGMA, EXPLAIN only).

Returns: JSON array of matching rows.

Cost: ZERO (local file access, instant).`),
		mcp.WithString("database", mcp.Required(), mcp.Description("Path to the SQLite database file")),
		mcp.WithString("query", mcp.Required(), mcp.Description("SQL SELECT query to execute")),
	)
}

func (s *Server) sqliteSchemaTool() mcp.Tool {
	return mcp.NewTool("clood_sqlite_schema",
		mcp.WithDescription(`Show schema for a SQLite database or specific table.

WHEN TO USE: Before querying, to understand table structure and column types.
WHEN NOT TO USE: If you already know the schema.

Returns: Column names, types, and constraints. Omit table param to list all tables.

Cost: ZERO (local file access, instant).`),
		mcp.WithString("database", mcp.Required(), mcp.Description("Path to the SQLite database file")),
		mcp.WithString("table", mcp.Description("Table name (omit to show all tables and schemas)")),
	)
}

// SQLite handlers

func (s *Server) sqliteQueryHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	dbPath, ok := args["database"].(string)
	if !ok || dbPath == "" {
		return mcp.NewToolResultError("database path is required"), nil
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	result, err := sqlite.Query(dbPath, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("query failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) sqliteSchemaHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	dbPath, ok := args["database"].(string)
	if !ok || dbPath == "" {
		return mcp.NewToolResultError("database path is required"), nil
	}

	tableName, _ := args["table"].(string)

	if tableName != "" {
		// Show schema for specific table
		info, err := sqlite.Schema(dbPath, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("schema failed: %v", err)), nil
		}
		data, _ := json.MarshalIndent(info, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}

	// Show schema for all tables
	infos, err := sqlite.DatabaseInfo(dbPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("database info failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(infos, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// =============================================================================
// CLIPBOARD TOOLS - System clipboard access
// =============================================================================

func (s *Server) clipboardReadTool() mcp.Tool {
	return mcp.NewTool("clood_clipboard_read",
		mcp.WithDescription(`Read current system clipboard contents.

WHEN TO USE: When user says "use what I copied" or references clipboard content.
WHEN NOT TO USE: For reading files (use file tools instead).

Returns: Text currently in clipboard.

Cost: ZERO (local system call, instant).`),
	)
}

func (s *Server) clipboardWriteTool() mcp.Tool {
	return mcp.NewTool("clood_clipboard_write",
		mcp.WithDescription(`Write text to the system clipboard.

WHEN TO USE: When user asks to copy something, or to share generated code/results.
WHEN NOT TO USE: For saving to files (use file tools instead).

Cost: ZERO (local system call, instant).`),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to copy to clipboard")),
	)
}

// Clipboard handlers

func (s *Server) clipboardReadHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text, err := clipboard.Read()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("clipboard read failed: %v", err)), nil
	}

	if text == "" {
		return mcp.NewToolResultText("Clipboard is empty"), nil
	}

	return mcp.NewToolResultText(text), nil
}

func (s *Server) clipboardWriteHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	text, ok := args["text"].(string)
	if !ok || text == "" {
		return mcp.NewToolResultError("text is required"), nil
	}

	if err := clipboard.Write(text); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("clipboard write failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Copied %d characters to clipboard", len(text))), nil
}

// =============================================================================
// CATFIGHT: Single-host model comparison
// =============================================================================

func (s *Server) catfightTool() mcp.Tool {
	return mcp.NewTool("clood_catfight",
		mcp.WithDescription(`Compare 2-5 models head-to-head on the same prompt.

WHEN TO USE: Testing which model handles a prompt best. Benchmarking local models.
WHEN NOT TO USE: For single queries (use clood_ask). When speed matters more than comparison.

Returns: All model responses with timing metrics for comparison.

Cost: Runs on LOCAL Ollama only. Tokens used per model queried.`),
		mcp.WithString("prompt", mcp.Required(), mcp.Description("The prompt to send to all models")),
		mcp.WithString("models", mcp.Description("Comma-separated models to compare (default: qwen2.5-coder:3b,mistral:7b,llama3.1:8b)")),
		mcp.WithString("host", mcp.Description("Target host (default: localhost)")),
	)
}

func (s *Server) catfightHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	prompt, _ := args["prompt"].(string)
	modelsStr, _ := args["models"].(string)
	hostName, _ := args["host"].(string)

	if prompt == "" {
		return mcp.NewToolResultError("prompt is required"), nil
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return mcp.NewToolResultError("Error loading config: " + err.Error()), nil
	}

	// Setup host manager
	mgr := hosts.NewManager()
	mgr.AddHosts(cfg.Hosts)

	// Find host
	if hostName == "" {
		hostName = "localhost"
	}

	var targetHost *hosts.Host
	if hostName == "localhost" {
		targetHost = &hosts.Host{Name: "localhost", URL: "http://localhost:11434"}
	} else {
		targetHost = mgr.GetHost(hostName)
	}

	if targetHost == nil {
		return mcp.NewToolResultError("Host not found: " + hostName), nil
	}

	// Check host status
	client := ollama.NewClient(targetHost.URL, 5*time.Minute)
	availableModels, err := client.ListModels()
	if err != nil {
		return mcp.NewToolResultError("Host offline or error: " + err.Error()), nil
	}

	// Parse models or use defaults
	var modelsToTest []string
	if modelsStr != "" {
		for _, m := range strings.Split(modelsStr, ",") {
			modelsToTest = append(modelsToTest, strings.TrimSpace(m))
		}
	} else {
		// Default models - use first 3 available
		defaults := []string{"qwen2.5-coder:3b", "mistral:7b", "llama3.1:8b"}
		for _, d := range defaults {
			for _, a := range availableModels {
				if a.Name == d {
					modelsToTest = append(modelsToTest, d)
					break
				}
			}
		}
		// If no defaults found, use first 3 available
		if len(modelsToTest) == 0 {
			for i, m := range availableModels {
				if i >= 3 {
					break
				}
				modelsToTest = append(modelsToTest, m.Name)
			}
		}
	}

	if len(modelsToTest) == 0 {
		return mcp.NewToolResultError("No models available for catfight"), nil
	}

	// Run catfight
	type catfightResult struct {
		Model    string
		Response string
		Duration time.Duration
		Tokens   int
		TokSec   float64
		Error    string
	}

	var results []catfightResult
	for _, model := range modelsToTest {
		start := time.Now()
		resp, err := client.Generate(model, prompt)
		duration := time.Since(start)

		result := catfightResult{
			Model:    model,
			Duration: duration,
		}

		if err != nil {
			result.Error = err.Error()
		} else {
			result.Response = resp.Response
			result.Tokens = resp.EvalCount
			if resp.EvalDuration > 0 {
				result.TokSec = float64(resp.EvalCount) / (float64(resp.EvalDuration) / 1e9)
			}
		}

		results = append(results, result)
	}

	// Build output
	var out strings.Builder
	out.WriteString("🐱 CATFIGHT RESULTS\n")
	out.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	out.WriteString(fmt.Sprintf("Host: %s | Models: %d | Prompt: %s\n\n", hostName, len(modelsToTest), truncateStr(prompt, 50)))

	for i, r := range results {
		out.WriteString(fmt.Sprintf("── %d. %s ", i+1, r.Model))
		if r.Error != "" {
			out.WriteString(fmt.Sprintf("(ERROR: %s)\n\n", r.Error))
			continue
		}
		out.WriteString(fmt.Sprintf("(%.1fs, %d tok, %.1f tok/s) ──\n", r.Duration.Seconds(), r.Tokens, r.TokSec))
		out.WriteString(r.Response)
		out.WriteString("\n\n")
	}

	return mcp.NewToolResultText(out.String()), nil
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
