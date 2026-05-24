package tool

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type listFilesArgs struct {
	Path string `json:"path"`
}

// ListFiles lists a directory non-recursively.
func ListFiles(argsJSON string) (string, error) {
	var args listFilesArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("list_files: invalid arguments: %w", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("list_files: path is required")
	}

	entries, err := os.ReadDir(args.Path)
	if err != nil {
		return "", fmt.Errorf("list_files: %w", err)
	}

	if len(entries) == 0 {
		return "(empty directory)", nil
	}

	var sb strings.Builder
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		kind := "file"
		size := ""
		if e.IsDir() {
			kind = "dir"
		} else {
			size = fmt.Sprintf(" (%d bytes)", info.Size())
		}
		fmt.Fprintf(&sb, "%s\t%s%s\n", e.Name(), kind, size)
	}

	return sb.String(), nil
}
