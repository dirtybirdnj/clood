package sd

import (
	"fmt"
	"strings"
	"time"
)

// Severity indicates how critical an issue is.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// GenerationStatus represents the outcome of a generation attempt.
type GenerationStatus int

const (
	StatusUnknown GenerationStatus = iota
	StatusSuccess
	StatusTimeout
	StatusFailed
	StatusPartial
	StatusSlow
)

func (s GenerationStatus) String() string {
	switch s {
	case StatusSuccess:
		return "SUCCESS"
	case StatusTimeout:
		return "TIMEOUT"
	case StatusFailed:
		return "FAILED"
	case StatusPartial:
		return "PARTIAL"
	case StatusSlow:
		return "SLOW"
	default:
		return "UNKNOWN"
	}
}

// Issue represents a detected problem with a generation.
type Issue struct {
	Severity   Severity `json:"severity"`
	Layer      string   `json:"layer"`      // Which stack layer: checkpoint, lora, prompt, etc.
	Message    string   `json:"message"`
	Resolution string   `json:"resolution"` // Suggested fix
}

// Suggestion is a recommendation for improving generation.
type Suggestion struct {
	Priority    int    `json:"priority"` // 1 = high, 5 = low
	Category    string `json:"category"` // speed, quality, compatibility
	Description string `json:"description"`
	Action      string `json:"action"` // Concrete step to take
}

// DebugReport provides analysis of a generation attempt.
type DebugReport struct {
	GenerationID string           `json:"generation_id"`
	Duration     time.Duration    `json:"duration"`
	Expected     time.Duration    `json:"expected"`
	Status       GenerationStatus `json:"status"`
	Issues       []Issue          `json:"issues"`
	Suggestions  []Suggestion     `json:"suggestions"`
	RawError     string           `json:"raw_error,omitempty"`
}

// ExpectedDuration estimates how long a generation should take.
type ExpectedDuration struct {
	Steps      int
	Resolution string // "1024x1024", "512x512", etc.
	HasLoRA    bool
	LoRACount  int
}

// EstimateDuration returns expected generation time based on config.
func EstimateDuration(cfg *WorkflowConfig) time.Duration {
	// Base time per step (rough estimate for SDXL on decent GPU)
	basePerStepMs := 1500 // 1.5 seconds in milliseconds

	steps := cfg.Steps
	if steps == 0 {
		steps = 25
	}

	base := time.Duration(steps*basePerStepMs) * time.Millisecond

	// Adjust for resolution
	pixels := cfg.Width * cfg.Height
	basePixels := 1024 * 1024
	if pixels > basePixels {
		// Higher res takes longer
		factor := float64(pixels) / float64(basePixels)
		base = time.Duration(float64(base) * factor)
	}

	// Add overhead for model loading, encoding, etc.
	base += 5 * time.Second

	return base
}

// AnalyzeResult creates a debug report from a generation result.
func AnalyzeResult(result *GenerateResult, err error) *DebugReport {
	report := &DebugReport{
		Issues:      []Issue{},
		Suggestions: []Suggestion{},
	}

	if result != nil {
		report.GenerationID = result.PromptID
		report.Duration = result.Duration

		if result.Workflow != nil {
			report.Expected = EstimateDuration(result.Workflow)
		}
	}

	if err != nil {
		report.RawError = err.Error()
		report.Status = StatusFailed
		analyzeError(report, err)
	} else if result != nil {
		if len(result.ImagePaths) == 0 {
			report.Status = StatusPartial
			report.Issues = append(report.Issues, Issue{
				Severity:   SeverityError,
				Layer:      "output",
				Message:    "Generation completed but no images were produced",
				Resolution: "Check ComfyUI logs for node errors",
			})
		} else if report.Expected > 0 && report.Duration > report.Expected*2 {
			report.Status = StatusSlow
			report.Issues = append(report.Issues, Issue{
				Severity:   SeverityWarning,
				Layer:      "performance",
				Message:    fmt.Sprintf("Generation took %.1fs, expected ~%.1fs", report.Duration.Seconds(), report.Expected.Seconds()),
				Resolution: "Consider reducing steps or resolution, or check GPU utilization",
			})
		} else {
			report.Status = StatusSuccess
		}
	}

	// Add general suggestions based on config
	if result != nil && result.Workflow != nil {
		addConfigSuggestions(report, result.Workflow)
	}

	return report
}

