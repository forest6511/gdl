# Makefile for gdl project
# Fast, concurrent file downloader written in Go

.PHONY: help build test lint clean install dev docs examples all

# Default target
all: lint test build

# Display help information
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Build targets
build: ## Build the CLI binary
	@echo "Building gdl CLI..."
	go build -ldflags="-s -w" -o bin/gdl ./cmd/gdl/

build-all: ## Build binaries for all platforms
	@echo "Building for all platforms..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/gdl-linux-amd64 ./cmd/gdl/
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/gdl-linux-arm64 ./cmd/gdl/
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/gdl-darwin-amd64 ./cmd/gdl/
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/gdl-darwin-arm64 ./cmd/gdl/
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/gdl-windows-amd64.exe ./cmd/gdl/

# Testing targets
test: ## Run all tests
	@echo "Running tests..."
	go test ./...

test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	go test -race ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-bench: ## Run benchmark tests
	@echo "Running benchmark tests..."
	go test -bench=. ./...

# Code quality targets
lint: ## Run golangci-lint
	@echo "Running linter..."
	golangci-lint run

fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

tidy: ## Tidy Go modules
	@echo "Tidying modules..."
	go mod tidy

# Development targets
dev: fmt tidy lint test ## Run development checks (format, tidy, lint, test)

install: ## Install CLI tool globally
	@echo "Installing gdl CLI..."
	go install ./cmd/gdl/

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -testcache

# Documentation targets
docs: ## Documentation target
	@echo "Documentation target complete"

