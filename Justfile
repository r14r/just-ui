set shell := ["sh", "-cu"]

default:
    just --list

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
