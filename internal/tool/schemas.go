package tool

import "encoding/json"

// ToolDefinition is the OpenAI function definition for a tool.
type ToolDefinition struct {
	Type     string       `json:"type"`
	Function FunctionSpec `json:"function"`
}

// FunctionSpec describes a callable function.
type FunctionSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// AllTools returns all tool definitions for the mdscript tool loop.
func AllTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Type: "function",
			Function: FunctionSpec{
				Name:        "run_shell",
				Description: "Execute a shell command inside the sandbox. Use for general Unix operations — grep, find, wc, sed, awk, curl, and so on. Output is truncated at 2KB with head and tail preserved.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"command": {"type": "string", "description": "The shell command to execute"}
					},
					"required": ["command"]
				}`),
			},
		},
		{
			Type: "function",
			Function: FunctionSpec{
				Name:        "read_file",
				Description: "Read a file. Text files come back as strings. Image files (jpg, png, gif, webp) and audio files (mp3, wav, m4a, ogg) are attached to the next message so you can see or hear them directly.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"path": {"type": "string", "description": "Path to the file to read"}
					},
					"required": ["path"]
				}`),
			},
		},
		{
			Type: "function",
			Function: FunctionSpec{
				Name:        "write_file",
				Description: "Write text content to a file. Fails if outside the sandbox's writable region.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"path": {"type": "string", "description": "Path to write to"},
						"content": {"type": "string", "description": "Text content to write"}
					},
					"required": ["path", "content"]
				}`),
			},
		},
		{
			Type: "function",
			Function: FunctionSpec{
				Name:        "edit_file",
				Description: "Replace an exact string in a file. The 'old' string must match exactly and appear exactly once. Use for targeted edits; use write_file for full rewrites.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"path": {"type": "string", "description": "Path to the file to edit"},
						"old": {"type": "string", "description": "Exact string to replace (must appear exactly once)"},
						"new": {"type": "string", "description": "Replacement string"}
					},
					"required": ["path", "old", "new"]
				}`),
			},
		},
		{
			Type: "function",
			Function: FunctionSpec{
				Name:        "list_files",
				Description: "List a directory (non-recursive). Returns names, types, and sizes.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"path": {"type": "string", "description": "Path to the directory to list"}
					},
					"required": ["path"]
				}`),
			},
		},
		{
			Type: "function",
			Function: FunctionSpec{
				Name:        "search_files",
				Description: "Content search across files, ripgrep-style. Returns matching lines with file and line number.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"pattern": {"type": "string", "description": "Search pattern (regex)"},
						"path": {"type": "string", "description": "Path to search in"}
					},
					"required": ["pattern", "path"]
				}`),
			},
		},
		{
			Type: "function",
			Function: FunctionSpec{
				Name:        "question",
				Description: "Ask the user a question interactively. The runtime will display the prompt, collect the answer, and return it to you.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"prompt": {"type": "string", "description": "The question to ask the user"}
					},
					"required": ["prompt"]
				}`),
			},
		},
	}
}
