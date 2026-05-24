package sandbox

// NoSandbox is the --no-sandbox fallback.
type NoSandbox struct{}

// NewNoSandbox creates a sandbox that does nothing.
func NewNoSandbox() Sandbox {
	return &NoSandbox{}
}

// Wrap returns argv unchanged (no sandboxing).
func (s *NoSandbox) Wrap(argv []string, pwd, temp string, allowRead, allowWrite []string) ([]string, error) {
	return argv, nil
}
