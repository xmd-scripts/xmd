# xmd specification

## Purpose

`xmd` is a runtime for markdown files containing natural-language task descriptions. It reads a script, prepends a declared variables block, concatenates any declared includes, and hands the result to a backend LLM, all while running inside a filesystem sandbox. It is glue between the shell and the LLM ecosystem, deliberately small.

## What xmd provides

xmd treats LLMs as unix scripts. A script is a markdown file you `chmod +x` and run: variables are arguments, stdout is the result, stderr is progress, exit codes signal success or failure. The model executes the logic; everything around it is fully specified, reproducible, and inspectable. xmd also manipulates LLM conversation state: context persistence enables multi-turn workflows, chat harnesses, and accumulating-context loops while keeping every turn fully inspectable.

- **The prompt** is fully inspectable. Use `--debug` to print it to stderr before it runs. No ambient context from loaded skills, no implicit behavior pulled in from elsewhere. What the model saw is what you wrote plus the variables you passed plus any files you explicitly composed in.
- **The script** is a file on disk. What you invoked is what runs.
- **The backend** is configurable. Any OpenAI-compatible server or agent CLI works — llama.cpp, Cursor, Codex, or similar.
- **The sandbox** is on by default. The filesystem is scoped to `$PWD`; everything outside — including the home directory — is inaccessible.

This is the inverse of skill-heavy agent ecosystems, which optimize for capability at the cost of reproducibility. xmd is for engineers and teams who want to use LLMs in pipelines but need to know, precisely and repeatably, what question the model is being asked and under what constraints.

## File format

An xmd script is a UTF-8 markdown file recognized as executable by xmd through either:

1. A shebang on the first line: `#!/usr/bin/env xmd`, which makes the file executable via `chmod +x script.md && ./script.md`, OR
2. A frontmatter key `xmd: 1` (the version number), which makes the file executable via `xmd script.md` without requiring a shebang.

Both mechanisms are supported because they serve different contexts. The shebang is the idiomatic Unix approach and works with `chmod +x`. The frontmatter key exists because some markdown editors (notably Obsidian) render the shebang line as a broken heading, and because third-party tools may want to recognize xmd files without relying on shebang parsing. A script can use either or both; xmd accepts any file that satisfies one of the two conditions.

The version number in `xmd: 1` reserves space for future format changes. The accepted value is `1`; future versions may accept higher numbers.

A minimal script using the shebang form:

```
#!/usr/bin/env xmd
---
description: Summarize a file with a title and three-sentence abstract
vars:
  file: required
---
Read the file at $FILE and produce a markdown summary with a title
and a three-sentence abstract.
```

A minimal script using the frontmatter form:

```
---
xmd: 1
description: Summarize a file with a title and three-sentence abstract
vars:
  file: required
---
Read the file at $FILE and produce a markdown summary with a title
and a three-sentence abstract.
```

### Frontmatter

Frontmatter is optional for shebang-form scripts and required for frontmatter-form scripts (they need at least the `xmd: 1` key to be recognized). When present, it is parsed as YAML. Frontmatter describes the script's interface: what variables it needs, what files it composes with, and what the script does. It does not configure execution (backend, sandbox rules, etc.); those are the runtime user's concern and live in environment variables and CLI flags.

A markdown file without a shebang or `xmd:` key is treated as a plain include fragment — a reusable body of text that other scripts can pull in via `include`. It may still have frontmatter (to declare variables it contributes), but the absence of a recognition marker means it is not directly executable by xmd.

Supported fields:

- `xmd` (integer, optional): the script format version. Required when no shebang is present. Accepted value: `1`.
- `description` (string, optional): one-line summary of what the script does. Shown in `--help` output.
- `role` (string, optional): `user` (default) or `system`. Determines which slot the rendered script occupies when used with `--context`. See the **Invocation** section.
- `vars` (map, optional): declares variables the script uses. Each entry is either `required` or a map with a `default` value.
- `include` (list, optional): file paths, relative to the current script, to concatenate into the prompt before the body. Order is preserved. When a script is used as an include fragment, its `role` field is ignored.

