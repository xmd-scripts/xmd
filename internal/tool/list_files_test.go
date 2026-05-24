package tool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListFiles_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	result, err := ListFiles(`{"path":"` + tmp + `"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "(empty directory)" {
		t.Errorf("expected '(empty directory)', got %q", result)
	}
}

func TestListFiles_OneFile(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "hello.txt")
	if err := os.WriteFile(p, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ListFiles(`{"path":"` + tmp + `"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "hello.txt") {
		t.Errorf("expected 'hello.txt' in result, got %q", result)
	}
	if !strings.Contains(result, "file") {
		t.Errorf("expected 'file' in result, got %q", result)
	}
	if !strings.Contains(result, "7") {
		t.Errorf("expected byte count '7' in result, got %q", result)
	}
}

func TestListFiles_WithSubdirectory(t *testing.T) {
	tmp := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmp, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := ListFiles(`{"path":"` + tmp + `"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "dir") {
		t.Errorf("expected 'dir' in result, got %q", result)
	}
	if !strings.Contains(result, "subdir") {
		t.Errorf("expected 'subdir' in result, got %q", result)
	}
}

func TestListFiles_NonExistentPath(t *testing.T) {
	_, err := ListFiles(`{"path":"/tmp/nonexistent_xmd_list_test_dir"}`)
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestListFiles_EmptyPath(t *testing.T) {
	_, err := ListFiles(`{"path":""}`)
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestListFiles_InvalidJSON(t *testing.T) {
	_, err := ListFiles(`not json`)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
