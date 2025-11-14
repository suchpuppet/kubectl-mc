.PHONY: build install test clean fmt vet

# Build the plugin
build:
	go build -o kubectl-mc .

# Install the plugin to GOPATH/bin
install:
	go install .

# Run tests
test:
	go test -v -race -cover ./...

# Run unit tests with coverage report
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run integration tests (requires cluster access)
test-integration:
	go test -v -race -tags=integration ./test/integration/...

# Format code
fmt:
	go fmt ./...

# Check if code is formatted
check-fmt:
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "The following files need formatting:"; \
		gofmt -s -l .; \
		exit 1; \
	fi

# Run go vet
vet:
	go vet ./...

# Run linters
lint:
	golangci-lint run

# Run all CI checks locally
ci: check-fmt vet test
	@echo "All CI checks passed!"

# Clean build artifacts
clean:
	rm -f kubectl-mc
	rm -f coverage.out coverage.html

# Tidy dependencies
tidy:
	go mod tidy

# Run the plugin locally (requires hub context)
run:
	go run . get pods -n test --hub-context kind-ocm-hub

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o dist/kubectl-mc-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o dist/kubectl-mc-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o dist/kubectl-mc-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o dist/kubectl-mc-windows-amd64.exe .
