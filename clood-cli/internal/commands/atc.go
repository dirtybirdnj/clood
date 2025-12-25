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

// Static hardware specs for known hosts
var hostHardware = map[string]HardwareSpec{
	"local-gpu": {CPU: "Apple M4", GPU: "M4 10-core", Memory: "32GB"},
	"mac-mini":  {CPU: "Apple M4", GPU: "M4 10-core", Memory: "16GB"},
	"ubuntu25":  {CPU: "i7-8700", GPU: "RX 590 8GB", Memory: "64GB"},
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
}

func newHub() *Hub {
	return &Hub{
		clients:      make(map[*websocket.Conn]bool),
		broadcast:    make(chan ATCMessage),
		register:     make(chan *websocket.Conn),
		unregister:   make(chan *websocket.Conn),
		events:       make([]CatfightEvent, 0),
		pollInterval: 10 * time.Second,
	}
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
			if mode == "active" {
				htmlContent = atcActiveHTML
			} else {
				htmlContent = atcPlanningHTML
			}

			// Start the appropriate poller with dynamic interval
			go func() {
				fetchAndBroadcast := func() {
					var msg ATCMessage
					if mode == "active" {
						hostsData := atcFetchHostStatus()
						msg = ATCMessage{
							Type: "hosts",
							Data: hostsData,
							Mode: mode,
							Time: time.Now().Format(time.RFC3339),
						}
					} else {
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
