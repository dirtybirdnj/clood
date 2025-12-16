package manifest

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

// Detect attempts to detect the project at the given path
func Detect(path string) (*Project, error) {
	// TODO: Implement full detection
	return &Project{
		Name: "unknown",
		Type: "unknown",
		Root: path,
	}, nil
}
