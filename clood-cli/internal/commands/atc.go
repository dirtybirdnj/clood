package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/google/go-github/v50/github"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

// IssuePacket represents an issue for the frontend display
type IssuePacket struct {
	ID             int64    `json:"id"`
	Number         int      `json:"number"`
	Title          string   `json:"title"`
	State          string   `json:"state"`
	Labels         []string `json:"labels"`
	Assignee       string   `json:"assignee,omitempty"`
	AssigneeAvatar string   `json:"assignee_avatar,omitempty"`
	Velocity       int      `json:"velocity"`
	Delta          int      `json:"delta"`
	Epic           string   `json:"epic,omitempty"`
	IsPR           bool     `json:"is_pr"`
	UpdatedAt      string   `json:"updated_at"`
	Body           string   `json:"body,omitempty"`
}

// HardwareSpec contains static hardware info for a host
type HardwareSpec struct {
	CPU    string `json:"cpu"`
	GPU    string `json:"gpu"`
	Memory string `json:"memory"`
}

// HostStatus represents a host's current state for active mode
type HostStatus struct {
	Name      string       `json:"name"`
	URL       string       `json:"url"`
	Online    bool         `json:"online"`
	Latency   int64        `json:"latency_ms"`
	Models    []string     `json:"models"`
	ActiveReq int          `json:"active_requests"`
	LastSeen  string       `json:"last_seen"`
	Hardware  HardwareSpec `json:"hardware"`
}

// CatfightEvent represents a catfight event for the dashboard
type CatfightEvent struct {
	Type      string      `json:"type"` // "start", "progress", "complete"
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// === Hierarchical Event Types for Experiments ===

// ExperimentSession tracks the overall experiment state
type ExperimentSession struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	StartTime   string                 `json:"start_time"`
	Status      string                 `json:"status"` // "running", "completed", "failed"
	CurrentStep int                    `json:"current_step"`
	TotalSteps  int                    `json:"total_steps"`
	Steps       []ExperimentStep       `json:"steps"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ExperimentStep represents a step within a session
type ExperimentStep struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Number     int                   `json:"number"`
	Status     string                `json:"status"` // "pending", "running", "completed", "failed"
	StartTime  string                `json:"start_time,omitempty"`
	EndTime    string                `json:"end_time,omitempty"`
	Iterations []ExperimentIteration `json:"iterations"`
	Validation *ValidationResult     `json:"validation,omitempty"`
}

// ExperimentIteration represents a single LLM attempt within a step
type ExperimentIteration struct {
	Number      int     `json:"number"`
	Model       string  `json:"model"`
	Host        string  `json:"host"`
	Status      string  `json:"status"` // "running", "completed", "failed"
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time,omitempty"`
	DurationSec float64 `json:"duration_sec,omitempty"`
	Tokens      int     `json:"tokens,omitempty"`
	TokensSec   float64 `json:"tokens_sec,omitempty"`
	ContextSize int     `json:"context_size,omitempty"`
	Error       string  `json:"error,omitempty"`
}

// ValidationResult represents the outcome of a validation check
type ValidationResult struct {
	Command  string   `json:"command"`
	Status   string   `json:"status"` // "pass", "fail", "skip"
	Output   string   `json:"output,omitempty"`
	Errors   []string `json:"errors,omitempty"`
	Duration float64  `json:"duration_sec,omitempty"`
}

// ExperimentEvent is the unified event type for experiment tracking
type ExperimentEvent struct {
	Type      string      `json:"type"` // "session_start", "session_complete", "step_start", "step_complete", "iteration_start", "iteration_complete", "validation"
	Timestamp string      `json:"timestamp"`
	SessionID string      `json:"session_id"`
	StepID    string      `json:"step_id,omitempty"`
	Data      interface{} `json:"data"`
}

// Static hardware specs for known hosts
var hostHardware = map[string]HardwareSpec{
	"mac-laptop": {CPU: "Apple M4 Max", GPU: "M4 Max 40-core", Memory: "128GB"},
	"mac-mini":   {CPU: "Apple M4", GPU: "M4 10-core", Memory: "24GB"},
	"ubuntu25":   {CPU: "i7-8700", GPU: "RX 590 8GB", Memory: "64GB"},
}

// ATCMessage is the WebSocket message format
type ATCMessage struct {
	Type   string      `json:"type"` // "issues", "hosts", "event"
	Data   interface{} `json:"data"`
	Mode   string      `json:"mode"` // "planning" or "active"
	Time   string      `json:"time"`
}

// Hub maintains active WebSocket connections and broadcasts updates
type Hub struct {
	clients      map[*websocket.Conn]bool
	broadcast    chan ATCMessage
	register     chan *websocket.Conn
	unregister   chan *websocket.Conn
	mu           sync.Mutex
	lastData     *ATCMessage // Cache last data for new clients
	lastDataMu   sync.RWMutex
	events       []CatfightEvent // Recent catfight events
	eventsMu     sync.RWMutex
	pollInterval time.Duration
	pollMu       sync.RWMutex

	// Experiment tracking
	sessions   map[string]*ExperimentSession // Active experiment sessions
	sessionsMu sync.RWMutex
	expEvents  []ExperimentEvent // Recent experiment events
	expEventsMu sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		clients:      make(map[*websocket.Conn]bool),
		broadcast:    make(chan ATCMessage),
		register:     make(chan *websocket.Conn),
		unregister:   make(chan *websocket.Conn),
		events:       make([]CatfightEvent, 0),
		pollInterval: 10 * time.Second,
		sessions:     make(map[string]*ExperimentSession),
		expEvents:    make([]ExperimentEvent, 0),
	}
}

// Session management methods
func (h *Hub) getSession(id string) *ExperimentSession {
	h.sessionsMu.RLock()
	defer h.sessionsMu.RUnlock()
	return h.sessions[id]
}

func (h *Hub) setSession(session *ExperimentSession) {
	h.sessionsMu.Lock()
	h.sessions[session.ID] = session
	h.sessionsMu.Unlock()
}

func (h *Hub) getAllSessions() []*ExperimentSession {
	h.sessionsMu.RLock()
	defer h.sessionsMu.RUnlock()
	result := make([]*ExperimentSession, 0, len(h.sessions))
	for _, s := range h.sessions {
		result = append(result, s)
	}
	return result
}

func (h *Hub) addExpEvent(event ExperimentEvent) {
	h.expEventsMu.Lock()
	h.expEvents = append(h.expEvents, event)
	// Keep only last 100 experiment events
	if len(h.expEvents) > 100 {
		h.expEvents = h.expEvents[len(h.expEvents)-100:]
	}
	h.expEventsMu.Unlock()
}

func (h *Hub) getExpEvents() []ExperimentEvent {
	h.expEventsMu.RLock()
	defer h.expEventsMu.RUnlock()
	result := make([]ExperimentEvent, len(h.expEvents))
	copy(result, h.expEvents)
	return result
}

func (h *Hub) setLastData(msg ATCMessage) {
	h.lastDataMu.Lock()
	h.lastData = &msg
	h.lastDataMu.Unlock()
}

func (h *Hub) getLastData() *ATCMessage {
	h.lastDataMu.RLock()
	defer h.lastDataMu.RUnlock()
	return h.lastData
}

func (h *Hub) addEvent(event CatfightEvent) {
	h.eventsMu.Lock()
	h.events = append(h.events, event)
	// Keep only last 50 events
	if len(h.events) > 50 {
		h.events = h.events[len(h.events)-50:]
	}
	h.eventsMu.Unlock()
}

func (h *Hub) getEvents() []CatfightEvent {
	h.eventsMu.RLock()
	defer h.eventsMu.RUnlock()
	result := make([]CatfightEvent, len(h.events))
	copy(result, h.events)
	return result
}

func (h *Hub) setPollInterval(d time.Duration) {
	h.pollMu.Lock()
	h.pollInterval = d
	h.pollMu.Unlock()
}

func (h *Hub) getPollInterval() time.Duration {
	h.pollMu.RLock()
	defer h.pollMu.RUnlock()
	return h.pollInterval
}

func (h *Hub) run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.Lock()
			data, _ := json.Marshal(msg)
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
					conn.Close()
					delete(h.clients, conn)
				}
			}
			h.mu.Unlock()
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	hub.register <- conn

	// Send cached data immediately so client doesn't wait for next poll
	if lastData := hub.getLastData(); lastData != nil {
		data, _ := json.Marshal(lastData)
		conn.WriteMessage(websocket.TextMessage, data)
	}

	// Send any recent events
	events := hub.getEvents()
	if len(events) > 0 {
		eventMsg := ATCMessage{
			Type: "events",
			Data: events,
			Time: time.Now().Format(time.RFC3339),
		}
		data, _ := json.Marshal(eventMsg)
		conn.WriteMessage(websocket.TextMessage, data)
	}

	go func() {
		defer func() { hub.unregister <- conn }()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// atcFetchIssues fetches issues and calculates velocity scores
func atcFetchIssues(owner, repo string) []IssuePacket {
	client := github.NewClient(nil)
	ctx := context.Background()

	issues, _, err := client.Issues.ListByRepo(ctx, owner, repo, &github.IssueListByRepoOptions{
		State:     "open",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 50,
		},
	})
	if err != nil {
		return nil
	}

	var packets []IssuePacket
	for _, issue := range issues {
		packet := IssuePacket{
			ID:        issue.GetID(),
			Number:    issue.GetNumber(),
			Title:     issue.GetTitle(),
			State:     issue.GetState(),
			IsPR:      issue.IsPullRequest(),
			UpdatedAt: issue.GetUpdatedAt().Format(time.RFC3339),
			Body:      truncateStr(issue.GetBody(), 200),
		}

		for _, label := range issue.Labels {
			packet.Labels = append(packet.Labels, label.GetName())
			if len(label.GetName()) > 5 && label.GetName()[:5] == "epic:" {
				packet.Epic = label.GetName()[5:]
			}
		}

		if issue.Assignee != nil {
			packet.Assignee = issue.Assignee.GetLogin()
			packet.AssigneeAvatar = issue.Assignee.GetAvatarURL()
		}

		packet.Velocity = calculateIssueVelocity(issue)
		packets = append(packets, packet)
	}

	return packets
}

