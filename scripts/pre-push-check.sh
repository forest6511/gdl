#!/bin/bash
#
# Pre-Push Check Script
# Comprehensive checks to run before pushing to remote
# Combines all necessary CI checks in one place
#
# Usage:
#   ./scripts/pre-push-check.sh [options]
#
# Options:
#   --quick    Run only essential checks (faster)
#   --full     Run all checks including benchmarks
#   --fix      Auto-fix issues where possible

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Options
QUICK_MODE=false
FULL_MODE=false
FIX_MODE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --quick)
            QUICK_MODE=true
            shift
            ;;
        --full)
            FULL_MODE=true
            shift
            ;;
        --fix)
            FIX_MODE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--quick|--full|--fix]"
            exit 1
            ;;
    esac
done

# Track failed checks
FAILED_CHECKS=()
WARNINGS=()

print_header() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}     Pre-Push Checks for godl Project${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

print_section() {
    echo ""
    echo -e "${BLUE}▶ $1${NC}"
    echo "─────────────────────────────────"
}

print_check() {
    echo -e "  ${BLUE}◆${NC} $1..."
}

print_success() {
    echo -e "    ${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "    ${RED}✗${NC} $1"
}

print_warning() {
    echo -e "    ${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "    ${CYAN}ℹ${NC} $1"
}

# Start checks
print_header

# 1. Git Status Check
print_section "Git Status"
print_check "Checking working directory"

UNCOMMITTED=$(git status --porcelain | wc -l)
if [ "$UNCOMMITTED" -gt 0 ]; then
    print_warning "You have $UNCOMMITTED uncommitted changes"
    WARNINGS+=("uncommitted changes")
    git status --short
else
    print_success "Working directory clean"
fi

CURRENT_BRANCH=$(git branch --show-current)
print_info "Current branch: $CURRENT_BRANCH"

# 2. Code Formatting
print_section "Code Formatting"

if [ "$FIX_MODE" = true ]; then
    print_check "Formatting code"
    gofmt -s -w .
    goimports -w . 2>/dev/null || true
    print_success "Code formatted"
else
    print_check "Checking code formatting"
    UNFORMATTED=$(gofmt -s -l . | wc -l)
    if [ "$UNFORMATTED" -gt 0 ]; then
        print_error "Found $UNFORMATTED unformatted files"
        gofmt -s -l .
        FAILED_CHECKS+=("formatting")
        print_info "Run with --fix to auto-format"
    else
        print_success "All files properly formatted"
    fi
fi

# 3. Module Dependencies
print_section "Dependencies"

print_check "Checking go.mod tidiness"
cp go.mod go.mod.backup
cp go.sum go.sum.backup 2>/dev/null || true

go mod tidy

if ! diff go.mod go.mod.backup >/dev/null 2>&1; then
    if [ "$FIX_MODE" = true ]; then
        print_success "go.mod tidied"
        rm go.mod.backup go.sum.backup 2>/dev/null || true
    else
        print_error "go.mod needs tidying"
        FAILED_CHECKS+=("go mod tidy")
        mv go.mod.backup go.mod
        mv go.sum.backup go.sum 2>/dev/null || true
        print_info "Run 'go mod tidy' or use --fix"
    fi
else
    print_success "go.mod is tidy"
    rm go.mod.backup go.sum.backup 2>/dev/null || true
fi

# 4. Static Analysis
print_section "Static Analysis"

print_check "Running go vet"
if go vet ./... 2>&1; then
    print_success "go vet passed"
else
    print_error "go vet found issues"
    FAILED_CHECKS+=("go vet")
fi

# 5. Linting
print_section "Linting"

if command -v golangci-lint &> /dev/null; then
    print_check "Running golangci-lint"

    if [ "$FIX_MODE" = true ]; then
        if golangci-lint run --fix; then
            print_success "Linting passed (with fixes)"
        else
            print_error "Linting failed (unfixable issues)"
            FAILED_CHECKS+=("golangci-lint")
        fi
    else
        if golangci-lint run; then
            print_success "All linting checks passed"
        else
            print_error "Linting issues found"
            FAILED_CHECKS+=("golangci-lint")
            print_info "Run with --fix to auto-fix some issues"
        fi
    fi
else
    print_warning "golangci-lint not installed"
    WARNINGS+=("golangci-lint not installed")
fi

# 6. Security Checks
print_section "Security"

# 6.1 gosec - Security scanner
if command -v gosec &> /dev/null; then
    print_check "Running gosec security scan"

    # Run gosec and check for HIGH/MEDIUM severity issues in production code
    if gosec -fmt text -exclude-dir=examples -exclude-dir=test ./... > /tmp/gosec-output.txt 2>&1; then
        print_success "No security issues found (gosec)"
    else
        # Count HIGH and MEDIUM severity issues (excluding G115)
        HIGH_ISSUES=$(grep "^\\[.*\\] - G" /tmp/gosec-output.txt | grep -v "G115" | grep -c "Severity: HIGH" || echo "0")
        MEDIUM_ISSUES=$(grep "^\\[.*\\] - G" /tmp/gosec-output.txt | grep -v "G115" | grep -c "Severity: MEDIUM" || echo "0")

        if [ "$HIGH_ISSUES" -gt 0 ]; then
            print_error "Found $HIGH_ISSUES HIGH severity security issues"
            FAILED_CHECKS+=("gosec HIGH severity")
            grep "^\\[.*\\] - G" /tmp/gosec-output.txt | grep -v "G115" | grep "Severity: HIGH" | head -3
        elif [ "$MEDIUM_ISSUES" -gt 0 ]; then
            print_warning "Found $MEDIUM_ISSUES MEDIUM severity security issues"
            WARNINGS+=("gosec MEDIUM severity")
        else
            print_success "No significant security issues (gosec)"
        fi
    fi
    rm -f /tmp/gosec-output.txt
else
    print_warning "gosec not installed"
    WARNINGS+=("gosec not installed")
    print_info "Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"
fi

# 6.2 staticcheck - Static analysis
if command -v staticcheck &> /dev/null; then
    print_check "Running staticcheck"

    if staticcheck ./... > /tmp/staticcheck-output.txt 2>&1; then
        print_success "No staticcheck issues found"
    else
        STATIC_ISSUES=$(wc -l < /tmp/staticcheck-output.txt | tr -d ' ')
        print_error "Found $STATIC_ISSUES staticcheck issues"
        FAILED_CHECKS+=("staticcheck")
        head -5 /tmp/staticcheck-output.txt
        if [ "$STATIC_ISSUES" -gt 5 ]; then
            print_info "... and $((STATIC_ISSUES - 5)) more issues"
        fi
    fi
    rm -f /tmp/staticcheck-output.txt
else
    print_warning "staticcheck not installed"
    WARNINGS+=("staticcheck not installed")
    print_info "Install: go install honnef.co/go/tools/cmd/staticcheck@latest"
fi

# 6.3 govulncheck - Vulnerability check
if command -v govulncheck &> /dev/null; then
    print_check "Checking for known vulnerabilities"

    if govulncheck ./... > /tmp/vuln-output.txt 2>&1; then
        print_success "No known vulnerabilities found"
    else
        if grep -q "Vulnerability" /tmp/vuln-output.txt; then
            print_error "Known vulnerabilities found"
            FAILED_CHECKS+=("govulncheck")
            grep -A 3 "Vulnerability" /tmp/vuln-output.txt | head -10
        else
            print_success "No known vulnerabilities found"
        fi
    fi
    rm -f /tmp/vuln-output.txt
else
    print_warning "govulncheck not installed"
    WARNINGS+=("govulncheck not installed")
    print_info "Install: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

# 7. Tests
print_section "Testing"

if [ "$QUICK_MODE" = true ]; then
    print_check "Running quick tests"
    if go test -short -race ./... >/dev/null 2>&1; then
        print_success "Quick tests passed"
    else
        print_error "Quick tests failed"
        FAILED_CHECKS+=("tests")
        print_info "Run 'go test -v ./...' for details"
    fi
elif [ "$FULL_MODE" = true ]; then
    print_check "Running full test suite with coverage"
    if go test -race -coverprofile=coverage.out ./... >/dev/null 2>&1; then
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
        print_success "All tests passed (coverage: $COVERAGE)"
        rm coverage.out
    else
        print_error "Tests failed"
        FAILED_CHECKS+=("tests")
    fi

    # Run benchmarks in full mode
    print_check "Running benchmarks"
    if go test -bench=. -benchmem -run=^$ ./... >/dev/null 2>&1; then
        print_success "Benchmarks completed"
    else
        print_warning "Some benchmarks failed"
        WARNINGS+=("benchmark failures")
    fi
else
    print_check "Running standard tests"
    if go test -race ./... >/dev/null 2>&1; then
        print_success "All tests passed"
    else
        print_error "Tests failed"
        FAILED_CHECKS+=("tests")
        print_info "Run 'go test -v ./...' for details"
    fi
fi

# 8. Build Verification
print_section "Build Verification"

print_check "Building CLI binary"
if go build -o /tmp/gdl-test ./cmd/gdl 2>/dev/null; then
    print_success "CLI builds successfully"
    rm /tmp/gdl-test
else
    print_error "CLI build failed"
    FAILED_CHECKS+=("build")
fi

print_check "Testing cross-compilation"
PLATFORMS=("windows/amd64" "darwin/amd64" "linux/arm64")
BUILD_FAILED=false

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    if GOOS=$GOOS GOARCH=$GOARCH go build -o /dev/null ./cmd/gdl 2>/dev/null; then
        echo -e "      ${GREEN}✓${NC} $platform"
    else
        echo -e "      ${RED}✗${NC} $platform"
        BUILD_FAILED=true
    fi
done

if [ "$BUILD_FAILED" = false ]; then
    print_success "Cross-compilation successful"
else
    print_error "Some platforms failed to build"
    FAILED_CHECKS+=("cross-compilation")
fi

# 9. Examples Check
if [ "$FULL_MODE" = true ]; then
    print_section "Examples"
    print_check "Verifying examples compile"

    EXAMPLE_FAILED=false
    for dir in examples/*/; do
        if [ -f "$dir/main.go" ]; then
            if go build -o /dev/null "$dir" 2>/dev/null; then
                echo -e "      ${GREEN}✓${NC} $(basename "$dir")"
            else
                echo -e "      ${RED}✗${NC} $(basename "$dir")"
                EXAMPLE_FAILED=true
            fi
        fi
    done

    if [ "$EXAMPLE_FAILED" = false ]; then
        print_success "All examples compile"
    else
        print_error "Some examples failed"
        FAILED_CHECKS+=("examples")
    fi
fi

# 10. Documentation Check
print_section "Documentation"

print_check "Checking for TODO/FIXME comments"
TODO_COUNT=$(grep -r "TODO\|FIXME" --include="*.go" . 2>/dev/null | grep -v vendor | wc -l)
if [ "$TODO_COUNT" -gt 0 ]; then
    print_warning "Found $TODO_COUNT TODO/FIXME comments"
    WARNINGS+=("$TODO_COUNT TODOs")
else
    print_success "No TODO/FIXME comments"
fi

print_check "Checking godoc comments"
EXPORTED_WITHOUT_DOC=$(golint ./... 2>/dev/null | grep -c "should have comment" || echo "0")
EXPORTED_WITHOUT_DOC="${EXPORTED_WITHOUT_DOC:-0}"
if [ "$EXPORTED_WITHOUT_DOC" -eq 0 ]; then
    print_success "All exported symbols documented"
else
    print_warning "$EXPORTED_WITHOUT_DOC exported symbols lack documentation"
    WARNINGS+=("missing godoc comments")
fi

# Summary
echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}                         Summary${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ ${#FAILED_CHECKS[@]} -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✅ All checks passed!${NC}"

    if [ ${#WARNINGS[@]} -gt 0 ]; then
        echo ""
        echo -e "${YELLOW}⚠️  Warnings:${NC}"
        for warning in "${WARNINGS[@]}"; do
            echo "   • $warning"
        done
    fi

    echo ""
    echo "Your code is ready to push! 🚀"
    echo ""
    echo "Next steps:"
    echo "  1. git add -A"
    echo "  2. git commit -m 'your message'"
    echo "  3. git push origin $CURRENT_BRANCH"

    exit 0
else
    echo ""
    echo -e "${RED}❌ Checks failed:${NC}"
    for check in "${FAILED_CHECKS[@]}"; do
        echo "   • $check"
    done

    if [ ${#WARNINGS[@]} -gt 0 ]; then
        echo ""
        echo -e "${YELLOW}⚠️  Warnings:${NC}"
        for warning in "${WARNINGS[@]}"; do
            echo "   • $warning"
        done
    fi

    echo ""
    echo "Please fix the issues above before pushing."

    if [ "$FIX_MODE" = false ]; then
        echo ""
        echo "Tips:"
        echo "  • Run with --fix to auto-fix some issues"
        echo "  • Run with --quick for faster checks"
        echo "  • Run with --full for comprehensive checks"
    fi

    exit 1
fi