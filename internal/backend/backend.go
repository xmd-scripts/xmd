package backend

import (
	"io"

	"github.com/xmd-scripts/xmd/internal/tool"
)

// Backend is the interface all backends implement.
//
// contextID is the conversation identifier passed via --context. Its meaning
// differs by backend:
//   - openai_completion: path to a JSONL file; the backend reads history from it
//     and appends new turns after each run.
//   - agent_command: a stable name used only to derive a deterministic session ID
//     passed as XMD_SESSION_ID; the agent manages its own history, and the file is
//     never read or written by xmd.
type Backend interface {
	Run(contextID string, prompt tool.Message, stdout, stderr io.Writer) error
}