// analyzeError examines an error and populates the report with issues.
func analyzeError(report *DebugReport, err error) {
	errStr := strings.ToLower(err.Error())

	// Connection errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "dial tcp") {
		report.Issues = append(report.Issues, Issue{
			Severity:   SeverityCritical,
			Layer:      "connection",
			Message:    "Cannot connect to ComfyUI server",
			Resolution: "Ensure ComfyUI is running: python main.py --listen",
		})
		return
	}

	// Timeout errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "context canceled") {
		report.Status = StatusTimeout
		report.Issues = append(report.Issues, Issue{
			Severity:   SeverityError,
			Layer:      "execution",
			Message:    "Generation timed out",
			Resolution: "Try reducing steps, resolution, or check if model is too large for VRAM",
		})
		report.Suggestions = append(report.Suggestions, Suggestion{
			Priority:    1,
			Category:    "speed",
			Description: "Reduce generation complexity",
			Action:      "Use --steps 20 or lower resolution",
		})
		return
	}

	// VRAM errors
	if strings.Contains(errStr, "out of memory") ||
		strings.Contains(errStr, "cuda") ||
		strings.Contains(errStr, "vram") {
		report.Issues = append(report.Issues, Issue{
			Severity:   SeverityCritical,
			Layer:      "hardware",
			Message:    "GPU ran out of VRAM",
			Resolution: "Reduce resolution, batch size, or use a smaller model",
		})
		report.Suggestions = append(report.Suggestions, Suggestion{
			Priority:    1,
			Category:    "compatibility",
			Description: "Use less VRAM",
			Action:      "Try 768x768 resolution or SD 1.5 instead of SDXL",
		})
		return
	}

	// Missing model errors
	if strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "missing") ||
		strings.Contains(errStr, "does not exist") {
		report.Issues = append(report.Issues, Issue{
			Severity:   SeverityError,
			Layer:      "model",
			Message:    "Required model or resource not found",
			Resolution: "Check that the checkpoint/LoRA exists in ComfyUI models folder",
		})
		return
	}

	// Workflow errors
	if strings.Contains(errStr, "workflow") ||
		strings.Contains(errStr, "node") ||
		strings.Contains(errStr, "invalid") {
		report.Issues = append(report.Issues, Issue{
			Severity:   SeverityError,
			Layer:      "workflow",
			Message:    "Workflow or node configuration error",
			Resolution: "Check ComfyUI for missing custom nodes or invalid connections",
		})
		return
	}

	// Generic error
	report.Issues = append(report.Issues, Issue{
		Severity:   SeverityError,
		Layer:      "unknown",
		Message:    err.Error(),
		Resolution: "Check ComfyUI logs for more details",
	})
}

// addConfigSuggestions adds suggestions based on workflow config.
func addConfigSuggestions(report *DebugReport, cfg *WorkflowConfig) {
	// High step count
	if cfg.Steps > 40 {
		report.Suggestions = append(report.Suggestions, Suggestion{
			Priority:    2,
			Category:    "speed",
			Description: "High step count may not improve quality",
			Action:      fmt.Sprintf("Consider using 25-30 steps instead of %d", cfg.Steps),
		})
	}

	// Very low step count
	if cfg.Steps > 0 && cfg.Steps < 15 {
		report.Suggestions = append(report.Suggestions, Suggestion{
			Priority:    2,
			Category:    "quality",
			Description: "Low step count may produce artifacts",
			Action:      "Consider using at least 20 steps for better quality",
		})
	}

	// Extreme CFG
	if cfg.CFGScale > 12 {
		report.Suggestions = append(report.Suggestions, Suggestion{
			Priority:    3,
			Category:    "quality",
			Description: "Very high CFG scale can cause oversaturation",
			Action:      fmt.Sprintf("Try CFG 7-10 instead of %.1f", cfg.CFGScale),
		})
	}
	if cfg.CFGScale > 0 && cfg.CFGScale < 3 {
		report.Suggestions = append(report.Suggestions, Suggestion{
			Priority:    3,
			Category:    "quality",
			Description: "Very low CFG scale may produce incoherent results",
			Action:      "Try CFG 5-7 for better prompt adherence",
		})
	}

	// Large resolution
	if cfg.Width*cfg.Height > 1536*1536 {
		report.Suggestions = append(report.Suggestions, Suggestion{
			Priority:    2,
			Category:    "speed",
			Description: "Large resolution significantly increases generation time",
			Action:      "Consider generating at 1024x1024 and upscaling afterward",
		})
	}
}

