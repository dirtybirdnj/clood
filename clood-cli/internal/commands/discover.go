package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// DiscoveredHost represents a found Ollama instance
type DiscoveredHost struct {
	IP       string   `json:"ip"`
	Port     int      `json:"port"`
	Version  string   `json:"version,omitempty"`
	Models   []string `json:"models,omitempty"`
	Latency  int64    `json:"latency_ms"`
	Hostname string   `json:"hostname,omitempty"`
}

// DiscoverResult holds the full scan results
type DiscoverResult struct {
	Timestamp   string           `json:"timestamp"`
	LocalIP     string           `json:"local_ip"`
	Subnet      string           `json:"subnet"`
	ScannedIPs  int              `json:"scanned_ips"`
	FoundHosts  int              `json:"found_hosts"`
	ScanTimeMs  int64            `json:"scan_time_ms"`
	Hosts       []DiscoveredHost `json:"hosts"`
}

func DiscoverCmd() *cobra.Command {
	var jsonOutput bool
	var timeout int
	var port int
	var subnet string

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Scan local network for Ollama instances",
		Long: `Discover Ollama instances on your local network.

Scans the local subnet for machines with Ollama running on port 11434.
Useful for multi-machine setup and debugging.

Examples:
  clood discover                    # Scan local subnet
  clood discover --subnet 192.168.1.0/24  # Scan specific subnet
  clood discover --timeout 200      # Faster scan with shorter timeout
  clood discover --json             # JSON output for scripts`,
		Run: func(cmd *cobra.Command, args []string) {
			startTime := time.Now()

			// Get local IP and subnet
			localIP, detectedSubnet := getLocalNetwork()
			if subnet == "" {
				subnet = detectedSubnet
			}

			if !jsonOutput {
				fmt.Println(tui.RenderHeader("Network Discovery"))
				fmt.Println()
				fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Local IP:"), localIP)
				fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Scanning:"), subnet)
				fmt.Printf("  %s %d\n", tui.MutedStyle.Render("Port:"), port)
				fmt.Println()
				fmt.Println(tui.MutedStyle.Render("Scanning for Ollama instances..."))
				fmt.Println()
			}

			// Generate IPs to scan
			ips := generateIPs(subnet)

			// Scan in parallel with rate limiting
			hosts := scanForOllama(ips, port, timeout)

			scanDuration := time.Since(startTime)

			result := DiscoverResult{
				Timestamp:   time.Now().Format(time.RFC3339),
				LocalIP:     localIP,
				Subnet:      subnet,
				ScannedIPs:  len(ips),
				FoundHosts:  len(hosts),
				ScanTimeMs:  scanDuration.Milliseconds(),
				Hosts:       hosts,
			}

			if jsonOutput {
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return
			}

			// Display results
			if len(hosts) == 0 {
				fmt.Println(tui.WarningStyle.Render("  No Ollama instances found on the network."))
				fmt.Println()
				fmt.Println(tui.MutedStyle.Render("  Tips:"))
				fmt.Println(tui.MutedStyle.Render("  • Ensure Ollama is running: ollama serve"))
				fmt.Println(tui.MutedStyle.Render("  • Check if bound to network: OLLAMA_HOST=0.0.0.0:11434"))
				fmt.Println(tui.MutedStyle.Render("  • Check firewall allows port 11434"))
			} else {
				fmt.Println(tui.RenderHeader(fmt.Sprintf("Found %d Ollama Instance(s)", len(hosts))))
				fmt.Println()

				for _, host := range hosts {
					status := tui.SuccessStyle.Render("●")
					fmt.Printf("  %s %s:%d", status, host.IP, host.Port)
					if host.Hostname != "" {
						fmt.Printf(" (%s)", host.Hostname)
					}
					fmt.Println()
					fmt.Printf("    %s %s\n", tui.MutedStyle.Render("Version:"), host.Version)
					fmt.Printf("    %s %dms\n", tui.MutedStyle.Render("Latency:"), host.Latency)
					if len(host.Models) > 0 {
						fmt.Printf("    %s %s\n", tui.MutedStyle.Render("Models:"), strings.Join(host.Models, ", "))
					}
					fmt.Println()
				}

				// Suggest config additions
				fmt.Println(tui.RenderHeader("Suggested Config"))
				fmt.Println()
				fmt.Println(tui.MutedStyle.Render("Add to ~/.config/clood/config.yaml:"))
				fmt.Println()
				fmt.Println("hosts:")
				for i, host := range hosts {
					name := fmt.Sprintf("host-%d", i+1)
					if host.Hostname != "" {
						name = sanitizeHostname(host.Hostname)
					}
					fmt.Printf("  - name: %s\n", name)
					fmt.Printf("    url: http://%s:%d\n", host.IP, host.Port)
					fmt.Printf("    priority: %d\n", i+1)
					fmt.Printf("    enabled: true\n")
				}
			}

			fmt.Println()
			fmt.Printf("%s Scanned %d IPs in %dms\n",
				tui.MutedStyle.Render("Done:"),
				len(ips),
				scanDuration.Milliseconds())
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().IntVar(&timeout, "timeout", 500, "Connection timeout per host in ms")
	cmd.Flags().IntVar(&port, "port", 11434, "Port to scan for Ollama")
	cmd.Flags().StringVar(&subnet, "subnet", "", "Subnet to scan (default: auto-detect)")

	return cmd
}

// getLocalNetwork returns the local IP and subnet
func getLocalNetwork() (string, string) {
	// Get all network interfaces
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown", "192.168.1.0/24"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				// Calculate /24 subnet
				parts := strings.Split(ip, ".")
				if len(parts) == 4 {
					subnet := fmt.Sprintf("%s.%s.%s.0/24", parts[0], parts[1], parts[2])
					return ip, subnet
				}
			}
		}
	}

	return "unknown", "192.168.1.0/24"
}

