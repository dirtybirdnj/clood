// cmd/atc-viewer.go
// ATC Viewer - Visual dashboard for GitHub issues
// Racing leaderboard + train station departure board aesthetics
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

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
	Velocity       int      `json:"velocity"`      // Score for ranking
	Delta          int      `json:"delta"`         // Position change (+/-)
	Epic           string   `json:"epic,omitempty"` // Parsed from labels
	IsPR           bool     `json:"is_pr"`
	UpdatedAt      string   `json:"updated_at"`
	Body           string   `json:"body,omitempty"` // Synopsis
}

// Hub maintains active WebSocket connections and broadcasts updates
type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []IssuePacket
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []IssuePacket),
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

		case issues := <-h.broadcast:
			h.mu.Lock()
			data, _ := json.Marshal(issues)
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

	// Keep connection alive, handle disconnects
	go func() {
		defer func() { hub.unregister <- conn }()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// fetchAndScoreGitHubIssues fetches issues and calculates velocity scores
func fetchAndScoreGitHubIssues(owner, repo string) []IssuePacket {
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
		fmt.Printf("Error fetching issues: %v\n", err)
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
			Body:      truncate(issue.GetBody(), 200),
		}

		// Extract labels
		for _, label := range issue.Labels {
			packet.Labels = append(packet.Labels, label.GetName())
			// Check for epic label
			if len(label.GetName()) > 5 && label.GetName()[:5] == "epic:" {
				packet.Epic = label.GetName()[5:]
			}
		}

		// Assignee
		if issue.Assignee != nil {
			packet.Assignee = issue.Assignee.GetLogin()
			packet.AssigneeAvatar = issue.Assignee.GetAvatarURL()
		}

		// Calculate velocity score
		packet.Velocity = calculateVelocity(issue)

		packets = append(packets, packet)
	}

	return packets
}

func calculateVelocity(issue *github.Issue) int {
	velocity := 0

	// Recent activity boost
	hourAgo := time.Now().Add(-1 * time.Hour)
	dayAgo := time.Now().Add(-24 * time.Hour)

	if issue.GetUpdatedAt().After(hourAgo) {
		velocity += 100
	} else if issue.GetUpdatedAt().After(dayAgo) {
		velocity += 50
	}

	// Priority labels
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
		}
	}

	// Comments indicate activity
	velocity += issue.GetComments() * 10

	return velocity
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

var atcCmd = &cobra.Command{
	Use:   "atc-viewer",
	Short: "Launch the Air Traffic Control display for issues",
	Long: `ATC Viewer provides a visual dashboard for GitHub issues.

Features:
- Racing leaderboard showing issues ranked by velocity/priority
- Train station departure board with flapper animation
- Real-time updates via WebSocket
- Themes: planes (planning) or trains (catfight)`,
	Run: func(cmd *cobra.Command, args []string) {
		owner := "dirtybirdnj"
		repo := "clood"

		// Start the Hub
		hub := newHub()
		go hub.run()

		// Start the GitHub Poller
		go func() {
			// Initial fetch
			issues := fetchAndScoreGitHubIssues(owner, repo)
			hub.broadcast <- issues

			ticker := time.NewTicker(30 * time.Second)
			for range ticker.C {
				issues := fetchAndScoreGitHubIssues(owner, repo)
				hub.broadcast <- issues
			}
		}()

		// Serve static files (TODO: embed these)
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(atcHTML))
		})
		http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			serveWs(hub, w, r)
		})

		fmt.Println("🛫 ATC Tower online at http://localhost:8080")
		fmt.Println("   Tracking:", owner+"/"+repo)
		http.ListenAndServe(":8080", nil)
	},
}

