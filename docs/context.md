# Context and multi-turn sessions

`--context FILE` enables persistent conversation state across multiple xmd invocations. The file is append-only JSONL — each turn adds a message record. This is how you build chat harnesses, accumulating-context loops, and programmatic agent sessions.

## System prompt

A script with `role: system` in its frontmatter writes a persona into the context file without making a model call:

```sh
./init-writer.md --context session.jsonl
```

`init-writer.md`:

```markdown
---
xmd: 1
role: system
---
You are a technical writer specializing in clear, concise documentation.
Respond only with revised text — no commentary, no explanations.
```

## Sending messages

Subsequent invocations with `--context FILE` append a user turn and run the model:

```sh
echo "Revise the introduction to lead with the benefit." \
  | xmd --context session.jsonl
```

Or pass a script body as the user turn:

```sh
./message.md --context session.jsonl target_section="Introduction"
```

Each turn sees the full accumulated history: system prompt, all prior user messages, all prior model responses.

## Shell orchestration

Because each turn is a plain shell command, you can drive multi-turn sessions from a script:

```bash
#!/usr/bin/env bash
CTX=$(mktemp /tmp/session-XXXXXX.jsonl)
trap 'rm -f "$CTX"' EXIT

./persona.md --context "$CTX"

for round in 1 2 3; do
  feedback=$(./critic.md draft_file=draft.md)
  echo "$feedback" | xmd --context "$CTX"
done

# Final pass
echo "Finalize." | xmd --context "$CTX"
```
