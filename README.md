# just-ui

Version: **0.1.6**

Interactive **terminal** command chooser and runner for Justfiles. No HTTP server, browser, or frontend build. Built using Go and Bubble Tea/Bubbles.

## Requirements

- Go 1.22+ (dependencies downloaded on first bootstrap)
- [just](https://github.com/casey/just)
- Interactive terminal (macOS and Linux)

## Getting started

```sh
git pull --ff-only origin main
just bootstrap
just check
just build
just install
just-ui
```

The CLI locates a Justfile from the current directory upward. Use `--justfile path/to/justfile` to choose a specific one.

```sh
just-ui --justfile examples/justfile
just-ui --list
just-ui --version
```

### Installation

Run `just install` to execute the checks, build the binary and install it as `/usr/local/bin/just-ui`. Installation uses `sudo` only when the directory or existing executable is not writable. Verify with `just-ui --version` (make sure `/usr/local/bin` is on `PATH`). The `update-cli.yaml` install workflow invokes the same recipe.

### Two-column layout

The left pane takes **30%** of terminal width for searchable recipes; the right pane takes **70%** for parameter input, dropdowns, values, and the generated shell command. Select a recipe with Enter to configure it. Once all parameters are entered, the right pane displays the **exact POSIX-shell-quoted command** and requests confirmation before execution. The command is also printed above the live recipe output. Password fields are masked in displayed commands, while their actual values are still supplied to `just`.

### Navigation

| Key | Action |
| --- | --- |
| Up / Down, j / k | Browse recipes and dropdown values |
| / | Fuzzy search by recipe name, group, label, description |
| Enter | Select recipe, edit parameter, confirm execution |
| Esc | Return to previous parameter / recipe selection |
| e | Edit parameters again from the confirmation screen |
| q | Exit recipe list |
| Ctrl+C | Quit TUI (recipes get terminal signals directly when running) |

Recipe groups appear on each item. Output from the selected `just` recipe streams **directly to the terminal**. Recipes can prompt for input, including terminal-based programs. After the recipe finishes, its output remains on the terminal and just-ui displays `Press Enter to go back to just-ui`. Only Enter returns to the recipe chooser, including after failed commands. The original exit status is preserved.

### Just --choose compatibility

```sh
export JUST_CHOOSER="just-ui --chooser"
just --choose
```

When used as a chooser, just-ui reads newline-separated recipe names from standard input, displays a selectable list on the controlling terminal, and writes **only the chosen recipe name** to stdout. In this compatibility mode, `just` executes the recipe and handles its own parameters. In normal `just-ui` mode, parameters are entered in the terminal UI before execution.

### Rich parameters

Optional `@ui` comments provide information not available from Justfile recipe signatures:

```just
# @ui group Books
# @ui label Build PDF
# @ui description Build one book with a chosen layout
# @ui param book type=text label=Book_ID
# @ui param layout type=select options=advent|modern|classic default=advent
build-book book layout="advent":
    echo "Building {{book}} using {{layout}}"
```

The form supports `text`, `password`, `number`, `path`, `select` and `checkbox` input types, static dropdown options, defaults, and required-parameter validation. `@ui confirm` requires confirmation before running a recipe. Recipe group attributes, for example `[group('Build')]`, are also recognized. Place a recipe attribute **immediately before the recipe**, after any `@ui` comment annotations. Putting comments between a `[group(...)]` attribute and its recipe causes a `just` parser error.

**Limitations:** Simple root-level recipe signatures are parsed by just-ui for UI metadata. Imported Justfiles, modules, aliases, expression-valued defaults and variadic parameters require a more complete `just --dump --dump-format json` integration. Dynamic dropdown sources are not yet implemented. The `--chooser` adapter supports a single selected recipe.

## Development and releases

- Work directly on `main`, without feature branches or PRs.
- Maintain `VERSION` and `update-cli.yaml`.
- Increment the semantic-version **patch** component whenever implementation code changes.
- Run `just bootstrap`, `just check`, and `just build`.

`update-cli.yaml` uses schema v2 and runs `just bootstrap` before Go checks/builds.
