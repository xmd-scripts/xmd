package tool

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestRunShell_EchoHello(t *testing.T) {
	out, err := RunShell(`{"command":"echo hello"}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello\n" {
		t.Errorf("expected 'hello\\n', got %q", out)
	}
}

func TestRunShell_EmptyCommand(t *testing.T) {
	_, err := RunShell(`{"command":""}`, nil)
	if err == nil {
		t.Error("expected error for empty command, got nil")
	}
}

func TestRunShell_InvalidJSON(t *testing.T) {
	_, err := RunShell(`not json`, nil)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestRunShell_NonZeroExit(t *testing.T) {
	out, err := RunShell(`{"command":"exit 3"}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[exit 3]") {
		t.Errorf("expected '[exit 3]' in output, got %q", out)
	}
}

func TestRunShell_LargeOutput(t *testing.T) {
	// Generate >2048 bytes of output
	out, err := RunShell(`{"command":"python3 -c \"print('x'*3000)\""}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "truncated") {
		t.Errorf("expected 'truncated' in output for large output, got %q (len=%d)", out, len(out))
	}
}

func TestRunShell_WithDebugOut(t *testing.T) {
	var debugBuf bytes.Buffer
	out, err := RunShell(`{"command":"echo hello"}`, &debugBuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello\n" {
		t.Errorf("expected 'hello\\n', got %q", out)
	}
	// Should not panic; debugBuf may or may not have content
}

func TestRunShell_NonZeroWithStderr(t *testing.T) {
	// Command exits non-zero AND writes to stderr
	out, err := RunShell(`{"command":"echo errline >&2; exit 1"}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[exit 1]") {
		t.Errorf("expected '[exit 1]' in output, got %q", out)
	}
	if !strings.Contains(out, "[stderr]") {
		t.Errorf("expected '[stderr]' in output, got %q", out)
	}
}

func TestRunShell_NonZeroWithStdoutAndStderr(t *testing.T) {
	// Command exits non-zero AND writes to stdout AND stderr → covers "out != """ branch before [exit N]
	out, err := RunShell(`{"command":"echo stdout_line; echo err_line >&2; exit 2"}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "stdout_line") {
		t.Errorf("expected 'stdout_line' in output, got %q", out)
	}
	if !strings.Contains(out, "[exit 2]") {
		t.Errorf("expected '[exit 2]' in output, got %q", out)
	}
	if !strings.Contains(out, "[stderr]") {
		t.Errorf("expected '[stderr]' in output, got %q", out)
	}
}

func TestRunShell_StderrOnlyZeroExit(t *testing.T) {
	// Command exits 0 but writes to stderr → covers the stderr section with no exit code prefix
	out, err := RunShell(`{"command":"echo stderr_only >&2"}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[stderr]") {
		t.Errorf("expected '[stderr]' in output, got %q", out)
	}
}

func TestTruncateOutput_Short(t *testing.T) {
	data := []byte("hello world")
	result := truncateOutput(data)
	if result != "hello world" {
		t.Errorf("short data should be unchanged, got %q", result)
	}
}

func TestTruncateOutput_Large(t *testing.T) {
	data := make([]byte, 3000)
	for i := range data {
		data[i] = 'a'
	}
	result := truncateOutput(data)
	if !strings.Contains(result, "truncated") {
		t.Errorf("expected 'truncated' in result, got %q", result)
	}
	omitted := 3000 - maxOutputBytes
	if !strings.Contains(result, fmt.Sprintf("%d", omitted)) {
		t.Errorf("expected omitted count %d in result, got %q", omitted, result)
	}
}
