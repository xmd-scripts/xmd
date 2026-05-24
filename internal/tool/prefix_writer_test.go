package tool

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPrefixWriter_SingleLine(t *testing.T) {
	var buf bytes.Buffer
	pw := newPrefixWriter("> ", &buf)
	pw.Write([]byte("hello\n"))
	if buf.String() != "> hello\n" {
		t.Errorf("expected '> hello\\n', got %q", buf.String())
	}
}

func TestPrefixWriter_TwoLinesOneCall(t *testing.T) {
	var buf bytes.Buffer
	pw := newPrefixWriter("> ", &buf)
	pw.Write([]byte("line1\nline2\n"))
	if buf.String() != "> line1\n> line2\n" {
		t.Errorf("expected '>line1\\n> line2\\n', got %q", buf.String())
	}
}

func TestPrefixWriter_NoNewline_Buffered(t *testing.T) {
	var buf bytes.Buffer
	pw := newPrefixWriter("> ", &buf)
	pw.Write([]byte("no newline here"))
	// Data without newline should be buffered, not flushed
	if buf.Len() != 0 {
		t.Errorf("expected nothing written to output (buffered), got %q", buf.String())
	}
}

func TestPrefixWriter_WriteError(t *testing.T) {
	// Use an errWriter that always returns an error to trigger the p.w.Write error branch
	ew := &errWriter{}
	pw := newPrefixWriter("> ", ew)
	n, err := pw.Write([]byte("hello\n"))
	if err == nil {
		t.Error("expected error from underlying writer, got nil")
	}
	if n != 0 {
		t.Errorf("expected n=0 on error, got %d", n)
	}
}

// errWriter always returns an error on Write.
type errWriter struct{}

func (e *errWriter) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("write error")
}

func TestPrefixWriter_SplitAcrossMultipleCalls(t *testing.T) {
	var buf bytes.Buffer
	pw := newPrefixWriter("> ", &buf)
	pw.Write([]byte("hel"))
	pw.Write([]byte("lo\n"))
	if buf.String() != "> hello\n" {
		t.Errorf("expected '>hello\\n', got %q", buf.String())
	}
}
