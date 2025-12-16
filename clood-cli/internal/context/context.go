package context

// Package context handles project context detection and injection.
// This includes reading project manifests, README files, and other
// context sources to provide to LLMs.

import (
	"os"
	"path/filepath"
)

// ProjectContext holds detected project information
type ProjectContext struct {
	Name        string
	Type        string
	Root        string
	HasReadme   bool
	HasAgentsMD bool
	HasClaudeMD bool
}

// Detect scans the current directory for project context
func Detect(path string) (*ProjectContext, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	ctx := &ProjectContext{
		Name: filepath.Base(absPath),
		Root: absPath,
	}

	// Check for context files
	if _, err := os.Stat(filepath.Join(absPath, "README.md")); err == nil {
		ctx.HasReadme = true
	}
	if _, err := os.Stat(filepath.Join(absPath, "AGENTS.md")); err == nil {
		ctx.HasAgentsMD = true
	}
	if _, err := os.Stat(filepath.Join(absPath, "CLAUDE.md")); err == nil {
		ctx.HasClaudeMD = true
	}

	// Detect project type
	ctx.Type = detectType(absPath)

	return ctx, nil
}

func detectType(path string) string {
	indicators := map[string]string{
		"go.mod":         "go",
		"Cargo.toml":     "rust",
		"package.json":   "node",
		"pyproject.toml": "python",
		"requirements.txt": "python",
	}

	for file, projectType := range indicators {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			return projectType
		}
	}

	return "unknown"
}
