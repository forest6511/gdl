# CI Workflow Best Practices

## 🚨 Preventing CI Failures

### Common Issues and Solutions

#### 1. Formatting Issues
**Problem**: Code formatting doesn't match CI standards  
**Solution**: Always run formatting before committing

```bash
# Option 1: Auto-fix and commit if needed
make fix-and-commit

# Option 2: Manual fix
make ci-format
git add .
git commit
```

#### 2. Pre-commit Hook Behavior
**Important**: When the pre-commit hook fixes formatting, it will:
- Auto-format your files
- **STOP the commit** 
- Ask you to add the changes and commit again

This is intentional to ensure formatted code is included in your commit.

### Recommended Workflow

#### Before Every Push
```bash
# Run complete pre-push validation
make pre-push

# This runs:
# 1. Code formatting
# 2. go mod tidy
# 3. go vet
# 4. golangci-lint
# 5. Race detection tests
```

#### Quick Commands Reference

| Command | Purpose | When to Use |
|---------|---------|-------------|
| `make ci-check` | Run all CI checks locally | Before pushing |
| `make pre-push` | Format + all CI checks | Before pushing (recommended) |
| `make fix-and-commit` | Auto-fix formatting and commit | When formatting issues exist |
| `make ci-format` | Format code to CI standards | Before committing |
| `make test-ci-all` | Test all platforms with act | Cross-platform testing |
| `make test-ci-ubuntu` | Test Ubuntu CI locally | Linux-specific testing |
| `make test-ci-windows` | Test Windows CI locally | Windows-specific testing |
| `make test-ci-macos` | Test macOS CI locally | macOS-specific testing |
| `make test-cross-compile` | Quick Windows cross-compile | Fast Windows compatibility check |

### CI/CD Pipeline Stages

1. **Quick Checks** - Basic validation
2. **Lint / Code Quality** - Format, vet, tidy, golangci-lint
3. **Unit Tests** - All test suites with race detection
4. **Cross-Platform Tests** - Windows, macOS, Linux compatibility
5. **Integration Tests** - End-to-end testing

### Local Cross-Platform Testing with act

The `act` tool allows you to run GitHub Actions locally, providing an exact match to the CI environment.

#### Setup act

```bash
# Install act (macOS)
brew install act

# For other platforms, see: https://github.com/nektos/act#installation
```

#### Test Individual Platforms

```bash
# Test Ubuntu (Linux)
make test-ci-ubuntu

# Test Windows
make test-ci-windows  

# Test macOS
make test-ci-macos

# Test all platforms sequentially
make test-ci-all
```

#### Manual act Commands

```bash
# Run specific platform/version combination
act -j cross-platform --matrix os:windows-latest --matrix go-version:1.23

# Run entire cross-platform workflow
act -W .github/workflows/cross-platform.yml

# Debug mode (see detailed output)
act -j cross-platform --matrix os:windows-latest -v

# Use specific Docker image
act -P windows-latest=catthehacker/ubuntu:act-latest
```

#### Alternative: Quick Cross-Compilation Check

For rapid feedback without full CI simulation:

```bash
# Quick Windows compatibility check
make test-cross-compile

# Manual cross-compilation
GOOS=windows GOARCH=amd64 go build ./...
GOOS=windows GOARCH=amd64 go test -c ./pkg/plugin/...
GOOS=darwin GOARCH=amd64 go build ./...
```

#### When to Use Each Method

- **`make test-ci-all`** - Before creating PR (comprehensive)
- **`make test-ci-windows`** - When fixing Windows-specific issues  
- **`make test-ci-macos`** - When fixing macOS-specific issues
- **`make test-cross-compile`** - Quick build verification (fastest)

### Troubleshooting

#### "Works locally but fails in CI"
1. Check formatting: `gofmt -s -l .`
2. Check go.mod: `go mod tidy && git diff go.mod go.sum`
3. Run full CI check: `make ci-check`

#### Windows-specific Test Failures
- Use runtime checks: `if runtime.GOOS == "windows"`
- Handle path differences: Use `filepath.Join()` instead of hardcoded paths
- Be aware of executable permissions differences

#### Pre-commit Hook Issues
If pre-commit hook reformats files:
```bash
# Files were reformatted, commit was stopped
git add .
git commit  # Try again with formatted files
```

### Best Practices

1. **Always run `make pre-push` before pushing**
2. **Pay attention to pre-commit hook messages**
3. **Use `git status` after commits to check for unstaged changes**
4. **Test on multiple platforms when making OS-specific changes**
5. **Keep local tools updated** (golangci-lint, Go version)

### Tool Installation

```bash
# Install golangci-lint (required for full CI compatibility)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install act (for local CI testing)
brew install act

# Verify installation
golangci-lint version
act --version
```

## Quick Start for New Contributors

```bash
# 1. Clone the repository
git clone https://github.com/forest6511/godl.git
cd godl

# 2. Install dependencies
go mod download

# 3. Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 4. Before your first commit
make ci-check  # Verify everything works

# 5. Development workflow
make fmt       # Format code
make test      # Run tests
make pre-push  # Before pushing
```

## CI Configuration Files

- `.github/workflows/main.yml` - Main CI/CD pipeline
- `.golangci.yml` - Linter configuration
- `.git/hooks/pre-commit` - Local pre-commit checks
- `Makefile` - Build and test automation