// Enhanced ATC Viewer HTML with split-flap animation
var atcHTML = `<!DOCTYPE html>
<html>
<head>
    <title>ATC Viewer - Issue Radar</title>
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

        /* Header */
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
        .status {
            display: flex;
            gap: 20px;
            font-size: 14px;
            color: #888;
        }
        .status .live { color: #00ff88; animation: pulse 2s infinite; }
        @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }

        /* Main Layout */
        .main { display: grid; grid-template-columns: 1fr 400px; gap: 20px; }

        /* Leaderboard */
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
            position: relative;
        }
        .issue:hover {
            background: rgba(0,255,136,0.1);
            transform: translateX(5px);
        }
        .issue.moving-up { animation: slideUp 0.5s ease-out; }
        .issue.moving-down { animation: slideDown 0.5s ease-out; }
        @keyframes slideUp { from { transform: translateY(20px); opacity: 0.5; } }
        @keyframes slideDown { from { transform: translateY(-20px); opacity: 0.5; } }

        .rank {
            width: 40px;
            font-size: 20px;
            font-weight: bold;
            color: #00ff88;
        }
        .rank-1 { color: #ffd700; text-shadow: 0 0 10px rgba(255,215,0,0.5); }
        .rank-2 { color: #c0c0c0; }
        .rank-3 { color: #cd7f32; }

        .delta {
            width: 35px;
            font-size: 16px;
            text-align: center;
        }
        .delta.up { color: #00ff88; }
        .delta.down { color: #ff4444; }
        .delta.same { color: #666; }

        .number {
            width: 70px;
            color: #888;
            font-size: 14px;
        }
        .title {
            flex: 1;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            padding-right: 10px;
        }
        .labels {
            display: flex;
            gap: 5px;
            flex-wrap: wrap;
            max-width: 200px;
        }
        .label {
            background: #333;
            padding: 3px 8px;
            border-radius: 4px;
            font-size: 11px;
            color: #aaa;
        }
        .label.bug { background: #5c2020; color: #ff8888; }
        .label.feature, .label.enhancement { background: #1a3a1a; color: #88ff88; }
        .label.P0, .label.critical { background: #5c1a1a; color: #ff4444; }
        .label.P1, .label.high { background: #5c3a1a; color: #ffaa44; }

        .velocity {
            width: 80px;
            text-align: right;
            color: #ffaa00;
            font-weight: bold;
        }

        /* Departures Board */
        .departures-panel {
            display: flex;
            flex-direction: column;
            gap: 15px;
        }

        .departures {
            background: #000;
            border-radius: 12px;
            padding: 20px;
            border: 3px solid #333;
        }
        .departures h2 {
            color: #ffaa00;
            margin-bottom: 15px;
            font-size: 16px;
            text-transform: uppercase;
            letter-spacing: 3px;
        }

        /* Split-Flap Display */
        .flap-board {
            background: #111;
            border-radius: 8px;
            padding: 15px;
        }
        .flap-row {
            display: flex;
            align-items: center;
            padding: 8px 0;
            border-bottom: 1px solid #222;
        }
        .flap-row:last-child { border-bottom: none; }
        .flap-index {
            width: 30px;
            color: #ffaa00;
            font-weight: bold;
        }
        .flap-text {
            flex: 1;
            display: flex;
            gap: 2px;
        }
        .flap-char {
            width: 14px;
            height: 24px;
            background: #1a1a1a;
            border: 1px solid #333;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 14px;
            color: #fff;
            text-transform: uppercase;
        }
        .flap-char.flipping {
            animation: flip 0.1s ease-out;
        }
        @keyframes flip {
            0% { transform: rotateX(90deg); }
            100% { transform: rotateX(0deg); }
        }

        /* Currently Boarding */
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
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .current-box h3::before {
            content: '●';
            animation: blink 1s infinite;
        }
        @keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.3; } }

        .current-issue {
            font-size: 18px;
            margin-bottom: 10px;
        }
        .current-issue .num { color: #888; }
        .current-synopsis {
            font-size: 13px;
            color: #888;
            line-height: 1.5;
            max-height: 80px;
            overflow: hidden;
        }

        /* Footer Status */
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
            <h1>✈️ ATC ISSUE RADAR</h1>
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
                <div id="leaderboard">
                    <p style="padding:20px;color:#666">Connecting to tower...</p>
                </div>
            </div>

            <div class="departures-panel">
                <div class="departures">
                    <h2>📋 Upcoming Departures</h2>
                    <div class="flap-board" id="departures"></div>
                </div>

                <div class="current-box">
                    <h3>Currently Boarding</h3>
                    <div id="current">
                        <div class="current-issue">Awaiting assignment...</div>
                    </div>
                </div>
            </div>
        </div>

        <div class="footer">
            <span>clood atc-viewer | dirtybirdnj/clood</span>
            <span id="connection-status">Connecting...</span>
        </div>
    </div>

    <script>
        let previousRanks = {};
        let ws;

        function connect() {
            ws = new WebSocket('ws://' + location.host + '/ws');

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
                const issues = JSON.parse(e.data);
                renderLeaderboard(issues);
                renderDepartures(issues);
                document.getElementById('issue-count').textContent = issues.length + ' issues';
                document.getElementById('last-update').textContent = new Date().toLocaleTimeString();
            };
        }

        function renderLeaderboard(issues) {
            const sorted = issues.sort((a, b) => b.velocity - a.velocity);

            const html = sorted.slice(0, 15).map((issue, i) => {
                const rank = i + 1;
                const prevRank = previousRanks[issue.number];
                let deltaClass = 'same';
                let deltaSymbol = '─';
                let moveClass = '';

                if (prevRank !== undefined) {
                    if (prevRank > rank) {
                        deltaClass = 'up';
                        deltaSymbol = '▲' + (prevRank - rank);
                        moveClass = 'moving-up';
                    } else if (prevRank < rank) {
                        deltaClass = 'down';
                        deltaSymbol = '▼' + (rank - prevRank);
                        moveClass = 'moving-down';
                    }
                }

                previousRanks[issue.number] = rank;

                const rankClass = rank <= 3 ? 'rank-' + rank : '';
                const labels = (issue.labels || []).map(l => {
                    const cls = ['bug','feature','enhancement','P0','P1','critical','high'].includes(l) ? l : '';
                    return '<span class="label ' + cls + '">' + l + '</span>';
                }).join('');

                return '<div class="issue ' + moveClass + '">' +
                    '<span class="rank ' + rankClass + '">' + rank + '</span>' +
                    '<span class="delta ' + deltaClass + '">' + deltaSymbol + '</span>' +
                    '<span class="number">#' + issue.number + '</span>' +
                    '<span class="title">' + escapeHtml(issue.title) + '</span>' +
                    '<span class="labels">' + labels + '</span>' +
                    '<span class="velocity">' + issue.velocity + '</span>' +
                '</div>';
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

            // Animate flap effect
            document.querySelectorAll('.flap-char').forEach((el, i) => {
                setTimeout(() => el.classList.add('flipping'), i * 10);
            });

            if (sorted[0]) {
                document.getElementById('current').innerHTML =
                    '<div class="current-issue"><span class="num">#' + sorted[0].number + '</span> ' + escapeHtml(sorted[0].title) + '</div>' +
                    '<div class="current-synopsis">' + escapeHtml(sorted[0].body || 'No description available') + '</div>';
            }
        }

        function escapeHtml(text) {
            if (!text) return '';
            return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
        }

        connect();
    </script>
</body>
</html>`

func init() {
	// Will be registered with root command
	// rootCmd.AddCommand(atcCmd)
}
