#!/bin/bash
#
# Security Check Script
# Run comprehensive security checks before pushing
#
# Usage:
#   ./scripts/security-check.sh [options]
#
# Options:
#   --full     Run all security checks including advanced analysis
#   --quick    Run only essential security checks
#   --fix      Attempt to fix issues where possible

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Options
FULL_MODE=false
QUICK_MODE=false
FIX_MODE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --full)
            FULL_MODE=true
            shift
            ;;
        --quick)
            QUICK_MODE=true
            shift
            ;;
        --fix)
            FIX_MODE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

print_header() {
    echo -e "${BLUE}🔒 Security Check for godl${NC}"
    echo "================================"
    echo ""
}

print_check() {
    echo -e "${BLUE}[CHECK]${NC} $1"
}

print_success() {
    echo -e "${GREEN}  ✓${NC} $1"
}

print_error() {
    echo -e "${RED}  ✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}  !${NC} $1"
}

FAILED_CHECKS=()

print_header

# 1. Gosec Security Scan
print_check "Running gosec security scanner..."
if command -v gosec &> /dev/null; then
    # Exclude test files and examples from security scan
    if gosec -quiet -exclude-dir=examples -exclude-dir=test -fmt=json ./... 2>/dev/null | jq -e '.Issues | length == 0' >/dev/null 2>&1; then
        print_success "No security vulnerabilities found"
    else
        # Show detailed report
        ISSUES=$(gosec -quiet -exclude-dir=examples -exclude-dir=test -fmt=json ./... 2>/dev/null | jq -r '.Issues[] | "  [\(.severity)] \(.file):\(.line) - \(.rule_id): \(.details)"')

        if [ -n "$ISSUES" ]; then
            print_error "Security issues found:"
            echo "$ISSUES"
            FAILED_CHECKS+=("gosec")
        fi
    fi
else
    print_warning "gosec not installed"
    echo "     Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"
fi

# 2. Check for hardcoded secrets
print_check "Checking for hardcoded secrets..."
SECRET_PATTERNS=(
    'api[_-]?key.*=.*["\047][a-zA-Z0-9]{20,}'
    'secret.*=.*["\047][a-zA-Z0-9]{20,}'
    'token.*=.*["\047][a-zA-Z0-9]{20,}'
    'password.*=.*["\047][^"\047]{8,}'
    'BEGIN RSA PRIVATE KEY'
    'BEGIN EC PRIVATE KEY'
    'BEGIN OPENSSH PRIVATE KEY'
)

FOUND_SECRETS=false
for pattern in "${SECRET_PATTERNS[@]}"; do
    if grep -r -E "$pattern" --include="*.go" --exclude-dir=vendor --exclude-dir=.git . 2>/dev/null | grep -v "_test.go" | grep -v "example"; then
        FOUND_SECRETS=true
    fi
done

if [ "$FOUND_SECRETS" = false ]; then
    print_success "No hardcoded secrets detected"
else
    print_error "Potential hardcoded secrets found"
    FAILED_CHECKS+=("secrets")
fi

# 3. Check for unsafe file operations
print_check "Checking for unsafe file operations..."
UNSAFE_OPS=$(grep -r "os.Create\|os.Open\|os.ReadFile\|os.WriteFile" --include="*.go" . 2>/dev/null | \
    grep -v "#nosec" | \
    grep -v "_test.go" | \
    grep -v "// Safe:" | \
    wc -l)

if [ "$UNSAFE_OPS" -eq 0 ]; then
    print_success "All file operations are documented as safe"
else
    print_warning "Found $UNSAFE_OPS file operations without #nosec comments"
    echo "     Review these operations and add #nosec comments with justification if safe"
fi

# 4. Check for proper error handling
print_check "Checking for unhandled errors..."
if command -v errcheck &> /dev/null; then
    if errcheck ./... 2>/dev/null; then
        print_success "All errors are properly handled"
    else
        print_error "Unhandled errors found"
        FAILED_CHECKS+=("errcheck")
    fi
