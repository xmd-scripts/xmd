# Sandbox

xmd runs its entire process inside a filesystem sandbox — not just `run_shell`, but all tool calls (`read_file`, `write_file`, `edit_file`, `list_files`, `search_files`). The sandbox is on by default.

## What is and isn't accessible

Linux and macOS enforce the same policy:

- **Read-write**: `$PWD` (frozen at startup) and a per-invocation temp directory.
- **Read-only**: system directories needed to run shell commands (`/usr`, `/bin`, `/lib`, `/etc`, and equivalents).
- **Inaccessible**: the home directory and everything else not explicitly listed above.
- **Network**: allowed.

`$PWD` is resolved once when xmd starts. Changing directory inside a script or shell command does not expand the writable region.

## Platform details

**Linux** uses bubblewrap (`bwrap`). A minimal mount namespace is created containing only the explicitly bound paths. Anything not bound simply does not exist inside the namespace. `bubblewrap` must be installed (`bwrap` on PATH).

**macOS** uses `sandbox-exec`, which is part of the base system. The policy uses `(allow default)` as a base (required for exec and network), then globally denies file writes and re-grants them for `$PWD`, temp, `/tmp`, `/private/tmp`, `/var/folders`, `/dev/null`, and `/dev/fd`. Reads from `/Users` and `/home` are denied and re-granted only for `$PWD`, temp, and any `--allow-read` paths.

## Flags

- `--no-sandbox`: disable sandboxing entirely. A warning is printed to stderr.
- `--allow-read PATH`: grant read access to an additional path.
- `--allow-write PATH`: grant read-write access to an additional path.

`--context FILE` automatically adds the context file to the sandbox's writable paths.

## Nested xmd invocations

When xmd is already running inside a sandbox, nested xmd invocations (via `run_shell` or as an `agent_command` sub-process) do not re-apply a sandbox. xmd detects the `XMD_SANDBOXED=1` environment variable set by the parent and skips sandbox setup, inheriting the parent's constraints. Applying a sub-policy from inside an existing sandbox is not reliably supported on either platform.

This means subagents and nested scripts are sandboxed — just by the outermost xmd's policy rather than their own.

## Threat model

The sandbox limits filesystem blast radius from prompt injection and model mistakes. It does not protect against:

- Scripts that deliberately use `--no-sandbox` or `--allow-write`
- Information disclosure through stdout/stderr output
- Resource exhaustion (CPU, memory, network)
- Kernel exploits
