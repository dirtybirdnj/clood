// Package clipboard provides system clipboard access for clood.
// Cross-platform support via golang.design/x/clipboard.
package clipboard

import (
	"context"
	"fmt"
	"time"

	"golang.design/x/clipboard"
)

var initialized = false

// ensureInit initializes the clipboard once
func ensureInit() error {
	if initialized {
		return nil
	}

	err := clipboard.Init()
	if err != nil {
		return fmt.Errorf("clipboard init failed: %w", err)
	}
	initialized = true
	return nil
}

// Read returns the current clipboard contents as text
func Read() (string, error) {
	if err := ensureInit(); err != nil {
		return "", err
	}

	// Read with a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Read returns bytes
	data := clipboard.Read(clipboard.FmtText)

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("clipboard read timed out")
	default:
		return string(data), nil
	}
}

// Write sets the clipboard contents
func Write(text string) error {
	if err := ensureInit(); err != nil {
		return err
	}

	clipboard.Write(clipboard.FmtText, []byte(text))
	return nil
}

// ReadImage returns clipboard image data if available
func ReadImage() ([]byte, error) {
	if err := ensureInit(); err != nil {
		return nil, err
	}

	data := clipboard.Read(clipboard.FmtImage)
	if len(data) == 0 {
		return nil, fmt.Errorf("no image in clipboard")
	}
	return data, nil
}

// HasText checks if the clipboard contains text
func HasText() bool {
	if err := ensureInit(); err != nil {
		return false
	}
	data := clipboard.Read(clipboard.FmtText)
	return len(data) > 0
}
