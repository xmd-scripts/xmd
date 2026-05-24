package script

import (
	"strings"
	"testing"
)

func TestParse_Shebang(t *testing.T) {
	content := `#!/usr/bin/env xmd
---
description: Test script
vars:
  name: required
---
Hello $NAME
`
	s, err := Parse("/tmp/test.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.HasShebang {
		t.Error("expected HasShebang=true")
	}
	if s.Description != "Test script" {
		t.Errorf("expected description 'Test script', got %q", s.Description)
	}
	if _, ok := s.Vars["name"]; !ok {
		t.Error("expected var 'name'")
	}
	if !s.Vars["name"].Required {
		t.Error("expected var 'name' to be required")
	}
}

func TestParse_FrontmatterKey(t *testing.T) {
	content := `---
xmd: 1
description: Frontmatter form
---
Body text.
`
	s, err := Parse("/tmp/test.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.XMD != 1 {
		t.Errorf("expected XMD=1, got %d", s.XMD)
	}
	if s.Description != "Frontmatter form" {
		t.Errorf("unexpected description: %q", s.Description)
	}
}

func TestParse_NoRecognition(t *testing.T) {
	content := `---
description: No key
---
Body.
`
	_, err := Parse("/tmp/test.md", []byte(content))
	if err == nil {
		t.Error("expected error for missing shebang and xmd key")
	}
}

func TestParse_UnknownField(t *testing.T) {
	content := `---
xmd: 1
unknown_field: value
---
Body.
`
	_, err := Parse("/tmp/test.md", []byte(content))
	if err == nil {
		t.Error("expected error for unknown field")
	}
}

