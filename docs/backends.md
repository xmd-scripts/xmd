# Backends

xmd delegates model execution to a configurable backend. Set `XMD_BACKEND` (or `--backend`) to select one.

## openai_completion (default)

Calls any OpenAI-compatible chat completions endpoint. xmd runs its own tool loop.

```sh
XMD_BACKEND=openai_completion          # default
XMD_COMPLETION_URL=http://localhost:11434/v1/chat/completions
XMD_COMPLETION_MODEL=llama3            # auto-detected if omitted
XMD_COMPLETION_API_KEY=sk-...         # omit for local servers
```

Compatible servers: Ollama, llama-server, LM Studio, vLLM, OpenAI API, any OpenAI-compatible host.

```sh
# Ollama (default)
ollama serve
./script.md

# llama-server
llama-server -hf unsloth/gemma-4-26B-A4B-it-GGUF --port 11434
./script.md

# OpenAI
XMD_COMPLETION_URL=https://api.openai.com/v1/chat/completions \
XMD_COMPLETION_API_KEY=sk-... \
XMD_COMPLETION_MODEL=gpt-4o \
./script.md
```

## agent_command

Shells out to an external agent CLI via `sh -c`. The agent handles its own tool loop — xmd does not run one.

```sh
XMD_BACKEND=agent_command
XMD_AGENT_CMD='...'          # bash one-liner (see recipes below)
```

xmd sets the following environment variables before running `XMD_AGENT_CMD`:

| Variable | Description |
|---|---|
| `XMD_PROMPT_FILE` | Path to a temp file containing the user prompt |
| `XMD_SYSTEM_PROMPT_FILE` | Path to a temp file with the system message, or empty if none |
| `XMD_SESSION_ID` | Stable ID derived from `--context FILE`, or empty if no context |

The agent must write its response to stdout.

### Recipes

#### Claude Code

Claude Code stores credentials in `~/.claude`:

```sh
export XMD_BACKEND=agent_command
export XMD_AGENT_CMD='claude --print ${XMD_SESSION_ID:+--session-id "$XMD_SESSION_ID"} ${XMD_SYSTEM_PROMPT_FILE:+--system-prompt "$(cat "$XMD_SYSTEM_PROMPT_FILE")"} < "$XMD_PROMPT_FILE"'
./script.md --allow-read ~/.claude
```

#### Cursor

Cursor stores credentials in `~/.cursor`:

```sh
export XMD_BACKEND=agent_command
export XMD_AGENT_CMD='cat "$XMD_PROMPT_FILE" | cursor --headless --no-sandbox'
./script.md --allow-read ~/.cursor
```

#### Custom agent

Any program that reads a prompt and writes a response works:

```sh
export XMD_BACKEND=agent_command
export XMD_AGENT_CMD='cat "$XMD_PROMPT_FILE" | my-agent'
```

## Flags

`--backend NAME` overrides `XMD_BACKEND` for a single invocation:

```sh
./script.md --backend agent_command
./script.md --backend openai_completion
```
