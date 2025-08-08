# Makefile for Azure OIDC Action

.PHONY: build clean test help install deps

# Default target
all: build

# Build the binary
build:
	go build -o azure-oidc-action .

# Clean build artifacts
clean:
	rm -f azure-oidc-action

# Download and tidy dependencies
deps:
	go mod download
	go mod tidy

# Run tests
test:
	go test -v ./...

# Install the binary to GOPATH/bin
install:
	go install .

# Run the tool with help flag
help: build
	./azure-oidc-action --help

# Build for multiple platforms
build-all: clean
	GOOS=linux GOARCH=amd64 go build -o azure-oidc-action-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o azure-oidc-action-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build -o azure-oidc-action-windows-amd64.exe .
	GOOS=darwin GOARCH=amd64 go build -o azure-oidc-action-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o azure-oidc-action-darwin-arm64 .