func calculateIssueVelocity(issue *github.Issue) int {
	velocity := 0

	hourAgo := time.Now().Add(-1 * time.Hour)
	dayAgo := time.Now().Add(-24 * time.Hour)

	if issue.GetUpdatedAt().After(hourAgo) {
		velocity += 100
	} else if issue.GetUpdatedAt().After(dayAgo) {
		velocity += 50
	}

	for _, label := range issue.Labels {
		switch label.GetName() {
		case "P0", "critical", "urgent":
			velocity += 500
		case "P1", "high":
			velocity += 300
		case "P2", "medium":
			velocity += 100
		case "bug":
			velocity += 50
		case "enhancement", "feature":
			velocity += 25
		case "epic":
			velocity += 200
		}
	}

	velocity += issue.GetComments() * 10
	return velocity
}

// atcFetchHostStatus gets current status of all Ollama hosts
func atcFetchHostStatus() []HostStatus {
	mgr := hosts.NewManager()
	mgr.AddHosts(hosts.DefaultHosts())
	hostStatuses := mgr.CheckAllHosts()

	var statuses []HostStatus
	for _, hs := range hostStatuses {
		status := HostStatus{
			Name:     hs.Host.Name,
			URL:      hs.Host.URL,
			Online:   hs.Online,
			Latency:  hs.Latency.Milliseconds(),
			LastSeen: time.Now().Format(time.RFC3339),
		}
		// Add hardware specs if known
		if hw, ok := hostHardware[hs.Host.Name]; ok {
			status.Hardware = hw
		}
		if hs.Online {
			for _, m := range hs.Models {
				status.Models = append(status.Models, m.Name)
			}
		}
		statuses = append(statuses, status)
	}

	return statuses
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// Helper functions for parsing event data
func getString(data map[string]interface{}, key string) string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(data map[string]interface{}, key string) int {
	if v, ok := data[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func getFloat(data map[string]interface{}, key string) float64 {
	if v, ok := data[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func ATCCmd() *cobra.Command {
	var port int
	var mode string
	var owner string
	var repo string

	cmd := &cobra.Command{
		Use:   "atc",
		Short: "Air Traffic Control - Visual dashboard for issues and hosts",
		Long: `ATC (Air Traffic Control) provides real-time visual dashboards.

Two modes available:
  planning  - GitHub issues ranked by velocity (racing leaderboard)
  active    - Host status, model availability, request throughput

Examples:
  clood atc --mode planning        # Issues dashboard
  clood atc --mode active          # Host monitoring
  clood atc --port 8080            # Custom port`,
		Run: func(cmd *cobra.Command, args []string) {
			hub := newHub()
			go hub.run()

			// Select HTML based on mode
			var htmlContent string
			switch mode {
			case "active":
				htmlContent = atcActiveHTML
			case "experiment":
				htmlContent = atcExperimentHTML
			default:
				htmlContent = atcPlanningHTML
			}

			// Start the appropriate poller with dynamic interval
			go func() {
				fetchAndBroadcast := func() {
					var msg ATCMessage
					switch mode {
					case "active":
						hostsData := atcFetchHostStatus()
						msg = ATCMessage{
							Type: "hosts",
							Data: hostsData,
							Mode: mode,
							Time: time.Now().Format(time.RFC3339),
						}
					case "experiment":
						// For experiment mode, send session state
						sessions := hub.getAllSessions()
						hostsData := atcFetchHostStatus()
						msg = ATCMessage{
							Type: "experiment_state",
							Data: map[string]interface{}{
								"sessions": sessions,
								"hosts":    hostsData,
							},
							Mode: mode,
							Time: time.Now().Format(time.RFC3339),
						}
					default:
						issues := atcFetchIssues(owner, repo)
						msg = ATCMessage{
							Type: "issues",
							Data: issues,
							Mode: mode,
							Time: time.Now().Format(time.RFC3339),
						}
					}
					hub.setLastData(msg)
					hub.broadcast <- msg
				}

				// Initial fetch
				fetchAndBroadcast()

				// Dynamic interval polling
				for {
					time.Sleep(hub.getPollInterval())
					fetchAndBroadcast()
				}
			}()

			// HTTP handlers
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(htmlContent))
			})
			http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
				serveWs(hub, w, r)
			})

			// Events endpoint for catfight to POST to
			http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "POST only", http.StatusMethodNotAllowed)
					return
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				var event CatfightEvent
				if err := json.Unmarshal(body, &event); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				event.Timestamp = time.Now().Format(time.RFC3339)
				hub.addEvent(event)

				// Broadcast event to all clients
				hub.broadcast <- ATCMessage{
					Type: "event",
					Data: event,
					Mode: mode,
					Time: event.Timestamp,
				}
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{"status":"ok"}`)
			})

			// Experiment events endpoint for hierarchical session/step/iteration tracking
			http.HandleFunc("/experiment", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "POST only", http.StatusMethodNotAllowed)
					return
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				var event ExperimentEvent
				if err := json.Unmarshal(body, &event); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				event.Timestamp = time.Now().Format(time.RFC3339)

				// Handle session state updates based on event type
				switch event.Type {
				case "session_start":
					if data, ok := event.Data.(map[string]interface{}); ok {
						session := &ExperimentSession{
							ID:        event.SessionID,
							Name:      getString(data, "name"),
							StartTime: event.Timestamp,
							Status:    "running",
							TotalSteps: getInt(data, "total_steps"),
							Steps:     make([]ExperimentStep, 0),
						}
						hub.setSession(session)
					}
				case "session_complete", "session_fail":
					if session := hub.getSession(event.SessionID); session != nil {
						if event.Type == "session_complete" {
							session.Status = "completed"
						} else {
							session.Status = "failed"
						}
						hub.setSession(session)
					}
				case "step_start":
					if session := hub.getSession(event.SessionID); session != nil {
						if data, ok := event.Data.(map[string]interface{}); ok {
							step := ExperimentStep{
								ID:        event.StepID,
								Name:      getString(data, "name"),
								Number:    getInt(data, "number"),
								Status:    "running",
								StartTime: event.Timestamp,
								Iterations: make([]ExperimentIteration, 0),
							}
							session.Steps = append(session.Steps, step)
							session.CurrentStep = step.Number
							hub.setSession(session)
						}
					}
				case "step_complete", "step_fail":
					if session := hub.getSession(event.SessionID); session != nil {
						for i := range session.Steps {
							if session.Steps[i].ID == event.StepID {
								if event.Type == "step_complete" {
									session.Steps[i].Status = "completed"
								} else {
									session.Steps[i].Status = "failed"
								}
								session.Steps[i].EndTime = event.Timestamp
								break
							}
						}
						hub.setSession(session)
					}
				case "iteration_start", "iteration_complete", "iteration_fail":
					if session := hub.getSession(event.SessionID); session != nil {
						for i := range session.Steps {
							if session.Steps[i].ID == event.StepID {
								if event.Type == "iteration_start" {
									if data, ok := event.Data.(map[string]interface{}); ok {
										iter := ExperimentIteration{
											Number:    getInt(data, "number"),
											Model:     getString(data, "model"),
											Host:      getString(data, "host"),
											Status:    "running",
											StartTime: event.Timestamp,
										}
										session.Steps[i].Iterations = append(session.Steps[i].Iterations, iter)
									}
								} else {
									// Update last iteration
									iters := &session.Steps[i].Iterations
									if len(*iters) > 0 {
										last := &(*iters)[len(*iters)-1]
										if event.Type == "iteration_complete" {
											last.Status = "completed"
										} else {
											last.Status = "failed"
										}
										last.EndTime = event.Timestamp
										if data, ok := event.Data.(map[string]interface{}); ok {
											last.DurationSec = getFloat(data, "duration_sec")
											last.Tokens = getInt(data, "tokens")
											last.TokensSec = getFloat(data, "tokens_sec")
											last.Error = getString(data, "error")
										}
									}
								}
								break
							}
						}
						hub.setSession(session)
					}
				case "validation":
					if session := hub.getSession(event.SessionID); session != nil {
						for i := range session.Steps {
							if session.Steps[i].ID == event.StepID {
								if data, ok := event.Data.(map[string]interface{}); ok {
									session.Steps[i].Validation = &ValidationResult{
										Command:  getString(data, "command"),
										Status:   getString(data, "status"),
										Output:   getString(data, "output"),
										Duration: getFloat(data, "duration_sec"),
									}
									if errors, ok := data["errors"].([]interface{}); ok {
										for _, e := range errors {
											if s, ok := e.(string); ok {
												session.Steps[i].Validation.Errors = append(session.Steps[i].Validation.Errors, s)
											}
										}
									}
								}
								break
							}
						}
						hub.setSession(session)
					}
				}

				hub.addExpEvent(event)

				// Broadcast event to all clients
				hub.broadcast <- ATCMessage{
					Type: "experiment",
					Data: event,
					Mode: mode,
					Time: event.Timestamp,
				}
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{"status":"ok"}`)
			})

			// Sessions endpoint to get current experiment state
			http.HandleFunc("/sessions", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				sessions := hub.getAllSessions()
				json.NewEncoder(w).Encode(sessions)
			})

			// Poll interval control
			http.HandleFunc("/poll", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					// Return current interval
					interval := hub.getPollInterval()
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprintf(w, `{"interval_seconds":%d}`, int(interval.Seconds()))
					return
				}
				if r.Method == http.MethodPost {
					// Set new interval
					seconds := r.URL.Query().Get("seconds")
					if seconds == "" {
						http.Error(w, "seconds param required", http.StatusBadRequest)
						return
					}
					var secs int
					fmt.Sscanf(seconds, "%d", &secs)
					if secs < 1 || secs > 300 {
						http.Error(w, "seconds must be 1-300", http.StatusBadRequest)
						return
					}
					hub.setPollInterval(time.Duration(secs) * time.Second)
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprintf(w, `{"status":"ok","interval_seconds":%d}`, secs)
					return
				}
				http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
			})

			addr := fmt.Sprintf(":%d", port)
			fmt.Println(tui.RenderHeader("ATC Tower"))
			fmt.Printf("  %s Mode: %s\n", tui.SuccessStyle.Render("●"), mode)
			fmt.Printf("  %s http://localhost%s\n", tui.MutedStyle.Render("URL:"), addr)
			if mode == "planning" {
				fmt.Printf("  %s %s/%s\n", tui.MutedStyle.Render("Repo:"), owner, repo)
			}
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("  Press Ctrl+C to stop"))
			fmt.Println()

			http.ListenAndServe(addr, nil)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to serve dashboard")
	cmd.Flags().StringVarP(&mode, "mode", "m", "planning", "Dashboard mode: planning or active")
	cmd.Flags().StringVar(&owner, "owner", "dirtybirdnj", "GitHub repo owner")
	cmd.Flags().StringVar(&repo, "repo", "clood", "GitHub repo name")

	return cmd
}