func TestParse_VarWithDefault(t *testing.T) {
	content := `#!/usr/bin/env xmd
---
vars:
  style:
    default: "terse"
---
Body.
`
	s, err := Parse("/tmp/test.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := s.Vars["style"]
	if !ok {
		t.Fatal("expected var 'style'")
	}
	if v.Required {
		t.Error("expected var 'style' to not be required")
	}
	if v.Default != "terse" {
		t.Errorf("expected default 'terse', got %q", v.Default)
	}
}

func TestParse_IncludeField(t *testing.T) {
	content := `#!/usr/bin/env xmd
---
include:
  - ../common/rules.md
  - ../common/format.md
---
Body.
`
	s, err := Parse("/tmp/test.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Include) != 2 {
		t.Errorf("expected 2 includes, got %d", len(s.Include))
	}
}

func TestParse_BodyExtraction(t *testing.T) {
	content := "#!/usr/bin/env xmd\n---\ndescription: Test\n---\nThe body text.\n"
	s, err := Parse("/tmp/test.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Body != "The body text.\n" {
		t.Errorf("unexpected body: %q", s.Body)
	}
}

func TestParse_RoleSystem(t *testing.T) {
	content := `---
xmd: 1
role: system
---
You are a helpful assistant.
`
	s, err := Parse("/tmp/test.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Role != "system" {
		t.Errorf("expected role 'system', got %q", s.Role)
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	content := "---\nxmd: 1\nbad: : yaml\n---\nBody.\n"
	_, err := Parse("/tmp/test.md", []byte(content))
	if err == nil {
		t.Error("expected error for invalid YAML frontmatter, got nil")
	}
}

func TestParseFragment_PlainMarkdown(t *testing.T) {
	content := "This is just a plain markdown body.\nNo frontmatter.\n"
	s, err := ParseFragment("/tmp/fragment.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Body != content {
		t.Errorf("expected body=%q, got %q", content, s.Body)
	}
	if len(s.Vars) != 0 {
		t.Errorf("expected empty Vars, got %v", s.Vars)
	}
}

func TestParseFragment_WithFrontmatterVars(t *testing.T) {
	content := `---
vars:
  name: required
---
Hello $NAME
`
	s, err := ParseFragment("/tmp/fragment.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := s.Vars["name"]; !ok {
		t.Error("expected var 'name' to be parsed")
	}
	if !s.Vars["name"].Required {
		t.Error("expected var 'name' to be required")
	}
	if !strings.Contains(s.Body, "Hello $NAME") {
		t.Errorf("expected body to contain 'Hello $NAME', got %q", s.Body)
	}
}

// Additional tests to cover missing branches in ParseFragment and parseFrontmatter

func TestParseFragment_ShebangNoNewline(t *testing.T) {
	// Shebang with no trailing newline → Body=""
	content := "#!/usr/bin/env xmd"
	s, err := ParseFragment("/tmp/frag.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Body != "" {
		t.Errorf("expected empty body, got %q", s.Body)
	}
}

func TestParseFragment_NonXmdShebang(t *testing.T) {
	// Shebang that does NOT contain "xmd" → HasShebang=false
	content := "#!/bin/bash\nBody text.\n"
	s, err := ParseFragment("/tmp/frag.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.HasShebang {
		t.Error("expected HasShebang=false for non-xmd shebang")
	}
	if s.Body != "Body text.\n" {
		t.Errorf("expected body='Body text.\\n', got %q", s.Body)
	}
}

func TestParseFragment_UnclosedFrontmatter(t *testing.T) {
	// Frontmatter with no closing ---
	content := "---\nvars:\n  name: required\n"
	_, err := ParseFragment("/tmp/frag.md", []byte(content))
	if err == nil {
		t.Fatal("expected error for unclosed frontmatter, got nil")
	}
	if !strings.Contains(err.Error(), "unclosed") {
		t.Errorf("expected 'unclosed' in error, got %q", err.Error())
	}
}

func TestParseFragment_FrontmatterCRLF(t *testing.T) {
	// Frontmatter with \r\n line endings after opening and closing ---
	content := "---\r\nxmd: 1\r\ndescription: crlf test\r\n---\r\nBody.\r\n"
	s, err := ParseFragment("/tmp/frag.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = s
}

func TestParseFragment_XmdShebangWithBody(t *testing.T) {
	// ParseFragment with xmd shebang → HasShebang=true (line 112-114)
	content := "#!/usr/bin/env xmd\nBody content here.\n"
	s, err := ParseFragment("/tmp/frag.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.HasShebang {
		t.Error("expected HasShebang=true for xmd shebang in fragment")
	}
	if s.Body != "Body content here.\n" {
		t.Errorf("expected body='Body content here.\\n', got %q", s.Body)
	}
}

func TestParseFragment_FrontmatterParseError(t *testing.T) {
	// ParseFragment with frontmatter that fails parseFrontmatter (line 139-141)
	content := "---\nunknown_field: value\n---\nBody.\n"
	_, err := ParseFragment("/tmp/frag.md", []byte(content))
	if err == nil {
		t.Fatal("expected error for unknown frontmatter field, got nil")
	}
}

func TestParse_ShebangNoNewline(t *testing.T) {
	// Parse with shebang but no newline → error (lines 46-48)
	content := "#!/usr/bin/env xmd"
	_, err := Parse("/tmp/test.md", []byte(content))
	if err == nil {
		t.Fatal("expected error for shebang with no newline, got nil")
	}
}

func TestParse_ShebangNoFrontmatterBody(t *testing.T) {
	// Parse with shebang but no frontmatter → body is text directly (line 85-87)
	content := "#!/usr/bin/env xmd\nHello body.\n"
	s, err := Parse("/tmp/test.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Body != "Hello body.\n" {
		t.Errorf("expected body='Hello body.\\n', got %q", s.Body)
	}
}

func TestParseFrontmatter_InvalidRole(t *testing.T) {
	// Invalid role value (line 186-187)
	content := "---\nxmd: 1\nrole: invalid\n---\nBody.\n"
	_, err := Parse("/tmp/test.md", []byte(content))
	if err == nil {
		t.Fatal("expected error for invalid role, got nil")
	}
	if !strings.Contains(err.Error(), "invalid role") {
		t.Errorf("expected 'invalid role' in error, got %q", err.Error())
	}
}

func TestParseFrontmatter_VarMapUnknownKey(t *testing.T) {
	// Var declared as map with unknown key (line 202-204)
	content := "#!/usr/bin/env xmd\n---\nvars:\n  name:\n    unknown_key: value\n---\nHello\n"
	_, err := Parse("/tmp/test.md", []byte(content))
	if err == nil {
		t.Fatal("expected error for unknown key in var map, got nil")
	}
	if !strings.Contains(err.Error(), "unknown key") {
		t.Errorf("expected 'unknown key' in error, got %q", err.Error())
	}
}

func TestParseFrontmatter_VarStdinAndDefaultMutuallyExclusive(t *testing.T) {
	// Var with both stdin and default (line 208-210)
	content := "#!/usr/bin/env xmd\n---\nvars:\n  name:\n    stdin: true\n    default: foo\n---\nHello\n"
	_, err := Parse("/tmp/test.md", []byte(content))
	if err == nil {
		t.Fatal("expected error for stdin+default, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in error, got %q", err.Error())
	}
}

func TestParseFrontmatter_VarStdinNotTrue(t *testing.T) {
	// Var with stdin: false (not valid, must be true) (lines 212-214)
	content := "#!/usr/bin/env xmd\n---\nvars:\n  name:\n    stdin: false\n---\nHello\n"
	_, err := Parse("/tmp/test.md", []byte(content))
	if err == nil {
		t.Fatal("expected error for stdin: false, got nil")
	}
	if !strings.Contains(err.Error(), "'stdin' must be true") {
		t.Errorf("expected 'stdin must be true' in error, got %q", err.Error())
	}
}

func TestParseFrontmatter_VarMapNoDefaultNoStdin(t *testing.T) {
	// Var as map with neither default nor stdin (lines 218-220)
	content := "#!/usr/bin/env xmd\n---\nvars:\n  name: {}\n---\nHello\n"
	_, err := Parse("/tmp/test.md", []byte(content))
	if err == nil {
		t.Fatal("expected error for empty var map, got nil")
	}
	if !strings.Contains(err.Error(), "map must have") {
		t.Errorf("expected 'map must have' in error, got %q", err.Error())
	}
}

func TestParseFrontmatter_VarNonStringNonMap(t *testing.T) {
	// Var declared as integer (non-string, non-map) (lines 221-222)
	content := "#!/usr/bin/env xmd\n---\nvars:\n  count: 42\n---\nHello\n"
	_, err := Parse("/tmp/test.md", []byte(content))
	if err == nil {
		t.Fatal("expected error for integer var declaration, got nil")
	}
}

func TestParseFrontmatter_StdinVar(t *testing.T) {
	// A var with stdin: true
	content := "#!/usr/bin/env xmd\n---\nvars:\n  prompt:\n    stdin: true\n---\nUse $PROMPT\n"
	s, err := Parse("/tmp/test.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := s.Vars["prompt"]
	if !ok {
		t.Fatal("expected var 'prompt'")
	}
	if !v.Stdin {
		t.Error("expected Stdin=true for 'prompt' var")
	}
}

func TestParseFrontmatter_InvalidVarDeclPlainString(t *testing.T) {
	// A var with a non-"required" string value is an error
	content := "#!/usr/bin/env xmd\n---\nvars:\n  name: \"some default\"\n---\nHello\n"
	_, err := Parse("/tmp/test.md", []byte(content))
	if err == nil {
		t.Fatal("expected error for invalid var declaration with plain non-required string, got nil")
	}
}

func TestParse_UnclosedFrontmatter(t *testing.T) {
	// Parse with unclosed frontmatter
	content := "#!/usr/bin/env xmd\n---\ndescription: test\n"
	_, err := Parse("/tmp/test.md", []byte(content))
	if err == nil {
		t.Fatal("expected error for unclosed frontmatter, got nil")
	}
	if !strings.Contains(err.Error(), "unclosed") {
		t.Errorf("expected 'unclosed' in error, got %q", err.Error())
	}
}

func TestParse_FrontmatterCRLF(t *testing.T) {
	// Parse with \r\n line endings
	content := "#!/usr/bin/env xmd\r\n---\r\nxmd: 1\r\ndescription: crlf\r\n---\r\nBody.\r\n"
	s, err := Parse("/tmp/test.md", []byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.XMD != 1 {
		t.Errorf("expected XMD=1, got %d", s.XMD)
	}
}
