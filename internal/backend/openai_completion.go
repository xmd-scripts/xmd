package backend

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/xmd-scripts/xmd/internal/convo"
	"github.com/xmd-scripts/xmd/internal/tool"
)

// OpenAICompletion is the openai_completion backend adapter.
type OpenAICompletion struct {
	URL      string
	Model    string
	APIKey   string
	DebugOut io.Writer // if non-nil, tool results are written here
	client   *http.Client
}

// NewOpenAICompletion creates a new OpenAI-compatible completion backend.
func NewOpenAICompletion(url, model, apiKey string) *OpenAICompletion {
	return &OpenAICompletion{
		URL:    url,
		Model:  model,
		APIKey: apiKey,
		client: &http.Client{},
	}
}

// Run implements Backend.
//
// For role:system prompts the message is persisted to contextID and no LLM call
// is made. For role:user prompts history is loaded from contextID, the loop runs,
// and new turns are appended.
func (b *OpenAICompletion) Run(contextID string, prompt tool.Message, stdout, stderr io.Writer) error {
	if prompt.Role == "system" {
		if contextID == "" {
			fmt.Fprintln(stderr, "xmd: warning: role:system script — system prompt will be discarded (use --context to persist it)")
			return nil
		}
		return convo.Append(contextID, []tool.Message{prompt})
	}

	if err := b.EnsureModel(); err != nil {
		return err
	}

	var messages []tool.Message
	if contextID != "" {
		var err error
		messages, err = convo.Read(contextID)
		if err != nil {
			return fmt.Errorf("xmd: context: failed to read: %w", err)
		}
	}
	initialLen := len(messages)
	messages = append(messages, prompt)

	completer := &openAICompleter{OpenAICompletion: b, thinkingOut: stderr}
	result, err := tool.Loop(completer, messages, stdout, stderr, b.DebugOut)

	if contextID != "" && result != nil {
		if appendErr := convo.Append(contextID, result[initialLen:]); appendErr != nil {
			fmt.Fprintf(stderr, "xmd: context: failed to append: %s\n", appendErr)
		}
	}

	return err
}

// openAICompleter wraps OpenAICompletion to implement tool.Completer with a
// per-call ThinkingOut — keeping the public struct free of mutable streaming state.
type openAICompleter struct {
	*OpenAICompletion
	thinkingOut io.Writer
}

// EnsureModel auto-detects the model if not set.
func (b *OpenAICompletion) EnsureModel() error {
	if b.Model != "" {
		return nil
	}

	modelsURL := strings.TrimSuffix(b.URL, "/chat/completions") + "/models"
	req, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		return fmt.Errorf("xmd: backend: completion endpoint unreachable at %s", b.URL)
	}
	if b.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.APIKey)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("xmd: backend: completion endpoint unreachable at %s", b.URL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("xmd: backend: models endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return fmt.Errorf("xmd: backend: failed to parse models response: %w", err)
	}

	if len(modelsResp.Data) == 0 {
		return fmt.Errorf("xmd: backend: no models available at %s", b.URL)
	}

	b.Model = modelsResp.Data[0].ID
	return nil
}

type completionRequest struct {
	Model    string                `json:"model"`
	Messages []tool.Message        `json:"messages"`
	Tools    []tool.ToolDefinition `json:"tools,omitempty"`
	Stream   bool                  `json:"stream"`
}

// streamChunk is a single SSE chunk from the streaming API.
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"` // DeepSeek-style reasoning field
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete implements tool.Completer.
func (b *openAICompleter) Complete(messages []tool.Message, tools []tool.ToolDefinition, w io.Writer) (*tool.Message, error) {
	reqBody := completionRequest{
		Model:    b.Model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("backend: failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", b.URL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("backend: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if b.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.APIKey)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xmd: backend: completion endpoint unreachable at %s", b.URL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("xmd: backend: API error %d: %s", resp.StatusCode, string(body))
	}

	// Accumulate content and tool calls across all SSE chunks.
	// fullContent includes raw content (with <think> tags) for trailing-newline detection.
	// cleanContent excludes thinking blocks and is used for msg.Content / history.
	var fullContent strings.Builder
	var cleanContent strings.Builder
	// toolCallsAcc maps delta index -> accumulated ToolCall
	type partialToolCall struct {
		id          string
		typ         string
		name        string
		argsBuilder strings.Builder
	}
	toolCallsAcc := map[int]*partialToolCall{}

	thinkingOut := b.thinkingOut
	if thinkingOut == nil {
		thinkingOut = io.Discard
	}
	inThinking := false

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}

		if chunk.Error != nil {
			return nil, fmt.Errorf("xmd: backend: API error: %s", chunk.Error.Message)
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		// Route explicit reasoning_content field (DeepSeek-style) to ThinkingOut.
		if delta.ReasoningContent != "" {
			fmt.Fprint(thinkingOut, delta.ReasoningContent)
		}

		// Stream text content, routing <think>…</think> to ThinkingOut.
		if delta.Content != "" {
			fullContent.WriteString(delta.Content)
			remaining := delta.Content
			for len(remaining) > 0 {
				if inThinking {
					if idx := strings.Index(remaining, "</think>"); idx >= 0 {
						fmt.Fprint(thinkingOut, remaining[:idx])
						fmt.Fprintln(thinkingOut)
						inThinking = false
						remaining = remaining[idx+len("</think>"):]
					} else {
						fmt.Fprint(thinkingOut, remaining)
						remaining = ""
					}
				} else {
					if idx := strings.Index(remaining, "<think>"); idx >= 0 {
						fmt.Fprint(w, remaining[:idx])
						cleanContent.WriteString(remaining[:idx])
						inThinking = true
						remaining = remaining[idx+len("<think>"):]
					} else {
						fmt.Fprint(w, remaining)
						cleanContent.WriteString(remaining)
						remaining = ""
					}
				}
			}
		}

		// Accumulate tool call deltas
		for _, tc := range delta.ToolCalls {
			ptc, ok := toolCallsAcc[tc.Index]
			if !ok {
				ptc = &partialToolCall{}
				toolCallsAcc[tc.Index] = ptc
			}
			if tc.ID != "" {
				ptc.id = tc.ID
			}
			if tc.Type != "" {
				ptc.typ = tc.Type
			}
			if tc.Function.Name != "" {
				ptc.name = tc.Function.Name
			}
			ptc.argsBuilder.WriteString(tc.Function.Arguments)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("backend: error reading stream: %w", err)
	}

	// Ensure trailing newline when content was written
	if fullContent.Len() > 0 {
		content := fullContent.String()
		if content[len(content)-1] != '\n' {
			fmt.Fprintln(w)
		}
	}

	// Build the final Message
	msg := tool.Message{Role: "assistant"}

	if len(toolCallsAcc) > 0 {
		toolCalls := make([]tool.ToolCall, len(toolCallsAcc))
		for idx, ptc := range toolCallsAcc {
			toolCalls[idx] = tool.ToolCall{
				ID:   ptc.id,
				Type: ptc.typ,
				Function: tool.FunctionCall{
					Name:      ptc.name,
					Arguments: ptc.argsBuilder.String(),
				},
			}
		}
		msg.ToolCalls = toolCalls
	}
	msg.Content = cleanContent.String()

	return &msg, nil
}
