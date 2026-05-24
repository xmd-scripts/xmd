package tool

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var imageExtensions = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
}

var audioExtensions = map[string]string{
	".mp3": "audio/mpeg",
	".wav": "audio/wav",
	".m4a": "audio/mp4",
	".ogg": "audio/ogg",
}

// ReadFileResult holds the result of a read_file call.
type ReadFileResult struct {
	// TextContent is set for text files.
	TextContent string
	// IsMultimodal is true for image/audio files.
	IsMultimodal bool
	// MIMEType is set for multimodal files.
	MIMEType string
	// Data is base64-encoded data for multimodal files.
	Data string
	// MediaCategory is "image" or "audio"
	MediaCategory string
}

type readFileArgs struct {
	Path string `json:"path"`
}

// ReadFile reads a file and returns its content or multimodal attachment info.
func ReadFile(argsJSON string) (*ReadFileResult, error) {
	var args readFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("read_file: invalid arguments: %w", err)
	}
	if args.Path == "" {
		return nil, fmt.Errorf("read_file: path is required")
	}

	ext := strings.ToLower(filepath.Ext(args.Path))

	if mimeType, ok := imageExtensions[ext]; ok {
		data, err := os.ReadFile(args.Path)
		if err != nil {
			return nil, fmt.Errorf("read_file: %w", err)
		}
		return &ReadFileResult{
			IsMultimodal:  true,
			MIMEType:      mimeType,
			Data:          base64.StdEncoding.EncodeToString(data),
			MediaCategory: "image",
		}, nil
	}

	if mimeType, ok := audioExtensions[ext]; ok {
		data, err := os.ReadFile(args.Path)
		if err != nil {
			return nil, fmt.Errorf("read_file: %w", err)
		}
		return &ReadFileResult{
			IsMultimodal:  true,
			MIMEType:      mimeType,
			Data:          base64.StdEncoding.EncodeToString(data),
			MediaCategory: "audio",
		}, nil
	}

	// Text file
	data, err := os.ReadFile(args.Path)
	if err != nil {
		return nil, fmt.Errorf("read_file: %w", err)
	}
	return &ReadFileResult{TextContent: string(data)}, nil
}