// Planning mode HTML - Issues leaderboard
var atcPlanningHTML = `<!DOCTYPE html>
<html>
<head>
    <title>ATC Tower - Planning Mode</title>
    <meta charset="utf-8">
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            background: linear-gradient(135deg, #0a0a1a 0%, #1a1a2e 100%);
            color: #eee;
            font-family: 'Courier New', monospace;
            min-height: 100vh;
        }
        .container { max-width: 1400px; margin: 0 auto; padding: 20px; }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
            padding-bottom: 15px;
            border-bottom: 2px solid #333;
        }
        .header h1 {
            color: #00ff88;
            font-size: 28px;
            text-shadow: 0 0 20px rgba(0,255,136,0.5);
        }
        .status { display: flex; gap: 20px; font-size: 14px; color: #888; }
        .status .live { color: #00ff88; animation: pulse 2s infinite; }
        @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
        .main { display: grid; grid-template-columns: 1fr 400px; gap: 20px; }
        .leaderboard {
            background: rgba(22, 33, 62, 0.8);
            border-radius: 12px;
            padding: 20px;
            border: 1px solid #333;
        }
        .leaderboard-header {
            display: flex;
            padding: 10px 15px;
            color: #666;
            font-size: 12px;
            text-transform: uppercase;
            border-bottom: 1px solid #333;
            margin-bottom: 10px;
        }
        .issue {
            display: flex;
            align-items: center;
            padding: 12px 15px;
            border-radius: 8px;
            margin-bottom: 4px;
            background: rgba(0,0,0,0.3);
            transition: all 0.5s ease-out;
        }
        .issue:hover { background: rgba(0,255,136,0.1); transform: translateX(5px); }
        .rank { width: 40px; font-size: 20px; font-weight: bold; color: #00ff88; }
        .rank-1 { color: #ffd700; text-shadow: 0 0 10px rgba(255,215,0,0.5); }
        .rank-2 { color: #c0c0c0; }
        .rank-3 { color: #cd7f32; }
        .delta { width: 35px; font-size: 16px; text-align: center; }
        .delta.up { color: #00ff88; }
        .delta.down { color: #ff4444; }
        .number { width: 70px; color: #888; font-size: 14px; }
        .title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        .labels { display: flex; gap: 5px; flex-wrap: wrap; max-width: 200px; }
        .label { background: #333; padding: 3px 8px; border-radius: 4px; font-size: 11px; color: #aaa; }
        .label.bug { background: #5c2020; color: #ff8888; }
        .label.epic { background: #1a3a5c; color: #88aaff; }
        .label.enhancement { background: #1a3a1a; color: #88ff88; }
        .velocity { width: 80px; text-align: right; color: #ffaa00; font-weight: bold; }
        .departures-panel { display: flex; flex-direction: column; gap: 15px; }
        .departures {
            background: #000;
            border-radius: 12px;
            padding: 20px;
            border: 3px solid #333;
        }
        .departures h2 { color: #ffaa00; margin-bottom: 15px; font-size: 16px; text-transform: uppercase; letter-spacing: 3px; }
        .flap-board { background: #111; border-radius: 8px; padding: 15px; }
        .flap-row { display: flex; align-items: center; padding: 8px 0; border-bottom: 1px solid #222; }
        .flap-row:last-child { border-bottom: none; }
        .flap-index { width: 30px; color: #ffaa00; font-weight: bold; }
        .flap-text { flex: 1; display: flex; gap: 2px; }
        .flap-char {
            width: 14px; height: 24px; background: #1a1a1a; border: 1px solid #333;
            display: flex; align-items: center; justify-content: center;
            font-size: 14px; color: #fff; text-transform: uppercase;
        }
        .current-box {
            background: linear-gradient(135deg, #1a2a1a 0%, #16213e 100%);
            border-radius: 12px;
            padding: 20px;
            border-left: 4px solid #00ff88;
        }
        .current-box h3 {
            color: #00ff88;
            font-size: 14px;
            text-transform: uppercase;
            letter-spacing: 2px;
            margin-bottom: 15px;
        }
        .current-box h3::before { content: '● '; animation: blink 1s infinite; }
        @keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.3; } }
        .current-issue { font-size: 18px; margin-bottom: 10px; }
        .current-synopsis { font-size: 13px; color: #888; line-height: 1.5; }
        .footer {
            margin-top: 20px;
            padding-top: 15px;
            border-top: 1px solid #333;
            display: flex;
            justify-content: space-between;
            color: #666;
            font-size: 12px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>✈️ ATC TOWER - PLANNING MODE</h1>
            <div class="status">
                <span class="live">● LIVE</span>
                <span id="issue-count">-- issues</span>
                <span id="last-update">--</span>
            </div>
        </div>
        <div class="main">
            <div class="leaderboard">
                <div class="leaderboard-header">
                    <span style="width:40px">#</span>
                    <span style="width:35px">Δ</span>
                    <span style="width:70px">Issue</span>
                    <span style="flex:1">Title</span>
                    <span style="width:200px">Labels</span>
                    <span style="width:80px;text-align:right">Score</span>
                </div>
                <div id="leaderboard"><p style="padding:20px;color:#666">Connecting to tower...</p></div>
            </div>
            <div class="departures-panel">
                <div class="departures">
                    <h2>📋 Top Priority</h2>
                    <div class="flap-board" id="departures"></div>
                </div>
                <div class="current-box">
                    <h3>Currently Active</h3>
                    <div id="current"><div class="current-issue">Awaiting data...</div></div>
                </div>
            </div>
        </div>
        <div class="footer">
            <span>clood atc --mode planning</span>
            <span id="connection-status">Connecting...</span>
        </div>
    </div>
    <script>
        let previousRanks = {};
        function connect() {
            const ws = new WebSocket('ws://' + location.host + '/ws');
            ws.onopen = () => {
                document.getElementById('connection-status').textContent = 'Connected';
                document.getElementById('connection-status').style.color = '#00ff88';
            };
            ws.onclose = () => {
                document.getElementById('connection-status').textContent = 'Reconnecting...';
                document.getElementById('connection-status').style.color = '#ff4444';
                setTimeout(connect, 3000);
            };
            ws.onmessage = function(e) {
                const msg = JSON.parse(e.data);
                if (msg.type === 'issues') {
                    renderLeaderboard(msg.data);
                    renderDepartures(msg.data);
                    document.getElementById('issue-count').textContent = msg.data.length + ' issues';
                    document.getElementById('last-update').textContent = new Date().toLocaleTimeString();
                }
            };
        }
        function renderLeaderboard(issues) {
            const sorted = issues.sort((a, b) => b.velocity - a.velocity);
            const html = sorted.slice(0, 15).map((issue, i) => {
                const rank = i + 1;
                const prevRank = previousRanks[issue.number];
                let deltaClass = 'same', deltaSymbol = '─';
                if (prevRank !== undefined) {
                    if (prevRank > rank) { deltaClass = 'up'; deltaSymbol = '▲' + (prevRank - rank); }
                    else if (prevRank < rank) { deltaClass = 'down'; deltaSymbol = '▼' + (rank - prevRank); }
                }
                previousRanks[issue.number] = rank;
                const rankClass = rank <= 3 ? 'rank-' + rank : '';
                const labels = (issue.labels || []).map(l => {
                    const cls = ['bug','epic','enhancement'].includes(l) ? l : '';
                    return '<span class="label ' + cls + '">' + l + '</span>';
                }).join('');
                return '<div class="issue"><span class="rank ' + rankClass + '">' + rank + '</span>' +
                    '<span class="delta ' + deltaClass + '">' + deltaSymbol + '</span>' +
                    '<span class="number">#' + issue.number + '</span>' +
                    '<span class="title">' + escapeHtml(issue.title) + '</span>' +
                    '<span class="labels">' + labels + '</span>' +
                    '<span class="velocity">' + issue.velocity + '</span></div>';
            }).join('');
            document.getElementById('leaderboard').innerHTML = html;
        }
        function renderDepartures(issues) {
            const sorted = issues.sort((a, b) => b.velocity - a.velocity);
            const rows = sorted.slice(0, 5).map((issue, i) => {
                const text = ('#' + issue.number + ' ' + issue.title).toUpperCase().substring(0, 45).padEnd(45);
                const chars = text.split('').map(c => '<span class="flap-char">' + escapeHtml(c) + '</span>').join('');
                return '<div class="flap-row"><span class="flap-index">' + (i + 1) + '</span><div class="flap-text">' + chars + '</div></div>';
            }).join('');
            document.getElementById('departures').innerHTML = rows;
            if (sorted[0]) {
                document.getElementById('current').innerHTML =
                    '<div class="current-issue">#' + sorted[0].number + ' ' + escapeHtml(sorted[0].title) + '</div>' +
                    '<div class="current-synopsis">' + escapeHtml(sorted[0].body || 'No description') + '</div>';
            }
        }
        function escapeHtml(text) {
            if (!text) return '';
            return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
        }
        connect();
    </script>
</body>
</html>`

