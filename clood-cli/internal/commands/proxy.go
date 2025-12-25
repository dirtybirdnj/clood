package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// ProxyState tracks active requests and host status
type ProxyState struct {
	mu            sync.RWMutex
	activeReqs    map[string]*ActiveRequest
	hostStats     map[string]*HostStats
	totalRequests int64
	atcURL        string
}

// ActiveRequest tracks an in-flight request
type ActiveRequest struct {
	ID        string    `json:"id"`
	Model     string    `json:"model"`
	Host      string    `json:"host"`
	StartTime time.Time `json:"start_time"`
	Prompt    string    `json:"prompt"`
}

// HostStats tracks per-host statistics
type HostStats struct {
	Name         string  `json:"name"`
	URL          string  `json:"url"`
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
	AvgLatency   float64 `json:"avg_latency_ms"`
	LastUsed     time.Time
	Online       bool
}

// OpenAI-compatible request/response types
type ChatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []ProxyChatMessage  `json:"messages"`
	Stream      bool                `json:"stream"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
}

type ProxyChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int              `json:"index"`
	Message      ProxyChatMessage `json:"message,omitempty"`
	Delta        ProxyChatMessage `json:"delta,omitempty"`
	FinishReason string           `json:"finish_reason,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ModelsResponse struct {
	Object string            `json:"object"`
	Data   []OpenAIModelInfo `json:"data"`
}

type OpenAIModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func newProxyState(atcURL string) *ProxyState {
	return &ProxyState{
		activeReqs: make(map[string]*ActiveRequest),
		hostStats:  make(map[string]*HostStats),
		atcURL:     atcURL,
	}
}

func (ps *ProxyState) sendATCEvent(eventType string, data interface{}) {
	if ps.atcURL == "" {
		return
	}
	event := map[string]interface{}{
		"type":      eventType,
		"timestamp": time.Now().Format(time.RFC3339),
		"data":      data,
	}
	body, _ := json.Marshal(event)
	go func() {
		resp, err := http.Post(ps.atcURL+"/events", "application/json", bytes.NewReader(body))
		if err == nil {
			resp.Body.Close()
		}
	}()
}

func (ps *ProxyState) selectHost(mgr *hosts.Manager, model string) (*hosts.Host, error) {
	statuses := mgr.CheckAllHosts()

	var bestHost *hosts.Host
	var bestLatency int64 = -1

	for _, s := range statuses {
		if !s.Online {
			continue
		}
		// Check if host has the model
		hasModel := false
		for _, m := range s.Models {
			if m.Name == model || strings.HasPrefix(m.Name, strings.Split(model, ":")[0]) {
				hasModel = true
				break
			}
		}
		if !hasModel {
			continue
		}

		// Prefer host with lowest latency
		if bestLatency < 0 || s.Latency.Milliseconds() < bestLatency {
			bestHost = s.Host
			bestLatency = s.Latency.Milliseconds()
		}
	}

	if bestHost == nil {
		return nil, fmt.Errorf("no host available with model %s", model)
	}

	return bestHost, nil
}

func ProxyCmd() *cobra.Command {
	var port int
	var atcURL string

	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "OpenAI-compatible proxy server for multi-host routing",
		Long: `Starts an OpenAI-compatible API server that routes requests to available Ollama hosts.

This allows you to use any OpenAI-compatible chat UI (like Open WebUI) with your
clood host garden. Requests are routed to the best available host, and events
are streamed to ATC for monitoring.

Examples:
  # Start proxy on default port
  clood proxy

  # Start with ATC integration
  clood proxy --atc http://localhost:8080

  # Custom port
  clood proxy --port 8000 --atc http://localhost:8080

Then configure Open WebUI to use http://localhost:8000 as the API endpoint.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Load host configuration
			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error loading config: "+err.Error()))
				return
			}

			mgr := hosts.NewManager()
			mgr.AddHosts(cfg.Hosts)
			mgr.AddHosts(hosts.DefaultHosts())

			state := newProxyState(atcURL)

			// Check initial host status
			statuses := mgr.CheckAllHosts()
			onlineCount := 0
			for _, s := range statuses {
				state.hostStats[s.Host.Name] = &HostStats{
					Name:   s.Host.Name,
					URL:    s.Host.URL,
					Online: s.Online,
				}
				if s.Online {
					onlineCount++
				}
			}

			// Setup HTTP handlers
			mux := http.NewServeMux()

			// Health check
			mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{"status":"ok"}`)
			})

			// List models (aggregate from all hosts)
			mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
				handleModels(w, r, mgr)
			})

			// Chat completions
			mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
				handleChatCompletions(w, r, mgr, state)
			})

			// Status endpoint for debugging
			mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
				state.mu.RLock()
				defer state.mu.RUnlock()

				status := map[string]interface{}{
					"total_requests": state.totalRequests,
					"active_requests": len(state.activeReqs),
					"hosts": state.hostStats,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(status)
			})

			// Start server
			addr := fmt.Sprintf(":%d", port)
			srv := &http.Server{
				Addr:    addr,
				Handler: mux,
			}

			fmt.Println(tui.RenderHeader("clood Proxy"))
			fmt.Println()
			fmt.Printf("  %s OpenAI-compatible API server\n", tui.SuccessStyle.Render("●"))
			fmt.Printf("  %s http://localhost%s\n", tui.MutedStyle.Render("URL:"), addr)
			fmt.Printf("  %s %d/%d online\n", tui.MutedStyle.Render("Hosts:"), onlineCount, len(statuses))
			if atcURL != "" {
				fmt.Printf("  %s %s\n", tui.MutedStyle.Render("ATC:"), atcURL)
			}
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("  Endpoints:"))
			fmt.Printf("    %s /v1/chat/completions\n", tui.MutedStyle.Render("POST"))
			fmt.Printf("    %s /v1/models\n", tui.MutedStyle.Render("GET"))
			fmt.Printf("    %s /status\n", tui.MutedStyle.Render("GET"))
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("  Configure Open WebUI with:"))
			fmt.Printf("    API Base URL: %s\n", tui.AccentStyle.Render(fmt.Sprintf("http://localhost:%d/v1", port)))
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("  Press Ctrl+C to stop"))
			fmt.Println()

			// Send startup event to ATC
			state.sendATCEvent("proxy_start", map[string]interface{}{
				"port":   port,
				"hosts":  onlineCount,
			})

			// Handle graceful shutdown
			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					fmt.Println(tui.ErrorStyle.Render("Server error: " + err.Error()))
					stop <- syscall.SIGTERM
				}
			}()

			<-stop
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("  Shutting down..."))
			state.sendATCEvent("proxy_stop", nil)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8000, "Port to listen on")
	cmd.Flags().StringVar(&atcURL, "atc", "", "ATC server URL for event streaming (e.g., http://localhost:8080)")

	return cmd
}

