set shell := ["sh", "-cu"]

# @ui group General
# @ui description List all available Justfile recipes.
default:
    just --list

# @ui group Development
# @ui description Check the local Go and just toolchain and resolve module dependencies.
bootstrap:
    @command -v go >/dev/null || { echo "ERROR: Go is not installed" >&2; exit 1; }
    @command -v just >/dev/null || { echo "ERROR: just is not installed" >&2; exit 1; }
    @go version
    @just --version
    go mod tidy

# @ui group Development
# @ui description Run the complete Go unit test suite.
test: bootstrap
    go test ./...

# @ui group Development
# @ui description Run go vet and all Go unit tests.
check: bootstrap
    go vet ./...
    go test ./...

# @ui group Build
# @ui description Validate and compile dist/just-ui for the current platform.
build: check
    mkdir -p dist
    go build -o dist/just-ui .

# @ui group Installation
# @ui description Build and install just-ui into /usr/local/bin, using sudo if necessary.
install: build
    @if [ -w /usr/local/bin ] && { [ ! -e /usr/local/bin/just-ui ] || [ -w /usr/local/bin/just-ui ]; }; then install -m 755 dist/just-ui /usr/local/bin/just-ui; else command -v sudo >/dev/null || { echo "ERROR: sudo is required to install in /usr/local/bin" >&2; exit 1; }; sudo install -d /usr/local/bin && sudo install -m 755 dist/just-ui /usr/local/bin/just-ui; fi
    @/usr/local/bin/just-ui --version

# @ui group Development
# @ui description Start the example Justfile in the interactive terminal UI.
run: bootstrap
    go run . --justfile examples/justfile

# @ui group Development
# @ui description Open the interactive recipe chooser with the example Justfile.
choose: bootstrap
    go run . --justfile examples/justfile

# @ui group General
# @ui description Display the version from the VERSION file.
version:
    cat VERSION