Unknown fields are an error.

### Variables

Variables are passed on the command line as positional `key=value` pairs:

```
./summarize.md file=report.txt
./summarize.md file=report.txt style=terse
```

Keys must match `[a-zA-Z_][a-zA-Z0-9_]*`. Values are strings. Quote values containing spaces: `file="my document.pdf"`.

Variables are referenced in the script body using the form `$NAME` (uppercase, preceded by a dollar sign). This syntax is unambiguous — it cannot be confused with regular prose, and it gives the model a clear signal about which words are placeholders and which are ordinary content.

Before sending the body to the backend, xmd prepends a variables block to the prompt when variables have been declared:

```
Variables:
- $FILE = "report.txt"
- $STYLE = "terse"

---

<body of the script>
```

Multi-line variable values (from stdin, see below) are placed last in the block without quotes, with the `---` separator acting as the terminator:

```
Variables:
- $TITLE = "NotionFlow"
- $DESCRIPTION =
This micro-SaaS allows users to transform their existing Notion workspaces...
second paragraph here

---

<body of the script>
```

Scripts with no declared variables receive no variables block.

Binding rules:

- Every `required` variable must be passed on the command line. Missing required variables are an xmd error (exit 2).
- Variables with a `default` use it if not passed.
- Variables declared `stdin: true` read their value from stdin (see below).
- Variables passed on the command line that aren't declared in frontmatter are an xmd error. This is strict by design: it catches typos (`flie=report.txt` does not silently become an unused variable).
- Scripts with no `vars` block accept no variables.

### Stdin variables

A variable can be declared with `stdin: true` to read its value from stdin rather than the command line:

```yaml
vars:
  description:
    stdin: true
```

This enables idiomatic Unix piping between xmd scripts and shell commands:

```
cat report.txt | ./summarize.md title="Q1 Report"
./generate.md | ./review.md title="Draft"
```

xmd reads stdin in full and injects the content as the variable's value. The content appears in the preamble's variables block, placed last so the `---` separator terminates it cleanly without quoting.

At most one variable per script may declare `stdin: true`. If a script file is provided as an argument and stdin is not piped, xmd exits with an error. If no script file is provided (stdin is the script body), `stdin: true` variables are not allowed.

`xmd --help SCRIPT` (or `./SCRIPT --help`) prints the description and declared variables, marking required versus defaulted.

### Composition via includes

The `include` field lists files to concatenate into the prompt before the body, in order:

```
---
description: Assess whether an article is substantive
vars:
  article_path: required
include:
  - ../common/output-contract.md
  - ../common/editorial-voice.md
---
Read the article at $ARTICLE_PATH. Assess whether it is substantive
(at least 3 paragraphs of actual prose, not a paywall stub). Output
PASS or FAIL on a single line.
```

Included files do not need a shebang or `xmd: 1` key — a plain markdown file with no xmd-specific markers is a valid include fragment. If frontmatter is present it is parsed normally (for `vars` and nested `include` directives); if absent, the entire file content becomes the fragment body. Each included file is concatenated before the body after stripping any shebang line and frontmatter block. Included files may themselves have frontmatter with `include` directives, which compose transitively. Variable declarations merge with the parent's; conflicts are an xmd error.

Include paths resolve relative to the file doing the including. A file reachable via multiple paths in the include graph is included only once (first occurrence wins, subsequent references are silently skipped). A true cycle — where a file transitively includes itself — is detected and reported as an error.

No size caps. The filesystem tells you how big files are and the backend tells you when a prompt exceeds its context window.

### Comments

Comments are supported in two forms:

**Frontmatter comments** use YAML's native `#` syntax and are handled by the YAML parser.

**Body comments** use HTML comment syntax (`<!-- ... -->`). xmd strips them from the body before sending it to the backend, so they cost zero tokens and do not influence model behavior. They render invisibly in every markdown editor.

```
Read the file at $FILE and produce a summary.

Output the summary as markdown with a title and three paragraphs.
```

