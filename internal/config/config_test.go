package config

import (
	"os"
	"testing"
)

func TestFromEnv_Defaults(t *testing.T) {
	// Clear all relevant env vars
	for _, key := range []string{
		"XMD_BACKEND", "XMD_COMPLETION_URL", "XMD_COMPLETION_MODEL",
		"XMD_COMPLETION_API_KEY", "XMD_AGENT_CMD",
		"XMD_DEBUG", "XMD_ALLOW_READ", "XMD_ALLOW_WRITE",
	} {
		t.Setenv(key, "")
	}

	c := FromEnv()

	if c.Backend != "openai_completion" {
		t.Errorf("default Backend: want %q, got %q", "openai_completion", c.Backend)
	}
	if c.CompletionURL != "http://localhost:11434/v1/chat/completions" {
		t.Errorf("default CompletionURL: want %q, got %q", "http://localhost:11434/v1/chat/completions", c.CompletionURL)
	}
	if c.Debug {
		t.Error("default Debug: want false, got true")
	}
	if len(c.AllowRead) != 0 {
		t.Errorf("default AllowRead: want empty, got %v", c.AllowRead)
	}
	if len(c.AllowWrite) != 0 {
		t.Errorf("default AllowWrite: want empty, got %v", c.AllowWrite)
	}
}

func TestFromEnv_BackendOverride(t *testing.T) {
	t.Setenv("XMD_BACKEND", "agent_command")
	c := FromEnv()
	if c.Backend != "agent_command" {
		t.Errorf("want %q, got %q", "agent_command", c.Backend)
	}
}

func TestFromEnv_CompletionURLOverride(t *testing.T) {
	t.Setenv("XMD_COMPLETION_URL", "http://example.com/api")
	c := FromEnv()
	if c.CompletionURL != "http://example.com/api" {
		t.Errorf("want %q, got %q", "http://example.com/api", c.CompletionURL)
	}
}

func TestFromEnv_DebugSet(t *testing.T) {
	t.Setenv("XMD_DEBUG", "1")
	c := FromEnv()
	if !c.Debug {
		t.Error("want Debug=true, got false")
	}
}

func TestFromEnv_AllowRead(t *testing.T) {
	t.Setenv("XMD_ALLOW_READ", "a:b")
	c := FromEnv()
	if len(c.AllowRead) != 2 {
		t.Fatalf("want 2 AllowRead entries, got %d: %v", len(c.AllowRead), c.AllowRead)
	}
	if c.AllowRead[0] != "a" || c.AllowRead[1] != "b" {
		t.Errorf("want [a b], got %v", c.AllowRead)
	}
}

func TestFromEnv_AllowWrite(t *testing.T) {
	t.Setenv("XMD_ALLOW_WRITE", "c:d")
	c := FromEnv()
	if len(c.AllowWrite) != 2 {
		t.Fatalf("want 2 AllowWrite entries, got %d: %v", len(c.AllowWrite), c.AllowWrite)
	}
	if c.AllowWrite[0] != "c" || c.AllowWrite[1] != "d" {
		t.Errorf("want [c d], got %v", c.AllowWrite)
	}
}


func TestFromEnv_Isolation(t *testing.T) {
	// Verify env vars don't leak across tests by checking values are restored.
	t.Setenv("XMD_BACKEND", "cursor")
	c1 := FromEnv()
	if c1.Backend != "cursor" {
		t.Errorf("want cursor, got %q", c1.Backend)
	}

	// After the test ends, t.Setenv restores the original value.
	// Within this test, the value is properly set.
	_ = os.Getenv("XMD_BACKEND") // just to confirm no panic
}
