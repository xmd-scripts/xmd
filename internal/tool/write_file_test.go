package tool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFile_WritesContent(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "output.txt")
	argsJSON := `{"path":"` + p + `","content":"hello world"}`
	msg, err := WriteFile(argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(data))
	}
	if !strings.Contains(msg, "wrote") {
		t.Errorf("expected 'wrote' in message, got %q", msg)
	}
}

func TestWriteFile_CreatesNestedDirs(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "a", "b", "c", "file.txt")
	_, err := WriteFile(`{"path":"` + p + `","content":"nested"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("file not found: %v", err)
	}
}

func TestWriteFile_EmptyPath(t *testing.T) {
	_, err := WriteFile(`{"path":"","content":"data"}`)
	if err == nil {
		t.Error("expected error for empty path, got nil")
	}
}

func TestWriteFile_InvalidJSON(t *testing.T) {
	_, err := WriteFile(`not json`)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestWriteFile_MkdirAllError(t *testing.T) {
	// Block MkdirAll by placing a regular file where a directory is needed
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// blocker is a file; trying to create a dir underneath it should fail
	p := filepath.Join(blocker, "sub", "file.txt")
	argsJSON := `{"path":"` + p + `","content":"data"}`
	_, err := WriteFile(argsJSON)
	if err == nil {
		t.Error("expected error when MkdirAll fails (parent is a file), got nil")
	}
}

func TestWriteFile_WriteFileError(t *testing.T) {
	// Make the target directory read-only so os.WriteFile fails
	dir := t.TempDir()
	readonlyDir := filepath.Join(dir, "readonly")
	if err := os.Mkdir(readonlyDir, 0o555); err != nil {
		t.Fatalf("failed to create readonly dir: %v", err)
	}
	defer os.Chmod(readonlyDir, 0o755)
	p := filepath.Join(readonlyDir, "file.txt")
	argsJSON := `{"path":"` + p + `","content":"data"}`
	_, err := WriteFile(argsJSON)
	if err == nil {
		t.Skip("permission check not enforced (may be running as root)")
	}
}

func TestWriteFile_ReturnsNBytesMessage(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "size.txt")
	content := "abc"
	msg, err := WriteFile(`{"path":"` + p + `","content":"` + content + `"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, "3") {
		t.Errorf("expected byte count 3 in message, got %q", msg)
	}
}
