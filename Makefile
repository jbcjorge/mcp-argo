.PHONY: build install clean test lint vet vuln check fmt coverage release complexity tools shadow gosec gitleaks

BINARY    = mcp-argo
MODULE    = github.com/jbcjorge/mcp-argo
LOG_LEVEL ?= info
REPORTS_DIR = reports
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

## tools: Install development tools
tools:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install gotest.tools/gotestsum@latest
	go install github.com/boumenot/gocover-cobertura@latest

## build: Compile binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## build-race: Build with race detector
build-race:
	go build -race -ldflags "$(LDFLAGS)" -o $(BINARY) .

## install: Build, codesign (macOS), and install to ~/.local/bin
install: build
	@mkdir -p $(dir $(INSTALL_PATH))
	cp $(BINARY) $(INSTALL_PATH)
	@codesign -s - -f $(INSTALL_PATH) 2>/dev/null || true
	@echo "Installed $(INSTALL_PATH)"

## fmt: Format code
fmt:
	gofmt -s -w .

## vet: Run go vet
vet:
	go vet ./...

## shadow: Check for variable shadowing
shadow:
	@which shadow >/dev/null 2>&1 || go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest
	go vet -vettool=$$(go env GOPATH)/bin/shadow ./...

## lint: Run staticcheck
lint:
	@which staticcheck >/dev/null 2>&1 || go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...

## vuln: Run govulncheck
vuln:
	@which govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

## gosec: Security-focused static analysis
gosec:
	@which gosec >/dev/null 2>&1 || go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -quiet ./...

## gitleaks: Scan for secrets
gitleaks:
	@which gitleaks >/dev/null 2>&1 || (echo "gitleaks not installed, skipping" && exit 0)
	gitleaks detect --no-git -v

## complexity: Check cyclomatic complexity (threshold: 15)
complexity:
	@which gocyclo >/dev/null 2>&1 || go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	@output=$$(gocyclo -over 15 .); \
	if [ -n "$$output" ]; then \
		echo "Functions with cyclomatic complexity over 15:"; \
		echo "$$output"; \
		exit 1; \
	fi
	@gocyclo -avg . | grep '^Average'

## test: Run tests
test:
	go test -count=1 ./...

## test-v: Run tests verbose
test-v:
	go test -v -count=1 ./...

## test-race: Run tests with race detector
test-race:
	go test -race -count=1 ./...

## test-report: Run tests with coverage and JUnit reports (for CI)
test-report:
	@mkdir -p $(REPORTS_DIR)
	gotestsum --junitfile $(REPORTS_DIR)/junit.xml -- -count=1 -race -coverprofile=$(REPORTS_DIR)/coverage.out -covermode=atomic ./...
	@go tool cover -func=$(REPORTS_DIR)/coverage.out | tail -1
	@go tool cover -html=$(REPORTS_DIR)/coverage.out -o $(REPORTS_DIR)/coverage.html
	gocover-cobertura < $(REPORTS_DIR)/coverage.out > $(REPORTS_DIR)/coverage.xml

## coverage: Run tests with coverage and enforce 80% threshold
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

## check: Run all quality gates (the "is this ready to push?" command)
check: fmt vet shadow lint vuln gosec gitleaks complexity test

## clean: Remove build artifacts
clean:
	rm -f $(BINARY) coverage.out
	rm -rf $(REPORTS_DIR) dist/

## release: Cross-compile for distribution
release: clean
	@mkdir -p dist
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)_darwin_arm64 .
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)_darwin_amd64 .
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)_linux_arm64 .
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)_linux_amd64 .
	@echo "Binaries in dist/"
	@ls -lh dist/
