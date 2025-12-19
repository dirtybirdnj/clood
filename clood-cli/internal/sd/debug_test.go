package sd

import (
	"testing"
	"time"
)

func TestQuickDiagnose(t *testing.T) {
	tests := []struct {
		name       string
		errMsg     string
		wantLayer  string
		wantSeverity Severity
	}{
		{
			name:       "connection refused",
			errMsg:     "dial tcp: connection refused",
			wantLayer:  "connection",
			wantSeverity: SeverityCritical,
		},
		{
			name:       "timeout",
			errMsg:     "context deadline exceeded",
			wantLayer:  "execution",
			wantSeverity: SeverityError,
		},
		{
			name:       "out of memory",
			errMsg:     "CUDA out of memory",
			wantLayer:  "hardware",
			wantSeverity: SeverityCritical,
		},
		{
			name:       "model not found",
			errMsg:     "checkpoint not found: sdxl_base.safetensors",
			wantLayer:  "model",
			wantSeverity: SeverityError,
		},
		{
			name:       "workflow error",
			errMsg:     "invalid node connection",
			wantLayer:  "workflow",
			wantSeverity: SeverityError,
		},
		{
			name:       "generic error",
			errMsg:     "something went wrong",
			wantLayer:  "unknown",
			wantSeverity: SeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := QuickDiagnose(tt.errMsg)
			if len(issues) == 0 {
				t.Fatal("expected at least one issue")
			}

			issue := issues[0]
			if issue.Layer != tt.wantLayer {
				t.Errorf("Layer = %q, want %q", issue.Layer, tt.wantLayer)
			}
			if issue.Severity != tt.wantSeverity {
				t.Errorf("Severity = %v, want %v", issue.Severity, tt.wantSeverity)
			}
			if issue.Resolution == "" {
				t.Error("expected non-empty Resolution")
			}
		})
	}
}

func TestEstimateDuration(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *WorkflowConfig
		wantMin  time.Duration
		wantMax  time.Duration
	}{
		{
			name: "default config",
			cfg: &WorkflowConfig{
				Steps:  25,
				Width:  1024,
				Height: 1024,
			},
			wantMin: 30 * time.Second,
			wantMax: 60 * time.Second,
		},
		{
			name: "high steps",
			cfg: &WorkflowConfig{
				Steps:  50,
				Width:  1024,
				Height: 1024,
			},
			wantMin: 60 * time.Second,
			wantMax: 120 * time.Second,
		},
		{
			name: "large resolution",
			cfg: &WorkflowConfig{
				Steps:  25,
				Width:  2048,
				Height: 2048,
			},
			wantMin: 100 * time.Second,
			wantMax: 300 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateDuration(tt.cfg)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("EstimateDuration() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestAnalyzeResult_Success(t *testing.T) {
	result := &GenerateResult{
		PromptID:   "test-123",
		ImagePaths: []string{"/path/to/image.png"},
		Duration:   30 * time.Second,
		Workflow: &WorkflowConfig{
			Steps:  25,
			Width:  1024,
			Height: 1024,
		},
	}

	report := AnalyzeResult(result, nil)

	if report.Status != StatusSuccess {
		t.Errorf("Status = %v, want StatusSuccess", report.Status)
	}
	if report.GenerationID != "test-123" {
		t.Errorf("GenerationID = %q, want %q", report.GenerationID, "test-123")
	}
	if len(report.Issues) > 0 {
		t.Errorf("expected no issues for successful generation, got %d", len(report.Issues))
	}
}

func TestAnalyzeResult_SlowGeneration(t *testing.T) {
	result := &GenerateResult{
		PromptID:   "slow-123",
		ImagePaths: []string{"/path/to/image.png"},
		Duration:   5 * time.Minute, // Much longer than expected
		Workflow: &WorkflowConfig{
			Steps:  25,
			Width:  1024,
			Height: 1024,
		},
	}

	report := AnalyzeResult(result, nil)

	if report.Status != StatusSlow {
		t.Errorf("Status = %v, want StatusSlow", report.Status)
	}

	// Should have a warning about slow generation
	hasSlowWarning := false
	for _, issue := range report.Issues {
		if issue.Layer == "performance" {
			hasSlowWarning = true
			break
		}
	}
	if !hasSlowWarning {
		t.Error("expected performance warning for slow generation")
	}
}

func TestAnalyzeResult_NoImages(t *testing.T) {
	result := &GenerateResult{
		PromptID:   "empty-123",
		ImagePaths: []string{}, // No images produced
		Duration:   30 * time.Second,
	}

	report := AnalyzeResult(result, nil)

	if report.Status != StatusPartial {
		t.Errorf("Status = %v, want StatusPartial", report.Status)
	}

	// Should have an error about no images
	hasOutputError := false
	for _, issue := range report.Issues {
		if issue.Layer == "output" {
			hasOutputError = true
			break
		}
	}
	if !hasOutputError {
		t.Error("expected output error for generation with no images")
	}
}

func TestFormatReport(t *testing.T) {
	report := &DebugReport{
		GenerationID: "test-123",
		Duration:     30 * time.Second,
		Expected:     40 * time.Second,
		Status:       StatusSuccess,
		Issues: []Issue{
			{
				Severity:   SeverityWarning,
				Layer:      "config",
				Message:    "Test warning",
				Resolution: "Fix it",
			},
		},
		Suggestions: []Suggestion{
			{
				Priority:    1,
				Category:    "speed",
				Description: "Speed it up",
				Action:      "Do the thing",
			},
		},
	}

	formatted := FormatReport(report)

	if formatted == "" {
		t.Error("FormatReport returned empty string")
	}

	// Check key content is present
	mustContain := []string{
		"SUCCESS",
		"test-123",
		"30.0s",
		"ISSUES",
		"WARNING",
		"SUGGESTIONS",
		"speed",
	}

	for _, want := range mustContain {
		if !containsString(formatted, want) {
			t.Errorf("formatted report should contain %q", want)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsString(s[1:], substr) || s[:len(substr)] == substr)
}

func TestSeverityString(t *testing.T) {
	tests := []struct {
		severity Severity
		want     string
	}{
		{SeverityInfo, "INFO"},
		{SeverityWarning, "WARNING"},
		{SeverityError, "ERROR"},
		{SeverityCritical, "CRITICAL"},
		{Severity(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.severity.String(); got != tt.want {
				t.Errorf("Severity.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerationStatusString(t *testing.T) {
	tests := []struct {
		status GenerationStatus
		want   string
	}{
		{StatusSuccess, "SUCCESS"},
		{StatusTimeout, "TIMEOUT"},
		{StatusFailed, "FAILED"},
		{StatusPartial, "PARTIAL"},
		{StatusSlow, "SLOW"},
		{StatusUnknown, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("GenerationStatus.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
