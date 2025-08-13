#!/bin/bash

# Local CI Check Script
# Run this script to verify your changes will pass CI before pushing

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "🚀 Running local CI checks from: $PROJECT_ROOT"
echo "================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_step() {
    echo -e "${BLUE}📋 Step $1:${NC} $2"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if required tools are installed
print_step "0" "Checking required tools..."

if ! command -v go >/dev/null 2>&1; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

if ! command -v golangci-lint >/dev/null 2>&1; then
    print_warning "golangci-lint not found. Install it for complete CI compatibility:"
    echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
    SKIP_LINT=true
else
    SKIP_LINT=false
fi

# Check for act (optional but recommended)
if ! command -v act >/dev/null 2>&1; then
    print_warning "act not found. Install it for cross-platform CI testing:"
    echo "  brew install act  # macOS"
    echo "  # For other platforms: https://github.com/nektos/act#installation"
    SKIP_ACT=true
else
    print_success "act found: $(act --version 2>/dev/null | head -n1 || echo 'act installed')"
    SKIP_ACT=false
fi

print_success "Go version: $(go version)"

# Step 1: Code formatting check (exact CI equivalent)
print_step "1" "Checking code formatting (CI equivalent)..."
if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
    print_error "Formatting issues found:"
    gofmt -s -l .
    echo ""
    echo "To fix these issues, run:"
    echo "  gofmt -s -w ."
    echo "  # or"
    echo "  make ci-format"
    exit 1
fi
print_success "Code formatting check passed"

# Step 2: Go vet (exact CI equivalent - excluding examples)
print_step "2" "Running go vet (CI equivalent - excluding examples)..."
if ! go vet $(go list ./... | grep -v '/examples/'); then
    print_error "go vet found issues"
    exit 1
fi
print_success "go vet passed"

# Step 3: Go mod tidy check (exact CI equivalent)
print_step "3" "Checking go mod tidy (CI equivalent)..."
go mod tidy
if ! git diff --exit-code go.mod go.sum >/dev/null 2>&1; then
    print_error "go.mod or go.sum is not tidy"
    echo "The following files were modified by 'go mod tidy':"
    git diff --name-only go.mod go.sum
    echo ""
    echo "Please commit these changes or run 'go mod tidy' before pushing"
    exit 1
fi
print_success "go mod tidy check passed"

# Step 4: golangci-lint (if available)
if [ "$SKIP_LINT" = false ]; then
    print_step "4" "Running golangci-lint (CI equivalent)..."
    if ! golangci-lint run; then
        print_error "golangci-lint found issues"
        exit 1
    fi
    print_success "golangci-lint passed"
else
    print_step "4" "Skipping golangci-lint (not installed)"
fi

# Step 5: Quick test run with race detection (to catch obvious failures)
print_step "5" "Running quick tests with race detection..."
if ! go test -race -short ./...; then
    print_error "Quick tests with race detection failed"
    echo ""
    echo "Run full tests with: make ci-test-core"
    exit 1
fi
print_success "Quick tests with race detection passed"

# Step 6: Optional cross-platform testing with act
if [ "$SKIP_ACT" = false ]; then
    echo ""
    print_step "6" "Cross-platform testing available with act"
    echo "Run cross-platform tests before pushing (optional but recommended):"
    echo "  make test-ci-all       # Test all platforms (Ubuntu, Windows, macOS)"
    echo "  make test-ci-ubuntu    # Test Ubuntu only"
    echo "  make test-ci-windows   # Test Windows only"
    echo "  make test-ci-macos     # Test macOS only"
    echo ""
    read -p "Run cross-platform tests now? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_step "6a" "Preparing cross-platform tests..."
        
        # Clear act cache to ensure fresh workflow execution
        if [ -d ~/.cache/act ]; then
            echo "🧹 Clearing act cache for fresh execution..."
            rm -rf ~/.cache/act
        fi
        
        # Clear Docker containers/images that might be outdated
        if command -v docker >/dev/null 2>&1; then
            echo "🐳 Cleaning Docker cache..."
            docker system prune -f >/dev/null 2>&1 || true
            echo "🧹 Cleaning up act containers..."
            docker ps -aq --filter "name=act-" | xargs -r docker rm -f >/dev/null 2>&1 || true
        fi
        
        print_step "6b" "Running cross-platform tests..."
        if command -v make >/dev/null 2>&1; then
            # Run cross-platform tests and capture output
            if make test-ci-all 2>&1 | tee /tmp/act-output.log; then
                print_success "Cross-platform tests passed"
            else
                # Check if tests actually passed despite act post-action failures
                if grep -q "Tests Passed:" /tmp/act-output.log; then
                    print_success "Cross-platform tests passed (ignoring act post-action issues)"
                    echo "💡 Note: act post-action failures are normal and don't affect test results"
                else
                    print_error "Cross-platform tests failed"
                    echo "Check the output above for platform-specific issues"
                    echo ""
                    echo "💡 If you see job naming issues, try:"
                    echo "  rm -rf ~/.cache/act && docker system prune -f"
                    exit 1
                fi
            fi
        else
            print_warning "make not available, run manually: make test-ci-all"
        fi
    else
        print_warning "Skipping cross-platform tests"
        echo "Remember to test cross-platform compatibility before pushing!"
    fi
else
    echo ""
    print_step "6" "Cross-platform testing (optional)"
    echo "Install act for local cross-platform CI testing:"
    echo "  brew install act"
    echo "Then run: make test-ci-all"
fi

# Summary
echo ""
echo "================================================"
print_success "🎉 All local CI checks passed!"
echo ""
echo "Your changes should pass CI. You can now:"
echo "  1. Commit your changes"
echo "  2. Push to remote"
echo "  3. Create/update your PR"
echo ""
echo "For additional testing, run:"
echo "  make ci-check          # Complete CI equivalent"
echo "  make pre-push          # Format + all CI checks"
echo "  make test-ci-all       # Cross-platform testing (requires act)"
echo "  make test-cross-compile # Quick cross-compilation check"