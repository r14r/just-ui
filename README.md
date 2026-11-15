# just-ui

Version: **0.1.1**

Local browser UI for selecting, configuring, and executing recipes from a Justfile. Implemented in Go (standard library only).

## Requirements

- Go 1.22+
- [just](https://github.com/casey/just) on PATH

## Getting started

```sh
git clone https://github.com/r14r/just-ui.git
cd just-ui
just check
just build
./dist/just-ui --version
./dist/just-ui --justfile examples/justfile
```

Alternatively: `go run . --justfile examples/justfile`.

To use the UI with another project's Justfile:

```sh
./dist/just-ui --justfile /path/to/project/justfile
```

Use `--no-browser` for terminal-only startup, or `--listen 127.0.0.1:8080` for a fixed local port.

## Inline UI annotations

The UI reads ordinary Justfile recipe signatures plus optional `@ui` comments directly above each recipe:

```just
# @ui group Books
# @ui label Build PDF
# @ui description Build one book with the chosen layout
# @ui param book type=text label=Book_ID
# @ui param layout type=select options=advent|modern|classic default=advent
build-book book layout="advent":
    echo "Build {{book}} with {{layout}}"
```

| Annotation | Purpose |
| --- | --- |
| `@ui group NAME` | Group in sidebar |
| `@ui label TEXT` | Display label |
| `@ui description TEXT` | Help text |
| `@ui confirm` | Require confirmation before executing |
| `@ui param NAME type=TYPE` | Parameter input type |
| `options=a\|b\|c` | Static dropdown values |
| `default=VALUE` | Default value |
| `label=TEXT` | Parameter display label (use underscores for spaces) |

Valid input types: `text`, `number`, `password`, `checkbox`, `path`, `select`.

The prototype also recognizes Justfile `[group('Name')]` attributes. UI annotations may override group labels. Metadata cannot introduce arguments that are absent from the recipe signature.

## Versioning and repository workflow

The authoritative version lives in [VERSION](VERSION) and is compiled into the Go binary. Print it with `just-ui --version` or `just version`.

**Repository policy: all changes go directly to `main`. Whenever code changes, increment the patch number in VERSION (e.g. `0.1.1` → `0.1.2`) in the same change.** Do not create branches or PRs unless this policy is explicitly revised.

The [update-cli.yaml](update-cli.yaml) follows the `schemaVersion: 2` convention used in other Go projects, targets `main`, and defines `ci`, `build`, and `setup` workflows. `Justfile` recipes provide local check, build, run, and version commands.

## Current scope / limitations

Version 0.1.1 supports straightforward root-level recipes and static dropdown lists. The initial parser does not fully support imported Justfiles, expression-valued defaults, aliases or variadic arguments. Execution is delegated to `just` and captured after completion (not streamed). Jobs time out after 30 minutes; output is truncated at 1 MiB.

The server only binds to loopback and uses origin and session-token checks. Do **not** expose it through a reverse proxy.

For future versions, prefer `just --dump --dump-format json` as the authoritative recipe parser; use an optional `.just-ui.yaml` overlay for dynamic dropdowns and complex validation.
