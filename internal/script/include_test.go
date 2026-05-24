package script

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeScript creates a temp .md file with the given content and returns its path.
func makeScript(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file %s: %v", p, err)
	}
	return p
}

func TestResolve_NoIncludes(t *testing.T) {
	dir := t.TempDir()
	p := makeScript(t, dir, "main.md", "#!/usr/bin/env xmd\n---\ndescription: test\n---\nHello world\n")
	s, err := Parse(p, mustReadFile(t, p))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	rs, err := Resolve(s)
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if !strings.Contains(rs.Body, "Hello world") {
		t.Errorf("expected body to contain 'Hello world', got %q", rs.Body)
	}
}

func TestResolve_WithInclude(t *testing.T) {
	dir := t.TempDir()
	incPath := makeScript(t, dir, "rules.md", "<!-- rule comment -->\nBe polite.\n")
	mainContent := "#!/usr/bin/env xmd\n---\ndescription: test\ninclude:\n  - rules.md\n---\nDo the task.\n"
	mainPath := makeScript(t, dir, "main.md", mainContent)

	s, err := Parse(mainPath, mustReadFile(t, mainPath))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	rs, err := Resolve(s)
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if !strings.Contains(rs.Body, "Be polite.") {
		t.Errorf("expected included body, got %q", rs.Body)
	}
	if !strings.Contains(rs.Body, "Do the task.") {
		t.Errorf("expected main body, got %q", rs.Body)
	}
	_ = incPath
}

func TestResolve_CircularInclude(t *testing.T) {
	dir := t.TempDir()
	// A includes B, B includes A
	aPath := filepath.Join(dir, "a.md")
	bPath := filepath.Join(dir, "b.md")
	os.WriteFile(aPath, []byte("#!/usr/bin/env xmd\n---\ndescription: a\ninclude:\n  - b.md\n---\nA body\n"), 0o644)
	os.WriteFile(bPath, []byte("---\ninclude:\n  - a.md\n---\nB body\n"), 0o644)

	s, err := Parse(aPath, mustReadFile(t, aPath))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = Resolve(s)
	if err == nil {
		t.Fatal("expected circular include error, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected 'circular' in error, got %q", err.Error())
	}
}

func TestResolve_MissingIncludeFile(t *testing.T) {
	dir := t.TempDir()
	mainContent := "#!/usr/bin/env xmd\n---\ndescription: test\ninclude:\n  - nonexistent.md\n---\nBody\n"
	mainPath := makeScript(t, dir, "main.md", mainContent)

	s, err := Parse(mainPath, mustReadFile(t, mainPath))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = Resolve(s)
	if err == nil {
		t.Fatal("expected error for missing include, got nil")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' in error, got %q", err.Error())
	}
}

func TestResolve_SameVarSameDeclaration(t *testing.T) {
	dir := t.TempDir()
	// Both main and included file declare the same var with the same decl
	incContent := "---\nvars:\n  style:\n    default: terse\n---\nRules.\n"
	mainContent := "#!/usr/bin/env xmd\n---\ndescription: test\nvars:\n  style:\n    default: terse\ninclude:\n  - rules.md\n---\nBody.\n"
	makeScript(t, dir, "rules.md", incContent)
	mainPath := makeScript(t, dir, "main.md", mainContent)

	s, err := Parse(mainPath, mustReadFile(t, mainPath))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = Resolve(s)
	if err != nil {
		t.Errorf("expected no conflict, got: %v", err)
	}
}

func TestResolve_ConflictingVarDeclarations(t *testing.T) {
	dir := t.TempDir()
	// Include declares style differently (required vs default)
	incContent := "---\nvars:\n  style: required\n---\nRules.\n"
	mainContent := "#!/usr/bin/env xmd\n---\ndescription: test\nvars:\n  style:\n    default: terse\ninclude:\n  - rules.md\n---\nBody.\n"
	makeScript(t, dir, "rules.md", incContent)
	mainPath := makeScript(t, dir, "main.md", mainContent)

	s, err := Parse(mainPath, mustReadFile(t, mainPath))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = Resolve(s)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "conflict") {
		t.Errorf("expected 'conflict' in error, got %q", err.Error())
	}
}

func TestResolve_NestedIncludes(t *testing.T) {
	dir := t.TempDir()
	// C is the deepest
	cContent := "C content.\n"
	// B includes C
	bContent := "---\ninclude:\n  - c.md\n---\nB content.\n"
	// A includes B
	mainContent := "#!/usr/bin/env xmd\n---\ndescription: test\ninclude:\n  - b.md\n---\nA content.\n"
	makeScript(t, dir, "c.md", cContent)
	makeScript(t, dir, "b.md", bContent)
	mainPath := makeScript(t, dir, "main.md", mainContent)

	s, err := Parse(mainPath, mustReadFile(t, mainPath))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	rs, err := Resolve(s)
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	// All bodies should appear
	if !strings.Contains(rs.Body, "C content.") {
		t.Errorf("expected C content in body, got %q", rs.Body)
	}
	if !strings.Contains(rs.Body, "B content.") {
		t.Errorf("expected B content in body, got %q", rs.Body)
	}
	if !strings.Contains(rs.Body, "A content.") {
		t.Errorf("expected A content in body, got %q", rs.Body)
	}

	// C should appear before B, B before A
	cIdx := strings.Index(rs.Body, "C content.")
	bIdx := strings.Index(rs.Body, "B content.")
	aIdx := strings.Index(rs.Body, "A content.")
	if cIdx >= bIdx || bIdx >= aIdx {
		t.Errorf("expected C < B < A order, indices: C=%d B=%d A=%d\nbody: %q", cIdx, bIdx, aIdx, rs.Body)
	}
}