// generateIPs creates a list of IPs to scan from a CIDR
func generateIPs(cidr string) []string {
	var ips []string

	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		// Try parsing as base IP and assume /24
		parts := strings.Split(cidr, ".")
		if len(parts) >= 3 {
			base := strings.Join(parts[:3], ".")
			for i := 1; i < 255; i++ {
				ips = append(ips, fmt.Sprintf("%s.%d", base, i))
			}
		}
		return ips
	}

	// Generate IPs from CIDR
	for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
		// Skip network and broadcast addresses
		if ip[3] != 0 && ip[3] != 255 {
			ips = append(ips, ip.String())
		}
	}

	return ips
}

// incrementIP increments an IP address
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// scanForOllama scans IPs for Ollama instances
func scanForOllama(ips []string, port int, timeoutMs int) []DiscoveredHost {
	var hosts []DiscoveredHost
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Semaphore for rate limiting (50 concurrent connections)
	sem := make(chan struct{}, 50)

	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			host := checkOllamaHost(ip, port, timeoutMs)
			if host != nil {
				mu.Lock()
				hosts = append(hosts, *host)
				mu.Unlock()
			}
		}(ip)
	}

	wg.Wait()

	// Sort by IP
	sort.Slice(hosts, func(i, j int) bool {
		return hosts[i].IP < hosts[j].IP
	})

	return hosts
}

// checkOllamaHost checks if an IP has Ollama running
func checkOllamaHost(ip string, port int, timeoutMs int) *DiscoveredHost {
	url := fmt.Sprintf("http://%s:%d/api/version", ip, port)

	client := &http.Client{
		Timeout: time.Duration(timeoutMs) * time.Millisecond,
	}

	start := time.Now()
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	latency := time.Since(start).Milliseconds()

	if resp.StatusCode != 200 {
		return nil
	}

	// Parse version response
	var versionResp struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&versionResp); err != nil {
		return nil
	}

	host := &DiscoveredHost{
		IP:      ip,
		Port:    port,
		Version: versionResp.Version,
		Latency: latency,
	}

	// Try to get hostname via reverse DNS
	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		host.Hostname = strings.TrimSuffix(names[0], ".")
	}

	// Try to get models
	modelsURL := fmt.Sprintf("http://%s:%d/api/tags", ip, port)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs*2)*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
	modelsResp, err := client.Do(req)
	if err == nil {
		defer modelsResp.Body.Close()
		var tagsResp struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if json.NewDecoder(modelsResp.Body).Decode(&tagsResp) == nil {
			for _, m := range tagsResp.Models {
				host.Models = append(host.Models, m.Name)
			}
		}
	}

	return host
}

// sanitizeHostname makes a hostname safe for config
func sanitizeHostname(hostname string) string {
	// Remove domain suffix
	parts := strings.Split(hostname, ".")
	name := parts[0]
	// Replace unsafe characters
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ToLower(name)
	return name
}
