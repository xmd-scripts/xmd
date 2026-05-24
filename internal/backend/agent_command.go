package backend

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xmd-scripts/xmd/internal/tool"
)

// AgentCommand is the agent_command backend.
// It runs XMD_AGENT_CMD as a bash one-liner via sh -c, with the prompt and
// system prompt written to temp files and exposed as env vars.
type AgentCommand struct {
	Cmd string
}

// NewAgentCommand creates a new agent_command backend.
func NewAgentCommand(cmd string) *AgentCommand {
	return &AgentCommand{Cmd: cmd}
}

// Run implements Backend. The role of the prompt determines which file receives
// the script content:
//   - role:user  → XMD_PROMPT_FILE; XMD_SYSTEM_PROMPT_FILE has the xmd framing only.
//   - role:system → XMD_SYSTEM_PROMPT_FILE gets the xmd framing plus the script
//     content; XMD_PROMPT_FILE is empty (this initialises the agent session).
//
// The agent CLI owns its own history. XMD_SESSION_ID derived from contextID chains
// runs together; xmd reads and writes no state of its own.
func (a *AgentCommand) Run(contextID string, p tool.Message, stdout, stderr io.Writer) error {
	if strings.TrimSpace(a.Cmd) == "" {
		return fmt.Errorf("xmd: backend: XMD_AGENT_CMD not set")
	}

	tmpDir, err := os.MkdirTemp("", "xmd-agent-*")
	if err != nil {
		return fmt.Errorf("xmd: agent: failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	content, _ := p.Content.(string)

	var promptContent, systemContent string
	if p.Role == "system" {
		systemContent = content
		promptContent = ""
	} else {
		promptContent = content
		systemContent = ""
	}

	promptFile := filepath.Join(tmpDir, "prompt")
	if err := os.WriteFile(promptFile, []byte(promptContent), 0o600); err != nil {
		return fmt.Errorf("xmd: agent: failed to write prompt file: %w", err)
	}

	systemFile := filepath.Join(tmpDir, "system-prompt")
	if err := os.WriteFile(systemFile, []byte(systemContent), 0o600); err != nil {
		return fmt.Errorf("xmd: agent: failed to write system prompt file: %w", err)
	}

	sessionID := ""
	if contextID != "" {
		h := sha256.Sum256([]byte(contextID))
		sessionID = fmt.Sprintf("%x", h[:8])
	}

	return a.run(promptFile, systemFile, sessionID, stdout, stderr)
}

func (a *AgentCommand) run(promptFile, systemFile, sessionID string, stdout, stderr io.Writer) error {
	cmd := exec.Command("sh", "-c", a.Cmd)
	cmd.Env = append(os.Environ(),
		"XMD_PROMPT_FILE="+promptFile,
		"XMD_SYSTEM_PROMPT_FILE="+systemFile,
		"XMD_SESSION_ID="+sessionID,
	)

	lbw := &lastByteWriter{w: stdout}
	var stderrBuf bytes.Buffer
	cmd.Stdout = lbw
	cmd.Stderr = io.MultiWriter(stderr, &stderrBuf)

	if err := cmd.Run(); err != nil {
		if stderrBuf.Len() > 0 {
			return fmt.Errorf("backend: agent failed: %s", stderrBuf.String())
		}
		return fmt.Errorf("backend: agent failed: %w", err)
	}

	if lbw.n > 0 && lbw.last != '\n' {
		fmt.Fprintln(stdout)
	}

	return nil
}

// lastByteWriter tracks the last byte written for trailing-newline detection.
type lastByteWriter struct {
	w    io.Writer
	last byte
	n    int64
}

func (lw *lastByteWriter) Write(p []byte) (int, error) {
	if len(p) > 0 {
		lw.last = p[len(p)-1]
		lw.n += int64(len(p))
	}
	return lw.w.Write(p)
}
