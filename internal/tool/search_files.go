package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type searchFilesArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}

// SearchFiles runs ripgrep to search file contents.
func SearchFiles(argsJSON string) (string, error) {
	var args searchFilesArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("search_files: invalid arguments: %w", err)
	}
	if args.Pattern == "" {
		return "", fmt.Errorf("search_files: pattern is required")
	}
	if args.Path == "" {
		return "", fmt.Errorf("search_files: path is required")
	}

	cmd := exec.Command("rg", "--line-number", "--no-heading", args.Pattern, args.Path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Exit code 1 means no matches (not an error for rg)
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			return "(no matches)", nil
		}
		if stderr.Len() > 0 {
			return "", fmt.Errorf("search_files: %s", stderr.String())
		}
		return "", fmt.Errorf("search_files: %w", err)
	}

	if stdout.Len() == 0 {
		return "(no matches)", nil
	}

	return truncateOutput(stdout.Bytes()), nil
}
