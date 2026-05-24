# Includes

The `include` field lets you compose reusable fragments into a script's prompt. Includes are concatenated before the body, in order.

## Basic usage

```yaml
---
description: Assess whether an article is substantive
vars:
  article_path: required
include:
  - ../common/output-contract.md
  - ../common/editorial-voice.md
---
Read the article at $ARTICLE_PATH and assess whether it is substantive.
Output PASS or FAIL on a single line.
```

Paths are relative to the file doing the including.

## What counts as a valid include

An included file does not need a shebang or `xmd: 1` key. A plain markdown file with no xmd-specific markers is a valid fragment — its entire content becomes the included body. If frontmatter is present it is parsed for `vars` and nested `include` directives; if absent, the file is used as-is.

```markdown
<!-- output-contract.md — no frontmatter, no shebang, just prose -->
Respond with a single line containing only PASS or FAIL. No explanation.
```

When a file with a shebang or `xmd: 1` key is included, the shebang line and frontmatter block are stripped before concatenation — only the body is included.

## Transitive includes

Included files may themselves declare `include` directives, which compose transitively. The full graph is resolved before anything is sent to the backend.

```
report.md
  └── editorial-voice.md
        └── house-style.md
```

All three bodies are concatenated in depth-first order before `report.md`'s own body.

## Variable merging

Included files may declare variables. Those declarations merge into the including script's variable set. The merged set is what appears in the preamble and what must be satisfied at the command line.

```yaml
# house-style.md
vars:
  tone:
    default: formal
```

```yaml
# report.md
include:
  - editorial-voice.md  # which includes house-style.md
vars:
  file: required
```

Running `./report.md file=report.txt` produces a preamble with both `$FILE` and `$TONE`. The `tone` default applies unless overridden: `./report.md file=report.txt tone=casual`.

A variable declared in more than one file in the include graph with conflicting definitions is an error.

## Diamond includes

When a file is reachable via multiple paths in the include graph, it is included only once. The first occurrence wins; subsequent references are silently skipped.

```
report.md
  ├── section-a.md
  │     └── house-style.md   ← included here (first occurrence)
  └── section-b.md
        └── house-style.md   ← skipped silently
```

`house-style.md` appears exactly once in the final prompt, regardless of how many files include it.

## Circular includes

A true cycle — where a file transitively includes itself — is detected and reported as an error at startup:

```
report.md → section.md → report.md   ← error: circular include
```

xmd exits with code 2 before making any backend call.
