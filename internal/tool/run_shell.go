package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
)

const (
	maxOutputBytes = 2048
	halfOutput     = maxOutputBytes / 2
)

type runShellArgs struct {
	Command string `json:"command"`
}

// RunShell executes a shell command and returns its stdout (truncated).
// If debugOut is non-nil, subprocess stderr is forwarded there.
func RunShell(argsJSON string, debugOut io.Writer) (string, error) {
	var args runShellArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("run_shell: invalid arguments: %w", err)
	}
	if args.Command == "" {
		return "", fmt.Errorf("run_shell: command is required")
	}

	cmd := exec.Command("sh", "-c", args.Command)
	var stdout, stderr bytes.Buffer
	if debugOut != nil {
		pw := newPrefixWriter("> ", debugOut)
		cmd.Stdout = io.MultiWriter(&stdout, pw)
		cmd.Stderr = io.MultiWriter(&stderr, pw)
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	runErr := cmd.Run()

	exitCode := 0
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = 1
		}
	}

	if debugOut != nil {
		fmt.Fprintf(debugOut, "> [exit %d]\n", exitCode)
	}

	var out string
	if stdout.Len() > 0 {
		out = truncateOutput(stdout.Bytes())
	}
	if exitCode != 0 {
		if out != "" {
			out += "\n"
		}
		out += fmt.Sprintf("[exit %d]", exitCode)
	}
	if stderr.Len() > 0 {
		if out != "" {
			out += "\n"
		}
		out += fmt.Sprintf("[stderr]\n%s", stderr.String())
	}
	return out, nil
}

func truncateOutput(data []byte) string {
	if len(data) <= maxOutputBytes {
		return string(data)
	}
	head := data[:halfOutput]
	tail := data[len(data)-halfOutput:]
	omitted := len(data) - maxOutputBytes
	return fmt.Sprintf("%s\n... [truncated %d bytes] ...\n%s", string(head), omitted, string(tail))
}
