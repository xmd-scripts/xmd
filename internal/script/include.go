package script

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolvedScript is the fully resolved script with all includes merged.
type ResolvedScript struct {
	Description string
	Vars        map[string]VarDecl // merged from all includes
	Body        string             // concatenated includes + body (after comment stripping)
}

// Resolve resolves all includes transitively, merges var declarations,
// and concatenates the body (includes prepended before script body).
func Resolve(s *Script) (*ResolvedScript, error) {
	rs := &ResolvedScript{
		Description: s.Description,
		Vars:        make(map[string]VarDecl),
	}

	// Merge vars from top-level script
	for k, v := range s.Vars {
		rs.Vars[k] = v
	}

	visiting := map[string]bool{s.Path: true}
	visited := map[string]bool{}

	// Resolve includes
	includedBody, err := resolveIncludes(s, rs, visiting, visited)
	if err != nil {
		return nil, err
	}

	// Build final body: included content + script body
	body := StripComments(s.Body)
	if includedBody != "" {
		rs.Body = includedBody + "\n" + body
	} else {
		rs.Body = body
	}

	return rs, nil
}

// resolveIncludes processes include directives and returns concatenated included content.
// visiting tracks files currently on the call stack (cycle detection).
// visited tracks files already fully processed (deduplication).
func resolveIncludes(s *Script, rs *ResolvedScript, visiting, visited map[string]bool) (string, error) {
	if len(s.Include) == 0 {
		return "", nil
	}

	var parts []string
	dir := filepath.Dir(s.Path)

	for _, inc := range s.Include {
		incPath := filepath.Join(dir, inc)
		if !filepath.IsAbs(incPath) {
			incPath, _ = filepath.Abs(incPath)
		}

		if visiting[incPath] {
			return "", fmt.Errorf("xmd: include: circular include detected: %s (included from %s)", incPath, s.Path)
		}

		if visited[incPath] {
			continue
		}

		content, err := os.ReadFile(incPath)
		if err != nil {
			return "", fmt.Errorf("xmd: include: file not found: %s (included from %s)", incPath, s.Path)
		}

		incScript, err := ParseFragment(incPath, content)
		if err != nil {
			return "", fmt.Errorf("xmd: include: error parsing %s (included from %s): %w", incPath, s.Path, err)
		}

		// Merge vars, checking for conflicts
		for k, v := range incScript.Vars {
			if existing, exists := rs.Vars[k]; exists {
				if existing != v {
					return "", fmt.Errorf("xmd: vars: declaration conflict for '%s' between %s and %s", k, s.Path, incPath)
				}
			} else {
				rs.Vars[k] = v
			}
		}

		visiting[incPath] = true
		subIncluded, err := resolveIncludes(incScript, rs, visiting, visited)
		delete(visiting, incPath)
		if err != nil {
			return "", err
		}
		visited[incPath] = true

		body := StripComments(incScript.Body)
		if subIncluded != "" {
			parts = append(parts, subIncluded+"\n"+body)
		} else {
			parts = append(parts, body)
		}
	}

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "\n"
		}
		result += p
	}
	return result, nil
}
