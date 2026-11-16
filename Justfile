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

# Build and install the CLI in /usr/local/bin (sudo only when needed).
install: build
    @if [ -w /usr/local/bin ] && { [ ! -e /usr/local/bin/just-ui ] || [ -w /usr/local/bin/just-ui ]; }; then install -m 755 dist/just-ui /usr/local/bin/just-ui; else command -v sudo >/dev/null || { echo "ERROR: sudo is required to install in /usr/local/bin" >&2; exit 1; }; sudo install -d /usr/local/bin && sudo install -m 755 dist/just-ui /usr/local/bin/just-ui; fi
    @/usr/local/bin/just-ui --version

run: bootstrap
    go run . --justfile examples/justfile

choose: bootstrap
    go run . --justfile examples/justfile

version:
    cat VERSION
