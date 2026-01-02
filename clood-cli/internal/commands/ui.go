package commands

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func UICmd() *cobra.Command {
	var proxyPort int
	var uiPort int
	var noBrowser bool
	var stop bool

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Start the clood web interface (one command, everything works)",
		Long: `Launches the complete clood web UI stack:

  1. Starts clood proxy (multi-host routing)
  2. Starts Open WebUI + SearXNG (Docker)
  3. Opens your browser

Just run 'clood ui' and you're chatting with local LLMs.

Requirements:
  - Docker running
  - At least one Ollama host online

Examples:
  clood ui              # Start everything
  clood ui --stop       # Stop everything
  clood ui --no-browser # Start without opening browser`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if stop {
				return stopUI()
			}
			return startUI(proxyPort, uiPort, noBrowser)
		},
	}

	cmd.Flags().IntVar(&proxyPort, "proxy-port", 4000, "Port for clood proxy")
	cmd.Flags().IntVar(&uiPort, "ui-port", 3000, "Port for Open WebUI")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Don't open browser automatically")
	cmd.Flags().BoolVar(&stop, "stop", false, "Stop all UI services")

	return cmd
}

func startUI(proxyPort, uiPort int, noBrowser bool) error {
	fmt.Println(tui.RenderHeader("clood ui"))
	fmt.Println()

	// Step 1: Check Docker
	fmt.Print("  Checking Docker... ")
	if !isDockerRunning() {
		fmt.Println(tui.ErrorStyle.Render("not running"))
		fmt.Println()
		fmt.Println("  Docker Desktop needs to be running.")
		fmt.Println("  Start Docker and try again.")
		return fmt.Errorf("docker not running")
	}
	fmt.Println(tui.SuccessStyle.Render("ok"))

	// Step 2: Check for Ollama hosts
	fmt.Print("  Checking Ollama hosts... ")
	hostCount := checkOllamaHosts()
	if hostCount == 0 {
		fmt.Println(tui.ErrorStyle.Render("none online"))
		fmt.Println()
		fmt.Println("  No Ollama hosts found. Start Ollama:")
		fmt.Println("    ollama serve")
		return fmt.Errorf("no ollama hosts")
	}
	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("%d online", hostCount)))

	// Step 3: Start proxy if not running
	fmt.Print("  Starting clood proxy... ")
	if isProxyRunning(proxyPort) {
		fmt.Println(tui.SuccessStyle.Render("already running"))
	} else {
		if err := startProxy(proxyPort); err != nil {
			fmt.Println(tui.ErrorStyle.Render("failed"))
			return err
		}
		fmt.Println(tui.SuccessStyle.Render("started"))
	}

	// Step 4: Create Docker network if needed
	fmt.Print("  Docker network... ")
	createDockerNetwork()
	fmt.Println(tui.SuccessStyle.Render("ok"))

	// Step 5: Start Open WebUI
	fmt.Print("  Starting Open WebUI... ")
	if err := startOpenWebUI(); err != nil {
		fmt.Println(tui.ErrorStyle.Render("failed"))
		fmt.Println()
		fmt.Println("  Error:", err)
		return err
	}
	fmt.Println(tui.SuccessStyle.Render("started"))

	// Wait for UI to be ready
	fmt.Print("  Waiting for UI... ")
	uiURL := fmt.Sprintf("http://localhost:%d", uiPort)
	if waitForURL(uiURL, 30*time.Second) {
		fmt.Println(tui.SuccessStyle.Render("ready"))
	} else {
		fmt.Println(tui.ErrorStyle.Render("timeout"))
		fmt.Println()
		fmt.Println("  UI may still be starting. Try opening manually:")
		fmt.Println(" ", uiURL)
	}

	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render("  Ready!"))
	fmt.Println()
	fmt.Printf("  Open WebUI: %s\n", tui.SuccessStyle.Render(uiURL))
	fmt.Printf("  Proxy API:  http://localhost:%d/v1\n", proxyPort)
	fmt.Println()
	fmt.Println(tui.MutedStyle.Render("  Stop with: clood ui --stop"))
	fmt.Println()

	// Open browser
	if !noBrowser {
		openBrowserURL(uiURL)
	}

	return nil
}

