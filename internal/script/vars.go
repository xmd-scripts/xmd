package script

import (
	"fmt"
	"regexp"
	"strings"
)

var validVarName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ParseCLIVars parses key=value pairs from command-line arguments.
// Returns an error for invalid variable names or malformed pairs.
func ParseCLIVars(args []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, arg := range args {
		idx := strings.Index(arg, "=")
		if idx < 0 {
			return nil, fmt.Errorf("xmd: vars: argument '%s' is not a key=value pair", arg)
		}
		key := arg[:idx]
		value := arg[idx+1:]
		if !validVarName.MatchString(key) {
			return nil, fmt.Errorf("xmd: vars: invalid variable name '%s'", key)
		}
		result[key] = value
	}
	return result, nil
}

// BindVars validates and binds CLI-supplied vars against the declared vars.
// Returns the final map of variable name (lowercase key) -> value.
func BindVars(declared map[string]VarDecl, supplied map[string]string) (map[string]string, error) {
	result := make(map[string]string)

	// Check for undeclared supplied vars
	for k := range supplied {
		lk := strings.ToLower(k)
		found := false
		for dk := range declared {
			if strings.ToLower(dk) == lk {
				found = true
				break
			}
		}
		if !found {
			if len(declared) == 0 {
				return nil, fmt.Errorf("xmd: vars: undeclared variable '%s' passed on command line", k)
			}
			return nil, fmt.Errorf("xmd: vars: undeclared variable '%s' passed on command line", k)
		}
	}

	// Bind declared vars
	for name, decl := range declared {
		lname := strings.ToLower(name)
		// Check if supplied (case-insensitive match)
		var suppliedVal string
		var suppliedKey string
		for k, v := range supplied {
			if strings.ToLower(k) == lname {
				suppliedVal = v
				suppliedKey = k
				_ = suppliedKey
				break
			}
		}

		if suppliedVal != "" || hasKey(supplied, lname) {
			result[name] = suppliedVal
		} else if decl.Required {
			return nil, fmt.Errorf("xmd: vars: required variable '%s' not provided", name)
		} else {
			result[name] = decl.Default
		}
	}

	return result, nil
}

func hasKey(m map[string]string, lowerKey string) bool {
	for k := range m {
		if strings.ToLower(k) == lowerKey {
			return true
		}
	}
	return false
}
