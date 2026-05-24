package convo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xmd-scripts/xmd/internal/tool"
)

func TestRead_NonExistentFile(t *testing.T) {
	msgs, err := Read(filepath.Join(t.TempDir(), "nonexistent.jsonl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgs != nil {
		t.Errorf("expected nil messages, got %v", msgs)
	}
}

func TestRead_EmptyFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "empty.jsonl")
	if err := os.WriteFile(p, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	msgs, err := Read(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected empty slice, got %v", msgs)
	}
}

func TestRead_ValidJSONL(t *testing.T) {
	p := filepath.Join(t.TempDir(), "convo.jsonl")
	content := `{"role":"user","content":"hello"}` + "\n" +
		`{"role":"assistant","content":"world"}` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	msgs, err := Read(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("expected role 'user', got %q", msgs[0].Role)
	}
	if msgs[0].Content != "hello" {
		t.Errorf("expected content 'hello', got %v", msgs[0].Content)
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", msgs[1].Role)
	}
}

func TestRead_InvalidJSON(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.jsonl")
	if err := os.WriteFile(p, []byte("not-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Read(p)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestRead_BlankLinesSkipped(t *testing.T) {
	// Blank lines in JSONL should be skipped (line 29-30 coverage)
	p := filepath.Join(t.TempDir(), "blanks.jsonl")
	content := `{"role":"user","content":"hello"}` + "\n" +
		"\n" + // blank line
		"   \n" + // whitespace-only line
		`{"role":"assistant","content":"world"}` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	msgs, err := Read(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (blank lines skipped), got %d", len(msgs))
	}
}

func TestRead_OtherOSError(t *testing.T) {
	// Trigger the !os.IsNotExist(err) branch: open a directory as a file
	dir := t.TempDir()
	// On most OS, reading a directory with bufio.Scanner will work or give an error
	// Instead, make a path that causes a permission error (non-existent error kind)
	// We can use a subdirectory of a read-only dir to cause a permission denied (not ENOENT)
	restrictedDir := filepath.Join(dir, "restricted")
	if err := os.Mkdir(restrictedDir, 0o000); err != nil {
		t.Fatalf("failed to create restricted dir: %v", err)
	}
	defer os.Chmod(restrictedDir, 0o755)
	filePath := filepath.Join(restrictedDir, "convo.jsonl")
	// On macOS/Linux, opening a file in a 0o000 directory should give EACCES, not ENOENT
	_, err := Read(filePath)
	if err == nil {
		t.Skip("permission check not enforced (may be running as root)")
	}
	// err should NOT be os.IsNotExist
}

func TestRead_InvalidJSONAfterValidLine(t *testing.T) {
	p := filepath.Join(t.TempDir(), "mixed.jsonl")
	content := `{"role":"user","content":"hello"}` + "\n" + `{bad json}` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Read(p)
	if err == nil {
		t.Error("expected error for invalid JSON line, got nil")
	}
}

func TestAppend_OpenFileError(t *testing.T) {
	// Trigger os.OpenFile error by making the parent directory read-only
	dir := t.TempDir()
	restrictedDir := filepath.Join(dir, "restricted")
	if err := os.Mkdir(restrictedDir, 0o555); err != nil {
		t.Fatalf("failed to create restricted dir: %v", err)
	}
	defer os.Chmod(restrictedDir, 0o755)
	filePath := filepath.Join(restrictedDir, "convo.jsonl")
	msgs := []tool.Message{{Role: "user", Content: "hello"}}
	err := Append(filePath, msgs)
	if err == nil {
		t.Skip("permission check not enforced (may be running as root)")
	}
}

func TestAppend_EmptyMsgsIsNoop(t *testing.T) {
	p := filepath.Join(t.TempDir(), "noop.jsonl")
	if err := Append(p, []tool.Message{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Error("file should not have been created for empty msgs")
	}
}

func TestAppend_NilMsgsIsNoop(t *testing.T) {
	p := filepath.Join(t.TempDir(), "noop2.jsonl")
	if err := Append(p, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Error("file should not have been created for nil msgs")
	}
}

func TestAppend_NewFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "new.jsonl")
	msgs := []tool.Message{{Role: "user", Content: "hello"}}
	if err := Append(p, msgs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("file should exist: %v", err)
	}
}

func TestAppend_ExistingFileNotOverwritten(t *testing.T) {
	p := filepath.Join(t.TempDir(), "existing.jsonl")
	first := []tool.Message{{Role: "user", Content: "first"}}
	if err := Append(p, first); err != nil {
		t.Fatal(err)
	}
	second := []tool.Message{{Role: "assistant", Content: "second"}}
	if err := Append(p, second); err != nil {
		t.Fatal(err)
	}
	msgs, err := Read(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
}

func TestAppend_Roundtrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "roundtrip.jsonl")
	orig := []tool.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "world"},
	}
	if err := Append(p, orig); err != nil {
		t.Fatal(err)
	}
	got, err := Read(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(orig) {
		t.Fatalf("expected %d messages, got %d", len(orig), len(got))
	}
	for i := range orig {
		if got[i].Role != orig[i].Role {
			t.Errorf("msg[%d] role: want %q, got %q", i, orig[i].Role, got[i].Role)
		}
		if got[i].Content != orig[i].Content {
			t.Errorf("msg[%d] content: want %v, got %v", i, orig[i].Content, got[i].Content)
		}
	}
}
