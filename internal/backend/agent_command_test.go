package backend

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xmd-scripts/xmd/internal/tool"
)

func TestAgentCommand_PromptFileContainsPromptContent(t *testing.T) {
	a := NewAgentCommand(`cat "$XMD_PROMPT_FILE"`)
	p := tool.Message{Role: "user", Content: "hello from test"}
	var stdout, stderr bytes.Buffer
	if err := a.Run("", p, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "hello from test") {
		t.Errorf("expected prompt content in stdout, got: %q", stdout.String())
	}
}

func TestAgentCommand_SystemRoleGoesToSystemFile(t *testing.T) {
	a := NewAgentCommand(`cat "$XMD_SYSTEM_PROMPT_FILE"`)
	p := tool.Message{Role: "system", Content: "you are a helpful pirate"}
	var stdout, stderr bytes.Buffer
	if err := a.Run("", p, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "you are a helpful pirate") {
		t.Errorf("expected system content in XMD_SYSTEM_PROMPT_FILE, got: %q", stdout.String())
	}
}

func TestAgentCommand_SystemRolePromptFileIsEmpty(t *testing.T) {
	a := NewAgentCommand(`cat "$XMD_PROMPT_FILE"`)
	p := tool.Message{Role: "system", Content: "you are a helpful pirate"}
	var stdout, stderr bytes.Buffer
	if err := a.Run("", p, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimRight(stdout.String(), "\n") != "" {
		t.Errorf("expected empty XMD_PROMPT_FILE for role:system, got: %q", stdout.String())
	}
}

func TestAgentCommand_UserRoleGoesToPromptFile(t *testing.T) {
	a := NewAgentCommand(`cat "$XMD_PROMPT_FILE"`)
	p := tool.Message{Role: "user", Content: "what is the weather?"}
	var stdout, stderr bytes.Buffer
	if err := a.Run("", p, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "what is the weather?") {
		t.Errorf("expected user content in XMD_PROMPT_FILE, got: %q", stdout.String())
	}
}


func TestAgentCommand_SessionIDDerivedFromContextID(t *testing.T) {
	a := NewAgentCommand(`printf '%s' "$XMD_SESSION_ID"`)
	p := tool.Message{Role: "user", Content: "hi"}
	var stdout, stderr bytes.Buffer
	if err := a.Run("/some/path/context.jsonl", p, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sessionID := strings.TrimRight(stdout.String(), "\n")
	if sessionID == "" {
		t.Errorf("expected non-empty session ID, got empty string")
	}
}

func TestAgentCommand_SameContextIDSameSessionID(t *testing.T) {
	a := NewAgentCommand(`printf '%s' "$XMD_SESSION_ID"`)
	p := tool.Message{Role: "user", Content: "hi"}

	var out1, out2, stderr bytes.Buffer
	if err := a.Run("/path/to/ctx.jsonl", p, &out1, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stderr.Reset()
	if err := a.Run("/path/to/ctx.jsonl", p, &out2, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	id1 := strings.TrimRight(out1.String(), "\n")
	id2 := strings.TrimRight(out2.String(), "\n")
	if id1 != id2 {
		t.Errorf("expected same session ID for same context ID, got %q and %q", id1, id2)
	}
}

func TestAgentCommand_EmptySessionIDWhenNoContextID(t *testing.T) {
	a := NewAgentCommand(`printf '%s' "$XMD_SESSION_ID"`)
	p := tool.Message{Role: "user", Content: "hi"}
	var stdout, stderr bytes.Buffer
	if err := a.Run("", p, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimRight(stdout.String(), "\n") != "" {
		t.Errorf("expected empty session ID when no context ID, got %q", stdout.String())
	}
}

func TestAgentCommand_EmptyCmd(t *testing.T) {
	a := NewAgentCommand("   ")
	p := tool.Message{Role: "user", Content: "hi"}
	var stdout, stderr bytes.Buffer
	if err := a.Run("", p, &stdout, &stderr); err == nil {
		t.Error("expected error for empty command, got nil")
	}
}

func TestAgentCommand_StdoutForwarded(t *testing.T) {
	a := NewAgentCommand(`printf 'agent output'`)
	p := tool.Message{Role: "user", Content: "hi"}
	var stdout, stderr bytes.Buffer
	if err := a.Run("", p, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimRight(stdout.String(), "\n") != "agent output" {
		t.Errorf("expected agent stdout to be forwarded, got: %q", stdout.String())
	}
}
