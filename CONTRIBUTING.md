# Contributing to xmd

xmd is deliberately small. Its value is that the prompt sent to the model is exactly what you wrote — no ambient context, no magic, no framework. That simplicity is worth defending, and it shapes what contributions are welcome.

## What we love

- **Bug fixes** — especially cross-platform issues and edge cases in parsing or variable binding.
- **Documentation** — corrections, clarifications, examples. The spec and quickstart see the most traffic.
- **New examples** in [xmd-examples](https://github.com/xmd-scripts/xmd-examples) — self-contained `.md` scripts or shell-orchestrated pipelines demonstrating a pattern that isn't already covered.
- **Platform fixes** — sandbox support improvements, install script portability, path handling, etc.

If you're looking for something concrete, check [open issues](https://github.com/xmd-scripts/xmd/issues) for confirmed bugs.

## Before building a larger feature

If you're thinking about a non-trivial feature, open a GitHub discussion first. We'll tell you quickly whether it fits the project's direction — better to spend five minutes on alignment than to write code that won't be merged. The one-line test: does it add capability that can't be composed from what already exists?

## Sending a PR

- **One thing per PR.** A bugfix plus a refactoring plus a new feature is three PRs.
- **Write tests** for any new logic. Pure functions always; I/O paths via the `var`-hook pattern (see `question.go`).
- **Keep coverage above 95%** in `internal/` — run `make cover` before opening the PR.
- **`gofmt` your code.**
- **Describe the why**, not just the what. What broke, or what gap does this fill?

We review every PR. We're selective about scope — a small tool is worth keeping small.

## What won't get merged

- Features that add syntax or frontmatter fields to the script format without clear justification.
- New tool types that duplicate what `run_shell` already covers.
- Giant refactors. The codebase is small enough to read in an afternoon.
- Stylistic changes with no functional effect (code formatting, renaming for aesthetic reasons, etc.).
- Changes without tests.

## Codebase overview

xmd is a Go CLI. The main packages:

| Path | What it does |
|------|--------------|
| `cmd/xmd/` | CLI entry point, flag parsing, sandbox invocation |
| `internal/script/` | Parse `.md` files, extract frontmatter and variables |
| `internal/tool/` | Tool implementations (`read_file`, `write_file`, `run_shell`, `search_files`, `question`) and the tool loop |
| `internal/backend/` | Backend adapters: `anthropic` (default), `openai_completion`, `agent_command` |
| `internal/sandbox/` | Filesystem sandbox: `sandbox-exec` on macOS, `bubblewrap` on Linux |
| `internal/convo/` | Context file (`.jsonl`) read/write for `--context` |
| `internal/config/` | Config struct and environment variable resolution |
| `docs/` | Quickstart, author guide, backend reference |
| `skill/` | Claude Code skill file — drop in your project to get `/xmd` |

The tool loop in `internal/tool/loop.go` is the heart of the runtime: it calls the backend, dispatches tool calls, and loops until the model produces a final message.

## Building and testing

```sh
git clone https://github.com/xmd-scripts/xmd
cd xmd
go build -o bin/xmd ./cmd/xmd/
```

Go 1.22+ required. No other build dependencies.

```sh
# Unit tests
go test ./...

# Coverage (internal packages only, target ≥95%)
make cover

# Integration tests (uses canned responses, no live backend needed)
go test -tags=integration ./...

# End-to-end
XMD_BIN=./xmd bash test/e2e.sh
```

## CI

CI runs on every push and PR to `main`. Releases are managed by release-please: it opens a release PR when conventional commits accumulate on `main`, and triggers GoReleaser to build and publish binaries when that PR is merged.
