package tool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditFile_ReplacesExactString(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "edit.txt")
	if err := os.WriteFile(p, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	argsJSON := `{"path":"` + p + `","old":"world","new":"Go"}`
	msg, err := EditFile(argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(p)
	if string(data) != "hello Go" {
		t.Errorf("expected 'hello Go', got %q", string(data))
	}
	if !strings.Contains(msg, "edited") {
		t.Errorf("expected 'edited' in message, got %q", msg)
	}
}

func TestEditFile_OldNotFound(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "edit.txt")
	os.WriteFile(p, []byte("hello world"), 0o644)
	_, err := EditFile(`{"path":"` + p + `","old":"missing","new":"x"}`)
	if err == nil {
		t.Error("expected error when old string not found")
	}
}

func TestEditFile_OldAppearsMoreThanOnce(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "dup.txt")
	os.WriteFile(p, []byte("abc abc abc"), 0o644)
	_, err := EditFile(`{"path":"` + p + `","old":"abc","new":"x"}`)
	if err == nil {
		t.Error("expected error when old string appears more than once")
	}
}

func TestEditFile_FileNotExist(t *testing.T) {
	_, err := EditFile(`{"path":"/tmp/nonexistent_xmd_edit_test.txt","old":"x","new":"y"}`)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestEditFile_EmptyPath(t *testing.T) {
	_, err := EditFile(`{"path":"","old":"x","new":"y"}`)
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestEditFile_EmptyOldString(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "edit.txt")
	os.WriteFile(p, []byte("content"), 0o644)
	_, err := EditFile(`{"path":"` + p + `","old":"","new":"x"}`)
	if err == nil {
		t.Error("expected error for empty old string")
	}
}

func TestEditFile_InvalidJSON(t *testing.T) {
	_, err := EditFile(`not json`)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestEditFile_WriteFileError(t *testing.T) {
	// Read succeeds, replacement succeeds, but os.WriteFile fails because the file is read-only
	dir := t.TempDir()
	p := filepath.Join(dir, "readonly.txt")
	if err := os.WriteFile(p, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Make file read-only so write fails
	if err := os.Chmod(p, 0o444); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(p, 0o644)

	argsJSON := `{"path":"` + p + `","old":"hello","new":"goodbye"}`
	_, err := EditFile(argsJSON)
	if err == nil {
		t.Skip("permission check not enforced (may be running as root)")
	}
}
