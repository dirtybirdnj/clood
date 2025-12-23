package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/system"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// GardenIdentity holds information about the current machine
type GardenIdentity struct {
	Hostname     string        `json:"hostname"`
	LoreName     string        `json:"lore_name"`
	Role         string        `json:"role"`
	IP           string        `json:"ip,omitempty"`
	OllamaOnline bool          `json:"ollama_online"`
	Models       []string      `json:"models"`
	Hardware     *HardwareInfo `json:"hardware,omitempty"`
	Siblings     []SiblingInfo `json:"siblings,omitempty"`
	Spirit       string        `json:"spirit,omitempty"`
}

type HardwareInfo struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
	GPU    string `json:"gpu,omitempty"`
}

type SiblingInfo struct {
	Name   string `json:"name"`
	Online bool   `json:"online"`
	Models int    `json:"models"`
}

// LoreNames maps hostnames to garden identity names
var LoreNames = map[string]struct {
	Name   string
	Spirit string
	Emoji  string
}{
	"ubuntu25":         {"Iron Keep", "Heavy lifting, background processing, the patient work.", "🏰"},
	"mac-mini":         {"Sentinel Tower", "Always watching, always ready.", "🗼"},
	"macbook-air":      {"Jade Palace", "Mobile command, agile decisions.", "🏯"},
	"mgilberts-air":    {"Jade Palace", "Mobile command, agile decisions.", "🏯"},
	"mgilberts-laptop": {"Jade Palace", "Mobile command, agile decisions.", "🏯"},
	"localhost":        {"Local Node", "Development and testing ground.", "💻"},
}

func WhoamiCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Display current garden identity and capabilities",
		Long: `Shows the identity of the current machine in the server garden.

Displays:
- Hostname and lore name
- Available Ollama models
- Hardware summary
- Sibling status (other hosts in garden)

Use -v for detailed hardware and model information.`,
		Run: func(cmd *cobra.Command, args []string) {
			identity := getGardenIdentity(verbose)

			if output.JSONMode {
				data, _ := json.Marshal(identity)
				fmt.Println(string(data))
				return
			}

			printIdentity(identity, verbose)
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information")

	return cmd
}

func getGardenIdentity(verbose bool) *GardenIdentity {
	hostname, _ := os.Hostname()
	hostname = strings.ToLower(strings.Split(hostname, ".")[0])

	identity := &GardenIdentity{
		Hostname: hostname,
		Role:     "unknown",
	}

	// Look up lore name
	if lore, ok := LoreNames[hostname]; ok {
		identity.LoreName = lore.Name
		identity.Spirit = lore.Spirit
	} else {
		identity.LoreName = hostname
		identity.Spirit = "A node in the garden."
	}

	// Determine role based on hostname
	switch {
	case strings.Contains(hostname, "ubuntu"):
		identity.Role = "worker"
	case strings.Contains(hostname, "mini"):
		identity.Role = "sentinel"
	case strings.Contains(hostname, "air") || strings.Contains(hostname, "laptop"):
		identity.Role = "commander"
	default:
		identity.Role = "node"
	}

	// Check local Ollama
	client := ollama.NewClient("http://localhost:11434", 10e9)
	models, err := client.ListModels()
	if err == nil {
		identity.OllamaOnline = true
		for _, m := range models {
			identity.Models = append(identity.Models, m.Name)
		}
	}

	// Get hardware info
	if verbose {
		hw, err := system.DetectHardware()
		if err == nil {
			identity.Hardware = &HardwareInfo{
				CPU:    hw.CPUModel,
				Memory: fmt.Sprintf("%.0fGB", hw.MemoryGB),
			}
			if hw.GPU != nil && hw.GPU.Name != "" {
				identity.Hardware.GPU = hw.GPU.Name
			}
		}
	}

	// Check siblings
	mgr := hosts.NewManager()
	mgr.AddHosts(hosts.DefaultHosts())
	statuses := mgr.CheckAllHosts()

	for _, s := range statuses {
		if s.Host.Name == "localhost" || strings.EqualFold(s.Host.Name, hostname) {
			continue
		}
		identity.Siblings = append(identity.Siblings, SiblingInfo{
			Name:   s.Host.Name,
			Online: s.Online,
			Models: len(s.Models),
		})
	}

	return identity
}

func printIdentity(identity *GardenIdentity, verbose bool) {
	// Get emoji for lore name
	emoji := "🌿"
	for _, lore := range LoreNames {
		if lore.Name == identity.LoreName {
			emoji = lore.Emoji
			break
		}
	}

	fmt.Println()
	fmt.Printf("  %s %s (%s)\n",
		emoji,
		tui.AccentStyle.Render(identity.LoreName),
		identity.Hostname)

	fmt.Printf("     %s %s\n",
		tui.MutedStyle.Render("Role:"),
		identity.Role)

	// Ollama status
	if identity.OllamaOnline {
		fmt.Printf("     %s %d models available\n",
			tui.SuccessStyle.Render("●"),
			len(identity.Models))
	} else {
		fmt.Printf("     %s Ollama offline\n",
			tui.ErrorStyle.Render("○"))
	}

	// Siblings
	if len(identity.Siblings) > 0 {
		onlineSiblings := []string{}
		offlineSiblings := []string{}
		for _, s := range identity.Siblings {
			if s.Online {
				onlineSiblings = append(onlineSiblings, fmt.Sprintf("%s (%d)", s.Name, s.Models))
			} else {
				offlineSiblings = append(offlineSiblings, s.Name)
			}
		}
		if len(onlineSiblings) > 0 {
			fmt.Printf("     %s %s\n",
				tui.MutedStyle.Render("Siblings:"),
				strings.Join(onlineSiblings, ", "))
		}
	}

	// Spirit tagline
	if identity.Spirit != "" {
		fmt.Println()
		fmt.Printf("     %s\n", tui.MutedStyle.Render("\""+identity.Spirit+"\""))
	}

	// Verbose details
	if verbose {
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("  ─────────────────────────────────────"))

		if identity.Hardware != nil {
			fmt.Printf("  %s\n", tui.MutedStyle.Render("Hardware:"))
			if identity.Hardware.CPU != "" {
				fmt.Printf("     CPU:    %s\n", identity.Hardware.CPU)
			}
			if identity.Hardware.Memory != "" {
				fmt.Printf("     Memory: %s\n", identity.Hardware.Memory)
			}
			if identity.Hardware.GPU != "" {
				fmt.Printf("     GPU:    %s\n", identity.Hardware.GPU)
			}
		}

		if len(identity.Models) > 0 {
			fmt.Println()
			fmt.Printf("  %s\n", tui.MutedStyle.Render("Available Models:"))
			for _, m := range identity.Models {
				fmt.Printf("     • %s\n", m)
			}
		}
	}

	fmt.Println()
}
