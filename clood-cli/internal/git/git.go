package git

// Package git provides git-related utilities for clood.
// This includes status, blame, diff, and other git operations.

import (
	"os/exec"
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
	cmd := exec.Command("git", "-C", path, "log", "--oneline", "-n", string(rune(count)))
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
