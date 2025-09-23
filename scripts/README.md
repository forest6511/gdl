# Scripts Directory

This directory contains utility scripts for the godl project development and maintenance.

## CI/CD and Quality Checks

### 🚀 local-ci-check.sh
**Purpose**: Run CI-equivalent checks locally before pushing
**Usage**: `./scripts/local-ci-check.sh`

Performs the same checks that GitHub Actions CI will run:
- Code formatting (gofmt)
- Go vet analysis
- Go mod tidy verification
- Golangci-lint checks
- Quick tests with race detection
- Cross-compilation verification

**Features**:
- Supports `act` for local GitHub Actions testing
- Interactive cross-platform testing option
- Docker cleanup for fresh test runs

### 🔐 security-check.sh
**Purpose**: Comprehensive security analysis
**Usage**: `./scripts/security-check.sh [--full|--quick|--fix]`

Security-focused checks including:
- Gosec security scanner
- Hardcoded secrets detection
- Unsafe file operations audit
- Error handling verification
- Integer overflow risk assessment
- Dependency vulnerability scanning

**Options**:
- `--full`: Run advanced security checks including race conditions and TLS configs
- `--quick`: Essential security checks only
- `--fix`: Attempt to fix issues where possible

### 📋 pre-push-check.sh
**Purpose**: Comprehensive pre-push validation
**Usage**: `./scripts/pre-push-check.sh [--quick|--full|--fix]`

All-in-one validation before pushing to remote:
- Git status and branch info
- Code formatting verification
- Module dependency checks
- Static analysis (go vet)
- Linting (golangci-lint)
- Security scanning
- Test execution
- Build verification (including cross-platform)
- Examples compilation
- Documentation checks

**Options**:
- `--quick`: Fast essential checks only
- `--full`: Complete checks including benchmarks and examples
- `--fix`: Auto-fix formatting and linting issues

## Performance Testing

### ⚡ performance_check.sh
**Purpose**: Basic performance benchmarking
**Usage**: `./scripts/performance_check.sh`

Runs performance benchmarks for core functionality.

### 📊 performance_comparison.sh
**Purpose**: Compare performance between versions
**Usage**: `./scripts/performance_comparison.sh`

Compares benchmark results between different code versions.

### 📈 performance_monitor.sh
**Purpose**: Monitor performance over time
**Usage**: `./scripts/performance_monitor.sh`

Tracks performance metrics and generates reports.

### 🗄️ large_file_benchmark.sh
**Purpose**: Test large file download performance
**Usage**: `./scripts/large_file_benchmark.sh`

Benchmarks download performance with various file sizes.

### 🌍 real_world_test.sh
**Purpose**: Test with real-world scenarios
**Usage**: `./scripts/real_world_test.sh`

Tests the downloader with actual remote files and various network conditions.

## Setup and Maintenance

### 🔧 setup-git-hooks.sh
**Purpose**: Install Git hooks for the project
**Usage**: `./scripts/setup-git-hooks.sh`

Sets up pre-commit hooks that run basic checks before each commit.

### 📦 prepare-release.sh
**Purpose**: Prepare a new release
**Usage**: `./scripts/prepare-release.sh`

Automates release preparation:
- Updates version numbers
- Generates changelog
- Creates release tags

### 🏷️ sync-labels.sh
**Purpose**: Sync GitHub issue labels
**Usage**: `./scripts/sync-labels.sh`

Synchronizes GitHub issue labels with project standards.

### 🍺 update-homebrew.sh
**Purpose**: Update Homebrew formula
**Usage**: `./scripts/update-homebrew.sh`

Updates the Homebrew formula after a new release.

## Recommended Workflow

### Before Committing
```bash
# Quick validation
./scripts/local-ci-check.sh

# Or use git hooks (auto-runs on commit)
./scripts/setup-git-hooks.sh
```

### Before Pushing
```bash
# Comprehensive check
./scripts/pre-push-check.sh

# With auto-fix
./scripts/pre-push-check.sh --fix

# Quick mode for iterative development
./scripts/pre-push-check.sh --quick
```

### Before PR/Release
```bash
# Full validation
./scripts/pre-push-check.sh --full

# Security audit
./scripts/security-check.sh --full

# Performance regression check
./scripts/performance_comparison.sh
```

## Installing Required Tools

### Essential Tools
```bash
# Linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Security scanner
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Static checker
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### Optional Tools
```bash
# Local GitHub Actions runner
brew install act  # macOS
# For other platforms: https://github.com/nektos/act#installation

# Vulnerability scanner
go install golang.org/x/vuln/cmd/govulncheck@latest

# Error checker
go install github.com/kisielk/errcheck@latest
```

## Script Exit Codes

All scripts follow consistent exit codes:
- `0`: All checks passed
- `1`: One or more checks failed
- `2`: Missing dependencies or configuration issues

## Tips

1. **Start with `pre-push-check.sh`** - It combines all essential checks
2. **Use `--fix` flag** when available to auto-correct issues
3. **Run `security-check.sh --full`** before releases
4. **Set up git hooks** for automatic pre-commit validation
5. **Use `--quick` mode** during active development for faster feedback

## Contributing

When adding new scripts:
1. Follow the existing naming convention
2. Add proper documentation headers
3. Use consistent color codes and output formatting
4. Update this README with script description
5. Ensure scripts are executable (`chmod +x`)
6. Test on macOS, Linux, and Windows (Git Bash)