package logging

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogEntry represents a single interaction log entry.
type LogEntry struct {
	Timestamp   time.Time         `json:"timestamp"`
	Type        string            `json:"type"` // "ask", "generate", "chat", "catfight", etc.
	Command     string            `json:"command,omitempty"`
	Model       string            `json:"model,omitempty"`
	Host        string            `json:"host,omitempty"`
	Tier        int               `json:"tier,omitempty"`
	Prompt      string            `json:"prompt,omitempty"`
	Response    string            `json:"response,omitempty"`
	Tokens      TokenUsage        `json:"tokens,omitempty"`
	Duration    time.Duration     `json:"duration_ns,omitempty"`
	DurationSec float64           `json:"duration_sec,omitempty"`
	Success     bool              `json:"success"`
	Error       string            `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	Prompt     int `json:"prompt,omitempty"`
	Completion int `json:"completion,omitempty"`
	Total      int `json:"total,omitempty"`
}

// Logger handles structured logging to JSONL file.
type Logger struct {
	path string
	file *os.File
	mu   sync.Mutex
}

// DefaultLogPath returns the default log file path.
func DefaultLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clood", "conversations.jsonl")
}

// NewLogger creates a new JSONL logger.
func NewLogger(path string) (*Logger, error) {
	if path == "" {
		path = DefaultLogPath()
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	// Open file in append mode
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	return &Logger{path: path, file: file}, nil
}

// Log writes a log entry to the file.
func (l *Logger) Log(entry *LogEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Calculate duration in seconds for readability
	if entry.Duration > 0 {
		entry.DurationSec = entry.Duration.Seconds()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}

	if _, err := l.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write entry: %w", err)
	}

	return nil
}

// Close closes the log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// Path returns the log file path.
func (l *Logger) Path() string {
	return l.path
}

// Query options for reading logs.
type QueryOptions struct {
	Type     string    // Filter by type
	Model    string    // Filter by model
	Host     string    // Filter by host
	Since    time.Time // Only entries after this time
	Until    time.Time // Only entries before this time
	Limit    int       // Maximum entries to return (0 = no limit)
	Tail     bool      // If true, return last N entries instead of first N
	OnlyErrs bool      // Only show entries with errors
}

// Query reads and filters log entries.
func Query(path string, opts QueryOptions) ([]LogEntry, error) {
	if path == "" {
		path = DefaultLogPath()
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEntry{}, nil
		}
		return nil, fmt.Errorf("open log: %w", err)
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large entries

	for scanner.Scan() {
		var entry LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip malformed entries
		}

		// Apply filters
		if opts.Type != "" && entry.Type != opts.Type {
			continue
		}
		if opts.Model != "" && entry.Model != opts.Model {
			continue
		}
		if opts.Host != "" && entry.Host != opts.Host {
			continue
		}
		if !opts.Since.IsZero() && entry.Timestamp.Before(opts.Since) {
			continue
		}
		if !opts.Until.IsZero() && entry.Timestamp.After(opts.Until) {
			continue
		}
		if opts.OnlyErrs && entry.Error == "" {
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan log: %w", err)
	}

	// Apply limit
	if opts.Limit > 0 && len(entries) > opts.Limit {
		if opts.Tail {
			entries = entries[len(entries)-opts.Limit:]
		} else {
			entries = entries[:opts.Limit]
		}
	}

	return entries, nil
}

// Stats provides aggregate statistics from logs.
type Stats struct {
	TotalEntries    int                `json:"total_entries"`
	ByType          map[string]int     `json:"by_type"`
	ByModel         map[string]int     `json:"by_model"`
	ByHost          map[string]int     `json:"by_host"`
	SuccessRate     float64            `json:"success_rate"`
	TotalTokens     int                `json:"total_tokens"`
	TotalDurationMs int64              `json:"total_duration_ms"`
	AvgDurationMs   float64            `json:"avg_duration_ms"`
	FirstEntry      time.Time          `json:"first_entry"`
	LastEntry       time.Time          `json:"last_entry"`
}

// GetStats returns aggregate statistics from the log file.
func GetStats(path string) (*Stats, error) {
	entries, err := Query(path, QueryOptions{})
	if err != nil {
		return nil, err
	}

	stats := &Stats{
		ByType:  make(map[string]int),
		ByModel: make(map[string]int),
		ByHost:  make(map[string]int),
	}

	successCount := 0

	for i, entry := range entries {
		stats.TotalEntries++

		if entry.Type != "" {
			stats.ByType[entry.Type]++
		}
		if entry.Model != "" {
			stats.ByModel[entry.Model]++
		}
		if entry.Host != "" {
			stats.ByHost[entry.Host]++
		}

		if entry.Success {
			successCount++
		}

		stats.TotalTokens += entry.Tokens.Total
		stats.TotalDurationMs += entry.Duration.Milliseconds()

		if i == 0 {
			stats.FirstEntry = entry.Timestamp
		}
		stats.LastEntry = entry.Timestamp
	}

	if stats.TotalEntries > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.TotalEntries) * 100
		stats.AvgDurationMs = float64(stats.TotalDurationMs) / float64(stats.TotalEntries)
	}

	return stats, nil
}

// Global logger instance
var globalLogger *Logger
var loggerOnce sync.Once

// GetLogger returns the global logger instance.
func GetLogger() *Logger {
	loggerOnce.Do(func() {
		var err error
		globalLogger, err = NewLogger("")
		if err != nil {
			// Silent fail - logging is optional
			globalLogger = nil
		}
	})
	return globalLogger
}

// LogInteraction is a convenience function for logging.
func LogInteraction(entryType, command, model, host string, tier int, prompt, response string, duration time.Duration, err error) {
	logger := GetLogger()
	if logger == nil {
		return
	}

	entry := &LogEntry{
		Type:     entryType,
		Command:  command,
		Model:    model,
		Host:     host,
		Tier:     tier,
		Prompt:   prompt,
		Response: response,
		Duration: duration,
		Success:  err == nil,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	logger.Log(entry)
}
