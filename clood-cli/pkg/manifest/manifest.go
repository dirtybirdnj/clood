package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Package manifest provides public APIs for project detection.
// This is the public interface for external tools to use.

// Project represents a detected project
type Project struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Root    string   `json:"root"`
	Deps    []string `json:"deps,omitempty"`
	Scripts []string `json:"scripts,omitempty"`
}

// projectIndicators maps manifest files to project types
var projectIndicators = map[string]string{
	"go.mod":           "go",
	"Cargo.toml":       "rust",
	"package.json":     "node",
	"pyproject.toml":   "python",
	"requirements.txt": "python",
	"Gemfile":          "ruby",
	"pom.xml":          "java",
	"build.gradle":     "java",
	"Makefile":         "make",
	"CMakeLists.txt":   "cmake",
}

// Detect attempts to detect the project at the given path
func Detect(path string) (*Project, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	project := &Project{
		Name: filepath.Base(absPath),
		Type: "unknown",
		Root: absPath,
	}

	// Detect project type from manifest files
	for file, projectType := range projectIndicators {
		manifestPath := filepath.Join(absPath, file)
		if _, err := os.Stat(manifestPath); err == nil {
			project.Type = projectType

			// Parse manifest for additional info
			switch file {
			case "go.mod":
				project.parseGoMod(manifestPath)
			case "package.json":
				project.parsePackageJSON(manifestPath)
			}
			break
		}
	}

	return project, nil
}

// parseGoMod extracts module name from go.mod
func (p *Project) parseGoMod(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			p.Name = strings.TrimPrefix(line, "module ")
			break
		}
	}
}

// parsePackageJSON extracts name and scripts from package.json
func (p *Project) parsePackageJSON(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var pkg struct {
		Name    string            `json:"name"`
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}

	if pkg.Name != "" {
		p.Name = pkg.Name
	}

	for scriptName := range pkg.Scripts {
		p.Scripts = append(p.Scripts, scriptName)
	}
}
