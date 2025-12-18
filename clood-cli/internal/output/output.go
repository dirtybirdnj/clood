// Package output provides shared output formatting for CLI commands.
// Supports both human-readable (tables, colors) and machine-readable (JSON) output.
package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSONMode indicates whether to output JSON instead of human-readable format.
// Set via --json flag on root command.
var JSONMode bool

// JSON outputs data as JSON to stdout.
// Returns error if marshaling fails.
func JSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// JSONCompact outputs data as compact JSON (no indentation).
func JSONCompact(v interface{}) error {
	return json.NewEncoder(os.Stdout).Encode(v)
}

// MustJSON outputs data as JSON, panics on error.
func MustJSON(v interface{}) {
	if err := JSON(v); err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %v", err))
	}
}

// IsJSON returns true if JSON output mode is enabled.
func IsJSON() bool {
	return JSONMode
}
