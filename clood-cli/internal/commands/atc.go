package commands

import (
	"context"
	"encoding/json"
	"fmt"
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

// HostStatus represents a host's current state for active mode
type HostStatus struct {
	Name      string   `json:"name"`
	Online    bool     `json:"online"`
	Latency   int64    `json:"latency_ms"`
	Models    []string `json:"models"`
	ActiveReq int      `json:"active_requests"`
	LastSeen  string   `json:"last_seen"`
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
	clients    map[*websocket.Conn]bool
	broadcast  chan ATCMessage
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan ATCMessage),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
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

			// Start the appropriate poller
			go func() {
				ticker := time.NewTicker(30 * time.Second)

				// Initial broadcast
				if mode == "active" {
					hosts := atcFetchHostStatus()
					hub.broadcast <- ATCMessage{
						Type: "hosts",
						Data: hosts,
						Mode: mode,
						Time: time.Now().Format(time.RFC3339),
					}
				} else {
					issues := atcFetchIssues(owner, repo)
					hub.broadcast <- ATCMessage{
						Type: "issues",
						Data: issues,
						Mode: mode,
						Time: time.Now().Format(time.RFC3339),
					}
				}

				for range ticker.C {
					if mode == "active" {
						hosts := atcFetchHostStatus()
						hub.broadcast <- ATCMessage{
							Type: "hosts",
							Data: hosts,
							Mode: mode,
							Time: time.Now().Format(time.RFC3339),
						}
					} else {
						issues := atcFetchIssues(owner, repo)
						hub.broadcast <- ATCMessage{
							Type: "issues",
							Data: issues,
							Mode: mode,
							Time: time.Now().Format(time.RFC3339),
						}
					}
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

// Active mode HTML - Host monitoring
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
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 30px;
            padding-bottom: 15px;
            border-bottom: 2px solid #2a4a2a;
        }
        .header h1 { color: #00ff88; font-size: 28px; text-shadow: 0 0 20px rgba(0,255,136,0.5); }
        .status { display: flex; gap: 20px; font-size: 14px; color: #888; }
        .status .live { color: #00ff88; animation: pulse 2s infinite; }
        @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
        .hosts { display: grid; grid-template-columns: repeat(auto-fit, minmax(350px, 1fr)); gap: 20px; }
        .host {
            background: rgba(22, 62, 33, 0.8);
            border-radius: 12px;
            padding: 20px;
            border: 2px solid #2a4a2a;
            transition: all 0.3s ease;
        }
        .host.offline { opacity: 0.5; border-color: #4a2a2a; background: rgba(62, 22, 22, 0.5); }
        .host-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; }
        .host-name { font-size: 20px; font-weight: bold; color: #00ff88; }
        .host-status { padding: 4px 12px; border-radius: 20px; font-size: 12px; text-transform: uppercase; }
        .host-status.online { background: #1a4a1a; color: #00ff88; }
        .host-status.offline { background: #4a1a1a; color: #ff4444; }
        .host-stats { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-bottom: 15px; }
        .stat { background: rgba(0,0,0,0.3); padding: 10px; border-radius: 8px; }
        .stat-label { font-size: 11px; color: #666; text-transform: uppercase; margin-bottom: 5px; }
        .stat-value { font-size: 18px; color: #fff; }
        .stat-value.good { color: #00ff88; }
        .stat-value.warn { color: #ffaa00; }
        .stat-value.bad { color: #ff4444; }
        .models { background: rgba(0,0,0,0.3); border-radius: 8px; padding: 15px; }
        .models h4 { font-size: 12px; color: #666; text-transform: uppercase; margin-bottom: 10px; }
        .model-list { display: flex; flex-wrap: wrap; gap: 5px; }
        .model-tag { background: #333; padding: 4px 10px; border-radius: 4px; font-size: 12px; color: #aaa; }
        .footer {
            margin-top: 30px;
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
                <span id="last-update">--</span>
            </div>
        </div>
        <div class="hosts" id="hosts">
            <p style="padding:40px;color:#666;text-align:center">Scanning hosts...</p>
        </div>
        <div class="footer">
            <span>clood atc --mode active</span>
            <span id="connection-status">Connecting...</span>
        </div>
    </div>
    <script>
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
            };
        }
        function renderHosts(hosts) {
            const html = hosts.map(host => {
                const statusClass = host.online ? 'online' : 'offline';
                const hostClass = host.online ? '' : 'offline';
                const latencyClass = host.latency_ms < 50 ? 'good' : host.latency_ms < 200 ? 'warn' : 'bad';
                const models = (host.models || []).map(m => '<span class="model-tag">' + m + '</span>').join('');
                return '<div class="host ' + hostClass + '">' +
                    '<div class="host-header"><span class="host-name">' + host.name + '</span>' +
                    '<span class="host-status ' + statusClass + '">' + statusClass + '</span></div>' +
                    '<div class="host-stats">' +
                    '<div class="stat"><div class="stat-label">Latency</div><div class="stat-value ' + latencyClass + '">' +
                    (host.online ? host.latency_ms + 'ms' : '--') + '</div></div>' +
                    '<div class="stat"><div class="stat-label">Models</div><div class="stat-value">' +
                    (host.models ? host.models.length : 0) + '</div></div></div>' +
                    '<div class="models"><h4>Available Models</h4><div class="model-list">' +
                    (models || '<span style="color:#666">None loaded</span>') + '</div></div></div>';
            }).join('');
            document.getElementById('hosts').innerHTML = html;
        }
        connect();
    </script>
</body>
</html>`