// AnalyzeSlowGeneration provides detailed analysis of why a generation was slow.
func AnalyzeSlowGeneration(result *GenerateResult) *DebugReport {
	report := &DebugReport{
		GenerationID: result.PromptID,
		Duration:     result.Duration,
		Status:       StatusSlow,
		Issues:       []Issue{},
		Suggestions:  []Suggestion{},
	}

	if result.Workflow == nil {
		return report
	}

	cfg := result.Workflow
	report.Expected = EstimateDuration(cfg)

	// Calculate time per step
	timePerStep := result.Duration / time.Duration(cfg.Steps)

	if timePerStep > 3*time.Second {
		report.Issues = append(report.Issues, Issue{
			Severity:   SeverityWarning,
			Layer:      "performance",
			Message:    fmt.Sprintf("%.1fs per step is slow (expected ~1.5s)", timePerStep.Seconds()),
			Resolution: "GPU may be thermal throttling or model may be too large",
		})
	}

	// Resolution impact
	pixels := cfg.Width * cfg.Height
	if pixels > 1024*1024 {
		factor := float64(pixels) / float64(1024*1024)
		report.Suggestions = append(report.Suggestions, Suggestion{
			Priority:    1,
			Category:    "speed",
			Description: fmt.Sprintf("Resolution is %.1fx larger than 1024x1024", factor),
			Action:      "Generate at 1024x1024 and use upscaler for final output",
		})
	}

	// Step count impact
	if cfg.Steps > 30 {
		report.Suggestions = append(report.Suggestions, Suggestion{
			Priority:    1,
			Category:    "speed",
			Description: "High step count is a major factor in generation time",
			Action:      "Most samplers produce good results at 20-25 steps",
		})
	}

	return report
}

// QuickDiagnose performs fast diagnosis based on error message alone.
func QuickDiagnose(errMsg string) []Issue {
	report := &DebugReport{Issues: []Issue{}}
	analyzeError(report, fmt.Errorf("%s", errMsg))
	return report.Issues
}

// FormatReport creates a human-readable string from a debug report.
func FormatReport(report *DebugReport) string {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("=== Debug Report: %s ===\n", report.Status))
	if report.GenerationID != "" {
		b.WriteString(fmt.Sprintf("Generation ID: %s\n", report.GenerationID))
	}
	if report.Duration > 0 {
		b.WriteString(fmt.Sprintf("Duration: %.1fs", report.Duration.Seconds()))
		if report.Expected > 0 {
			b.WriteString(fmt.Sprintf(" (expected: ~%.1fs)", report.Expected.Seconds()))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Issues
	if len(report.Issues) > 0 {
		b.WriteString("ISSUES:\n")
		for _, issue := range report.Issues {
			b.WriteString(fmt.Sprintf("  [%s] %s: %s\n", issue.Severity, issue.Layer, issue.Message))
			if issue.Resolution != "" {
				b.WriteString(fmt.Sprintf("         → %s\n", issue.Resolution))
			}
		}
		b.WriteString("\n")
	}

	// Suggestions
	if len(report.Suggestions) > 0 {
		b.WriteString("SUGGESTIONS:\n")
		for _, sug := range report.Suggestions {
			b.WriteString(fmt.Sprintf("  [P%d/%s] %s\n", sug.Priority, sug.Category, sug.Description))
			if sug.Action != "" {
				b.WriteString(fmt.Sprintf("         → %s\n", sug.Action))
			}
		}
	}

	return b.String()
}
