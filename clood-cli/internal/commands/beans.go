package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// Bean represents a feature vision in the JDD methodology
type Bean struct {
	ID         int       `json:"id"`
	Vision     string    `json:"vision"`
	Intensity  int       `json:"intensity"`  // 1-11 (Spinal Tap dial)
	Provenance string    `json:"provenance"` // user, ai, collab, ext
	Tags       []string  `json:"tags,omitempty"`
	Created    time.Time `json:"created"`
	ForgedTo   int       `json:"forged_to,omitempty"` // GitHub issue number if forged
	Pruned     bool      `json:"pruned"`
	Notes      string    `json:"notes,omitempty"`
}

// BeanGarden is the collection of all beans
type BeanGarden struct {
	Beans    []Bean `json:"beans"`
	NextID   int    `json:"next_id"`
	LastSync string `json:"last_sync,omitempty"`
}

func BeansCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "beans",
		Short: "Jellybean Driven Development - manage feature visions",
		Long: `The bean garden where visions grow before becoming issues.

Beans are feature visions at various stages:
  🫘 Dreaming     - just an idea, intensity 1-3
  🌱 Sprouting    - taking shape, intensity 4-6
  🌿 Growing      - needs planning, intensity 7-9
  ⭐ Star Seed    - ready to forge into issue, intensity 10-11

Provenance tracks where ideas come from:
  user   - your original idea
  ai     - AI suggested
  collab - emerged from discussion
  ext    - inspired by external source

Examples:
  clood beans add "batch overnight processing"
  clood beans add "semantic search" --intensity 8 --provenance ai
  clood beans list
  clood beans show 42
  clood beans forge 42
  clood beans prune 13 --reason "out of scope"`,
		Run: func(cmd *cobra.Command, args []string) {
			// Default: list beans
			listBeans(false)
		},
	}

	// Add subcommands
	cmd.AddCommand(beansAddCmd())
	cmd.AddCommand(beansListCmd())
	cmd.AddCommand(beansShowCmd())
	cmd.AddCommand(beansForgeCmd())
	cmd.AddCommand(beansPruneCmd())
	cmd.AddCommand(beansEditCmd())

	return cmd
}

func beansAddCmd() *cobra.Command {
	var intensity int
	var provenance string
	var tags []string

	cmd := &cobra.Command{
		Use:   "add [vision]",
		Short: "Plant a new bean in the garden",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vision := strings.Join(args, " ")

			// Validate intensity (Spinal Tap dial: 1-11)
			if intensity < 1 || intensity > 11 {
				return fmt.Errorf("intensity must be 1-11 (these go to eleven)")
			}

			// Validate provenance
			validProv := map[string]bool{"user": true, "ai": true, "collab": true, "ext": true}
			if !validProv[provenance] {
				return fmt.Errorf("provenance must be: user, ai, collab, or ext")
			}

			garden, err := loadGarden()
			if err != nil {
				return err
			}

			bean := Bean{
				ID:         garden.NextID,
				Vision:     vision,
				Intensity:  intensity,
				Provenance: provenance,
				Tags:       tags,
				Created:    time.Now(),
			}

			garden.Beans = append(garden.Beans, bean)
			garden.NextID++

			if err := saveGarden(garden); err != nil {
				return err
			}

			if output.JSONMode {
				data, _ := json.MarshalIndent(bean, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("🫘 Bean #%d planted", bean.ID)))
				fmt.Printf("   %s\n", vision)
				fmt.Printf("   Intensity: %s  Provenance: %s\n",
					renderIntensity(intensity),
					provenance)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&intensity, "intensity", "i", 5, "Intensity level 1-11 (these go to eleven)")
	cmd.Flags().StringVarP(&provenance, "provenance", "p", "user", "Origin: user, ai, collab, ext")
	cmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "Comma-separated tags")

	return cmd
}

func beansListCmd() *cobra.Command {
	var showPruned bool
	var filterProv string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all beans in the garden",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listBeans(showPruned)
		},
	}

	cmd.Flags().BoolVar(&showPruned, "pruned", false, "Include pruned beans")
	cmd.Flags().StringVar(&filterProv, "provenance", "", "Filter by provenance")

	return cmd
}

func beansShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [id]",
		Short: "Show details of a specific bean",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid bean ID: %s", args[0])
			}

			garden, err := loadGarden()
			if err != nil {
				return err
			}

			for _, bean := range garden.Beans {
				if bean.ID == id {
					if output.JSONMode {
						data, _ := json.MarshalIndent(bean, "", "  ")
						fmt.Println(string(data))
					} else {
						renderBeanDetail(bean)
					}
					return nil
				}
			}

			return fmt.Errorf("bean #%d not found", id)
		},
	}
}

