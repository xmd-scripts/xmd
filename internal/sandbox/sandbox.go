package sandbox

// Sandbox is the interface for platform-specific sandboxing.
type Sandbox interface {
	Wrap(argv []string, pwd, temp string, allowRead, allowWrite []string) ([]string, error)
}
