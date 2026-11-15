# just-ui

Local web UI for executing recipes from a Justfile, implemented in Go without third-party dependencies.

## Setup

Requires Go 1.22+ and `just` on PATH.

```sh
go test ./...
go build -o just-ui .
./just-ui --justfile examples/justfile
```

Use `--no-browser` for headless operation. The HTTP server is bound to loopback.

## UI annotations

Place optional comments directly before a recipe:

```just
# @ui group Books
# @ui label Build PDF
# @ui description Build one book with a chosen layout
# @ui param book type=text label=Book_ID
# @ui param layout type=select options=advent|modern|classic default=advent
build-book book layout="advent":
    echo "Build {{book}} with {{layout}}"
```

Annotations: `@ui group`, `@ui label`, `@ui description`, `@ui confirm`, `@ui param NAME type=text|number|password|checkbox|select|path options=A|B default=X label=Text`.

## Current scope

Prototype v0.1 supports simple root-level recipes and static dropdowns. Modules, imports, aliases, expressions, variadic arguments, live streaming and cancellation are not yet supported. Syntax validation and execution are delegated to `just`. Output is limited to 1 MiB and processes time out after 30 minutes. The local HTTP service checks request origin and a session token. Do not expose it over a proxy.

For future releases, prefer `just --dump --dump-format json` for recipe discovery. Support `.just-ui.yaml` overlays for dynamic selects and other metadata.