package tool

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

func TestQuestion_InvalidJSON(t *testing.T) {
	_, err := Question("not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestQuestion_EmptyPrompt(t *testing.T) {
	_, err := Question(`{"prompt":""}`)
	if err == nil {
		t.Error("expected error for empty prompt")
	}
}

// TestQuestion_TTYSuccess tests the /dev/tty happy path via an injected pipe.
func TestQuestion_TTYSuccess(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString("tty answer\n"); err != nil {
		t.Fatal(err)
	}
	w.Close()

	orig := openTTY
	openTTY = func() (*os.File, error) { return r, nil }
	defer func() {
		openTTY = orig
		r.Close()
	}()

	answer, err := Question(`{"prompt":"hello?"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer != "tty answer" {
		t.Errorf("expected %q, got %q", "tty answer", answer)
	}
}

// TestQuestion_TTYEOFError tests the scanner.Scan() → false branch (EOF with no data).
func TestQuestion_TTYEOFError(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w.Close() // immediate EOF

	orig := openTTY
	openTTY = func() (*os.File, error) { return r, nil }
	defer func() {
		openTTY = orig
		r.Close()
	}()

	_, err = Question(`{"prompt":"hello?"}`)
	if err == nil {
		t.Fatal("expected error on EOF tty read, got nil")
	}
	if err.Error() != "question: failed to read input from tty" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestQuestion_TTYOpenError tests that openTTY failure falls back to stderr+stdin.
// (Covered by StdinFallback below when /dev/tty is unavailable in CI.)
// This test exercises the same path with an explicit injected failure.
func TestQuestion_TTYOpenError(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString("fallback line\n"); err != nil {
		t.Fatal(err)
	}
	w.Close()

	orig := openTTY
	openTTY = func() (*os.File, error) { return nil, fmt.Errorf("no tty") }
	defer func() { openTTY = orig }()

	// stdinReaderOnce already fired (from StdinFallback test or naturally), so
	// we wire a fresh reader directly to avoid the once-per-process limitation.
	origReader := stdinReader
	origOnce := stdinReaderOnce
	stdinReader = nil
	stdinReaderOnce = &sync.Once{}
	defer func() {
		stdinReader = origReader
		stdinReaderOnce = origOnce
		r.Close()
	}()

	old := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = old }()

	answer, err := Question(`{"prompt":"anything?"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer != "fallback line" {
		t.Errorf("expected %q, got %q", "fallback line", answer)
	}
}

// TestQuestion_StdinFallback tests the /dev/tty fallback path.
// In test environments /dev/tty is not a terminal so Question falls back to stdinLine().
// stdinReaderOnce means this can only fire once per process — kept as a single test.
func TestQuestion_StdinFallback(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString("hello from stdin\n"); err != nil {
		t.Fatal(err)
	}
	w.Close()

	old := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = old
		r.Close()
	}()

	answer, err := Question(`{"prompt":"what is your name?"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer != "hello from stdin" {
		t.Errorf("expected %q, got %q", "hello from stdin", answer)
	}
}
