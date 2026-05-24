package backend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xmd-scripts/xmd/internal/tool"
)

// sseServer starts a test HTTP server that streams SSE chunks then [DONE].
func sseServer(t *testing.T, chunks []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
}

// newTestCompleter returns an openAICompleter pointed at url with a pre-set model.
func newTestCompleter(url string) *openAICompleter {
	return &openAICompleter{
		OpenAICompletion: NewOpenAICompletion(url, "test-model", ""),
		thinkingOut:      io.Discard,
	}
}

// --- EnsureModel ---

func TestEnsureModel_AutoDetectsFromModelsEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":"llama3"},{"id":"llama2"}]}`)
	}))
	defer srv.Close()

	b := NewOpenAICompletion(srv.URL+"/v1/chat/completions", "", "")
	if err := b.EnsureModel(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Model != "llama3" {
		t.Errorf("expected first model %q, got %q", "llama3", b.Model)
	}
}

func TestEnsureModel_SkipsWhenAlreadySet(t *testing.T) {
	// No server running — any HTTP call would fail, proving no request was made.
	b := NewOpenAICompletion("http://127.0.0.1:1/v1/chat/completions", "already-set", "")
	if err := b.EnsureModel(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Model != "already-set" {
		t.Errorf("model changed unexpectedly to %q", b.Model)
	}
}

func TestEnsureModel_ErrorOnUnreachableEndpoint(t *testing.T) {
	b := NewOpenAICompletion("http://127.0.0.1:1/v1/chat/completions", "", "")
	if err := b.EnsureModel(); err == nil {
		t.Error("expected error for unreachable endpoint, got nil")
	}
}

func TestEnsureModel_ErrorOnNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	b := NewOpenAICompletion(srv.URL+"/v1/chat/completions", "", "")
	if err := b.EnsureModel(); err == nil {
		t.Error("expected error for non-200 response, got nil")
	}
}

func TestEnsureModel_ErrorOnEmptyModelList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	b := NewOpenAICompletion(srv.URL+"/v1/chat/completions", "", "")
	if err := b.EnsureModel(); err == nil {
		t.Error("expected error for empty model list, got nil")
	}
}

func TestEnsureModel_SendsAPIKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":"gpt-4o"}]}`)
	}))
	defer srv.Close()

	b := NewOpenAICompletion(srv.URL+"/v1/chat/completions", "", "sk-test")
	if err := b.EnsureModel(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("expected Authorization header %q, got %q", "Bearer sk-test", gotAuth)
	}
}

// --- Complete ---

func TestComplete_StreamsTextToWriter(t *testing.T) {
	srv := sseServer(t, []string{
		`{"choices":[{"delta":{"content":"Hello"}}]}`,
		`{"choices":[{"delta":{"content":", world!"}}]}`,
	})
	defer srv.Close()

	c := newTestCompleter(srv.URL)
	var out bytes.Buffer
	msg, err := c.Complete([]tool.Message{{Role: "user", Content: "hi"}}, nil, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Hello, world!") {
		t.Errorf("expected streamed content in output, got %q", out.String())
	}
	if msg.Content != "Hello, world!" {
		t.Errorf("expected message content %q, got %q", "Hello, world!", msg.Content)
	}
}

