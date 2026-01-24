# List available recipes
_default:
    @just --list

# Build the game
build:
    go build -o incell ./cmd/incell

# Build and run
run: build
    ./incell

# Clean build artifacts
clean:
    rm -f incell incell-*
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
