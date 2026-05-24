//go:build linux

package sandbox

import (
	"fmt"
	"os"
	"os/exec"
)

// BwrapSandbox implements sandboxing on Linux using bubblewrap.
type BwrapSandbox struct{}

// New returns the platform sandbox implementation.
func New() (Sandbox, error) {
	return &BwrapSandbox{}, nil
}

// Wrap builds the bwrap command wrapping the given argv.
func (s *BwrapSandbox) Wrap(argv []string, pwd, temp string, allowRead, allowWrite []string) ([]string, error) {
	bwrapPath, err := exec.LookPath("bwrap")
	if err != nil {
		return nil, fmt.Errorf("xmd: sandbox: bubblewrap not found (install bubblewrap or use --no-sandbox)")
	}

	args := []string{bwrapPath}

	for _, dir := range []string{"/usr", "/bin", "/lib", "/lib64", "/etc", "/tmp"} {
		if _, err := os.Stat(dir); err == nil {
			args = append(args, "--ro-bind", dir, dir)
		}
	}

	args = append(args, "--bind", pwd, pwd)
	args = append(args, "--bind", temp, temp)

	for _, p := range allowRead {
		args = append(args, "--ro-bind", p, p)
	}

	for _, p := range allowWrite {
		args = append(args, "--bind", p, p)
	}

	args = append(args,
		"--chdir", pwd,
		"--unshare-pid",
		"--die-with-parent",
		"--proc", "/proc",
		"--dev", "/dev",
		"--",
	)

	args = append(args, argv...)
	return args, nil
}
