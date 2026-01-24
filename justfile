# List available recipes
_default:
    @just --list

# Build the game
build:
    go build -o incell ./cmd/incell

# Build and run
run: build
    ./incell

# Build WASM version
wasm:
    GOOS=js GOARCH=wasm go build -o dist/incell.wasm ./cmd/incell
    cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" dist/

# Serve WASM version locally
serve: wasm
    cd dist && python3 -m http.server 8080

# Clean build artifacts
clean:
    rm -f incell incell-*
    rm -rf dist
    go clean

# Run go mod tidy
tidy:
    go mod tidy

# Run tests
test:
    go test ./...

# Format code
fmt:
    go fmt ./...
