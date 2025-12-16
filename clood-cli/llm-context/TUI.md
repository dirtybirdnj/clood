# TUI Package Usage Guide

This file contains correct usage patterns for the `internal/tui` package.
Include this context when asking LLMs to generate clood commands.

## CRITICAL: Style Objects Require .Render()

The tui package uses `lipgloss.Style` objects. These are NOT functions.
You MUST call `.Render(string)` on them.

### WRONG (Common LLM Mistake)
```go
// DO NOT DO THIS
tui.ErrorStyle("error message")           // Wrong!
tui.MutedStyle("info")                    // Wrong!
fmt.Println(tui.HeaderStyle("title"))     // Wrong!
```

### CORRECT Usage
```go
// Style objects - call .Render()
fmt.Println(tui.ErrorStyle.Render("error message"))
fmt.Println(tui.MutedStyle.Render("secondary info"))
fmt.Println(tui.HeaderStyle.Render("Section Title"))
fmt.Println(tui.SuccessStyle.Render("Operation succeeded"))

// For stderr errors
fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error: "+err.Error()))
```

## Available Styles

| Style | Purpose | Example |
|-------|---------|---------|
| `tui.ErrorStyle` | Error messages | Red, bold |
| `tui.MutedStyle` | Secondary info | Gray/dim |
| `tui.HeaderStyle` | Section headers | Blue, bold, underline |
| `tui.SuccessStyle` | Success messages | Green |
| `tui.TierFastStyle` | Tier 1 indicator | Yellow/gold |
| `tui.TierDeepStyle` | Tier 2+ indicator | Blue |

## Helper Functions

These functions handle rendering internally - just call them directly:

```go
// Banner with logo
banner := tui.RenderBanner()
fmt.Println(banner)

// Section header (styled)
fmt.Println(tui.RenderHeader("My Section"))

// Tier indicator with icon
fmt.Println(tui.RenderTier(1))  // "⚡ Tier 1: Speed Mode"
fmt.Println(tui.RenderTier(2))  // "🧠 Tier 2: Deep Mode"
```

## Complete Command Example

```go
package commands

import (
    "fmt"
    "os"

    "github.com/dirtybirdnj/clood/internal/tui"
    "github.com/spf13/cobra"
)

func ExampleCmd() *cobra.Command {
    var jsonOutput bool

    cmd := &cobra.Command{
        Use:   "example [args]",
        Short: "Example command description",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Header
            fmt.Println(tui.RenderHeader("Example Output"))
            fmt.Println()

            // Normal output with styles
            fmt.Printf("  %s: %s\n",
                tui.HeaderStyle.Render("Status"),
                tui.SuccessStyle.Render("OK"))

            // Muted secondary info
            fmt.Println(tui.MutedStyle.Render("Additional details here"))

            // Error handling
            if someError != nil {
                fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error: "+someError.Error()))
                return nil  // Don't return error, we already printed it
            }

            return nil
        },
    }

    cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

    return cmd
}
```

## Common Patterns

### Printing a list with headers
```go
fmt.Println(tui.RenderHeader("Files Found"))
fmt.Println()
for _, file := range files {
    fmt.Printf("  %s\n", file)
}
```

### Status with colored indicator
```go
status := tui.SuccessStyle.Render("online")
// or
status := tui.ErrorStyle.Render("offline")
fmt.Printf("  Host: %s [%s]\n", hostname, status)
```

### JSON output alternative
```go
if jsonOutput {
    output, _ := json.MarshalIndent(data, "", "  ")
    fmt.Println(string(output))
} else {
    // Pretty printed output with styles
    printPretty(data)
}
```
