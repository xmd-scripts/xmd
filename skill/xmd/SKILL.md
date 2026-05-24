---
name: xmd
description: Use when the user wants to create, edit, run, or debug an xmd script (.md file with a shebang or xmd frontmatter). Also use when orchestrating multi-step LLM pipelines with xmd.
---

xmd runs markdown files as LLM prompts. A script is a plain `.md` file you `chmod +x` and run: variables are arguments, stdout is the result, exit codes signal success or failure.

## Install

```
curl -sSL https://raw.githubusercontent.com/xmd-scripts/xmd/main/install.sh | sh
```

## Format

```markdown
#!/usr/bin/env xmd
---
description: One-line summary of what the script does
vars:
  name: required
  style:
    default: concise
---
Write a bio for $NAME in a $STYLE style.
Output only the bio, no commentary.
```

## Variables

Declared in `vars`. Passed on the command line as `key=value`. Referenced in the body as `$NAME` (uppercase).

```
./bio.md name="Ada Lovelace" style=formal   # chmod +x bio.md first
```

Three declaration forms:

```yaml
vars:
  title: required          # must be passed on CLI
  format:
    default: markdown      # optional, falls back to default
  content:
    stdin: true            # read from stdin (pipe)
```

`stdin: true` enables Unix piping — at most one var per script:

```
cat report.txt | ./summarize.md title="Q1 Report"
./generate.md | ./review.md title="Draft"
```

## Tools

The model runs inside a sandbox restricted to `$PWD` and has access to file management (read, write, edit, list, search), shell execution, and a `question` tool for interactive prompts. Image and audio files are passed as multimodal input when read — no special handling needed in the script.

## Running

```
chmod +x script.md
./script.md key=value        # run
./script.md --debug ...      # print rendered prompt to stderr
./script.md --help           # show description and variables
```

## Composing scripts

Shell scripts are the natural orchestrator — capture output, pipe between scripts, branch on results:

```bash
idea=$(./ideate.md topic="$topic")
review=$(printf '%s' "$idea" | ./review.md title="Draft")
```

Each xmd invocation is an isolated process: fresh model context, its own tool loop, its own sandbox. Calling one script from another via `run_shell` or from a bash orchestrator gives you subagent semantics for free — the inner script has no memory of the outer one and evaluates its input independently.

Use `--context FILE` for the opposite: persistent multi-turn conversations where the model accumulates history across calls. A `role: system` script writes the persona; subsequent turns feed input via stdin with `./script.md --context FILE` or pipe directly: `echo "revise the intro" | ./chat.md --context session.jsonl`.

Use `include` in frontmatter to prepend shared fragments into the prompt:

```yaml
include:
  - ../common/tone.md
```

Includes are how knowledge is shared across scripts — the same idea as skills in agent frameworks. Since skills are markdown files, you can include them directly.

Full spec: [SPEC.md](https://github.com/xmd-scripts/xmd/blob/main/SPEC.md)