The `--debug` output reflects the post-strip body — what the model actually saw.

### Extensions

Scripts are conventionally `.md`. They're real markdown and every markdown-aware editor renders them correctly. xmd does not enforce the extension.

## Invocation

```
./SCRIPT [key=value ...] [flags]         # shebang form (chmod +x)
xmd SCRIPT [key=value ...] [flags]       # frontmatter-form (xmd: 1 key, no shebang)
xmd --context FILE [flags]               # stdin-as-input continuation (no script)
```

Positional `key=value` pairs become variables. Flags beginning with `-` or `--` are xmd's own options. A positional argument without `=` is an error.

### Script execution

Depending on the script's `role` and whether `--context` is present:

- **`role: user` (default)** — the rendered script body becomes a user-role message. xmd calls the backend and streams the response to stdout. With `--context FILE`, existing messages are loaded first and new turns are appended after.
- **`role: system` with `--context FILE`** — the rendered body is written as a system-role entry to the context file. No backend call is made (there is no user message to respond to).
- **`role: system` without `--context`** — xmd warns that the system prompt will be discarded, validates the script (variable and include errors are still caught), and exits.

### Stdin-as-input continuation

`xmd --context FILE` without a script path reads stdin as a user-role message, appends it to the context, calls the backend, and appends the resulting turns to the context file. This is the loop-turn operation for chat harnesses.

### Context file format

The `--context FILE` format is JSONL, one message object per line, using the native backend message format. Full fidelity is preserved: roles, content blocks, tool-use/tool-result pairs with IDs, multimodal content. The context file is automatically added to the sandbox's writable paths.

Concurrent writers to the same file are the caller's concern.

### Flags

- `--context FILE` — persist conversation state to a JSONL file.
- `--backend NAME` — override `XMD_BACKEND`.
- `--debug` — dump xmd's rendered prompt to stderr before running.
- `--help` — print description, role, and variable list. Also available as `./SCRIPT --help`.
- `--no-sandbox` — disable sandboxing (loud warning on stderr).
- `--allow-read PATH` (repeatable) — add a read-only sandbox bind.
- `--allow-write PATH` (repeatable) — add a read-write sandbox bind.

## Backends

A backend is how xmd reaches an LLM. Two backend types are supported:

### `openai_completion` backend (default)

