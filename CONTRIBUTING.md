# Contributing to godl

## Development Guidelines

### Code Style

1. **Comments must be in English only**
   - All code comments, documentation, and commit messages should be in English
   - Use clear and concise language

2. **Follow Go conventions**
   - Use `gofmt` for formatting
   - Follow [Effective Go](https://golang.org/doc/effective_go.html)
   - Use meaningful variable and function names

3. **Error handling**
   - Always handle errors explicitly
   - Wrap errors with context using `fmt.Errorf`
   - Create custom error types when appropriate

4. **Testing**
   - Write tests for all new features
   - Maintain test coverage above 80%
   - Use table-driven tests where appropriate

### For AI Assistants (Claude, GitHub Copilot, etc.)

When generating code for this project:

1. **All comments and documentation must be in English**
2. **Follow Go idioms and best practices**
3. **Include comprehensive error handling**
4. **Add unit tests for any new functions**
5. **Use descriptive variable names, not single letters**
6. **Keep functions small and focused (< 50 lines)**
7. **Add godoc comments for all exported functions**

### Commit Messages

Follow the conventional commits format:
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `test:` for test additions/changes
- `refactor:` for code refactoring
- `chore:` for maintenance tasks

Example:
```
feat: add concurrent download support

- Implement chunk-based downloading
- Add progress tracking for each chunk
- Support up to 10 concurrent connections
```

## 🚀 Quick Start for Contributors

### Setup Development Environment

```bash
# 1. Fork and clone the repository
git clone https://github.com/YOUR_USERNAME/godl.git
cd godl

# 2. Install dependencies
go mod download

# 3. Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
brew install act  # For cross-platform testing (optional but recommended)

# 4. Verify setup
make ci-check  # Run all CI checks locally
```

### Development Workflow

#### 1. Before Making Changes

```bash
# Create a feature branch
git checkout -b feature/your-feature-name

# Verify current state
./scripts/local-ci-check.sh
```

#### 2. During Development

```bash
# Quick testing during development
go test ./...
go test -race ./...

# Check specific package
go test -v ./pkg/your-package/
```

#### 3. Before Committing

```bash
# Complete validation (REQUIRED)
./scripts/local-ci-check.sh

# Format and fix issues
make ci-format
make fix-and-commit  # Auto-fix formatting if needed
```

#### 4. Before Pushing

```bash
# Final validation
make pre-push

# Cross-platform testing (recommended for new features)
make test-ci-all     # Test Ubuntu + Windows + macOS
# OR test specific platforms
make test-ci-windows # If you changed platform-specific code
make test-ci-macos   # If you changed platform-specific code
```

### Testing Strategy

#### Test Types

| Test Level | Command | When to Use |
|------------|---------|-------------|
| **Unit Tests** | `go test ./pkg/your-package/` | During feature development |
| **Integration Tests** | `go test ./...` | After implementing features |
| **CI Validation** | `./scripts/local-ci-check.sh` | Before every commit |
| **Cross-Platform** | `make test-ci-all` | Before pushing major changes |

#### Writing Tests

- **Coverage Target**: Maintain >90% test coverage
- **Test Structure**: Use table-driven tests for multiple scenarios
- **Test Names**: Use descriptive names: `TestDownloadWithResume_ResumeFromPartialFile`
- **Platform-Specific**: Use build tags for platform-specific tests

```go
//go:build !windows
// +build !windows

func TestUnixSpecificFeature(t *testing.T) {
    // Unix-only test
}
```

### Code Quality Standards

#### Automated Checks

The project uses automated tools to maintain code quality:

```bash
# All of these run automatically in CI and pre-commit hooks
gofmt -s -w .        # Code formatting
go vet ./...         # Static analysis
golangci-lint run    # Comprehensive linting
go test -race ./...  # Race condition detection
```

#### Manual Code Review Checklist

- [ ] **English Only**: All comments and documentation in English
- [ ] **Error Handling**: All errors properly handled and wrapped
- [ ] **Tests**: New features have comprehensive tests
- [ ] **Documentation**: Public APIs have godoc comments
- [ ] **Cross-Platform**: Consider Windows/macOS compatibility
- [ ] **Performance**: No obvious performance regressions

### Handling CI Failures

#### Common Issues and Solutions

1. **Formatting Issues**
   ```bash
   make ci-format
   git add .
   git commit --amend
   ```

2. **Race Conditions**
   ```bash
   go test -race -v ./problematic/package/
   # Fix using sync primitives or atomic operations
   ```

3. **Cross-Platform Failures**
   ```bash
   make test-ci-windows  # Test Windows locally
   make test-ci-macos    # Test macOS locally
   ```

4. **Lint Issues**
   ```bash
   golangci-lint run --fix  # Auto-fix some issues
   ```

### Platform-Specific Development

#### Windows Compatibility

- Use `filepath.Join()` instead of hardcoded paths
- Use build tags for Windows-specific code: `//go:build windows`
- Test with: `make test-ci-windows`
- Note: Go plugins are not supported on Windows

#### macOS Compatibility

- Use BSD-compatible commands (e.g., timeout differences)
- Test with: `make test-ci-macos`

#### Cross-Compilation Testing

```bash
# Quick build verification
GOOS=windows GOARCH=amd64 go build ./...
GOOS=darwin GOARCH=amd64 go build ./...
```

### Git Workflow

#### Branch Naming

- `feature/add-oauth-support`
- `fix/windows-path-handling`
- `docs/update-api-reference`
- `test/improve-coverage`

#### Pre-commit Hook

The repository automatically installs a pre-commit hook that:

1. Checks code formatting
2. Runs basic validation
3. **Stops commit if formatting is needed**

If the hook reformats your code:
```bash
git add .
git commit  # Try again with formatted code
```

### Performance Considerations

- **Benchmarks**: Include benchmarks for performance-critical code
- **Memory**: Avoid memory leaks in long-running operations
- **Concurrency**: Use race detection: `go test -race`
- **Profiling**: Profile with `go test -bench=. -cpuprofile=cpu.prof`

### Documentation Requirements

#### Code Documentation

```go
// DownloadWithOptions downloads a file with custom options.
// It returns download statistics and any error encountered.
//
// Example:
//   stats, err := godl.DownloadWithOptions(ctx, url, dest, &Options{
//       MaxConcurrency: 4,
//       ProgressCallback: func(p Progress) { ... },
//   })
func DownloadWithOptions(ctx context.Context, url, dest string, opts *Options) (*Stats, error) {
```

#### Update Documentation

- Update README.md for user-facing changes
- Update API_REFERENCE.md for API changes
- Add examples in `examples/` directory for new features

### Release Process

1. **Feature Development**: Complete feature with tests
2. **Local Validation**: `./scripts/local-ci-check.sh`
3. **Cross-Platform Testing**: `make test-ci-all`
4. **Documentation**: Update relevant docs
5. **PR Creation**: Create descriptive PR
6. **Code Review**: Address review feedback
7. **CI Validation**: Ensure all CI checks pass
8. **Merge**: Squash and merge to main

### Getting Help

- **Development Questions**: Open a discussion
- **Bug Reports**: Use the issue template
- **Feature Requests**: Open an issue with detailed requirements
- **CI/Testing Issues**: Check `docs/CI_WORKFLOW.md`

### Tools and Resources

- **Required**: Go 1.23+, golangci-lint
- **Recommended**: act (for local CI), goimports
- **Documentation**: 
  - [Go Code Review Guidelines](https://github.com/golang/go/wiki/CodeReviewComments)
  - [Effective Go](https://golang.org/doc/effective_go.html)
  - [CI Workflow Guide](docs/CI_WORKFLOW.md)
