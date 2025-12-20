// Package git provides git-related utilities for clood.
// This includes status, blame, diff, log, and other git operations.
package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// IsRepo checks if the current directory is a git repository
func IsRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// Status returns a summary of git status
func Status(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "status", "--short")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// RecentCommits returns recent commit messages
func RecentCommits(path string, count int) ([]string, error) {
	cmd := exec.Command("git", "-C", path, "log", "--oneline", "-n", strconv.Itoa(count))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	return lines, nil
}

// CurrentBranch returns the current branch name
func CurrentBranch(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// =============================================================================
// NEW: Enhanced git operations for MCP tools
// =============================================================================

// DiffOptions configures the diff output
type DiffOptions struct {
	Path     string // Repository path
	File     string // Specific file to diff (optional)
	Commit   string // Specific commit to diff (optional)
	Staged   bool   // Show staged changes only
	Context  int    // Lines of context (default 3)
	Stat     bool   // Show stat summary instead of full diff
}

// Diff returns git diff output
func Diff(opts DiffOptions) (string, error) {
	args := []string{"-C", opts.Path, "diff"}

	if opts.Staged {
		args = append(args, "--cached")
	}

	if opts.Stat {
		args = append(args, "--stat")
	} else if opts.Context > 0 {
		args = append(args, fmt.Sprintf("-U%d", opts.Context))
	}

	if opts.Commit != "" {
		args = append(args, opts.Commit)
	}

	if opts.File != "" {
		args = append(args, "--", opts.File)
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// git diff returns exit 1 if there are differences, which is not an error
		if len(output) > 0 {
			return string(output), nil
		}
		return "", fmt.Errorf("git diff failed: %w", err)
	}
	return string(output), nil
}

// BlameOptions configures blame output
type BlameOptions struct {
	Path      string
	File      string
	StartLine int // Start line (optional)
	EndLine   int // End line (optional)
	ShowEmail bool
}

// BlameLine represents a single line of blame output
type BlameLine struct {
	Commit  string `json:"commit"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// Blame returns annotated file with commit info per line
func Blame(opts BlameOptions) ([]BlameLine, error) {
	args := []string{"-C", opts.Path, "blame", "--porcelain"}

	if opts.StartLine > 0 && opts.EndLine > 0 {
		args = append(args, fmt.Sprintf("-L%d,%d", opts.StartLine, opts.EndLine))
	}

	args = append(args, opts.File)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git blame failed: %w - %s", err, string(output))
	}

	return parseBlameOutput(string(output))
}

func parseBlameOutput(output string) ([]BlameLine, error) {
	var lines []BlameLine
	currentLine := BlameLine{}
	lineNum := 0

	for _, line := range strings.Split(output, "\n") {
		if len(line) == 0 {
			continue
		}

		// Line starting with a commit hash (40 chars)
		if len(line) >= 40 && !strings.HasPrefix(line, "\t") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				currentLine.Commit = parts[0][:8] // Short hash
				lineNum, _ = strconv.Atoi(parts[2])
				currentLine.Line = lineNum
			}
		} else if strings.HasPrefix(line, "author ") {
			currentLine.Author = strings.TrimPrefix(line, "author ")
		} else if strings.HasPrefix(line, "author-time ") {
			// Unix timestamp - we'll skip parsing for now
			currentLine.Date = strings.TrimPrefix(line, "author-time ")
		} else if strings.HasPrefix(line, "\t") {
			currentLine.Content = strings.TrimPrefix(line, "\t")
			lines = append(lines, currentLine)
			currentLine = BlameLine{}
		}
	}

	return lines, nil
}

// LogOptions configures log output
type LogOptions struct {
	Path    string
	Count   int    // Number of commits (default 20)
	Author  string // Filter by author
	Since   string // Since date (e.g., "2024-01-01")
	Until   string // Until date
	Grep    string // Search commit messages
	OneLine bool   // One-line format
	Stat    bool   // Include stats
	File    string // Filter to specific file
}

// LogEntry represents a commit in the log
type LogEntry struct {
	Hash       string   `json:"hash"`
	ShortHash  string   `json:"short_hash"`
	Author     string   `json:"author"`
	AuthorEmail string  `json:"author_email"`
	Date       string   `json:"date"`
	Subject    string   `json:"subject"`
	Body       string   `json:"body,omitempty"`
	Files      []string `json:"files,omitempty"`
}

// Log returns commit history with filtering options
func Log(opts LogOptions) ([]LogEntry, error) {
	if opts.Count == 0 {
		opts.Count = 20
	}

	// Use a format that's easy to parse
	format := "%H%n%h%n%an%n%ae%n%ai%n%s%n%b%n---COMMIT_END---"

	args := []string{"-C", opts.Path, "log",
		fmt.Sprintf("-n%d", opts.Count),
		fmt.Sprintf("--format=%s", format),
	}

	if opts.Author != "" {
		args = append(args, fmt.Sprintf("--author=%s", opts.Author))
	}
	if opts.Since != "" {
		args = append(args, fmt.Sprintf("--since=%s", opts.Since))
	}
	if opts.Until != "" {
		args = append(args, fmt.Sprintf("--until=%s", opts.Until))
	}
	if opts.Grep != "" {
		args = append(args, fmt.Sprintf("--grep=%s", opts.Grep))
	}
	if opts.File != "" {
		args = append(args, "--", opts.File)
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	return parseLogOutput(string(output))
}

func parseLogOutput(output string) ([]LogEntry, error) {
	var entries []LogEntry

	commits := strings.Split(output, "---COMMIT_END---")
	for _, commit := range commits {
		commit = strings.TrimSpace(commit)
		if commit == "" {
			continue
		}

		lines := strings.Split(commit, "\n")
		if len(lines) < 6 {
			continue
		}

		entry := LogEntry{
			Hash:        lines[0],
			ShortHash:   lines[1],
			Author:      lines[2],
			AuthorEmail: lines[3],
			Date:        lines[4],
			Subject:     lines[5],
		}

		// Body is everything after subject
		if len(lines) > 6 {
			entry.Body = strings.TrimSpace(strings.Join(lines[6:], "\n"))
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// Branch represents a git branch
type Branch struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"is_current"`
	IsRemote  bool   `json:"is_remote"`
	Commit    string `json:"commit"`
	Upstream  string `json:"upstream,omitempty"`
}

// Branches returns all branches
func Branches(path string, includeRemote bool) ([]Branch, error) {
	args := []string{"-C", path, "branch", "-v", "--format=%(HEAD)|%(refname:short)|%(objectname:short)|%(upstream:short)"}

	if includeRemote {
		args = append(args[:3], append([]string{"-a"}, args[3:]...)...)
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git branch failed: %w", err)
	}

	var branches []Branch
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}

		branch := Branch{
			Name:      parts[1],
			IsCurrent: parts[0] == "*",
			Commit:    parts[2],
			IsRemote:  strings.HasPrefix(parts[1], "remotes/"),
		}

		if len(parts) > 3 && parts[3] != "" {
			branch.Upstream = parts[3]
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

// StashEntry represents a stash entry
type StashEntry struct {
	Index   int    `json:"index"`
	Branch  string `json:"branch"`
	Message string `json:"message"`
}

// Stash returns list of stash entries
func Stash(path string) ([]StashEntry, error) {
	cmd := exec.Command("git", "-C", path, "stash", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git stash list failed: %w", err)
	}

	var entries []StashEntry
	for i, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}

		// Format: stash@{0}: On branch: message
		entry := StashEntry{Index: i}

		// Parse "On branch:" if present
		if idx := strings.Index(line, ": On "); idx != -1 {
			rest := line[idx+5:]
			if colonIdx := strings.Index(rest, ": "); colonIdx != -1 {
				entry.Branch = rest[:colonIdx]
				entry.Message = rest[colonIdx+2:]
			}
		} else if idx := strings.Index(line, ": "); idx != -1 {
			entry.Message = line[idx+2:]
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// StashShow returns the diff of a specific stash entry
func StashShow(path string, index int) (string, error) {
	stashRef := fmt.Sprintf("stash@{%d}", index)
	cmd := exec.Command("git", "-C", path, "stash", "show", "-p", stashRef)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git stash show failed: %w - %s", err, string(output))
	}
	return string(output), nil
}

// Show returns the content of a specific commit
func Show(path, commit string, stat bool) (string, error) {
	args := []string{"-C", path, "show", commit}
	if stat {
		args = append(args, "--stat")
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git show failed: %w", err)
	}
	return string(output), nil
}