func stopUI() error {
	fmt.Println(tui.RenderHeader("Stopping clood ui"))
	fmt.Println()

	// Stop Open WebUI containers
	fmt.Print("  Stopping Open WebUI... ")
	infraDir := findInfraDir()
	if infraDir != "" {
		cmd := exec.Command("docker-compose", "down")
		cmd.Dir = infraDir
		cmd.Run()
	}
	// Also try stopping by container name
	exec.Command("docker", "stop", "open-webui").Run()
	exec.Command("docker", "stop", "searxng").Run()
	fmt.Println(tui.SuccessStyle.Render("stopped"))

	// Stop proxy
	fmt.Print("  Stopping clood proxy... ")
	exec.Command("pkill", "-f", "clood proxy").Run()
	fmt.Println(tui.SuccessStyle.Render("stopped"))

	fmt.Println()
	fmt.Println(tui.MutedStyle.Render("  All services stopped."))
	return nil
}

func isDockerRunning() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

func checkOllamaHosts() int {
	// Quick check of common hosts
	hosts := []string{
		"http://localhost:11434",
		"http://mac-mini:11434",
		"http://ubuntu25:11434",
	}

	count := 0
	for _, host := range hosts {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(host + "/api/tags")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				count++
			}
		}
	}
	return count
}

func isProxyRunning(port int) bool {
	// Use /status endpoint which is faster than /v1/models
	// (models endpoint waits for host checks)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/status", port))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func startProxy(port int) error {
	// Find clood binary
	cloodPath := "clood"
	if p, err := exec.LookPath("clood"); err == nil {
		cloodPath = p
	}

	// Start proxy as detached process
	cmd := exec.Command(cloodPath, "proxy", "--port", fmt.Sprintf("%d", port))
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Detach from parent process group
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}

	// Wait for proxy to be ready (up to 30 seconds - it checks all hosts on startup)
	for i := 0; i < 60; i++ {
		time.Sleep(500 * time.Millisecond)
		if isProxyRunning(port) {
			return nil
		}
	}

	// Check if process is still alive
	if cmd.Process != nil {
		if err := cmd.Process.Signal(syscall.Signal(0)); err == nil {
			return fmt.Errorf("proxy running (PID %d) but not responding - may still be starting", cmd.Process.Pid)
		}
	}

	return fmt.Errorf("proxy failed to start on port %d", port)
}

func createDockerNetwork() {
	exec.Command("docker", "network", "create", "webui-net").Run()
}

func startOpenWebUI() error {
	infraDir := findInfraDir()
	if infraDir == "" {
		return fmt.Errorf("infrastructure directory not found")
	}

	// Try docker-compose (standalone) first, then docker compose (plugin)
	cmd := exec.Command("docker-compose", "up", "-d")
	cmd.Dir = infraDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to docker compose plugin
		cmd = exec.Command("docker", "compose", "up", "-d")
		cmd.Dir = infraDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %v", strings.TrimSpace(string(output)), err)
		}
	}
	return nil
}

func findInfraDir() string {
	// Try relative to current dir
	candidates := []string{
		"infrastructure",
		"../infrastructure",
		"../../infrastructure",
	}

	// Try relative to executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "infrastructure"),
			filepath.Join(exeDir, "..", "infrastructure"),
			filepath.Join(exeDir, "..", "..", "infrastructure"),
		)
	}

	// Try home directory
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, "Code", "clood", "infrastructure"),
		)
	}

	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "docker-compose.yml")); err == nil {
			abs, _ := filepath.Abs(dir)
			return abs
		}
	}

	return ""
}

func waitForURL(url string, timeout time.Duration) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return true
			}
		}
		time.Sleep(time.Second)
	}
	return false
}

func openBrowserURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}