func TestComplete_ReturnsToolCalls(t *testing.T) {
	srv := sseServer(t, []string{
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":""}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\":\"/tmp/f\"}"}}]}}]}`,
	})
	defer srv.Close()

	c := newTestCompleter(srv.URL)
	var out bytes.Buffer
	msg, err := c.Complete([]tool.Message{{Role: "user", Content: "read it"}}, nil, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(msg.ToolCalls))
	}
	tc := msg.ToolCalls[0]
	if tc.Function.Name != "read_file" {
		t.Errorf("expected tool name %q, got %q", "read_file", tc.Function.Name)
	}
	if tc.ID != "call_1" {
		t.Errorf("expected tool call ID %q, got %q", "call_1", tc.ID)
	}
}

func TestComplete_ThinkTagsRoutedToThinkingOut(t *testing.T) {
	srv := sseServer(t, []string{
		`{"choices":[{"delta":{"content":"<think>internal reasoning</think>final answer"}}]}`,
	})
	defer srv.Close()

	c := newTestCompleter(srv.URL)
	var thinking bytes.Buffer
	c.thinkingOut = &thinking

	var out bytes.Buffer
	msg, err := c.Complete([]tool.Message{{Role: "user", Content: "think"}}, nil, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out.String(), "internal reasoning") {
		t.Errorf("thinking content should not appear in stdout, got %q", out.String())
	}
	if !strings.Contains(thinking.String(), "internal reasoning") {
		t.Errorf("expected thinking content in thinkingOut, got %q", thinking.String())
	}
	if msg.Content != "final answer" {
		t.Errorf("expected clean content %q, got %q", "final answer", msg.Content)
	}
}

func TestComplete_ReasoningContentRoutedToThinkingOut(t *testing.T) {
	srv := sseServer(t, []string{
		`{"choices":[{"delta":{"reasoning_content":"step by step","content":""}}]}`,
		`{"choices":[{"delta":{"content":"answer"}}]}`,
	})
	defer srv.Close()

	c := newTestCompleter(srv.URL)
	var thinking bytes.Buffer
	c.thinkingOut = &thinking

	var out bytes.Buffer
	if _, err := c.Complete([]tool.Message{{Role: "user", Content: "reason"}}, nil, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(thinking.String(), "step by step") {
		t.Errorf("expected reasoning_content in thinkingOut, got %q", thinking.String())
	}
	if strings.Contains(out.String(), "step by step") {
		t.Errorf("reasoning_content should not appear in stdout, got %q", out.String())
	}
}

func TestComplete_ErrorOnAPIErrorInStream(t *testing.T) {
	srv := sseServer(t, []string{
		`{"error":{"message":"rate limit exceeded"}}`,
	})
	defer srv.Close()

	c := newTestCompleter(srv.URL)
	var out bytes.Buffer
	_, err := c.Complete([]tool.Message{{Role: "user", Content: "hi"}}, nil, &out)
	if err == nil {
		t.Error("expected error for API error in stream, got nil")
	}
	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("expected error to contain 'rate limit exceeded', got %q", err.Error())
	}
}

func TestComplete_ErrorOnNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestCompleter(srv.URL)
	var out bytes.Buffer
	_, err := c.Complete([]tool.Message{{Role: "user", Content: "hi"}}, nil, &out)
	if err == nil {
		t.Error("expected error for non-200 response, got nil")
	}
}

func TestComplete_SendsAPIKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer srv.Close()

	c := &openAICompleter{
		OpenAICompletion: NewOpenAICompletion(srv.URL, "test-model", "sk-secret"),
		thinkingOut:      io.Discard,
	}
	var out bytes.Buffer
	if _, err := c.Complete([]tool.Message{{Role: "user", Content: "hi"}}, nil, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer sk-secret" {
		t.Errorf("expected Authorization header %q, got %q", "Bearer sk-secret", gotAuth)
	}
}

// --- Run (end-to-end) ---

func TestRun_SimpleResponse(t *testing.T) {
	srv := sseServer(t, []string{
		`{"choices":[{"delta":{"content":"done"}}]}`,
	})
	defer srv.Close()

	b := NewOpenAICompletion(srv.URL, "test-model", "")
	var stdout, stderr bytes.Buffer
	err := b.Run("", tool.Message{Role: "user", Content: "hi"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "done") {
		t.Errorf("expected response in stdout, got %q", stdout.String())
	}
}

func TestRun_PersistsToContextFile(t *testing.T) {
	srv := sseServer(t, []string{
		`{"choices":[{"delta":{"content":"hello"}}]}`,
	})
	defer srv.Close()

	contextID := filepath.Join(t.TempDir(), "context.jsonl")
	b := NewOpenAICompletion(srv.URL, "test-model", "")
	var stdout, stderr bytes.Buffer
	if err := b.Run(contextID, tool.Message{Role: "user", Content: "hi"}, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(contextID)
	if err != nil {
		t.Fatalf("context file not created: %v", err)
	}
	if !strings.Contains(string(data), `"user"`) {
		t.Errorf("expected context file to contain conversation, got: %s", data)
	}
}

func TestRun_LoadsExistingContext(t *testing.T) {
	var receivedCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"turn %d\"}}]}\n\ndata: [DONE]\n\n", receivedCount)
		receivedCount++
	}))
	defer srv.Close()

	contextID := filepath.Join(t.TempDir(), "context.jsonl")
	b := NewOpenAICompletion(srv.URL, "test-model", "")

	// First turn
	var out1, stderr bytes.Buffer
	if err := b.Run(contextID, tool.Message{Role: "user", Content: "first"}, &out1, &stderr); err != nil {
		t.Fatalf("first turn failed: %v", err)
	}

	// Second turn — context file should now exist with first turn's history
	var out2 bytes.Buffer
	stderr.Reset()
	if err := b.Run(contextID, tool.Message{Role: "user", Content: "second"}, &out2, &stderr); err != nil {
		t.Fatalf("second turn failed: %v", err)
	}

	data, _ := os.ReadFile(contextID)
	// Should have multiple lines (user + assistant from turn 1, plus turn 2)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 3 {
		t.Errorf("expected at least 3 lines in context file after 2 turns, got %d: %s", len(lines), data)
	}
}
