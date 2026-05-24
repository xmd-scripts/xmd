package tool

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// UnmarshalJSON handles polymorphic Content: string or []ContentPart.
func (m *Message) UnmarshalJSON(data []byte) error {
	var raw struct {
		Role       string          `json:"role"`
		Content    json.RawMessage `json:"content"`
		ToolCallID string          `json:"tool_call_id,omitempty"`
		ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
		Name       string          `json:"name,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Role = raw.Role
	m.ToolCallID = raw.ToolCallID
	m.ToolCalls = raw.ToolCalls
	m.Name = raw.Name
	if len(raw.Content) == 0 || string(raw.Content) == "null" {
		return nil
	}
	switch raw.Content[0] {
	case '"':
		var s string
		if err := json.Unmarshal(raw.Content, &s); err != nil {
			return err
		}
		m.Content = s
	case '[':
		var parts []ContentPart
		if err := json.Unmarshal(raw.Content, &parts); err != nil {
			return err
		}
		m.Content = parts
	}
	return nil
}

// Message represents a chat message (OpenAI format).
type Message struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // string or []ContentPart
	ToolCallID string      `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	Name       string      `json:"name,omitempty"`
}

// ContentPart is part of a multimodal message.
type ContentPart struct {
	Type       string      `json:"type"`
	Text       string      `json:"text,omitempty"`
	ImageURL   *ImageURL   `json:"image_url,omitempty"`
	InputAudio *InputAudio `json:"input_audio,omitempty"`
}

// ImageURL is the image_url content part.
type ImageURL struct {
	URL string `json:"url"`
}

// InputAudio is audio content for multimodal messages.
type InputAudio struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

// ToolCall is a tool invocation from the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall holds tool call details.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Completer is the interface the tool loop uses to call the LLM.
type Completer interface {
	// Complete sends messages and tools to the backend. Text content is written
	// to w as it arrives (streaming backends write deltas in real time;
	// non-streaming backends write the full content before returning).
	// The returned Message always contains the full accumulated content and
	// any tool calls, and is used by the loop for history and tool dispatch.
	Complete(messages []Message, tools []ToolDefinition, w io.Writer) (*Message, error)
}

// Loop runs the tool-calling loop until the model produces a final message.
// Text output is written to stdout as it arrives via Completer.Complete.
// stderr is used for subprocess/progress output from tool calls.
// debugOut, if non-nil, receives tool results after each tool call.
// Returns the full message slice (initial + new messages added during the loop).
func Loop(backend Completer, messages []Message, stdout, stderr io.Writer, debugOut io.Writer) ([]Message, error) {
	tools := AllTools()

	for {
		reply, err := backend.Complete(messages, tools, stdout)
		if err != nil {
			return messages, fmt.Errorf("backend error: %w", err)
		}

		// Add assistant message to history
		messages = append(messages, *reply)

		// No tool calls = final answer
		if len(reply.ToolCalls) == 0 {
			return messages, nil
		}

		// Process each tool call
		for _, tc := range reply.ToolCalls {
			fmt.Fprintf(stderr, "\n[%s]\n", formatToolCall(tc.Function.Name, tc.Function.Arguments))
			result, isMultimodal, mmResult, err := dispatchTool(tc, debugOut)

			// Log the tool call
			if debugOut != nil && tc.Function.Name != "run_shell" {
				pw := newPrefixWriter("> ", debugOut)
				if err != nil {
					fmt.Fprintf(pw, "error: %s\n", err.Error())
				} else if isMultimodal {
					fmt.Fprintf(pw, "[image]\n")
				} else {
					fmt.Fprintf(pw, "%s\n", result)
				}
			}

			if err != nil {
				// Return error as tool result
				messages = append(messages, Message{
					Role:       "tool",
					Content:    fmt.Sprintf("error: %s", err.Error()),
					ToolCallID: tc.ID,
				})
				continue
			}

			if isMultimodal && mmResult != nil {
				messages = append(messages, Message{
					Role:       "tool",
					Content:    "image attached",
					ToolCallID: tc.ID,
				})
				messages = append(messages, buildMultimodalMessage(mmResult))
			} else {
				messages = append(messages, Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: tc.ID,
				})
			}
		}
	}
}


// formatToolCall formats a tool call as: name key="value" key="value"
func formatToolCall(name, argsJSON string) string {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return name + " " + argsJSON
	}
	parts := []string{name}
	for k, v := range args {
		parts = append(parts, fmt.Sprintf("%s=%q", k, fmt.Sprintf("%v", v)))
	}
	return strings.Join(parts, " ")
}

// dispatchTool routes a tool call to the appropriate handler.
func dispatchTool(tc ToolCall, debugOut io.Writer) (result string, isMultimodal bool, mmResult *ReadFileResult, err error) {
	switch tc.Function.Name {
	case "run_shell":
		r, e := RunShell(tc.Function.Arguments, debugOut)
		return r, false, nil, e
	case "read_file":
		rfr, e := ReadFile(tc.Function.Arguments)
		if e != nil {
			return "", false, nil, e
		}
		if rfr.IsMultimodal {
			return "", true, rfr, nil
		}
		return rfr.TextContent, false, nil, nil
	case "write_file":
		r, e := WriteFile(tc.Function.Arguments)
		return r, false, nil, e
	case "edit_file":
		r, e := EditFile(tc.Function.Arguments)
		return r, false, nil, e
	case "list_files":
		r, e := ListFiles(tc.Function.Arguments)
		return r, false, nil, e
	case "search_files":
		r, e := SearchFiles(tc.Function.Arguments)
		return r, false, nil, e
	case "question":
		r, e := Question(tc.Function.Arguments)
		return r, false, nil, e
	default:
		return "", false, nil, fmt.Errorf("unknown tool: %s", tc.Function.Name)
	}
}

// buildMultimodalMessage creates a user message with a multimodal attachment.
func buildMultimodalMessage(rfr *ReadFileResult) Message {
	var parts []ContentPart
	parts = append(parts, ContentPart{
		Type: "text",
		Text: "Here is the file content:",
	})

	switch rfr.MediaCategory {
	case "image":
		parts = append(parts, ContentPart{
			Type: "image_url",
			ImageURL: &ImageURL{
				URL: fmt.Sprintf("data:%s;base64,%s", rfr.MIMEType, rfr.Data),
			},
		})
	case "audio":
		format := "mp3"
		switch rfr.MIMEType {
		case "audio/wav":
			format = "wav"
		case "audio/mp4":
			format = "mp4"
		case "audio/ogg":
			format = "ogg"
		}
		parts = append(parts, ContentPart{
			Type: "input_audio",
			InputAudio: &InputAudio{
				Data:   rfr.Data,
				Format: format,
			},
		})
	}

	return Message{
		Role:    "user",
		Content: parts,
	}
}
