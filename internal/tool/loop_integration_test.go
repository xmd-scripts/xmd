//go:build integration

package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// cannedBackend replays pre-recorded responses from a JSON file.
type cannedBackend struct {
	responses []json.RawMessage
	idx       int
}

func newCannedBackend(t *testing.T, path string) *cannedBackend {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read canned responses: %v", err)
	}
	var responses []json.RawMessage
	if err := json.Unmarshal(data, &responses); err != nil {
		t.Fatalf("failed to parse canned responses: %v", err)
	}
	return &cannedBackend{responses: responses}
}

func (b *cannedBackend) Complete(messages []Message, tools []ToolDefinition, w io.Writer) (*Message, error) {
	if b.idx >= len(b.responses) {
		panic("cannedBackend: out of responses")
	}
	resp := b.responses[b.idx]
	b.idx++

	var cr struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(resp, &cr); err != nil {
		return nil, err
	}
	msg := cr.Choices[0].Message
	// Write final content to w (simulating what a real streaming backend does)
	if content, ok := msg.Content.(string); ok && content != "" && len(msg.ToolCalls) == 0 {
		fmt.Fprint(w, content)
	}
	return &msg, nil
}

func TestLoop_SimpleAnswer(t *testing.T) {
	backend := newCannedBackend(t, filepath.Join("..", "..", "test", "testdata", "canned_responses", "simple_answer.json"))
	messages := []Message{
		{Role: "user", Content: "Say hello."},
	}
	var stdout, stderr bytes.Buffer
	if _, err := Loop(backend, messages, &stdout, &stderr, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.String() != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got %q", stdout.String())
	}
}

func TestLoop_ReadFileThenAnswer(t *testing.T) {
	// Create a temp file to read
	tmp := t.TempDir()
	testFile := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	responses := []json.RawMessage{
		json.RawMessage(`{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":` + string(mustMarshal(string(mustMarshal(map[string]string{"path": testFile})))) + `}}]},"finish_reason":"tool_calls"}]}`),
		json.RawMessage(`{"choices":[{"message":{"role":"assistant","content":"The file contains: hello world","tool_calls":null},"finish_reason":"stop"}]}`),
	}

	backend := &cannedBackend{responses: responses}
	messages := []Message{
		{Role: "user", Content: "Read the file."},
	}
	var stdout, stderr bytes.Buffer
	if _, err := Loop(backend, messages, &stdout, &stderr, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.String() != "The file contains: hello world" {
		t.Errorf("unexpected result: %q", stdout.String())
	}
}

func TestLoop_WriteFileThenAnswer(t *testing.T) {
	tmp := t.TempDir()
	outFile := filepath.Join(tmp, "output.txt")

	responses := []json.RawMessage{
		json.RawMessage(`{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"write_file","arguments":` + string(mustMarshal(string(mustMarshal(map[string]string{"path": outFile, "content": "test output"})))) + `}}]},"finish_reason":"tool_calls"}]}`),
		json.RawMessage(`{"choices":[{"message":{"role":"assistant","content":"Done. I wrote the output.","tool_calls":null},"finish_reason":"stop"}]}`),
	}

	backend := &cannedBackend{responses: responses}
	messages := []Message{
		{Role: "user", Content: "Write a file."},
	}
	var stdout, stderr bytes.Buffer
	if _, err := Loop(backend, messages, &stdout, &stderr, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.String() != "Done. I wrote the output." {
		t.Errorf("unexpected result: %q", stdout.String())
	}
	// Verify file was written
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}
	if string(data) != "test output" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
