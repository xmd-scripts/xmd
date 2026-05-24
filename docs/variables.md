# Variables

Variables are how you pass arguments to an xmd script. They are declared in frontmatter and passed on the command line as `key=value` pairs.

## Declaring variables

```yaml
---
description: Summarize a file
vars:
  file: required
  style:
    default: detailed
---
Read the file at $FILE and produce a $STYLE summary.
```

Each entry under `vars` is either the literal string `required` or a map with a `default` key.

## Passing variables

```sh
./summarize.md file=report.txt
./summarize.md file=report.txt style=terse
```

Positional `key=value` pairs bind variables. Key names are case-insensitive on input (`file=` binds `$FILE`). Quote values that contain spaces:

```sh
./summarize.md file="my document.pdf"
```

Passing a variable not declared in frontmatter is an error. This is strict by design: `flie=report.txt` is caught immediately rather than silently ignored.

## Required variables

A variable declared `required` must be supplied on every invocation. Missing it is an exit 2 error.

```yaml
vars:
  file: required
  topic: required
```

## Optional variables with defaults

A variable with a `default` uses that value when not passed:

```yaml
vars:
  style:
    default: detailed
  language:
    default: English
```

Defaults can be overridden at the command line like any other variable.

## How variables reach the model

xmd does not substitute `$NAME` in the script body — there is no template engine. Instead, it prepends a variables block to the prompt:

```
Variables:
- $FILE = "report.txt"
- $STYLE = "terse"

---

Read the file at $FILE and produce a $STYLE summary.
```

The model reads the variable list at the top and uses the values throughout its response. `$NAME` in the body is a convention for the model, not a preprocessor directive. Scripts with no declared variables receive no variables block.

## Stdin variables

A variable can be declared with `stdin: true` to read its value from stdin:

```yaml
vars:
  content:
    stdin: true
  title: required
```

This enables Unix piping:

```sh
cat report.txt | ./summarize.md title="Q1 Report"
./generate.md | ./review.md title="Draft"
```

xmd reads stdin in full and injects it as the variable's value. Multi-line stdin values are placed last in the variables block without quotes, with the `---` separator acting as the terminator:

```
Variables:
- $TITLE = "Q1 Report"
- $CONTENT =
Full text of the report here...
second paragraph...

---

Produce a markdown summary of $CONTENT with "$TITLE" as the heading.
```

At most one variable per script may declare `stdin: true`. Declaring it while stdin is not piped is an error.

## Variables in included files

Included files may declare their own variables. Those declarations merge with the including script's `vars` block. A variable declared in an include and also declared in the parent (or another include) with conflicting definitions is an error.

```yaml
# tone.md (include fragment)
vars:
  tone:
    default: formal
```

```yaml
# report.md
include:
  - tone.md
vars:
  file: required
```

The merged variable set for `report.md` is `file` (required) and `tone` (default: formal). Both appear in the preamble when the script runs.

## Listing variables

`./SCRIPT --help` (or `xmd --help SCRIPT`) prints the description and each declared variable, marking required versus defaulted:

```
Description: Summarize a file

Variables:
  file     required
  style    default: detailed
```