func TestResolve_DuplicateInclude(t *testing.T) {
	// Same file included twice → should be deduplicated (visited branch)
	dir := t.TempDir()
	sharedContent := "Shared rules.\n"
	makeScript(t, dir, "shared.md", sharedContent)
	mainContent := "#!/usr/bin/env xmd\n---\ndescription: test\ninclude:\n  - shared.md\n  - shared.md\n---\nBody.\n"
	mainPath := makeScript(t, dir, "main.md", mainContent)

	s, err := Parse(mainPath, mustReadFile(t, mainPath))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	rs, err := Resolve(s)
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	// Shared content should appear only once
	count := strings.Count(rs.Body, "Shared rules.")
	if count != 1 {
		t.Errorf("expected 'Shared rules.' to appear once, got %d times; body=%q", count, rs.Body)
	}
}

func TestResolve_MultipleIncludes(t *testing.T) {
	// Multiple distinct includes → tests multi-part joining (lines 113-115)
	dir := t.TempDir()
	makeScript(t, dir, "part1.md", "Part 1 content.\n")
	makeScript(t, dir, "part2.md", "Part 2 content.\n")
	mainContent := "#!/usr/bin/env xmd\n---\ndescription: test\ninclude:\n  - part1.md\n  - part2.md\n---\nMain body.\n"
	mainPath := makeScript(t, dir, "main.md", mainContent)

	s, err := Parse(mainPath, mustReadFile(t, mainPath))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	rs, err := Resolve(s)
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if !strings.Contains(rs.Body, "Part 1 content.") {
		t.Errorf("expected Part 1 in body, got %q", rs.Body)
	}
	if !strings.Contains(rs.Body, "Part 2 content.") {
		t.Errorf("expected Part 2 in body, got %q", rs.Body)
	}
}

func TestResolve_IncludeIntroducesNewVar(t *testing.T) {
	// Include fragment declares a var that main script does not → covers lines 90-92
	dir := t.TempDir()
	incContent := "---\nvars:\n  style:\n    default: terse\n---\nRules.\n"
	makeScript(t, dir, "rules.md", incContent)
	// Main has no vars
	mainContent := "#!/usr/bin/env xmd\n---\ndescription: test\ninclude:\n  - rules.md\n---\nBody.\n"
	mainPath := makeScript(t, dir, "main.md", mainContent)

	s, err := Parse(mainPath, mustReadFile(t, mainPath))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	rs, err := Resolve(s)
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if _, ok := rs.Vars["style"]; !ok {
		t.Error("expected 'style' var from include, not found")
	}
}

func TestResolve_IncludeFragmentParseError(t *testing.T) {
	// Include a file whose frontmatter has a parse error → covers lines 80-82
	dir := t.TempDir()
	// A fragment with an unknown frontmatter field
	incContent := "---\nunknown_bad_field: value\n---\nBody.\n"
	makeScript(t, dir, "bad.md", incContent)
	mainContent := "#!/usr/bin/env xmd\n---\ndescription: test\ninclude:\n  - bad.md\n---\nMain body.\n"
	mainPath := makeScript(t, dir, "main.md", mainContent)

	s, err := Parse(mainPath, mustReadFile(t, mainPath))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = Resolve(s)
	if err == nil {
		t.Fatal("expected error for include with parse error, got nil")
	}
	if !strings.Contains(err.Error(), "error parsing") {
		t.Errorf("expected 'error parsing' in error, got %q", err.Error())
	}
}

func TestResolve_NestedIncludeError(t *testing.T) {
	// Nested include that itself has a missing include → error propagated (lines 90-92)
	dir := t.TempDir()
	// b.md includes a file that doesn't exist
	bContent := "---\ninclude:\n  - missing.md\n---\nB body.\n"
	makeScript(t, dir, "b.md", bContent)
	mainContent := "#!/usr/bin/env xmd\n---\ndescription: test\ninclude:\n  - b.md\n---\nMain body.\n"
	mainPath := makeScript(t, dir, "main.md", mainContent)

	s, err := Parse(mainPath, mustReadFile(t, mainPath))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = Resolve(s)
	if err == nil {
		t.Fatal("expected error for nested missing include, got nil")
	}
}


func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return data
}
