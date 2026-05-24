package script

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Script represents a parsed xmd file.
type Script struct {
	Path        string
	XMD         int
	Description string
	Role        string // "user" (default) or "system"
	Vars        map[string]VarDecl
	Include     []string
	Body        string
	HasShebang  bool
}

// VarDecl is a variable declaration: either required, has a default, or reads from stdin.
type VarDecl struct {
	Required bool
	Default  string
	Stdin    bool
}

// rawFrontmatter is used for YAML unmarshaling with strict field checking.
type rawFrontmatter struct {
	XMD         int                    `yaml:"xmd"`
	Description string                 `yaml:"description"`
	Role        string                 `yaml:"role"`
	Vars        map[string]interface{} `yaml:"vars"`
	Include     []string               `yaml:"include"`
}

// Parse reads and parses an xmd file from content.
func Parse(path string, content []byte) (*Script, error) {
	s := &Script{Path: path}
	text := string(content)

	// Strip shebang if present
	if strings.HasPrefix(text, "#!/") {
		nl := strings.Index(text, "\n")
		if nl < 0 {
			return nil, fmt.Errorf("xmd: script: file has only a shebang line")
		}
		shebangLine := text[:nl]
		if strings.Contains(shebangLine, "xmd") {
			s.HasShebang = true
		}
		text = text[nl+1:]
	}

	// Parse frontmatter if present
	hasFrontmatter := false
	if strings.HasPrefix(strings.TrimSpace(text), "---") {
		trimmed := strings.TrimLeft(text, " \t\n\r")
		if strings.HasPrefix(trimmed, "---") {
			rest := trimmed[3:]
			if strings.HasPrefix(rest, "\n") {
				rest = rest[1:]
			} else if strings.HasPrefix(rest, "\r\n") {
				rest = rest[2:]
			}
			closingIdx := strings.Index(rest, "\n---")
			if closingIdx < 0 {
				return nil, fmt.Errorf("xmd: frontmatter: unclosed frontmatter block")
			}
			fmContent := rest[:closingIdx]
			afterFM := rest[closingIdx+4:]
			if strings.HasPrefix(afterFM, "\n") {
				afterFM = afterFM[1:]
			} else if strings.HasPrefix(afterFM, "\r\n") {
				afterFM = afterFM[2:]
			}

			hasFrontmatter = true
			if err := s.parseFrontmatter(fmContent); err != nil {
				return nil, err
			}
			s.Body = afterFM
		}
	} else {
		s.Body = text
	}

	// Validate recognition
	if !s.HasShebang && (!hasFrontmatter || s.XMD == 0) {
		return nil, fmt.Errorf("xmd: script: neither shebang nor xmd frontmatter key present")
	}

	return s, nil
}

// ParseFragment parses a file intended for use as an include fragment.
// Unlike Parse, it does not require a shebang or xmd frontmatter key —
// a plain markdown file with no frontmatter is valid and its entire
// content becomes the body.
func ParseFragment(path string, content []byte) (*Script, error) {
	s := &Script{Path: path}
	text := string(content)

	// Strip shebang if present
	if strings.HasPrefix(text, "#!/") {
		nl := strings.Index(text, "\n")
		if nl < 0 {
			s.Body = ""
			return s, nil
		}
		if strings.Contains(text[:nl], "xmd") {
			s.HasShebang = true
		}
		text = text[nl+1:]
	}

	// Parse frontmatter if present
	if strings.HasPrefix(strings.TrimSpace(text), "---") {
		trimmed := strings.TrimLeft(text, " \t\n\r")
		if strings.HasPrefix(trimmed, "---") {
			rest := trimmed[3:]
			if strings.HasPrefix(rest, "\n") {
				rest = rest[1:]
			} else if strings.HasPrefix(rest, "\r\n") {
				rest = rest[2:]
			}
			closingIdx := strings.Index(rest, "\n---")
			if closingIdx < 0 {
				return nil, fmt.Errorf("xmd: frontmatter: unclosed frontmatter block")
			}
			fmContent := rest[:closingIdx]
			afterFM := rest[closingIdx+4:]
			if strings.HasPrefix(afterFM, "\n") {
				afterFM = afterFM[1:]
			} else if strings.HasPrefix(afterFM, "\r\n") {
				afterFM = afterFM[2:]
			}
			if err := s.parseFrontmatter(fmContent); err != nil {
				return nil, err
			}
			s.Body = afterFM
			return s, nil
		}
	}

	// No frontmatter — treat entire content as body
	s.Body = text
	s.Role = "user"
	return s, nil
}

func (s *Script) parseFrontmatter(content string) error {
	var rawMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &rawMap); err != nil {
		return fmt.Errorf("xmd: frontmatter: invalid YAML: %w", err)
	}

	allowedFields := map[string]bool{
		"xmd":         true,
		"description": true,
		"role":        true,
		"vars":        true,
		"include":     true,
	}
	for k := range rawMap {
		if !allowedFields[k] {
			return fmt.Errorf("xmd: frontmatter: unknown field '%s'", k)
		}
	}

	var raw rawFrontmatter
	if err := yaml.Unmarshal([]byte(content), &raw); err != nil {
		return fmt.Errorf("xmd: frontmatter: invalid YAML: %w", err)
	}

	s.XMD = raw.XMD
	s.Description = raw.Description
	s.Include = raw.Include

	switch raw.Role {
	case "", "user":
		s.Role = "user"
	case "system":
		s.Role = "system"
	default:
		return fmt.Errorf("xmd: frontmatter: invalid role %q: must be 'user' or 'system'", raw.Role)
	}

	if raw.Vars != nil {
		s.Vars = make(map[string]VarDecl)
		for k, v := range raw.Vars {
			switch val := v.(type) {
			case string:
				if val == "required" {
					s.Vars[k] = VarDecl{Required: true}
				} else {
					return fmt.Errorf("xmd: frontmatter: invalid var declaration for '%s': expected 'required' or map with 'default'", k)
				}
			case map[string]interface{}:
				for key := range val {
					if key != "default" && key != "stdin" {
						return fmt.Errorf("xmd: frontmatter: unknown key '%s' in var declaration for '%s'", key, k)
					}
				}
				_, hasDefault := val["default"]
				_, hasStdin := val["stdin"]
				if hasStdin && hasDefault {
					return fmt.Errorf("xmd: frontmatter: var '%s': 'stdin' and 'default' are mutually exclusive", k)
				}
				if hasStdin {
					if b, ok := val["stdin"].(bool); !ok || !b {
						return fmt.Errorf("xmd: frontmatter: var '%s': 'stdin' must be true", k)
					}
					s.Vars[k] = VarDecl{Stdin: true}
				} else if hasDefault {
					s.Vars[k] = VarDecl{Default: fmt.Sprintf("%v", val["default"])}
				} else {
					return fmt.Errorf("xmd: frontmatter: invalid var declaration for '%s': map must have 'default' or 'stdin' key", k)
				}
			default:
				return fmt.Errorf("xmd: frontmatter: invalid var declaration for '%s'", k)
			}
		}
	}

	return nil
}
