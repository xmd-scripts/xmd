//go:build darwin

package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SandboxExecSandbox implements sandboxing on macOS using sandbox-exec.
type SandboxExecSandbox struct{}

// New returns the platform sandbox implementation.
func New() (Sandbox, error) {
	return &SandboxExecSandbox{}, nil
}

// Wrap builds the sandbox-exec command wrapping the given argv.
func (s *SandboxExecSandbox) Wrap(argv []string, pwd, temp string, allowRead, allowWrite []string) ([]string, error) {
	sandboxExecPath, err := exec.LookPath("sandbox-exec")
	if err != nil {
		return nil, fmt.Errorf("xmd: sandbox: sandbox-exec not found")
	}

	policy := buildPolicy(pwd, temp, allowRead, allowWrite)

	policyFile, err := os.CreateTemp(temp, "xmd-sandbox-*.sb")
	if err != nil {
		return nil, fmt.Errorf("xmd: sandbox: policy setup failed: %w", err)
	}
	defer policyFile.Close()

	if _, err := policyFile.WriteString(policy); err != nil {
		return nil, fmt.Errorf("xmd: sandbox: policy setup failed: %w", err)
	}

	args := []string{sandboxExecPath, "-f", policyFile.Name(), "--"}
	args = append(args, argv...)
	return args, nil
}

func buildPolicy(pwd, temp string, allowRead, allowWrite []string) string {
	var sb strings.Builder
	// macOS sandbox-exec works best as allow-default + selective deny.
	// Attempting deny-default blocks too many implicit kernel operations.
	sb.WriteString("(version 1)\n")
	sb.WriteString("(allow default)\n")
	// Deny writes globally, then re-open specific paths.
	sb.WriteString("(deny file-write* (subpath \"/\"))\n")
	// Allow /dev/null and /dev/fd/* so shell redirections (2>/dev/null, process
	// substitution) work. /dev/fd entries are aliases for the process's own fds.
	sb.WriteString("(allow file-write* (literal \"/dev/null\"))\n")
	sb.WriteString("(allow file-read* (literal \"/dev/null\"))\n")
	sb.WriteString("(allow file-write* (subpath \"/dev/fd\"))\n")
	sb.WriteString("(allow file-read* (subpath \"/dev/fd\"))\n")
	for _, p := range []string{pwd, temp} {
		fmt.Fprintf(&sb, "(allow file-write* (subpath %q))\n", p)
	}
	// Allow the real path of the temp dir (macOS /var -> /private/var symlink).
	if real, err := filepath.EvalSymlinks(temp); err == nil && real != temp {
		fmt.Fprintf(&sb, "(allow file-write* (subpath %q))\n", real)
	}
	// Allow nested xmd processes to create their own temp dirs.
	// They will fail at sandbox_apply and fall back to running within this sandbox.
	sb.WriteString("(allow file-write* (subpath \"/tmp\"))\n")
	sb.WriteString("(allow file-write* (subpath \"/private/tmp\"))\n")
	sb.WriteString("(allow file-write* (subpath \"/var/folders\"))\n")
	sb.WriteString("(allow file-write* (subpath \"/private/var/folders\"))\n")
	for _, p := range allowWrite {
		fmt.Fprintf(&sb, "(allow file-write* (subpath %q))\n", p)
	}
	// Deny reads to user home directories — the most sensitive area.
	// System paths remain readable via allow-default.
	sb.WriteString("(deny file-read* (subpath \"/Users\"))\n")
	sb.WriteString("(deny file-read* (subpath \"/home\"))\n")
	// Re-allow specific user paths the process legitimately needs.
	for _, p := range append(allowRead, pwd, temp) {
		fmt.Fprintf(&sb, "(allow file-read* (subpath %q))\n", p)
	}
	return sb.String()
}
