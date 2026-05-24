package tool

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type writeFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// WriteFile writes text content to a file.
func WriteFile(argsJSON string) (string, error) {
	var args writeFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("write_file: invalid arguments: %w", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("write_file: path is required")
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(args.Path), 0o755); err != nil {
		return "", fmt.Errorf("write_file: cannot create directory: %w", err)
	}

	if err := os.WriteFile(args.Path, []byte(args.Content), 0o644); err != nil {
		return "", fmt.Errorf("write_file: %w", err)
	}

	return fmt.Sprintf("wrote %d bytes to %s", len(args.Content), args.Path), nil
}
