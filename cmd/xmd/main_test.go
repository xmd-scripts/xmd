package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old
	return buf.String()
}

func TestRun_Version(t *testing.T) {
	var code int
	out := captureStdout(func() {
		code = run([]string{"--version"})
	})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if out == "" {
		t.Error("expected version output, got empty string")
	}
}

func TestRun_VersionShortFlag(t *testing.T) {
	var code int
	out := captureStdout(func() {
		code = run([]string{"-v"})
	})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if out == "" {
		t.Error("expected version output")
	}
}

func TestRun_VersionBeforeScript(t *testing.T) {
	var code int
	captureStdout(func() {
		code = run([]string{"--version", "somescript.md"})
	})
	if code != 0 {
		t.Errorf("expected exit code 0 for --version, got %d", code)
	}
}

func TestRun_UnknownFlag(t *testing.T) {
	code := run([]string{"--bad-flag"})
	if code != 2 {
		t.Errorf("expected exit code 2 for unknown flag, got %d", code)
	}
}

func TestRun_HelpNoScript(t *testing.T) {
	// --help with no script prints usage and exits 0.
	code := run([]string{"--help"})
	if code != 0 {
		t.Errorf("expected exit code 0 for --help with no script, got %d", code)
	}
}

func TestRun_HelpOnRealScript(t *testing.T) {
	// Create a minimal valid xmd script
	tmp := t.TempDir()
	scriptPath := filepath.Join(tmp, "test.md")
	content := `---
xmd: 1
description: A test script
vars:
  name: required
---
Hello $NAME
`
	if err := os.WriteFile(scriptPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var code int
	captureStdout(func() {
		code = run([]string{"--no-sandbox", "--help", scriptPath})
	})
	if code != 0 {
		t.Errorf("expected exit code 0 for --help, got %d", code)
	}
}