Any OpenAI-compatible completion endpoint. This includes local servers (Ollama, llama-server, LM Studio, vLLM), hosted APIs (OpenAI, Anthropic's OpenAI-compatible mode, Groq, Together), and anything else that speaks the OpenAI chat completions protocol with tool calling.

Configuration via environment variables:

- `XMD_COMPLETION_URL` — URL of the endpoint. Default: `http://localhost:11434/v1/chat/completions` (the conventional local port).
- `XMD_COMPLETION_MODEL` — model name to send. If unset, xmd queries the endpoint's models list and uses the first available model.
- `XMD_COMPLETION_API_KEY` — API key if the endpoint requires authentication. Optional for local endpoints.

xmd runs its own lightweight tool-calling loop against this backend, exposing the file tools, shell tool, and question tool described below. This is the default because it works against any compatible endpoint with zero configuration beyond having a local LLM server running on the default port.

### `agent_command` backend

Shells out to an external agent CLI via `sh -c`. The agent handles its own tool loop — xmd does not run one. Configuration:

- `XMD_AGENT_CMD` — a bash one-liner invoked via `sh -c`. xmd sets the following environment variables before running it:
  - `XMD_PROMPT_FILE` — path to a temp file containing the user prompt.
  - `XMD_SYSTEM_PROMPT_FILE` — path to a temp file with the system message (xmd's task-framing preamble plus any `role: system` script content), or empty if none.
  - `XMD_SESSION_ID` — stable ID derived from `--context FILE`, or empty if no context.

Agent backends run their own tool loops and do not use xmd's built-in tools. The agent's own tool vocabulary applies. See `docs/backends.md` for recipes covering Claude Code, Cursor, and custom agents.

### Backend selection

`XMD_BACKEND` environment variable or `--backend NAME` flag. Default: `openai_completion`. If this default's URL is unreachable, xmd fails with a clear message pointing at the environment variable.

## System prompt

xmd does not maintain a general-purpose system prompt. The system slot in a conversation is owned by the script author via `role: system` scripts. For `openai_completion` backends, tool declarations are sent in the API `tools` field — not as a system message — so the system slot remains available for the script's content.

For `agent_command` backends, xmd prepends a brief task-framing preamble (stdout/stderr contract, variables format, question tool) to the user prompt, since agent CLIs typically expect a combined input rather than structured message lists.

## Tools

For `openai_completion` backends, xmd runs a built-in tool loop exposing six tools. For `agent_command` backends, no xmd tools are exposed — the agent uses its own.

The tools:

- **`run_shell(command, timeout_sec)`** — execute a shell command inside the sandbox. stdout and stderr are captured and returned to the model. Output is truncated at 2KB with head-and-tail preservation: if total output exceeds 2KB, the model receives the first 1KB, a `... [truncated N bytes] ...` marker, and the last 1KB. Default timeout 30 seconds.
- **`read_file(path)`** — read a file. For text files, returns the contents as a string. **For image files (jpg, png, gif, webp) and audio files (mp3, wav, m4a, ogg), the file is attached as a multimodal part in the next user message so the model can see or hear it directly.** This is how vision scripts work against a vision-capable model: the model calls `read_file` on an image and receives the image as native input.
- **`write_file(path, content)`** — write text content to a file. Fails if outside the sandbox's writable region.
- **`edit_file(path, old, new)`** — exact-string replacement in an existing file. The `old` string must appear exactly once in the file; if zero or multiple matches, the call fails with a clear error. Simpler than write-after-read for small targeted changes.
- **`list_files(path)`** — non-recursive directory listing with names, types, and sizes.
- **`search_files(pattern, path)`** — content search across files, ripgrep-style, returning matching lines with file and line number.
- **`question(prompt)`** — ask the user a question interactively. xmd writes the prompt to `/dev/tty`, reads a line of input, and returns it to the model as the tool result. This is the mechanism for interactive scripts; agents handle their own interactivity but completion backends use this tool.

Tool calls follow the standard OpenAI-format protocol. There is no maximum turn limit: weak models that loop are handled via the model's own repetition penalties, and users can `ctrl-C` if a run goes rogue.

All tools run inside the sandbox — the sandbox wraps the entire xmd process, not just `run_shell`, so `read_file` and `write_file` respect the same filesystem boundaries as shell commands.

Intentionally omitted: network-specific tools beyond what shell provides.

## I/O contract

- **stdout**: the declared result of the script. The system prompt instructs the model to write only the final answer here. Output is streamed — text appears as it is generated rather than after the model finishes.
- **stderr**: the model's reasoning, tool call echoes, sandbox notices, and errors. For the `openai_completion` backend, xmd detects reasoning content and writes it to stderr as it streams, so the caller sees thinking separate from the final answer without any special handling in the script itself. For `agent_command` backends, whatever the agent writes to its own stderr is forwarded directly.
- **Exit code**: 0 success, 1 backend failure, 2 xmd failure (bad frontmatter, missing script, missing backend, invalid or missing variable, sandbox setup error).

### Streaming

xmd streams model output to stdout as tokens arrive. Both backend types stream:

- **`openai_completion`**: uses the OpenAI SSE streaming protocol (`stream: true`). Text deltas are written to stdout immediately. Tool call argument deltas are buffered internally and dispatched when the tool call is complete. Reasoning content is detected and written to stderr instead of stdout; two formats are supported: a `delta.reasoning_content` field (used by DeepSeek and some llama.cpp builds) and inline `<think>…</think>` tags (used by Gemma and others).
- **`agent_command`**: the subprocess stdout is piped directly to xmd's stdout, so whatever the agent CLI streams is forwarded without buffering.

Consumers that need the full output before acting (e.g. a script that greps the result) can pipe xmd into a buffer: `output=$(./script.md)`. Shell command substitution buffers all output and delivers it as a single string after xmd exits.

## Sandboxing

xmd runs its entire process (not just `run_shell`) inside a filesystem sandbox. This means all tool calls — `run_shell`, `read_file`, `write_file`, `edit_file`, `list_files`, `search_files` — inherit the same filesystem boundaries. The sandbox is on by default; use `--no-sandbox` to disable.

The model of both platforms is the same: read-write access to `$PWD` and a per-invocation temp directory; read-only access to the system paths needed to run commands; everything else — including the user's home directory — inaccessible. Network is allowed.

**Linux**: bubblewrap creates a minimal mount namespace. Only explicitly bound paths exist inside it. System directories (`/usr`, `/bin`, `/lib`, `/lib64`, `/etc`, present ones only) are bound read-only so shell commands work. `$PWD` and a per-invocation temp dir are bound read-write. `/proc` and `/dev` are provided via `--proc` and `--dev`. The home directory and all other paths are simply absent from the namespace. `--unshare-pid` and `--die-with-parent` isolate the process tree.

**macOS**: sandbox-exec with an `(allow default)` base policy (needed for exec and network), then file-write access denied globally and re-granted only for `$PWD`, the per-invocation temp dir, `/tmp`, `/private/tmp`, `/var/folders`, `/dev/null`, and `/dev/fd`. Read access to `/Users` and `/home` is also denied and re-granted only for `$PWD`, temp, and any `--allow-read` paths. The home directory is therefore unreadable and unwritable unless explicitly allowed.

**External dependencies:**

- Linux: `bubblewrap` package must be installed (`bwrap` binary on PATH). Check and error clearly if missing.
- macOS: `sandbox-exec` is part of the base system, no installation needed.
- Both: `rg` (ripgrep) for the `search_files` tool, typically present in `/usr/bin`.

**Flags that adjust the sandbox:**

- `--no-sandbox`: disable sandboxing entirely, loud warning.
- `--allow-read PATH`: add a read-only bind.
- `--allow-write PATH`: add a read-write bind.

**Enforcement notes:**

- `$PWD` is resolved once at startup. `cd ..` inside the backend does not expand the writable region.
- Child processes inherit the sandbox on both platforms.
- Nested xmd invocations (via `run_shell` or `agent_command`) do not re-apply a sandbox. xmd detects it is already running inside a sandbox (`XMD_SANDBOXED=1` env var) and skips sandbox setup, inheriting the parent's constraints. Applying a sub-policy from inside an existing sandbox is not reliably supported on either platform.

**Threat model, documented in README:** xmd's sandbox protects against prompt injection and model mistakes by limiting filesystem blast radius. It does not protect against reckless scripts, information disclosure through output, resource exhaustion, or determined attackers with kernel exploits.

## Example scripts

Examples are self-explanatory through their frontmatter `description` and prose body. No per-example READMEs.

- `hello.md` — greets a user by name. Takes a `NAME` variable. Minimal validation example for installations; runs in under five seconds against any backend.
- `summarize.md` — reads a file and produces a summary. Works on any backend.
- `ocr.md` — given an image, produces a title and description. Requires a vision-capable backend. Demonstrates multimodal `read_file`.
- `animal-guess.md` — interactive quiz where the model guesses an animal the user is thinking of by asking yes/no questions. Uses the question tool. The script prompts the model to ask sharp, trait-based questions (habitat, size, diet, behavior) rather than brute-forcing through a hardcoded decision tree — demonstrating how natural-language scripting leverages the model's knowledge instead of encoding it.
- `write.md` — demonstrates the subagent pattern with a real iteration loop. Takes a `TITLE` variable describing what to write about (defaulting to "how LLMs work, for kids"). The script writes two paragraphs on that title, then invokes `critic.md` on the result as a subprocess via `run_shell`, reads the critique, revises based on the feedback, and loops up to five iterations or until the critic returns `PASS`. This is the canonical example of xmd-in-xmd composition with real state (the draft on disk) flowing between the outer and inner processes. Each `critic.md` invocation is a separate process with its own fresh model context, which is what makes the critique genuinely independent — a critic that had just written the draft would defend its own choices rather than evaluate them honestly.
- `critic.md` — takes a `TITLE` describing what the piece was supposed to be about and a `FILE` path pointing at the draft. Outputs either `PASS` or `FAIL: <specific issues>` on the first line, with any additional feedback on subsequent lines. Evaluates the draft against the title (is it actually on-topic? does it match the audience the title implies?) and against general quality criteria (clarity, accuracy, flow). The strict output contract lets `write.md` parse it reliably, and the script is also useful standalone as a writing review tool.
- `rss-pipeline/` — multi-file pipeline showing bash orchestration plus xmd for judgment steps.
- `chat/` — multi-turn interactive chat with persistent memory across sessions, demonstrating the include-as-plugin pattern.
- `critic-writer/` — a critic drives a writer through multiple revision rounds using context persistence. `critic.md` uses `run_shell` to call two thin shell wrappers: `init-writer.sh` (initializes `writer.jsonl` from `writer.md`'s system prompt) and `message-writer.sh` (sends a message via stdin and returns the writer's response). Each `message-writer.sh` call continues the same conversation — the writer accumulates the full exchange and can be asked to revise, expand, or change direction across multiple turns. This is the multi-turn agent session pattern: one agent programmatically drives another the same way a human user drives Claude Code or Cursor, by sending a sequence of prompts into a persistent context rather than invoking a fresh subprocess per interaction. `term-chat.sh` runs in the terminal; `web-chat.sh` serves it over HTTP via ttyd. `chat.md` includes `use-memory.md` as a plugin: a fragment of instructions that tells the model to read `MEMORY.md` from the working directory at the start of each turn (recalling prior context) and update it via `write_file`/`edit_file` when it learns something worth keeping. The memory file lives in the user's working directory and is never included statically — the model reads and writes it as a tool call, so it works correctly even when the file doesn't exist yet.

The `critic-writer/` example demonstrates a second composition pattern that goes beyond subagents: **multi-turn agent sessions via `--context`**. The subagent pattern gives each nested invocation fresh context — appropriate when the inner agent needs an independent perspective (as in `write.md` + `critic.md`, where independence is the point). The multi-turn session pattern accumulates context across invocations — appropriate when the inner agent needs memory of prior exchanges to incorporate feedback, maintain a draft, or build toward a goal incrementally. The two patterns are complementary, and which to use depends on whether the inner agent should remember or forget.

The `write.md` + `critic.md` pair demonstrates the subagent pattern: **xmd gets subagent semantics for free from Unix process composition**, without needing to introduce subagents as a first-class concept. Each nested `xmd` invocation is a separate process with fresh model context, its own sandbox, and its own tool loop — which is exactly what a subagent should be. The outer script treats the inner script's stdout as data, the same way bash scripts treat the output of other commands. Context isolation and composition fall out of the process model rather than requiring a framework to implement them.

The `chat/` example demonstrates a second structural pattern: **includes as a plugin system**. `chat.md` includes `use-memory.md` — a reusable plugin fragment that tells the model to read `MEMORY.md` from the working directory at the start of each turn and update it when it learns something worth keeping across sessions. The two files are intentionally named differently: `use-memory.md` is the plugin (instructions, part of the repo), `MEMORY.md` is the data file (facts, gitignored, lives in the user's working directory). The pattern generalizes: any behavior you want to compose into a script — a tone, a constraint, a tool policy, a persona — can live in its own `.md` file and be listed under `include`. The include mechanism is xmd's plugin system — no special API, just file composition.

## Out of scope

- Windows native support (WSL only).
- Template language.
- Multi-script project scaffolder.
- Compilation of scripts into bash plus xmd calls.
- Package manager distribution.
- Plugin system for adapters.
- Per-script backend or model preferences.
- Parameterized includes.
- Conditional inclusion.
- Tool-call turn limit.