// Active mode HTML - Host monitoring with events panel
var atcActiveHTML = `<!DOCTYPE html>
<html>
<head>
    <title>ATC Tower - Active Mode</title>
    <meta charset="utf-8">
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            background: linear-gradient(135deg, #0a1a0a 0%, #1a2e1a 100%);
            color: #eee;
            font-family: 'Courier New', monospace;
            min-height: 100vh;
        }
        .container { max-width: 1900px; margin: 0 auto; padding: 20px; }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
            padding-bottom: 15px;
            border-bottom: 2px solid #2a4a2a;
        }
        .header h1 { color: #00ff88; font-size: 28px; text-shadow: 0 0 20px rgba(0,255,136,0.5); }
        .status { display: flex; gap: 20px; font-size: 14px; color: #888; align-items: center; }
        .status .live { color: #00ff88; animation: pulse 2s infinite; }
        @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
        .poll-controls { display: flex; gap: 5px; align-items: center; }
        .poll-btn { background: #2a4a2a; border: 1px solid #3a5a3a; color: #888; padding: 4px 10px; border-radius: 4px; cursor: pointer; font-size: 12px; }
        .poll-btn:hover { background: #3a5a3a; color: #fff; }
        .poll-btn.active { background: #00ff88; color: #000; }

        /* Three column layout for hosts */
        .hosts-row { display: grid; grid-template-columns: repeat(3, 1fr); gap: 15px; margin-bottom: 20px; }
        .host {
            background: rgba(22, 62, 33, 0.8);
            border-radius: 12px;
            padding: 15px;
            border: 3px solid #2a4a2a;
            transition: all 0.3s ease;
        }
        /* Host color coding */
        .host[data-host="local-gpu"] { border-color: #4488ff; }
        .host[data-host="local-gpu"] .host-name { color: #4488ff; }
        .host[data-host="local-gpu"] .host-color { background: #4488ff; }
        .host[data-host="ubuntu25"] { border-color: #ff8844; }
        .host[data-host="ubuntu25"] .host-name { color: #ff8844; }
        .host[data-host="ubuntu25"] .host-color { background: #ff8844; }
        .host[data-host="mac-mini"] { border-color: #44ff88; }
        .host[data-host="mac-mini"] .host-name { color: #44ff88; }
        .host[data-host="mac-mini"] .host-color { background: #44ff88; }

        .host.offline { opacity: 0.5; border-color: #4a2a2a !important; background: rgba(62, 22, 22, 0.5); }
        .host-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
        .host-name { font-size: 20px; font-weight: bold; display: flex; align-items: center; gap: 8px; }
        .host-color { width: 12px; height: 12px; border-radius: 50%; }
        .host-status { padding: 3px 10px; border-radius: 20px; font-size: 11px; text-transform: uppercase; }
        .host-status.online { background: #1a4a1a; color: #00ff88; }
        .host-status.offline { background: #4a1a1a; color: #ff4444; }
        .host-specs { background: rgba(0,0,0,0.3); border-radius: 6px; padding: 10px; margin-bottom: 10px; font-size: 12px; }
        .spec-row { display: flex; justify-content: space-between; padding: 3px 0; border-bottom: 1px solid #222; }
        .spec-row:last-child { border-bottom: none; }
        .spec-label { color: #666; }
        .spec-value { color: #aaa; }
        .host-stats { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin-bottom: 10px; }
        .stat { background: rgba(0,0,0,0.3); padding: 8px; border-radius: 6px; }
        .stat-label { font-size: 10px; color: #666; text-transform: uppercase; margin-bottom: 3px; }
        .stat-value { font-size: 16px; color: #fff; }
        .stat-value.good { color: #00ff88; }
        .stat-value.warn { color: #ffaa00; }
        .stat-value.bad { color: #ff4444; }
        .models { background: rgba(0,0,0,0.3); border-radius: 6px; padding: 10px; }
        .models h4 { font-size: 10px; color: #666; text-transform: uppercase; margin-bottom: 8px; }
        .model-list { display: flex; flex-wrap: wrap; gap: 4px; max-height: 60px; overflow-y: auto; }
        .model-tag { background: #333; padding: 3px 8px; border-radius: 4px; font-size: 11px; color: #aaa; transition: all 0.3s ease; }
        .model-tag.active { background: #ffaa00; color: #000; animation: model-pulse 1s infinite; box-shadow: 0 0 10px rgba(255,170,0,0.5); }
        @keyframes model-pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.7; } }

        /* Full width events section */
        .events-section {
            background: rgba(0, 0, 0, 0.6);
            border-radius: 12px;
            padding: 20px;
            border: 2px solid #2a4a2a;
        }
        .events-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; }
        .events-header h2 { color: #ffaa00; font-size: 18px; text-transform: uppercase; letter-spacing: 2px; }
        .battle-stats { display: flex; gap: 30px; }
        .battle-stat { text-align: center; }
        .battle-stat-value { font-size: 24px; font-weight: bold; color: #fff; }
        .battle-stat-label { font-size: 10px; color: #666; text-transform: uppercase; }
        .events-list { display: flex; flex-direction: column; gap: 8px; max-height: 400px; overflow-y: auto; }
        .event-item {
            background: rgba(0,0,0,0.4);
            border-radius: 8px;
            padding: 12px 16px;
            border-left: 4px solid #00ff88;
            display: flex;
            align-items: center;
            gap: 15px;
            width: 100%;
        }
        .event-item.start { border-left-color: #ffaa00; background: rgba(255,170,0,0.1); }
        .event-item.complete { border-left-color: #00ff88; background: rgba(0,255,136,0.1); }
        .event-item.progress { border-left-color: #888; }
        .event-item.analysis { border-left-color: #aa88ff; background: rgba(170,136,255,0.15); border-width: 4px; }
        /* Host-colored events */
        .event-item.host-local-gpu { border-left-color: #4488ff; }
        .event-item.host-ubuntu25 { border-left-color: #ff8844; }
        .event-item.host-mac-mini { border-left-color: #44ff88; }
        .event-type { font-size: 11px; font-weight: bold; text-transform: uppercase; padding: 4px 10px; border-radius: 4px; background: #333; white-space: nowrap; min-width: 80px; text-align: center; }
        .event-type.start { background: #ffaa00; color: #000; }
        .event-type.complete { background: #00ff88; color: #000; }
        .event-type.analysis { background: #aa88ff; color: #000; }
        .event-time { font-size: 11px; color: #666; white-space: nowrap; min-width: 70px; }
        .event-host { font-size: 11px; padding: 4px 10px; border-radius: 4px; font-weight: bold; white-space: nowrap; min-width: 90px; text-align: center; }
        .event-host.local-gpu, .event-host.localhost { background: rgba(68,136,255,0.3); color: #4488ff; }
        .event-host.ubuntu25 { background: rgba(255,136,68,0.3); color: #ff8844; }
        .event-host.mac-mini { background: rgba(68,255,136,0.3); color: #44ff88; }
        .event-content { flex: 1; font-size: 13px; color: #ccc; overflow: hidden; text-overflow: ellipsis; }
        .event-stats { display: flex; gap: 20px; font-size: 12px; color: #888; white-space: nowrap; }
        .event-stats .stat-item { display: flex; align-items: center; gap: 5px; }
        .event-stats .stat-value { color: #fff; font-weight: bold; }
        .no-events { color: #666; font-size: 14px; text-align: center; padding: 40px; }

        /* Analysis Panel */
        .analysis-section {
            background: linear-gradient(135deg, rgba(170,136,255,0.15) 0%, rgba(100,80,180,0.1) 100%);
            border-radius: 12px;
            padding: 20px;
            margin-top: 20px;
            border: 2px solid #aa88ff;
        }
        .analysis-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; }
        .analysis-header h2 { color: #aa88ff; font-size: 18px; text-transform: uppercase; letter-spacing: 2px; margin: 0; }
        .analysis-time { color: #666; font-size: 12px; }
        .analysis-content { display: flex; flex-direction: column; gap: 12px; }
        .analysis-summary { font-size: 16px; color: #fff; line-height: 1.6; }
        .analysis-summary strong { color: #aa88ff; }
        .analysis-rankings {
            background: rgba(0,0,0,0.3);
            padding: 12px 16px;
            border-radius: 8px;
            font-size: 13px;
            color: #aaa;
            font-family: monospace;
        }
        .analysis-rankings .rank { color: #ffaa00; font-weight: bold; }

        .footer {
            margin-top: 20px;
            padding-top: 15px;
            border-top: 1px solid #2a4a2a;
            display: flex;
            justify-content: space-between;
            color: #666;
            font-size: 12px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🖥️ ATC TOWER - ACTIVE MODE</h1>
            <div class="status">
                <span class="live">● MONITORING</span>
                <span id="host-count">-- hosts</span>
                <div class="poll-controls">
                    <span style="color:#666">Poll:</span>
                    <button class="poll-btn" onclick="setPoll(5)">5s</button>
                    <button class="poll-btn active" onclick="setPoll(10)">10s</button>
                    <button class="poll-btn" onclick="setPoll(30)">30s</button>
                    <button class="poll-btn" onclick="setPoll(60)">60s</button>
                </div>
                <span id="last-update">--</span>
            </div>
        </div>

        <!-- Hosts in 3 columns -->
        <div class="hosts-row" id="hosts">
            <p style="padding:40px;color:#666;text-align:center;grid-column:1/-1">Scanning hosts...</p>
        </div>

        <!-- Full width events section -->
        <div class="events-section">
            <div class="events-header">
                <h2>🏟️ Catfight Arena</h2>
                <div class="battle-stats">
                    <div class="battle-stat">
                        <div class="battle-stat-value" id="stat-battles">0</div>
                        <div class="battle-stat-label">Battles</div>
                    </div>
                    <div class="battle-stat">
                        <div class="battle-stat-value" id="stat-models">0</div>
                        <div class="battle-stat-label">Models Run</div>
                    </div>
                    <div class="battle-stat">
                        <div class="battle-stat-value" id="stat-tokens">0</div>
                        <div class="battle-stat-label">Total Tokens</div>
                    </div>
                    <div class="battle-stat">
                        <div class="battle-stat-value" id="stat-avgspeed">--</div>
                        <div class="battle-stat-label">Avg tok/s</div>
                    </div>
                </div>
            </div>
            <div class="events-list" id="events">
                <div class="no-events">No catfight events yet. Run: <code>clood catfight --atc http://localhost:8080 --all-hosts "prompt"</code></div>
            </div>
        </div>

        <!-- Analysis Panel -->
        <div class="analysis-section" id="analysis-section" style="display:none;">
            <div class="analysis-header">
                <h2>🔬 Battle Analysis</h2>
                <span class="analysis-time" id="analysis-time"></span>
            </div>
            <div class="analysis-content">
                <div class="analysis-summary" id="analysis-summary"></div>
                <div class="analysis-rankings" id="analysis-rankings"></div>
            </div>
        </div>

        <div class="footer">
            <span>clood atc --mode active | Host colors: <span style="color:#4488ff">●</span> local-gpu <span style="color:#ff8844">●</span> ubuntu25 <span style="color:#44ff88">●</span> mac-mini</span>
            <span id="connection-status">Connecting...</span>
        </div>
    </div>
    <script>
        let currentPoll = 10;
        let battleStats = { battles: 0, models: 0, tokens: 0, speeds: [] };
        let activeModels = {}; // { hostName: modelName }

        const hostColors = {
            'local-gpu': '#4488ff',
            'localhost': '#4488ff',
            'ubuntu25': '#ff8844',
            'mac-mini': '#44ff88'
        };

        function setPoll(seconds) {
            fetch('/poll?seconds=' + seconds, {method: 'POST'})
                .then(() => {
                    currentPoll = seconds;
                    document.querySelectorAll('.poll-btn').forEach(b => b.classList.remove('active'));
                    event.target.classList.add('active');
                });
        }
        function connect() {
            const ws = new WebSocket('ws://' + location.host + '/ws');
            ws.onopen = () => {
                document.getElementById('connection-status').textContent = 'Connected';
                document.getElementById('connection-status').style.color = '#00ff88';
            };
            ws.onclose = () => {
                document.getElementById('connection-status').textContent = 'Reconnecting...';
                document.getElementById('connection-status').style.color = '#ff4444';
                setTimeout(connect, 3000);
            };
            ws.onmessage = function(e) {
                const msg = JSON.parse(e.data);
                if (msg.type === 'hosts') {
                    renderHosts(msg.data);
                    const online = msg.data.filter(h => h.online).length;
                    document.getElementById('host-count').textContent = online + '/' + msg.data.length + ' online';
                    document.getElementById('last-update').textContent = new Date().toLocaleTimeString();
                }
                if (msg.type === 'event') {
                    addEvent(msg.data);
                    highlightActiveModel(msg.data);
                    if (msg.data.type === 'analysis') {
                        showAnalysis(msg.data);
                    }
                }
                if (msg.type === 'events') {
                    msg.data.forEach(e => {
                        addEvent(e);
                        if (e.type === 'analysis') showAnalysis(e);
                    });
                }
            };
        }
        function renderHosts(hosts) {
            const html = hosts.map(host => {
                const statusClass = host.online ? 'online' : 'offline';
                const hostClass = host.online ? '' : 'offline';
                const latencyClass = host.latency_ms < 50 ? 'good' : host.latency_ms < 200 ? 'warn' : 'bad';
                const models = (host.models || []).slice(0, 6).map(m => '<span class="model-tag">' + m + '</span>').join('');
                const moreModels = (host.models || []).length > 6 ? '<span class="model-tag">+' + ((host.models || []).length - 6) + ' more</span>' : '';
                const hw = host.hardware || {};
                return '<div class="host ' + hostClass + '" data-host="' + host.name + '">' +
                    '<div class="host-header"><span class="host-name"><span class="host-color"></span>' + host.name + '</span>' +
                    '<span class="host-status ' + statusClass + '">' + statusClass + '</span></div>' +
                    '<div class="host-specs">' +
                    '<div class="spec-row"><span class="spec-label">CPU</span><span class="spec-value">' + (hw.cpu || '--') + '</span></div>' +
                    '<div class="spec-row"><span class="spec-label">GPU</span><span class="spec-value">' + (hw.gpu || '--') + '</span></div>' +
                    '<div class="spec-row"><span class="spec-label">Memory</span><span class="spec-value">' + (hw.memory || '--') + '</span></div>' +
                    '</div>' +
                    '<div class="host-stats">' +
                    '<div class="stat"><div class="stat-label">Latency</div><div class="stat-value ' + latencyClass + '">' +
                    (host.online ? host.latency_ms + 'ms' : '--') + '</div></div>' +
                    '<div class="stat"><div class="stat-label">Models</div><div class="stat-value">' +
                    (host.models ? host.models.length : 0) + '</div></div></div>' +
                    '<div class="models"><h4>Available Models</h4><div class="model-list">' +
                    (models + moreModels || '<span style="color:#666">None loaded</span>') + '</div></div></div>';
            }).join('');
            document.getElementById('hosts').innerHTML = html;
        }
        function addEvent(event) {
            const container = document.getElementById('events');
            if (container.querySelector('.no-events')) {
                container.innerHTML = '';
            }
            const time = event.timestamp ? new Date(event.timestamp).toLocaleTimeString() : new Date().toLocaleTimeString();
            const typeClass = event.type || 'progress';
            const hostName = event.data?.host || 'localhost';
            const hostClass = 'host-' + hostName.replace(/[^a-z0-9]/gi, '-');

            // Update stats
            if (event.type === 'start') {
                battleStats.battles++;
            }
            if (event.type === 'progress' && event.data?.tokens) {
                battleStats.models++;
                battleStats.tokens += event.data.tokens || 0;
                if (event.data.tokens_sec) battleStats.speeds.push(event.data.tokens_sec);
            }
            updateStats();

            const formatted = formatEventRow(event.data, event.type);
            const showHost = hostName && hostName !== 'localhost' && event.type !== 'analysis' && event.type !== 'start' && event.type !== 'complete';
            const html = '<div class="event-item ' + typeClass + ' ' + hostClass + '">' +
                '<span class="event-type ' + typeClass + '">' + (event.type || 'event') + '</span>' +
                '<span class="event-time">' + time + '</span>' +
                (showHost ? '<span class="event-host ' + hostName + '">' + hostName + '</span>' : '') +
                '<span class="event-content">' + formatted.content + '</span>' +
                (formatted.stats ? '<div class="event-stats">' + formatted.stats + '</div>' : '') +
                '</div>';
            container.insertAdjacentHTML('afterbegin', html);
            // Keep only last 30 events in DOM
            while (container.children.length > 30) {
                container.removeChild(container.lastChild);
            }
        }
        function updateStats() {
            document.getElementById('stat-battles').textContent = battleStats.battles;
            document.getElementById('stat-models').textContent = battleStats.models;
            document.getElementById('stat-tokens').textContent = battleStats.tokens.toLocaleString();
            if (battleStats.speeds.length > 0) {
                const avg = battleStats.speeds.reduce((a,b) => a+b, 0) / battleStats.speeds.length;
                document.getElementById('stat-avgspeed').textContent = avg.toFixed(1);
            }
        }
        function showAnalysis(event) {
            const section = document.getElementById('analysis-section');
            const summary = document.getElementById('analysis-summary');
            const rankings = document.getElementById('analysis-rankings');
            const timeEl = document.getElementById('analysis-time');

            if (!event.data) return;

            // Show the section
            section.style.display = 'block';

            // Set timestamp
            const time = event.timestamp ? new Date(event.timestamp).toLocaleTimeString() : new Date().toLocaleTimeString();
            timeEl.textContent = time;

            // Set summary with highlighted winner
            let summaryText = event.data.analysis || '';
            // Highlight model names
            summaryText = summaryText.replace(/(qwen[^\s]+|llama[^\s]+|mistral[^\s]+|codestral[^\s]+|deepseek[^\s]+)/gi, '<strong>$1</strong>');
            summary.innerHTML = summaryText;

            // Set rankings with numbered highlights
            if (event.data.rankings) {
                let rankingsText = event.data.rankings;
                // Highlight rank numbers
                rankingsText = rankingsText.replace(/(\d+)\./g, '<span class="rank">$1.</span>');
                rankings.innerHTML = rankingsText;
                rankings.style.display = 'block';
            } else {
                rankings.style.display = 'none';
            }
        }
        function highlightActiveModel(event) {
            if (!event.data) return;
            const hostName = event.data.host || 'localhost';
            const modelName = event.data.model;

            // Map host names to DOM data-host values
            const hostMap = {
                'localhost': 'local-gpu',
                'local-gpu': 'local-gpu',
                'ubuntu25': 'ubuntu25',
                'mac-mini': 'mac-mini'
            };
            const domHost = hostMap[hostName] || hostName;

            // Clear ALL previous highlights first
            document.querySelectorAll('.model-tag.active').forEach(el => el.classList.remove('active'));

            if (event.type === 'progress' && modelName) {
                // Find the host card
                const hostCard = document.querySelector('.host[data-host="' + domHost + '"]');
                if (!hostCard) {
                    console.log('Host card not found for:', domHost);
                    return;
                }

                // Find and highlight the model tag (match full name or prefix)
                const modelTags = hostCard.querySelectorAll('.model-tag');
                const modelBase = modelName.split(':')[0];
                modelTags.forEach(tag => {
                    const tagText = tag.textContent.trim();
                    if (tagText === modelName || tagText.startsWith(modelBase)) {
                        tag.classList.add('active');
                    }
                });
            }
        }
        function formatEventRow(data, eventType) {
            if (!data) return { content: '', stats: '' };
            if (typeof data === 'string') return { content: data, stats: '' };

            // Progress event - model completed
            if (data.status === 'complete' && data.model) {
                return {
                    content: '<strong>' + data.model + '</strong>',
                    stats: '<span class="stat-item">⏱ <span class="stat-value">' + (data.time_sec?.toFixed(1) || '?') + 's</span></span>' +
                           '<span class="stat-item">📝 <span class="stat-value">' + (data.tokens || 0) + '</span> tokens</span>' +
                           '<span class="stat-item">⚡ <span class="stat-value">' + (data.tokens_sec?.toFixed(1) || '?') + '</span> tok/s</span>'
                };
            }
            // Error event
            if (data.status === 'error') {
                return { content: '❌ <strong>' + data.model + '</strong>: ' + data.message, stats: '' };
            }
            // Start event
            if (data.prompt) {
                const modelCount = data.models ? data.models.length : 0;
                const hostCount = data.hosts ? data.hosts.length : 0;
                return {
                    content: data.prompt.substring(0, 100) + (data.prompt.length > 100 ? '...' : ''),
                    stats: '<span class="stat-item">🐱 <span class="stat-value">' + modelCount + '</span> models</span>' +
                           '<span class="stat-item">🖥 <span class="stat-value">' + hostCount + '</span> hosts</span>'
                };
            }
            // Complete event - winner
            if (data.winner) {
                return {
                    content: '🏆 <strong>' + data.winner + '</strong> wins!',
                    stats: '<span class="stat-item">⏱ <span class="stat-value">' + (data.winner_time?.toFixed(1) || '?') + 's</span></span>' +
                           '<span class="stat-item">🖥 ' + (data.winner_host || 'localhost') + '</span>'
                };
            }
            // Analysis event
            if (data.analysis) {
                return {
                    content: '🔬 ' + data.analysis,
                    stats: data.rankings ? '<span class="stat-item">' + data.rankings + '</span>' : ''
                };
            }
            return { content: JSON.stringify(data).substring(0, 150), stats: '' };
        }
        connect();
    </script>
</body>
</html>`

