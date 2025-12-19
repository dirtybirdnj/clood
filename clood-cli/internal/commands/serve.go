package commands

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cloodmcp "github.com/dirtybirdnj/clood/internal/mcp"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

func ServeCmd() *cobra.Command {
	var port int
	var host string
	var useSSE bool
	var baseURL string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start clood as an MCP server",
		Long: `Starts clood as a Model Context Protocol (MCP) server.

This allows AI agents (like Claude Code, crush) to call clood tools via HTTP/SSE streaming.

Examples:
  # Start SSE server on default port (localhost only)
  clood serve --sse

  # Expose to network (for remote access)
  clood serve --sse --host 0.0.0.0

  # Start on custom port
  clood serve --sse --port 8080

  # Remote server setup (on ubuntu25)
  clood serve --sse --host 0.0.0.0 --base-url http://192.168.4.64:8765

  # With custom base URL (for reverse proxy)
  clood serve --sse --port 8765 --base-url https://clood.local`,
		Run: func(cmd *cobra.Command, args []string) {
			if !useSSE {
				fmt.Println(tui.ErrorStyle.Render("Currently only SSE transport is supported"))
				fmt.Println(tui.MutedStyle.Render("Use: clood serve --sse"))
				return
			}

			// Create MCP server with all tools
			mcpSrv, err := cloodmcp.NewServer()
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error creating MCP server: " + err.Error()))
				return
			}

			// Build base URL
			if baseURL == "" {
				if host == "0.0.0.0" || host == "" {
					baseURL = fmt.Sprintf("http://localhost:%d", port)
				} else {
					baseURL = fmt.Sprintf("http://%s:%d", host, port)
				}
			}

			// Create SSE server
			sseSrv := server.NewSSEServer(mcpSrv.MCPServer(),
				server.WithBaseURL(baseURL),
				server.WithKeepAlive(true),
				server.WithKeepAliveInterval(30*time.Second),
			)

			// Setup HTTP server
			addr := fmt.Sprintf("%s:%d", host, port)

			fmt.Println(tui.RenderHeader("clood MCP Server"))
			fmt.Println()
			fmt.Printf("  %s SSE server starting...\n", tui.SuccessStyle.Render("●"))
			fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Address:"), addr)
			fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Base URL:"), baseURL)
			fmt.Printf("  %s %s/sse\n", tui.MutedStyle.Render("SSE Endpoint:"), baseURL)
			fmt.Printf("  %s %s/message\n", tui.MutedStyle.Render("Message Endpoint:"), baseURL)
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("  crush.json config:"))
			fmt.Println(tui.MutedStyle.Render(fmt.Sprintf(`  "clood": {
    "type": "sse",
    "url": "%s/sse"
  }`, baseURL)))
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("  Press Ctrl+C to stop"))
			fmt.Println()

			// Handle graceful shutdown
			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

			// Start server in goroutine
			go func() {
				if err := sseSrv.Start(addr); err != nil && err != http.ErrServerClosed {
					fmt.Println(tui.ErrorStyle.Render("Server error: " + err.Error()))
					stop <- syscall.SIGTERM
				}
			}()

			// Wait for shutdown signal
			<-stop
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("  Shutting down..."))
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8765, "Port to listen on")
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "Host to bind to (use 0.0.0.0 for network access)")
	cmd.Flags().BoolVar(&useSSE, "sse", false, "Use SSE (Server-Sent Events) transport")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Base URL for SSE endpoints (default: http://localhost:PORT)")

	return cmd
}
