package tool

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type editFileArgs struct {
	Path string `json:"path"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

// EditFile performs an exact-string replacement in a file.
func EditFile(argsJSON string) (string, error) {
	var args editFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("edit_file: invalid arguments: %w", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("edit_file: path is required")
	}
	if args.Old == "" {
		return "", fmt.Errorf("edit_file: old string is required")
	}

	data, err := os.ReadFile(args.Path)
	if err != nil {
		return "", fmt.Errorf("edit_file: %w", err)
	}

	content := string(data)
	count := strings.Count(content, args.Old)
	if count == 0 {
		return "", fmt.Errorf("edit_file: old string not found in %s", args.Path)
	}
	if count > 1 {
		return "", fmt.Errorf("edit_file: old string found %d times in %s (must appear exactly once)", count, args.Path)
	}

	newContent := strings.Replace(content, args.Old, args.New, 1)
	if err := os.WriteFile(args.Path, []byte(newContent), 0o644); err != nil {
		return "", fmt.Errorf("edit_file: %w", err)
	}

	return fmt.Sprintf("edited %s: replaced %d bytes with %d bytes", args.Path, len(args.Old), len(args.New)), nil
}
