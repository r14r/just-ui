set shell := ["sh", "-cu"]

default:
    just --list

# Validate dependencies required to build and execute just-ui.
bootstrap:
    @command -v go >/dev/null || { echo "ERROR: Go is not installed" >&2; exit 1; }
    @command -v just >/dev/null || { echo "ERROR: just is not installed" >&2; exit 1; }
    @go version
    @just --version

test:
    go test ./...

check:
    go vet ./...
    go test ./...

build: check
    mkdir -p dist
    go build -o dist/just-ui .

run:
    go run . --justfile examples/justfile

version:
    cat VERSION