// atcExperimentHTML is the dashboard for tracking multi-step experiments
const atcExperimentHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>clood ATC - Experiment Mode</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        html, body {
            height: 100vh;
            overflow: hidden;
        }
        body {
            font-family: 'SF Mono', 'Monaco', 'Inconsolata', monospace;
            background: #0a0a0f;
            color: #e0e0e0;
            padding: 20px;
            display: flex;
            flex-direction: column;
        }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            flex-shrink: 0;
            padding: 15px 20px;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            border-radius: 12px;
            margin-bottom: 20px;
            border: 1px solid #2a2a4a;
        }
        .header h1 {
            font-size: 1.5rem;
            background: linear-gradient(90deg, #00ff88, #00ccff);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .status-badge {
            padding: 6px 14px;
            border-radius: 20px;
            font-size: 0.85rem;
            font-weight: 600;
        }
        .status-running { background: #1a4a1a; color: #00ff88; }
        .status-completed { background: #1a3a5a; color: #00ccff; }
        .status-failed { background: #4a1a1a; color: #ff6b6b; }

        .main-grid {
            display: grid;
            grid-template-columns: 1fr 350px;
            gap: 20px;
            flex: 1;
            min-height: 0;
            overflow: hidden;
        }

        .session-panel {
            background: #12121a;
            border-radius: 12px;
            padding: 20px;
            border: 1px solid #2a2a4a;
            overflow-y: auto;
            min-height: 0;
        }

        .session-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
        }
        .session-name {
            font-size: 1.3rem;
            color: #00ff88;
        }
        .session-progress {
            color: #888;
        }

        .progress-bar {
            height: 8px;
            background: #1a1a2e;
            border-radius: 4px;
            overflow: hidden;
            margin-bottom: 30px;
        }
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #00ff88, #00ccff);
            transition: width 0.5s ease;
        }

        .timeline {
            position: relative;
            padding-left: 30px;
        }
        .timeline::before {
            content: '';
            position: absolute;
            left: 8px;
            top: 0;
            bottom: 0;
            width: 2px;
            background: #2a2a4a;
        }

        .step {
            position: relative;
            margin-bottom: 25px;
            padding: 15px;
            background: #1a1a2e;
            border-radius: 8px;
            border: 1px solid #2a2a4a;
        }
        .step.active {
            border-color: #00ff88;
            box-shadow: 0 0 20px rgba(0, 255, 136, 0.1);
        }
        .step.completed {
            border-color: #00ccff;
            opacity: 0.8;
        }
        .step.failed {
            border-color: #ff6b6b;
        }

        .step::before {
            content: '';
            position: absolute;
            left: -26px;
            top: 20px;
            width: 12px;
            height: 12px;
            border-radius: 50%;
            background: #2a2a4a;
            border: 2px solid #0a0a0f;
        }
        .step.active::before { background: #00ff88; }
        .step.completed::before { background: #00ccff; }
        .step.failed::before { background: #ff6b6b; }

        .step-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
        }
        .step-name {
            font-weight: 600;
            color: #fff;
        }
        .step-status {
            font-size: 0.8rem;
            padding: 3px 10px;
            border-radius: 12px;
        }
        .step-status.running { background: #1a4a1a; color: #00ff88; }
        .step-status.completed { background: #1a3a5a; color: #00ccff; }
        .step-status.failed { background: #4a1a1a; color: #ff6b6b; }
        .step-status.pending { background: #2a2a3a; color: #888; }

        .iterations {
            margin-top: 10px;
        }
        .iteration {
            display: flex;
            align-items: center;
            flex-wrap: wrap;
            gap: 8px;
            padding: 8px 12px;
            background: #0a0a0f;
            border-radius: 6px;
            margin-top: 8px;
            font-size: 0.8rem;
        }
        .iteration-num {
            color: #666;
            min-width: 25px;
        }
        .iteration-model {
            color: #00ccff;
            min-width: 140px;
        }
        .iteration-host {
            color: #888;
            min-width: 80px;
        }
        .iteration-time {
            color: #666;
            font-size: 0.7rem;
            font-family: monospace;
        }
        .iteration-time.start { color: #888; }
        .iteration-time.end { color: #00ff88; }

        .iteration-details {
            width: 100%;
            margin-top: 8px;
            font-size: 0.75rem;
        }
        .iteration-prompt {
            background: #1a1a2e;
            padding: 8px;
            border-radius: 4px;
            color: #888;
            font-style: italic;
            margin-bottom: 6px;
            border-left: 2px solid #00ccff;
        }
        .iteration-output {
            background: #0f0f15;
            padding: 8px;
            border-radius: 4px;
            color: #aaa;
            font-family: monospace;
            white-space: pre-wrap;
            max-height: 150px;
            overflow-y: auto;
            border-left: 2px solid #00ff88;
        }
        .iteration-output.collapsed {
            max-height: 80px;
            overflow: hidden;
            cursor: pointer;
        }
        .iteration-output.collapsed::after {
            content: ' [click to expand]';
            color: #666;
            font-style: italic;
        }
        .iteration-toggle {
            color: #666;
            font-size: 0.7rem;
            cursor: pointer;
            margin-left: 10px;
        }
        .iteration-toggle:hover { color: #00ccff; }
        .iteration-stats {
            color: #00ff88;
            margin-left: auto;
        }
        .iteration.running {
            border: 1px solid #00ff88;
            animation: pulse 2s infinite;
        }
        .iteration.failed {
            border: 1px solid #ff6b6b;
            background: #1a0a0a;
        }

        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.7; }
        }

        .validation {
            margin-top: 15px;
            padding: 12px;
            background: #0a0a0f;
            border-radius: 6px;
            border-left: 3px solid #888;
        }
        .validation.pass { border-left-color: #00ff88; }
        .validation.fail { border-left-color: #ff6b6b; }
        .validation-header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 8px;
        }
        .validation-cmd {
            font-family: monospace;
            color: #888;
        }
        .validation-output {
            font-family: monospace;
            font-size: 0.8rem;
            color: #aaa;
            white-space: pre-wrap;
            max-height: 100px;
            overflow-y: auto;
        }
        .validation-errors {
            color: #ff6b6b;
            font-size: 0.8rem;
            margin-top: 8px;
        }

        .sidebar {
            display: flex;
            flex-direction: column;
            gap: 20px;
            min-height: 0;
            overflow: hidden;
        }

        .hosts-panel {
            background: #12121a;
            border-radius: 12px;
            padding: 15px;
            border: 1px solid #2a2a4a;
            flex-shrink: 0;
            max-height: 45%;
            overflow-y: auto;
        }
        .hosts-panel h3 {
            color: #888;
            font-size: 0.9rem;
            margin-bottom: 15px;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        .host-item {
            display: flex;
            flex-direction: column;
            gap: 4px;
            padding: 12px;
            background: #1a1a2e;
            border-radius: 6px;
            margin-bottom: 10px;
            border: 1px solid #2a2a4a;
        }
        .host-item.offline {
            opacity: 0.5;
        }
        .host-item.local {
            border-color: #00ff88;
            background: linear-gradient(135deg, #1a2e1a 0%, #1a1a2e 100%);
        }
        .host-item.local .host-name {
            color: #00ff88;
        }
        .host-header {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .host-status {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            flex-shrink: 0;
        }
        .host-status.online { background: #00ff88; }
        .host-status.offline { background: #ff6b6b; }
        .host-name {
            font-weight: 600;
            flex: 1;
        }
        .host-latency {
            font-size: 0.75rem;
            color: #00ff88;
        }
        .host-url {
            font-size: 0.7rem;
            color: #666;
            font-family: monospace;
            margin-left: 16px;
        }
        .host-hw {
            font-size: 0.75rem;
            color: #888;
            margin-left: 16px;
        }
        .hw-gpu {
            color: #00ccff;
        }
        .hw-mem {
            color: #888;
        }
        .host-status.busy { background: #ffaa00; animation: pulse 1s infinite; }

        .events-panel {
            background: #12121a;
            border-radius: 12px;
            padding: 15px;
            border: 1px solid #2a2a4a;
            flex: 1;
            min-height: 0;
            display: flex;
            flex-direction: column;
            overflow: hidden;
        }
        .events-panel h3 {
            color: #888;
            font-size: 0.9rem;
            margin-bottom: 15px;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        .events-feed {
            flex: 1;
            min-height: 0;
            overflow-y: auto;
            font-size: 0.65rem;
            font-family: 'SF Mono', 'Monaco', monospace;
        }
        .event-item {
            padding: 4px 6px;
            border-bottom: 1px solid #1a1a2e;
            display: flex;
            flex-wrap: wrap;
            gap: 4px;
        }
        .event-time {
            color: #666;
            min-width: 70px;
            flex-shrink: 0;
        }
        .event-type {
            color: #00ccff;
            min-width: 90px;
            flex-shrink: 0;
            font-weight: 600;
        }
        .event-msg {
            color: #aaa;
            flex: 1;
            word-break: break-word;
        }
        .event-duration {
            color: #00ff88;
            margin-left: auto;
            font-weight: 600;
        }
        .event-item.step { background: #1a1a2e; }
        .event-item.iteration { background: #12121a; }
        .event-item.error { background: #2a1a1a; border-left: 2px solid #ff6b6b; }

        .no-session {
            text-align: center;
            padding: 60px 20px;
            color: #666;
        }
        .no-session h2 {
            color: #888;
            margin-bottom: 15px;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>clood ATC - Experiment Mode</h1>
        <div id="connection-status" class="status-badge status-running">Connecting...</div>
    </div>

    <div class="main-grid">
        <div class="session-panel" id="session-panel">
            <div class="no-session" id="no-session">
                <h2>No Active Experiment</h2>
                <p>Waiting for experiment events...</p>
                <p style="margin-top: 20px; font-size: 0.9rem;">
                    Start an experiment with:<br>
                    <code style="color: #00ff88;">clood chimborazo --session "Session Name"</code>
                </p>
            </div>
            <div id="session-content" style="display: none;">
                <div class="session-header">
                    <span class="session-name" id="session-name">-</span>
                    <div style="display: flex; gap: 20px; align-items: center;">
                        <span class="session-timer" id="session-timer" style="font-family: monospace; color: #00ff88; font-size: 1.1rem;">00:00</span>
                        <span class="session-progress" id="session-progress">Step 0/0</span>
                    </div>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill" id="progress-fill" style="width: 0%"></div>
                </div>
                <div class="timeline" id="timeline"></div>
            </div>
        </div>

        <div class="sidebar">
            <div class="hosts-panel">
                <h3>Hosts</h3>
                <div id="hosts-list">
                    <div class="host-item">
                        <div class="host-status offline"></div>
                        <span class="host-name">Loading...</span>
                    </div>
                </div>
            </div>

            <div class="events-panel">
                <h3>Event Feed</h3>
                <div class="events-feed" id="events-feed"></div>
            </div>
        </div>
    </div>

    <script>
        let ws;
        let currentSession = null;
        let events = [];

        function connect() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            ws = new WebSocket(protocol + '//' + window.location.host + '/ws');

            ws.onopen = () => {
                document.getElementById('connection-status').textContent = 'Connected';
                document.getElementById('connection-status').className = 'status-badge status-completed';
            };

            ws.onclose = () => {
                document.getElementById('connection-status').textContent = 'Disconnected';
                document.getElementById('connection-status').className = 'status-badge status-failed';
                setTimeout(connect, 2000);
            };

            ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                handleEvent(data);
            };
        }

        function handleEvent(event) {
            // Handle wrapper messages from server broadcasts
            // Server sends: {type: "experiment", data: {type: "session_start", ...}}
            if (event.type === 'experiment' && event.data && event.data.type) {
                // Unwrap the experiment event and handle the inner event
                const innerEvent = event.data;
                addEventToFeed({type: innerEvent.type, data: innerEvent.data || innerEvent});
                handleInnerEvent(innerEvent.type, innerEvent.data || innerEvent);
                return;
            }

            // Add to event feed
            addEventToFeed(event);

            // Handle different event types
            handleInnerEvent(event.type, event.data);
        }

        function handleInnerEvent(type, data) {
            switch(type) {
                case 'session_start':
                    startSession(data);
                    break;
                case 'session_end':
                case 'session_complete':
                case 'session_fail':
                    endSession(data);
                    break;
                case 'step_start':
                    startStep(data);
                    break;
                case 'step_end':
                case 'step_complete':
                case 'step_fail':
                    endStep(data);
                    break;
                case 'iteration_start':
                    startIteration(data);
                    break;
                case 'iteration_end':
                case 'iteration_complete':
                case 'iteration_fail':
                    endIteration(data);
                    break;
                case 'validation':
                    addValidation(data);
                    break;
                case 'hosts':
                    updateHosts(data);
                    break;
                case 'experiment_state':
                    // Poller sends combined state - extract hosts
                    if (data && data.hosts) {
                        updateHosts(data.hosts);
                    }
                    break;
            }
        }

        function addEventToFeed(event) {
            const feed = document.getElementById('events-feed');
            const time = new Date().toLocaleTimeString('en-US', {hour12: false});
            const div = document.createElement('div');

            // Classify event for styling
            let eventClass = 'event-item';
            if (event.type.includes('step')) eventClass += ' step';
            if (event.type.includes('iteration')) eventClass += ' iteration';
            if (event.type.includes('fail') || event.type.includes('error')) eventClass += ' error';
            div.className = eventClass;

            // Build message with more detail
            let msg = '';
            let duration = '';
            const d = event.data || {};

            if (d.name) msg = d.name;
            else if (d.model) msg = d.model + (d.host ? ' @ ' + d.host : '');
            else if (d.error) msg = 'ERROR: ' + d.error;
            else msg = JSON.stringify(d).substring(0, 120);

            // Show duration if available
            if (d.duration_sec) duration = d.duration_sec.toFixed(1) + 's';
            else if (d.duration) duration = d.duration.toFixed(1) + 's';

            div.innerHTML = '<span class="event-time">' + time + '</span>' +
                           '<span class="event-type">' + event.type + '</span>' +
                           '<span class="event-msg">' + msg + '</span>' +
                           (duration ? '<span class="event-duration">' + duration + '</span>' : '');
            feed.insertBefore(div, feed.firstChild);

            // Keep only last 100 events
            while (feed.children.length > 100) {
                feed.removeChild(feed.lastChild);
            }
        }

        let sessionTimer = null;

        function startSession(data) {
            currentSession = {
                id: data.id,
                name: data.name,
                totalSteps: data.total_steps || 0,
                currentStep: 0,
                steps: [],
                startTime: Date.now()
            };

            document.getElementById('no-session').style.display = 'none';
            document.getElementById('session-content').style.display = 'block';
            document.getElementById('session-name').textContent = data.name;
            document.getElementById('session-progress').textContent = 'Step 0/' + currentSession.totalSteps;
            document.getElementById('progress-fill').style.width = '0%';
            document.getElementById('timeline').innerHTML = '';

            // Start session timer
            if (sessionTimer) clearInterval(sessionTimer);
            updateSessionTimer();
            sessionTimer = setInterval(updateSessionTimer, 1000);
        }

        function updateSessionTimer() {
            if (!currentSession || !currentSession.startTime) return;
            const elapsed = Math.floor((Date.now() - currentSession.startTime) / 1000);
            const mins = Math.floor(elapsed / 60);
            const secs = elapsed % 60;
            document.getElementById('session-timer').textContent =
                String(mins).padStart(2, '0') + ':' + String(secs).padStart(2, '0');
        }

        function endSession(data) {
            if (currentSession) {
                // Stop timer
                if (sessionTimer) {
                    clearInterval(sessionTimer);
                    sessionTimer = null;
                }

                // Show final time and status
                const elapsed = Math.floor((Date.now() - currentSession.startTime) / 1000);
                const mins = Math.floor(elapsed / 60);
                const secs = elapsed % 60;
                const finalTime = String(mins).padStart(2, '0') + ':' + String(secs).padStart(2, '0');

                document.getElementById('session-timer').textContent = finalTime + ' ✓';
                document.getElementById('session-timer').style.color = data.status === 'completed' ? '#00ccff' : '#ff6b6b';
                document.getElementById('session-progress').textContent =
                    data.status === 'completed' ? 'Completed' : data.status;
            }
        }

        function startStep(data) {
            if (!currentSession) return;

            const stepStartTime = Date.now();
            const stepStartTimeStr = new Date().toLocaleTimeString('en-US', {hour12: false});

            // Calculate gap from previous step
            let gapStr = '';
            if (currentSession.lastStepEndTime) {
                const gap = (stepStartTime - currentSession.lastStepEndTime) / 1000;
                if (gap > 0.5) {
                    gapStr = '<span style="color: #ff9900; font-size: 0.7rem; margin-left: 10px;">+' + gap.toFixed(1) + 's gap</span>';
                }
            }

            currentSession.currentStep = data.number;
            currentSession.stepStartTimes = currentSession.stepStartTimes || {};
            currentSession.stepStartTimes[data.number] = stepStartTime;

            document.getElementById('session-progress').textContent =
                'Step ' + data.number + '/' + currentSession.totalSteps;

            const progress = (data.number / currentSession.totalSteps) * 100;
            document.getElementById('progress-fill').style.width = progress + '%';

            const timeline = document.getElementById('timeline');
            const stepDiv = document.createElement('div');
            stepDiv.className = 'step active';
            stepDiv.id = 'step-' + data.number;
            stepDiv.innerHTML =
                '<div class="step-header">' +
                    '<span class="step-name">' + data.number + '. ' + data.name + '</span>' +
                    '<div style="display: flex; align-items: center; gap: 8px;">' +
                        '<span style="color: #666; font-size: 0.7rem; font-family: monospace;">▶ ' + stepStartTimeStr + '</span>' +
                        gapStr +
                        '<span class="step-status running">Running</span>' +
                    '</div>' +
                '</div>' +
                '<div class="iterations" id="iterations-' + data.number + '"></div>';
            timeline.appendChild(stepDiv);

            // Mark previous steps as completed
            for (let i = 1; i < data.number; i++) {
                const prevStep = document.getElementById('step-' + i);
                if (prevStep && !prevStep.classList.contains('completed')) {
                    prevStep.className = 'step completed';
                    prevStep.querySelector('.step-status').className = 'step-status completed';
                    prevStep.querySelector('.step-status').textContent = 'Completed';
                }
            }
        }

        function endStep(data) {
            const stepDiv = document.getElementById('step-' + data.number);
            if (stepDiv) {
                // Store end time for gap calculation
                if (currentSession) {
                    currentSession.lastStepEndTime = Date.now();

                    // Calculate step duration
                    if (currentSession.stepStartTimes && currentSession.stepStartTimes[data.number]) {
                        const stepDuration = (currentSession.lastStepEndTime - currentSession.stepStartTimes[data.number]) / 1000;
                        const durationStr = stepDuration.toFixed(1) + 's';

                        // Add duration to step header
                        const header = stepDiv.querySelector('.step-header > div');
                        if (header) {
                            const durationSpan = document.createElement('span');
                            durationSpan.style.cssText = 'color: #00ff88; font-size: 0.75rem; font-weight: 600;';
                            durationSpan.textContent = durationStr;
                            header.insertBefore(durationSpan, header.querySelector('.step-status'));
                        }
                    }
                }

                stepDiv.className = 'step ' + data.status;
                const statusEl = stepDiv.querySelector('.step-status');
                statusEl.className = 'step-status ' + data.status;
                statusEl.textContent = data.status.charAt(0).toUpperCase() + data.status.slice(1);
            }
        }

        function startIteration(data) {
            const container = document.getElementById('iterations-' + data.step);
            if (!container) return;

            const startTime = new Date().toLocaleTimeString('en-US', {hour12: false});
            const iterDiv = document.createElement('div');
            iterDiv.className = 'iteration running';
            iterDiv.id = 'iter-' + data.step + '-' + data.number;
            iterDiv.dataset.startTime = Date.now(); // Store for duration calc

            // Build prompt preview if available
            let promptHtml = '';
            if (data.prompt) {
                const promptPreview = data.prompt.substring(0, 200) + (data.prompt.length > 200 ? '...' : '');
                promptHtml = '<div class="iteration-details"><div class="iteration-prompt">' + escapeHtml(promptPreview) + '</div></div>';
            }

            iterDiv.innerHTML =
                '<span class="iteration-num">#' + data.number + '</span>' +
                '<span class="iteration-model">' + data.model + '</span>' +
                '<span class="iteration-host">' + data.host + '</span>' +
                '<span class="iteration-time start">▶ ' + startTime + '</span>' +
                '<span class="iteration-stats">Running...</span>' +
                promptHtml;
            container.appendChild(iterDiv);
            scrollToBottom();
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        function scrollToBottom() {
            const panel = document.getElementById('session-panel');
            if (panel) {
                setTimeout(() => {
                    panel.scrollTop = panel.scrollHeight;
                }, 100);
            }
        }

        function endIteration(data) {
            const iterDiv = document.getElementById('iter-' + data.step + '-' + data.number);
            if (iterDiv) {
                iterDiv.className = 'iteration ' + data.status;
                const endTime = new Date().toLocaleTimeString('en-US', {hour12: false});

                // Calculate actual duration if we have start time
                let duration = data.duration_sec || data.duration || 0;
                if (iterDiv.dataset.startTime && !duration) {
                    duration = (Date.now() - parseInt(iterDiv.dataset.startTime)) / 1000;
                }

                const durationStr = duration ? duration.toFixed(1) + 's' : '';
                const tokens = data.tokens ? ' ' + data.tokens + ' tok' : '';
                const tps = data.tokens_sec ? ' ' + data.tokens_sec.toFixed(1) + ' t/s' : '';

                // Update with end time
                const timeSpan = iterDiv.querySelector('.iteration-time');
                if (timeSpan) {
                    const startTime = timeSpan.textContent.replace('▶ ', '');
                    timeSpan.outerHTML = '<span class="iteration-time start">' + startTime + '</span>' +
                                        '<span class="iteration-time end">→ ' + endTime + '</span>';
                }

                iterDiv.querySelector('.iteration-stats').textContent = durationStr + tokens + tps;

                // Add output snippet if available
                if (data.output) {
                    let detailsDiv = iterDiv.querySelector('.iteration-details');
                    if (!detailsDiv) {
                        detailsDiv = document.createElement('div');
                        detailsDiv.className = 'iteration-details';
                        iterDiv.appendChild(detailsDiv);
                    }

                    const outputDiv = document.createElement('div');
                    outputDiv.className = 'iteration-output'; // Expanded by default
                    outputDiv.textContent = data.output;
                    outputDiv.onclick = function() {
                        this.classList.toggle('collapsed');
                    };
                    detailsDiv.appendChild(outputDiv);

                    // Auto-scroll to show latest content
                    scrollToBottom();
                }
            }
        }

        function addValidation(data) {
            const stepDiv = document.getElementById('step-' + data.step);
            if (!stepDiv) return;

            const validDiv = document.createElement('div');
            validDiv.className = 'validation ' + data.status;
            validDiv.innerHTML =
                '<div class="validation-header">' +
                    '<span class="validation-cmd">$ ' + data.command + '</span>' +
                    '<span class="step-status ' + data.status + '">' + data.status.toUpperCase() + '</span>' +
                '</div>' +
                (data.output ? '<div class="validation-output">' + escapeHtml(data.output) + '</div>' : '') +
                (data.errors && data.errors.length ? '<div class="validation-errors">' + data.errors.join('<br>') + '</div>' : '');
            stepDiv.appendChild(validDiv);
        }

        function updateHosts(hosts) {
            const container = document.getElementById('hosts-list');
            container.innerHTML = hosts.map(h => {
                const hw = h.hardware || {};
                const isLocal = h.url && h.url.includes('localhost');
                // Show full URL but clean up protocol
                let url = h.url ? h.url.replace('http://', '') : '';
                // For localhost, show the actual machine IP
                if (isLocal) {
                    url = '127.0.0.1:11434 (local)';
                }
                return '<div class="host-item' + (h.online ? '' : ' offline') + (isLocal ? ' local' : '') + '">' +
                    '<div class="host-header">' +
                        '<div class="host-status ' + (h.online ? 'online' : 'offline') + '"></div>' +
                        '<span class="host-name">' + h.name + '</span>' +
                        '<span class="host-latency">' + (h.online ? h.latency_ms + 'ms' : 'offline') + '</span>' +
                    '</div>' +
                    '<div class="host-url">' + url + '</div>' +
                    (hw.gpu ? '<div class="host-hw"><span class="hw-gpu">' + hw.gpu + '</span></div>' : '') +
                    (hw.memory ? '<div class="host-hw"><span class="hw-mem">' + hw.memory + '</span></div>' : '') +
                '</div>';
            }).join('');
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        connect();
    </script>
</body>
</html>`
