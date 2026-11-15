set shell := ["sh", "-cu"]

default:
    just --list

# Install/update dependencies and verify the Go and just toolchain.
bootstrap:
    @command -v go >/dev/null || { echo "ERROR: Go is not installed" >&2; exit 1; }
    @command -v just >/dev/null || { echo "ERROR: just is not installed" >&2; exit 1; }
    @go version
    @just --version
    go mod tidy

test: bootstrap
    go test ./...

check: bootstrap
    go vet ./...
    go test ./...

build: check
    mkdir -p dist
    go build -o dist/just-ui .

run: bootstrap
    go run . --justfile examples/justfile

choose: bootstrap
    go run . --justfile examples/justfile

version:
    cat VERSION