func beansForgeCmd() *cobra.Command {
	var issueNum int

	cmd := &cobra.Command{
		Use:   "forge [id]",
		Short: "Mark a bean as forged into a GitHub issue",
		Long: `Mark a bean as forged into a GitHub issue.

This doesn't create the issue - it records that you've forged
the bean into an issue manually or via gh CLI.

Example:
  gh issue create --title "Bean #42 vision" --body "..."
  clood beans forge 42 --issue 135`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid bean ID: %s", args[0])
			}

			if issueNum == 0 {
				return fmt.Errorf("--issue flag required (the GitHub issue number)")
			}

			garden, err := loadGarden()
			if err != nil {
				return err
			}

			for i, bean := range garden.Beans {
				if bean.ID == id {
					garden.Beans[i].ForgedTo = issueNum
					if err := saveGarden(garden); err != nil {
						return err
					}

					if output.JSONMode {
						data, _ := json.MarshalIndent(garden.Beans[i], "", "  ")
						fmt.Println(string(data))
					} else {
						fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("⭐ Bean #%d forged into issue #%d", id, issueNum)))
					}
					return nil
				}
			}

			return fmt.Errorf("bean #%d not found", id)
		},
	}

	cmd.Flags().IntVar(&issueNum, "issue", 0, "GitHub issue number")
	cmd.MarkFlagRequired("issue")

	return cmd
}

func beansPruneCmd() *cobra.Command {
	var reason string

	cmd := &cobra.Command{
		Use:   "prune [id]",
		Short: "Prune a bean from the garden",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid bean ID: %s", args[0])
			}

			garden, err := loadGarden()
			if err != nil {
				return err
			}

			for i, bean := range garden.Beans {
				if bean.ID == id {
					garden.Beans[i].Pruned = true
					garden.Beans[i].Notes = reason
					if err := saveGarden(garden); err != nil {
						return err
					}

					if output.JSONMode {
						data, _ := json.MarshalIndent(garden.Beans[i], "", "  ")
						fmt.Println(string(data))
					} else {
						fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("🗑️  Bean #%d pruned", id)))
						if reason != "" {
							fmt.Printf("   Reason: %s\n", reason)
						}
					}
					return nil
				}
			}

			return fmt.Errorf("bean #%d not found", id)
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "Reason for pruning")

	return cmd
}

func beansEditCmd() *cobra.Command {
	var intensity int
	var provenance string
	var notes string

	cmd := &cobra.Command{
		Use:   "edit [id]",
		Short: "Edit a bean's metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid bean ID: %s", args[0])
			}

			garden, err := loadGarden()
			if err != nil {
				return err
			}

			for i, bean := range garden.Beans {
				if bean.ID == id {
					if cmd.Flags().Changed("intensity") {
						if intensity < 1 || intensity > 11 {
							return fmt.Errorf("intensity must be 1-11")
						}
						garden.Beans[i].Intensity = intensity
					}
					if cmd.Flags().Changed("provenance") {
						garden.Beans[i].Provenance = provenance
					}
					if cmd.Flags().Changed("notes") {
						garden.Beans[i].Notes = notes
					}

					if err := saveGarden(garden); err != nil {
						return err
					}

					if output.JSONMode {
						data, _ := json.MarshalIndent(garden.Beans[i], "", "  ")
						fmt.Println(string(data))
					} else {
						fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("🫘 Bean #%d updated", id)))
						renderBeanDetail(garden.Beans[i])
					}
					return nil
				}
			}

			return fmt.Errorf("bean #%d not found", id)
		},
	}

	cmd.Flags().IntVarP(&intensity, "intensity", "i", 0, "New intensity level")
	cmd.Flags().StringVarP(&provenance, "provenance", "p", "", "New provenance")
	cmd.Flags().StringVarP(&notes, "notes", "n", "", "Add notes")

	return cmd
}

// Helper functions

func gardenPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "clood", "beans.json")
}

func loadGarden() (*BeanGarden, error) {
	path := gardenPath()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty garden
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &BeanGarden{NextID: 1}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var garden BeanGarden
	if err := json.Unmarshal(data, &garden); err != nil {
		return nil, err
	}

	return &garden, nil
}

