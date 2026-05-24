package prompt

import (
	"fmt"
	"sort"
	"strings"
)

// BuildVarsBlock renders the variables block prepended to the prompt.
// vars is a map of declared variable name -> bound value.
// If vars is empty, returns empty string (no block prepended).
func BuildVarsBlock(vars map[string]string, body string) string {
	if len(vars) == 0 {
		return body
	}

	var sb strings.Builder
	sb.WriteString("Variables:\n")

	// Sort keys for deterministic output
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var multiKeys []string
	for _, k := range keys {
		val := vars[k]
		if strings.Contains(val, "\n") {
			multiKeys = append(multiKeys, k)
			continue
		}
		upper := strings.ToUpper(k)
		fmt.Fprintf(&sb, "- $%s = %q\n", upper, val)
	}
	for _, k := range multiKeys {
		upper := strings.ToUpper(k)
		fmt.Fprintf(&sb, "- $%s =\n%s\n", upper, vars[k])
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString(body)
	return sb.String()
}