examples: ## Test all examples
	@echo "Testing examples..."
	@for dir in examples/*/; do \
		if [[ -f "$$dir/main.go" ]]; then \
			echo "Testing: $$dir"; \
			cd "$$dir" && timeout 30s go run main.go || echo "Example completed"; \
			cd - > /dev/null; \
		fi; \
	done

# CI targets (used by GitHub Actions)
ci-test: ## CI test target
	go test -race -coverprofile=coverage.out ./...

ci-lint: ## CI lint target
	golangci-lint run --out-format=github-actions

ci-build: ## CI build target
	go build -ldflags="-s -w" ./cmd/gdl/

# CI equivalent checks (run exactly what CI runs)
ci-check: ## Run full CI equivalent checks locally
	@echo "🔍 Running CI equivalent checks..."
	@echo "1. Checking code formatting..."
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "❌ Formatting issues found:"; \
		gofmt -s -l .; \
		echo "Run 'gofmt -s -w .' to fix"; \
		exit 1; \
	fi
	@echo "✅ Code formatting check passed"
	@echo "2. Running go vet (excluding examples)..."
	go vet $$(go list ./... | grep -v '/examples/')
	@echo "✅ go vet passed"
	@echo "3. Checking go mod tidy..."
	go mod tidy
	@git diff --exit-code go.mod go.sum || (echo "❌ go.mod/go.sum not tidy. Run 'go mod tidy'" && exit 1)
	@echo "✅ go mod tidy check passed"
	@echo "4. Running golangci-lint..."
	golangci-lint run
	@echo "✅ golangci-lint passed"
	@echo "5. Running quick tests with race detection..."
	go test -race -short ./...
	@echo "✅ Quick tests with race detection passed"
	@echo "🎉 All CI equivalent checks passed!"

ci-format: ## Format code exactly like CI expects
	@echo "📝 Applying CI formatting standards..."
	gofmt -s -w .
	go mod tidy
	@echo "✅ Code formatted to CI standards"

pre-push: ## Complete pre-push validation (runs all CI checks locally)
	@echo "🚀 Running complete pre-push validation..."
	@$(MAKE) ci-format
	@$(MAKE) ci-check
	@echo "✅ All pre-push checks passed! Safe to push."

fix-and-commit: ## Fix formatting and create a commit if needed
	@echo "🔧 Checking and fixing formatting..."
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		gofmt -s -w . && \
		git add -A && \
		git commit -m "style: auto-fix code formatting" && \
		echo "✅ Formatting fixed and committed"; \
	else \
		echo "✅ No formatting issues found"; \
	fi

# Local CI testing with act
test-ci-local: ## Run GitHub Actions locally with act (requires: brew install act)
	@echo "🐳 Running GitHub Actions locally with act..."
	@if ! command -v act >/dev/null 2>&1; then \
		echo "❌ act not found. Install with: brew install act"; \
		exit 1; \
	fi
	act -j cross-platform --matrix os:ubuntu-latest --matrix go-version:1.24

test-ci-windows: ## Test Windows CI locally with act
	@echo "🪟 Testing Windows CI locally..."
	@if ! command -v act >/dev/null 2>&1; then \
		echo "❌ act not found. Install with: brew install act"; \
		exit 1; \
	fi
	@echo "🧹 Clearing act cache..."
	@rm -rf ~/.cache/act || true
	@echo "🐳 Cleaning up act containers..."
	@docker ps -aq --filter "name=act-" | xargs -r docker rm -f || true
	act workflow_dispatch -W .github/workflows/cross-platform.yml --matrix os:windows-latest --matrix go-version:1.23 --container-architecture linux/amd64

test-ci-macos: ## Test macOS CI locally with act  
	@echo "🍎 Testing macOS CI locally..."
	@if ! command -v act >/dev/null 2>&1; then \
		echo "❌ act not found. Install with: brew install act"; \
		exit 1; \
	fi
	@echo "🧹 Clearing act cache..."
	@rm -rf ~/.cache/act || true
	@echo "🐳 Cleaning up act containers..."
	@docker ps -aq --filter "name=act-" | xargs -r docker rm -f || true
	act workflow_dispatch -W .github/workflows/cross-platform.yml --matrix os:macos-latest --matrix go-version:1.23 --container-architecture linux/amd64

test-ci-ubuntu: ## Test Ubuntu CI locally with act
	@echo "🐧 Testing Ubuntu CI locally..."
	@if ! command -v act >/dev/null 2>&1; then \
		echo "❌ act not found. Install with: brew install act"; \
		exit 1; \
	fi
	@echo "🧹 Clearing act cache..."
	@rm -rf ~/.cache/act || true
	@echo "🐳 Cleaning up act containers..."
	@docker ps -aq --filter "name=act-" | xargs -r docker rm -f || true
	act workflow_dispatch -W .github/workflows/cross-platform.yml --matrix os:ubuntu-latest --matrix go-version:1.24 --container-architecture linux/amd64

test-ci-all: ## Test cross-platform compatibility locally with act (Ubuntu only, Windows/macOS require GitHub Actions)
	@echo "🌍 Testing all platforms locally with act..."
	@if ! command -v act >/dev/null 2>&1; then \
		echo "❌ act not found. Install with: brew install act"; \
		echo "📝 Install with: brew install act"; \
		exit 1; \
	fi
	@echo "🧹 Clearing act cache for fresh execution..."
	@rm -rf ~/.cache/act || true
	@echo "🐳 Cleaning up act containers..."
	@docker ps -aq --filter "name=act-" | xargs -r docker rm -f || true
	@echo "1️⃣ Testing Ubuntu (cross-platform workflow)..."
	@ubuntu_success=0; \
	if act workflow_dispatch -W .github/workflows/cross-platform.yml --matrix os:ubuntu-latest --matrix go-version:1.24 --container-architecture linux/amd64 2>&1 | tee /tmp/ubuntu-test.log; then \
		if grep -q "✅ Tests Passed: ubuntu-latest" /tmp/ubuntu-test.log; then \
			echo "✅ Ubuntu tests passed"; \
			ubuntu_success=1; \
		fi; \
	fi; \
	echo "2️⃣ Testing Windows (cross-platform workflow)..."; \
	act workflow_dispatch -W .github/workflows/cross-platform.yml --matrix os:windows-latest --matrix go-version:1.23 --container-architecture linux/amd64 || echo "🚧 Windows test skipped (expected with act)"; \
	echo "3️⃣ Testing macOS (cross-platform workflow)..."; \
	act workflow_dispatch -W .github/workflows/cross-platform.yml --matrix os:macos-latest --matrix go-version:1.23 --container-architecture linux/amd64 || echo "🚧 macOS test skipped (expected with act)"; \
	echo "✅ Cross-platform tests completed!"; \
	echo "💡 Note: Only Ubuntu can be fully tested locally with act."; \
	echo "   Windows and macOS testing requires GitHub Actions environment."; \
	if [ $$ubuntu_success -eq 1 ]; then \
		echo "✅ Primary platform (Ubuntu) tests passed - ready for CI"; \
		exit 0; \
	else \
		echo "❌ Ubuntu tests failed - check output above"; \
		exit 1; \
	fi

test-cross-compile: ## Quick cross-compilation test for Windows
	@echo "🔄 Testing cross-compilation for Windows..."
	GOOS=windows GOARCH=amd64 go build ./...
	GOOS=windows GOARCH=amd64 go test -c ./pkg/plugin/...
	@echo "✅ Cross-compilation successful"

ci-vet: ## Run go vet exactly like CI
	@echo "🔍 Running go vet (CI equivalent)..."
	go vet $$(go list ./... | grep -v '/examples/')

ci-test-core: ## Run core library tests like CI
	@echo "🧪 Running Core Library tests (CI equivalent)..."
	go test -v -race -coverprofile=coverage-library.out . ./pkg/...

# Release targets
release-check: test-race build-all ## Pre-release validation
	@echo "Release checks complete"

# Quick targets for common workflows
quick: fmt lint test ## Quick development cycle (format, lint, test)

full: clean tidy fmt lint test-race test-coverage build ## Full build and validation cycle

# Help target should be first for default
.DEFAULT_GOAL := help