func handleModels(w http.ResponseWriter, r *http.Request, mgr *hosts.Manager) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	statuses := mgr.CheckAllHosts()

	// Collect unique models from all hosts
	modelSet := make(map[string]bool)
	for _, s := range statuses {
		if !s.Online {
			continue
		}
		for _, m := range s.Models {
			modelSet[m.Name] = true
		}
	}

	// Build response
	models := make([]OpenAIModelInfo, 0, len(modelSet))
	for name := range modelSet {
		models = append(models, OpenAIModelInfo{
			ID:      name,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "clood-proxy",
		})
	}

	resp := ModelsResponse{
		Object: "list",
		Data:   models,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleChatCompletions(w http.ResponseWriter, r *http.Request, mgr *hosts.Manager, state *ProxyState) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Select best host for this model
	host, err := state.selectHost(mgr, req.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Track request
	reqID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	prompt := ""
	if len(req.Messages) > 0 {
		prompt = req.Messages[len(req.Messages)-1].Content
	}

	state.mu.Lock()
	state.totalRequests++
	state.activeReqs[reqID] = &ActiveRequest{
		ID:        reqID,
		Model:     req.Model,
		Host:      host.Name,
		StartTime: time.Now(),
		Prompt:    truncateString(prompt, 100),
	}
	state.mu.Unlock()

	// Send start event to ATC
	state.sendATCEvent("request_start", map[string]interface{}{
		"id":     reqID,
		"model":  req.Model,
		"host":   host.Name,
		"prompt": truncateString(prompt, 100),
	})

	defer func() {
		state.mu.Lock()
		delete(state.activeReqs, reqID)
		state.mu.Unlock()
	}()

	// Forward request to selected host
	reqBody, _ := json.Marshal(req)
	proxyReq, err := http.NewRequest("POST", host.URL+"/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(proxyReq)
	if err != nil {
		state.sendATCEvent("request_error", map[string]interface{}{
			"id":    reqID,
			"host":  host.Name,
			"error": err.Error(),
		})
		http.Error(w, "Upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	// Handle streaming vs non-streaming
	if req.Stream {
		// Stream response and track tokens
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		totalTokens := 0
		for scanner.Scan() {
			line := scanner.Text()
			w.Write([]byte(line + "\n"))
			flusher.Flush()

			// Try to count tokens from streaming response
			if strings.HasPrefix(line, "data: ") && !strings.Contains(line, "[DONE]") {
				totalTokens++
			}
		}

		duration := time.Since(start)
		state.sendATCEvent("request_complete", map[string]interface{}{
			"id":       reqID,
			"host":     host.Name,
			"model":    req.Model,
			"duration": duration.Seconds(),
			"tokens":   totalTokens,
			"stream":   true,
		})
	} else {
		// Non-streaming: read full response
		body, _ := io.ReadAll(resp.Body)
		w.Write(body)

		// Parse response for token count
		var chatResp ChatCompletionResponse
		json.Unmarshal(body, &chatResp)

		duration := time.Since(start)
		state.sendATCEvent("request_complete", map[string]interface{}{
			"id":       reqID,
			"host":     host.Name,
			"model":    req.Model,
			"duration": duration.Seconds(),
			"tokens":   chatResp.Usage.TotalTokens,
			"stream":   false,
		})

		// Update host stats
		state.mu.Lock()
		if hs, ok := state.hostStats[host.Name]; ok {
			hs.RequestCount++
			hs.TotalTokens += int64(chatResp.Usage.TotalTokens)
			hs.LastUsed = time.Now()
		}
		state.mu.Unlock()
	}
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