else
    # Fall back to golangci-lint errcheck
    if command -v golangci-lint &> /dev/null; then
        if golangci-lint run --disable-all --enable=errcheck ./... 2>/dev/null; then
            print_success "All errors are properly handled"
        else
            print_error "Unhandled errors found"
            FAILED_CHECKS+=("errcheck")
        fi
    else
        print_warning "errcheck not available"
    fi
fi

# 5. Check for integer overflow vulnerabilities
print_check "Checking for integer overflow risks..."
OVERFLOW_RISKS=$(grep -r "int(" --include="*.go" . 2>/dev/null | \
    grep -v "#nosec G115" | \
    grep -v "_test.go" | \
    grep -v "// Safe:" | \
    wc -l)

if [ "$OVERFLOW_RISKS" -eq 0 ]; then
    print_success "Integer conversions are documented"
else
    print_warning "Found $OVERFLOW_RISKS integer conversions without documentation"
    echo "     Add #nosec G115 comments for safe conversions"
fi

# 6. Check dependencies for known vulnerabilities
print_check "Checking dependencies for vulnerabilities..."
if command -v nancy &> /dev/null; then
    if go list -json -m all | nancy sleuth 2>/dev/null; then
        print_success "No known vulnerabilities in dependencies"
    else
        print_error "Vulnerable dependencies found"
        FAILED_CHECKS+=("dependencies")
    fi
elif command -v govulncheck &> /dev/null; then
    if govulncheck ./... 2>/dev/null; then
        print_success "No known vulnerabilities in dependencies"
    else
        print_error "Vulnerable dependencies found"
        FAILED_CHECKS+=("dependencies")
    fi
else
    print_warning "Vulnerability scanner not installed"
    echo "     Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

if [ "$FULL_MODE" = true ]; then
    echo ""
    echo "Running advanced security checks..."

    # 7. Check for race conditions
    print_check "Testing for race conditions..."
    if go test -race -short ./... >/dev/null 2>&1; then
        print_success "No race conditions detected"
    else
        print_warning "Potential race conditions found"
        echo "     Run full race tests: go test -race ./..."
    fi

    # 8. Check TLS configuration
    print_check "Checking TLS configurations..."
    TLS_ISSUES=$(grep -r "tls.Config" --include="*.go" . 2>/dev/null | \
        grep -v "MinVersion.*tls.VersionTLS12\|MinVersion.*tls.VersionTLS13" | \
        grep -v "_test.go" | \
        wc -l)

    if [ "$TLS_ISSUES" -eq 0 ]; then
        print_success "TLS configurations use secure versions"
    else
        print_warning "Found $TLS_ISSUES TLS configs without explicit MinVersion"
        echo "     Ensure MinVersion is set to tls.VersionTLS12 or higher"
    fi

    # 9. Check for unsafe random number generation
    print_check "Checking random number generation..."
    UNSAFE_RAND=$(grep -r "math/rand" --include="*.go" . 2>/dev/null | \
        grep -v "#nosec G404" | \
        grep -v "_test.go" | \
        grep -v "// Non-cryptographic use" | \
        wc -l)

    if [ "$UNSAFE_RAND" -eq 0 ]; then
        print_success "Random number usage is documented"
    else
        print_warning "Found $UNSAFE_RAND uses of math/rand without documentation"
        echo "     Use crypto/rand for security-sensitive randomness"
        echo "     Add #nosec G404 comments for non-cryptographic uses"
    fi
fi

echo ""
echo "================================"

# Summary
if [ ${#FAILED_CHECKS[@]} -eq 0 ]; then
    echo -e "${GREEN}✅ Security checks passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Security checks failed:${NC}"
    for check in "${FAILED_CHECKS[@]}"; do
        echo "  - $check"
    done
    echo ""
    echo "Fix these security issues before pushing."
    exit 1
fi