func saveGarden(garden *BeanGarden) error {
	data, err := json.MarshalIndent(garden, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(gardenPath(), data, 0644)
}

func listBeans(showPruned bool) error {
	garden, err := loadGarden()
	if err != nil {
		return err
	}

	// Filter active beans
	var beans []Bean
	for _, b := range garden.Beans {
		if !b.Pruned || showPruned {
			beans = append(beans, b)
		}
	}

	if output.JSONMode {
		data, _ := json.MarshalIndent(beans, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(beans) == 0 {
		fmt.Println(tui.MutedStyle.Render("  The garden is empty. Plant some beans!"))
		fmt.Println(tui.MutedStyle.Render("  clood beans add \"your vision here\""))
		return nil
	}

	fmt.Println(tui.RenderHeader("Bean Garden"))
	fmt.Println()

	// Group by intensity for display
	dreaming := []Bean{}  // 1-3
	sprouting := []Bean{} // 4-6
	growing := []Bean{}   // 7-9
	starSeed := []Bean{}  // 10-11

	for _, b := range beans {
		switch {
		case b.Intensity <= 3:
			dreaming = append(dreaming, b)
		case b.Intensity <= 6:
			sprouting = append(sprouting, b)
		case b.Intensity <= 9:
			growing = append(growing, b)
		default:
			starSeed = append(starSeed, b)
		}
	}

	if len(starSeed) > 0 {
		fmt.Println(tui.SuccessStyle.Render("  ⭐ Star Seeds (10-11) - Ready to forge"))
		for _, b := range starSeed {
			renderBeanLine(b)
		}
		fmt.Println()
	}

	if len(growing) > 0 {
		fmt.Println(tui.TierDeepStyle.Render("  🌿 Growing (7-9) - Needs planning"))
		for _, b := range growing {
			renderBeanLine(b)
		}
		fmt.Println()
	}

	if len(sprouting) > 0 {
		fmt.Println(tui.TierFastStyle.Render("  🌱 Sprouting (4-6) - Taking shape"))
		for _, b := range sprouting {
			renderBeanLine(b)
		}
		fmt.Println()
	}

	if len(dreaming) > 0 {
		fmt.Println(tui.MutedStyle.Render("  🫘 Dreaming (1-3) - Just ideas"))
		for _, b := range dreaming {
			renderBeanLine(b)
		}
		fmt.Println()
	}

	// Summary
	forged := 0
	for _, b := range garden.Beans {
		if b.ForgedTo > 0 {
			forged++
		}
	}
	pruned := len(garden.Beans) - len(beans)

	fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("  Total: %d beans (%d forged, %d pruned)",
		len(beans), forged, pruned)))

	return nil
}

func renderBeanLine(b Bean) {
	status := ""
	if b.ForgedTo > 0 {
		status = fmt.Sprintf(" → #%d", b.ForgedTo)
	}
	if b.Pruned {
		status = " [pruned]"
	}

	prov := ""
	if b.Provenance != "user" {
		prov = fmt.Sprintf(" [%s]", b.Provenance)
	}

	vision := b.Vision
	if len(vision) > 50 {
		vision = vision[:47] + "..."
	}

	fmt.Printf("     #%-3d %s%s%s\n", b.ID, vision, prov, status)
}

func renderBeanDetail(b Bean) {
	fmt.Println()
	fmt.Printf("  🫘 Bean #%d\n", b.ID)
	fmt.Printf("  ─────────────────────────────────────\n")
	fmt.Printf("  Vision:     %s\n", b.Vision)
	fmt.Printf("  Intensity:  %s\n", renderIntensity(b.Intensity))
	fmt.Printf("  Provenance: %s\n", b.Provenance)
	fmt.Printf("  Created:    %s\n", b.Created.Format("2006-01-02 15:04"))

	if len(b.Tags) > 0 {
		fmt.Printf("  Tags:       %s\n", strings.Join(b.Tags, ", "))
	}
	if b.ForgedTo > 0 {
		fmt.Printf("  Forged to:  Issue #%d\n", b.ForgedTo)
	}
	if b.Notes != "" {
		fmt.Printf("  Notes:      %s\n", b.Notes)
	}
	if b.Pruned {
		fmt.Printf("  Status:     🗑️ Pruned\n")
	}
	fmt.Println()
}

func renderIntensity(level int) string {
	dial := ""
	for i := 1; i <= 11; i++ {
		if i == level {
			dial += "●"
		} else if i <= level {
			dial += "○"
		} else {
			dial += "·"
		}
	}

	label := ""
	switch {
	case level <= 3:
		label = "dreaming"
	case level <= 6:
		label = "sprouting"
	case level <= 9:
		label = "growing"
	case level == 10:
		label = "ready"
	case level == 11:
		label = "ELEVEN"
	}

	return fmt.Sprintf("[%s] %d (%s)", dial, level, label)
}
