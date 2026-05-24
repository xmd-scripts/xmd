package prompt

import (
	"strings"
	"testing"
)

func TestBuildVarsBlock_NoVars(t *testing.T) {
	body := "Do the thing."
	result := BuildVarsBlock(nil, body)
	if result != body {
		t.Errorf("expected body unchanged, got %q", result)
	}
}

func TestBuildVarsBlock_WithVars(t *testing.T) {
	vars := map[string]string{
		"file":  "report.pdf",
		"style": "terse",
	}
	body := "Read the file at $FILE."
	result := BuildVarsBlock(vars, body)

	if !strings.Contains(result, "Variables:") {
		t.Error("missing 'Variables:' header")
	}
	if !strings.Contains(result, `- $FILE = "report.pdf"`) {
		t.Errorf("missing FILE var in result:\n%s", result)
	}
	if !strings.Contains(result, `- $STYLE = "terse"`) {
		t.Errorf("missing STYLE var in result:\n%s", result)
	}
	if !strings.Contains(result, "---") {
		t.Error("missing separator")
	}
	if !strings.Contains(result, body) {
		t.Error("body not included in result")
	}
}

func TestBuildVarsBlock_EmptyMap(t *testing.T) {
	body := "No vars here."
	result := BuildVarsBlock(map[string]string{}, body)
	if result != body {
		t.Errorf("empty map: expected body unchanged, got %q", result)
	}
}

func TestBuildVarsBlock_MultilineValue(t *testing.T) {
	vars := map[string]string{
		"content": "line1\nline2\nline3",
	}
	body := "Use the content."
	result := BuildVarsBlock(vars, body)

	if !strings.Contains(result, "- $CONTENT =\nline1\nline2\nline3") {
		t.Errorf("expected multiline format for CONTENT, got:\n%s", result)
	}
	if !strings.Contains(result, body) {
		t.Error("body not included in result")
	}
}

func TestBuildVarsBlock_MultilineAfterSingleLine(t *testing.T) {
	vars := map[string]string{
		"name":    "alice",
		"content": "first\nsecond",
	}
	body := "body"
	result := BuildVarsBlock(vars, body)

	// Single-line vars should appear before multiline vars
	singleIdx := strings.Index(result, `- $NAME = "alice"`)
	multiIdx := strings.Index(result, "- $CONTENT =\n")

	if singleIdx < 0 {
		t.Error("missing single-line NAME var")
	}
	if multiIdx < 0 {
		t.Error("missing multiline CONTENT var")
	}
	if singleIdx > multiIdx {
		t.Error("single-line var should appear before multiline var")
	}
}
