.PHONY: build install clean test lint vet vuln check fmt coverage release complexity

BINARY    = mcp-argo
MODULE    = github.com/jbcjorge/mcp-argo
LOG_LEVEL ?= info
VERSION  ?= $(shell \
	if git describe --tags --exact-match >/dev/null 2>&1; then \
		git describe --tags --exact-match; \
	else \
		echo "$$(git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)-dev-$$(date +%m%d%H%M)"; \
	fi)
LDFLAGS   = -s -w -X main.version=$(VERSION) -X main.defaultLogLevel=$(LOG_LEVEL)
INSTALL_PATH = $(HOME)/.local/bin/$(BINARY)

# Default target
all: check build

# Build the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

# Build with race detector (for testing)
build-race:
	go build -race -ldflags "$(LDFLAGS)" -o $(BINARY) .

# Install to ~/.local/bin
install: build
	@mkdir -p $(dir $(INSTALL_PATH))
	cp $(BINARY) $(INSTALL_PATH)
	@codesign -s - -f $(INSTALL_PATH) 2>/dev/null || true
	@echo "Installed $(INSTALL_PATH)"

# Run tests
test:
	go test -count=1 ./...

# Run tests verbose
test-v:
	go test -v -count=1 ./...

# Run tests with race detector
test-race:
	go test -race -count=1 ./...

# Run tests with coverage
coverage:
	@go test -coverprofile=coverage.out ./... | tail -1
	@go tool cover -func=coverage.out | tail -1
	@pct=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$NF}' | tr -d '%'); \
	if [ $$(echo "$$pct < 80" | bc) -eq 1 ]; then \
		echo "FAIL: coverage $$pct% is below 80% threshold"; exit 1; \
	else \
		echo "OK: coverage $$pct% meets 80% threshold"; \
	fi
	@rm -f coverage.out

# Format code
fmt:
	gofmt -s -w .

# Run go vet
vet:
	go vet ./...

# Run staticcheck
lint:
	@which staticcheck >/dev/null 2>&1 || (echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	staticcheck ./...

# Run govulncheck
vuln:
	@which govulncheck >/dev/null 2>&1 || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

# Check cyclomatic complexity (threshold: 15)
complexity:
	@which gocyclo >/dev/null 2>&1 || (echo "Installing gocyclo..." && go install github.com/fzipp/gocyclo/cmd/gocyclo@latest)
	gocyclo -over 15 -avg .

# Run all checks (fmt, vet, lint, vuln, complexity, test)
check: fmt vet lint vuln complexity test

# Clean build artifacts
clean:
	rm -f $(BINARY) coverage.out

# Build release binaries for multiple platforms
release: clean
	@mkdir -p dist
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)_darwin_arm64 .
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)_darwin_amd64 .
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)_linux_arm64 .
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)_linux_amd64 .
	@echo "Binaries in dist/"
	@ls -lh dist/
