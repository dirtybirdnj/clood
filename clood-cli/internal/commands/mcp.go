package commands

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	cloodmcp "github.com/dirtybirdnj/clood/internal/mcp"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// copyToClipboard copies text to system clipboard, returns success
func copyToClipboard(text string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return false
		}
	default:
		return false
	}

	pipe, err := cmd.StdinPipe()
	if err != nil {
		return false
	}

	if err := cmd.Start(); err != nil {
		return false
	}

	pipe.Write([]byte(text))
	pipe.Close()

	return cmd.Wait() == nil
}

func McpCmd() *cobra.Command {
	var port int
	var quiet bool
	var copyConfig bool

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server for Claude Code (just works)",
		Long: `Starts the clood MCP server with sensible defaults.

This is the easiest way to make clood tools available to Claude Code.
Just run it and clood tools become available to AI agents.

Tools exposed:
  LOCAL (0 network, 0 LLM tokens):
    clood_grep      - Search codebase with regex
    clood_tree      - Directory structure
    clood_symbols   - Extract functions/types/classes
    clood_imports   - Dependency analysis
    clood_context   - Project summary
    clood_capabilities - What's available locally
    clood_system    - Hardware detection

  LOCAL OLLAMA:
    clood_ask       - Query local LLM
    clood_hosts     - Check Ollama hosts
    clood_models    - List available models
    clood_health    - System health check

Examples:
  clood mcp           # Start on default port 8765
  clood mcp -p 9000   # Custom port
  clood mcp -q        # Quiet mode (less output)
  clood mcp --copy    # Copy .mcp.json config to clipboard`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create MCP server with all tools
			mcpSrv, err := cloodmcp.NewServer()
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error: " + err.Error()))
				return
			}

			baseURL := fmt.Sprintf("http://localhost:%d", port)

			// JSON config for .mcp.json
			mcpJSON := fmt.Sprintf(`{
  "mcpServers": {
    "clood": {"type": "sse", "url": "%s/sse"}
  }
}`, baseURL)

			// Handle --copy flag
			if copyConfig {
				if copyToClipboard(mcpJSON) {
					fmt.Printf("%s .mcp.json config copied to clipboard\n", tui.SuccessStyle.Render("✓"))
				} else {
					fmt.Println(tui.MutedStyle.Render("Clipboard not available. Here's the config:"))
					fmt.Println()
					fmt.Println(mcpJSON)
				}
				fmt.Println()
			}

			// Create SSE server
			sseSrv := server.NewSSEServer(mcpSrv.MCPServer(),
				server.WithBaseURL(baseURL),
				server.WithKeepAlive(true),
				server.WithKeepAliveInterval(30*time.Second),
			)

			addr := fmt.Sprintf("127.0.0.1:%d", port)

			if !quiet {
				fmt.Println()
				fmt.Printf("  %s clood MCP server running\n", tui.SuccessStyle.Render("●"))
				fmt.Printf("  %s %s/sse\n", tui.MutedStyle.Render("URL:"), baseURL)
				fmt.Println()
				if !copyConfig {
					fmt.Println(tui.MutedStyle.Render("  Add to .mcp.json (use --copy to copy):"))
					fmt.Println(tui.MutedStyle.Render(mcpJSON))
				}
				fmt.Println()
				fmt.Println(tui.MutedStyle.Render("  Ctrl+C to stop"))
				fmt.Println()
			} else {
				fmt.Printf("clood mcp @ %s/sse\n", baseURL)
			}

			// Handle graceful shutdown
			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

			go func() {
				if err := sseSrv.Start(addr); err != nil && err != http.ErrServerClosed {
					fmt.Println(tui.ErrorStyle.Render("Error: " + err.Error()))
					stop <- syscall.SIGTERM
				}
			}()

			<-stop
			if !quiet {
				fmt.Println(tui.MutedStyle.Render("\n  Stopped"))
			}
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8765, "Port")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Minimal output")
	cmd.Flags().BoolVarP(&copyConfig, "copy", "c", false, "Copy .mcp.json config to clipboard")

	return cmd
}
