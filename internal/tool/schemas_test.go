package tool

import (
	"testing"
)

func TestAllTools_Count(t *testing.T) {
	tools := AllTools()
	if len(tools) != 7 {
		t.Errorf("expected 7 tools, got %d", len(tools))
	}
}

func TestAllTools_NonEmptyNamesAndDescriptions(t *testing.T) {
	tools := AllTools()
	for _, td := range tools {
		if td.Function.Name == "" {
			t.Errorf("tool has empty Name")
		}
		if td.Function.Description == "" {
			t.Errorf("tool %q has empty Description", td.Function.Name)
		}
	}
}

func TestAllTools_ExpectedNames(t *testing.T) {
	expected := map[string]bool{
		"run_shell":    false,
		"read_file":    false,
		"write_file":   false,
		"edit_file":    false,
		"list_files":   false,
		"search_files": false,
		"question":     false,
	}
	tools := AllTools()
	for _, td := range tools {
		if _, ok := expected[td.Function.Name]; !ok {
			t.Errorf("unexpected tool name: %q", td.Function.Name)
		}
		expected[td.Function.Name] = true
	}
	for name, found := range expected {
		if !found {
			t.Errorf("expected tool %q not found", name)
		}
	}
}
