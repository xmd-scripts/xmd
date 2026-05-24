# xmd

[![CI](https://github.com/xmd-scripts/xmd/actions/workflows/ci.yml/badge.svg)](https://github.com/xmd-scripts/xmd/actions/workflows/ci.yml)

**LLMs as unix scripts.**

xmd is a runtime for markdown files containing natural-language task descriptions. It reads a script, prepends declared variables, concatenates any declared includes, and hands the result to a backend LLM — all while running inside a filesystem sandbox.

Complex LLM-based systems are hard to build and harder to maintain. xmd makes them tractable by mapping agent concepts onto Unix primitives: scripts compose with pipes, subagents are processes, orchestration is shell. No framework to learn. The constructs you already know scale to multi-agent pipelines.

## Install

```sh
curl -sSL https://raw.githubusercontent.com/xmd-scripts/xmd/main/install.sh | sh
```

Or build from source:

```sh
git clone https://github.com/xmd-scripts/xmd
cd xmd
go build -o xmd ./cmd/xmd/
```

## Quick start

xmd connects to an OpenAI-compatible completion endpoint by default at `http://localhost:11434/v1/chat/completions`. Start any compatible local server ([Ollama](https://ollama.ai), llama-server, LM Studio, vLLM) or point it at a hosted API.

```sh
# Greet a user by name
./hello.md name=world

# Summarize a file
./summarize.md file=README.md

# Interactive animal guessing game
./animal-guess.md
```

Check out [xmd-examples](https://github.com/xmd-scripts/xmd-examples) for more complex examples: a Twitter thread author, a Turing tester, a software factory, a Taboo game player, and more.

## How it works

A script is a markdown file:

```markdown
#!/usr/bin/env xmd
---
description: Summarize a file
vars:
  file: required
---
Read the file at $FILE and produce a markdown summary with a title
and a three-sentence abstract.
```

Run it:

```sh
chmod +x summarize.md
./summarize.md file=report.pdf
```

xmd:
1. Parses the frontmatter (description, vars, includes)
2. Binds the supplied variables
3. Prepends the variables block to the prompt
4. Wraps the process in a filesystem sandbox
5. Sends the prompt to the backend LLM
6. Runs a tool-calling loop (read/write files, run shell commands, ask questions)
7. Writes the model's final answer to stdout

## What you get

- **Sandboxed by default.** The entire process runs inside a filesystem sandbox (bubblewrap on Linux, sandbox-exec on macOS). Writing outside `$PWD` is blocked. Use `--allow-read` / `--allow-write` to grant additional paths; use `--no-sandbox` to disable entirely.
- **No hidden prompt.** What you write in the markdown is what the model receives — no framework injecting hundreds of skill definitions behind the scenes. Use `--debug` to print the exact prompt to stderr before it runs.
- **Variables are strictly declared.** Passing an undeclared variable is an error. Missing a required one is an error. Typos are caught at startup.
- **Composable with bash.** Because xmd writes to stdout and exits with a code, it slots naturally into shell pipelines and scripts for deterministic orchestration around LLM steps.

## Configuration

| Variable | Default | Description |
|---|---|---|
| `XMD_BACKEND` | `openai_completion` | Backend: `openai_completion`, `agent_command` |
| `XMD_COMPLETION_URL` | `http://localhost:11434/v1/chat/completions` | OpenAI-compatible endpoint URL |
| `XMD_COMPLETION_MODEL` | (auto-detected) | Model name |
| `XMD_COMPLETION_API_KEY` | (none) | API key |
| `XMD_AGENT_CMD` | (none) | Agent command for `agent_command` backend (bash one-liner) |

## Agent backends

Point xmd at any agent CLI — Claude Code, Cursor, or your own:

```sh
export XMD_BACKEND=agent_command
export XMD_AGENT_CMD='claude --print ${XMD_SESSION_ID:+--session-id "$XMD_SESSION_ID"} ${XMD_SYSTEM_PROMPT_FILE:+--system-prompt "$(cat "$XMD_SYSTEM_PROMPT_FILE")"} < "$XMD_PROMPT_FILE"'
./summarize.md file=report.txt --allow-read ~/.claude
```

Agent backends run their own tool loops. xmd sets `XMD_PROMPT_FILE`, `XMD_SYSTEM_PROMPT_FILE`, and `XMD_SESSION_ID` and runs `XMD_AGENT_CMD` via `sh -c`. See [docs/backends.md](docs/backends.md) for recipes.

## Flags

```
--context FILE     Persist conversation state to a JSONL file
--backend NAME     Override XMD_BACKEND
--no-sandbox       Disable sandboxing (prints a warning)
--allow-read PATH  Add a read-only sandbox bind
--allow-write PATH Add a read-write sandbox bind
--debug            Print the rendered prompt to stderr before running
--help             Print description, role, and variable list
```

## Examples

See [xmd-examples](https://github.com/xmd-scripts/xmd-examples) for a full collection of examples.

## Design

xmd treats LLMs as unix scripts: a markdown file you `chmod +x` and run, with variables as arguments, stdout as the result, and exit codes for success or failure. The model executes the logic; the unix interface around it is fully specified, reproducible, and inspectable.

This is the inverse of skill-heavy agent ecosystems, which optimize for capability at the cost of reproducibility. xmd is for engineers and teams who want to use LLMs in pipelines but need to know, precisely and repeatably, what question the model is being asked and under what constraints.

**Variables without a template language.** Declared variables are prepended to the prompt as a plain list (`$NAME = "value"`). There's no obscure interpolation syntax to learn — the model reads the values the same way a human would and uses them throughout its response. Reusability comes from the model's comprehension, not a preprocessor.

**Subagent semantics come free from Unix process composition.** Each nested `xmd` invocation is a separate process with fresh model context, its own sandbox, and its own tool loop. The outer script treats the inner script's stdout as data. No framework needed. See [writer-critic](https://github.com/xmd-scripts/xmd-examples/tree/main/writer-critic).

**Multi-turn agent sessions via `--context`.** A script with `role: system` writes a persona into a JSONL context file. Subsequent `xmd --context FILE` calls (reading input from stdin) continue the conversation, accumulating history. This is how you drive a persistent agent programmatically — sending it a sequence of prompts over time, exactly like a user working with Claude Code or Cursor. The [critic-writer](https://github.com/xmd-scripts/xmd-examples/tree/main/critic-writer) example uses this: `init-writer.sh` sets up the writer's persona, and the critic sends multiple targeted revision prompts through `message-writer.sh`, each one building on the writer's full accumulated context.

**Includes are a plugin system.** Any behavior you want to compose into a script — a tone, a constraint, a tool policy, persistent memory — can live in its own `.md` file and be listed under `include`. The [chat](https://github.com/xmd-scripts/xmd-examples/tree/main/chat) example uses this to add persistent memory: `use-memory.md` is a plugin that tells the model to read and update `MEMORY.md` in the working directory, giving the chat persistent context across sessions without any framework support.

See [SPEC.md](SPEC.md) for the full design.

## License

MIT
