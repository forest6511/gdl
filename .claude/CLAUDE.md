# Claude Code Instructions for gdl Project

## Project Overview
gdl is a fast, resumable file downloader library and CLI tool written in Go. The project emphasizes clean Go code, comprehensive error handling, and user-friendly APIs.

## Code Requirements

### Language
- **ALL comments must be in English only**
- **ALL documentation must be in English**
- **ALL commit messages must be in English**
- Variable names must use descriptive English words

### Go Conventions
- Follow standard Go idioms and conventions
- Use `gofmt` for all code formatting
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Prefer simplicity over cleverness

### Linting and Code Quality
The project uses comprehensive golangci-lint configuration for code quality:

#### Basic Configuration (`.golangci.yml`)

- **DO NOT** update `.golangci.yml` without explicit user instruction
- This file contains critical linting configuration that should remain stable
- Any changes to linting rules must be requested by the user first

#### Pre-commit Checklist
```bash
# 1. Run basic lint checks (REQUIRED before push)
golangci-lint run

# 2. Run strict development checks (RECOMMENDED)
golangci-lint run --config .golangci-strict.yml

# 3. Run security checks (REQUIRED before push)
gosec ./...

# 4. Format code automatically
gofmt -s -w .
goimports -w .

# 5. Run tests
go test ./...
go test -race ./...
```

### Security Scanning Guidelines

The project includes comprehensive security scanning with gosec. Follow these guidelines:

#### Local Security Checks
```bash
# Basic security scan (excludes examples and test files)
gosec -exclude-dir=examples -exclude-dir=test ./...

# Full codebase security scan
gosec ./...

# Generate detailed security report
gosec -fmt=html -out=security-report.html ./...
```

#### Security Issue Guidelines
- **HIGH severity issues**: Must be fixed before merging
- **MEDIUM severity issues**: Should be addressed but won't block CI
- **G115 (Integer overflow)**: Often false positives, review case by case
- **G404 (Weak random)**: Use `#nosec G404 -- comment` for non-cryptographic randomness

#### Using #nosec Comments
When security findings are false positives or acceptable for the use case:

```go
// Good: Specific suppression with justification
// #nosec G404 -- Jitter for retry delays doesn't require cryptographic randomness
jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)

// Good: File operations with controlled input
// #nosec G304 -- File path is constructed internally, not from user input
data, err := os.ReadFile(resumeFilePath)
```

#### Security Best Practices
- Use secure file permissions: `0600` for files, `0750` for directories
- Replace weak crypto primitives: Use SHA256 instead of MD5
- Validate all external inputs and file paths
- Use crypto/rand for security-sensitive randomness
- Document security decisions with #nosec comments

#### Security Validation Policy (OSS-Ready)

This project uses a **3-tier security validation approach** to balance safety, performance, and maintainability for OSS distribution:

**Tier 1: Public API - Mandatory Code Validation**
- All public functions must validate inputs with actual code
- No #nosec comments allowed for user-provided parameters
- Must handle invalid inputs gracefully with clear error messages
- Example: URL validation, file path sanitization, size limits

```go
func (d *Downloader) DownloadFile(url, dest string) error {
    if err := validateURL(url); err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }
    if err := validateDestination(dest); err != nil {
        return fmt.Errorf("invalid destination: %w", err)
    }
    // ... internal processing
}
```

**Tier 2: Internal Processing - #nosec with Validation Context**
- Internal functions can use #nosec if input is validated by Tier 1
- Must reference the validation point in the comment
- Should include brief explanation of why it's safe
- Used for performance-critical internal operations

```go
// #nosec G304 -- dest validated by validateDestination() in DownloadFile()
file, err := os.Create(dest)
```

**Tier 3: System Values - #nosec with Technical Justification**
- System APIs and verified external data sources
- HTTP headers from trusted sources (Content-Length, etc.)
- OS-provided values (file sizes, disk space, etc.)
- Must explain the data source and why it's trusted

```go
// #nosec G115 -- HTTP Content-Length header, RFC-compliant servers guarantee non-negative values
contentSize := uint64(contentLength)
```

**Validation Function Guidelines:**
- Create reusable validation functions in `pkg/validation/`
- Use consistent error types and messages
- Include bounds checking for numeric values
- Validate file paths against directory traversal attacks
- Check URL schemes and hosts for security policies
- Document validation rules in function comments

**Testing Requirements:**
- All validation functions must have comprehensive tests
- Include boundary value testing and malicious input testing
- Test both valid and invalid inputs with clear expectations
- Use fuzzing tests for complex validation logic

This approach ensures OSS users can trust the library's security while maintaining good performance for internal operations.

### Code Style Guidelines
1. **Function Design**
   - Keep functions small and focused (< 50 lines)
   - Use early returns to reduce nesting
   - One function should do one thing well

2. **Error Handling**
   - Never ignore errors with `_`
   - Always wrap errors with context using `fmt.Errorf("context: %w", err)`
   - Create custom error types for domain-specific errors
   - Check errors immediately after function calls

3. **Naming Conventions**
   - Use descriptive names, avoid single letters except for loop indices
   - Exported functions must start with capital letters
   - Use camelCase for variables and functions
   - Use PascalCase for types and exported identifiers

4. **Documentation**
   - Add godoc comments for ALL exported types, functions, and methods
   - Start godoc comments with the name of the element
   - Include usage examples where appropriate

### Example Code Style

```go
// Download downloads a file from the given URL to the specified destination.
// It automatically resumes interrupted downloads and supports concurrent connections.
//
// Example:
//   err := Download(ctx, "https://example.com/file.zip", "./downloads/file.zip")
//   if err != nil {
//       return fmt.Errorf("failed to download file: %w", err)
//   }
func Download(ctx context.Context, url, dest string) error {
    // Validate inputs
    if url == "" {
        return fmt.Errorf("url cannot be empty")
    }
    if dest == "" {
        return fmt.Errorf("destination cannot be empty")
    }

    // Create downloader with default options
    dl := New()
    
    // Start download
    if err := dl.DownloadFile(ctx, url, dest); err != nil {
        return fmt.Errorf("download failed: %w", err)
    }
    
    return nil
}
```

### Testing Requirements
1. Write tests for all new functionality
2. Use table-driven tests for multiple test cases
3. Include both positive and negative test cases
4. Mock external dependencies (HTTP calls, file system)
5. Maintain test coverage above 80%

### Testing Example
```go
func TestDownload(t *testing.T) {
    tests := []struct {
        name    string
        url     string
        dest    string
        wantErr bool
    }{
        {
            name:    "valid download",
            url:     "https://example.com/file.zip",
            dest:    "./test.zip",
            wantErr: false,
        },
        {
            name:    "empty url",
            url:     "",
            dest:    "./test.zip",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Download(context.Background(), tt.url, tt.dest)
            if (err != nil) != tt.wantErr {
                t.Errorf("Download() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## CI/CD and Testing Strategy

### GitHub Actions Workflow Management
The project uses comprehensive GitHub Actions workflows for quality assurance. As new features are implemented, corresponding tests must be added to the CI pipeline.

### Test Categories and CI Jobs

#### Current Test Jobs (Optimized Pipeline - 7 Jobs)
1. **Code Quality** (`lint` job)
  - golangci-lint comprehensive rule set
  - go vet static analysis
  - go mod tidy verification

2. **Unit Tests** (`test` job)  
  - Standard unit tests for all packages with race detection
  - Coverage reporting to Codecov
  - Must maintain >90% coverage

3. **Integration Tests** (`integration` job)
  - Real-world scenarios and examples validation
  - Library API and CLI functionality tests
  - Uses real HTTP endpoints (httpbin.org)

4. **Cross-Platform Compatibility** (`cross-platform` job)
  - Ubuntu, Windows, macOS testing
  - Go versions 1.23, 1.24 (optimized matrix)
  - Core functionality and CLI builds

5. **Performance Tests** (`benchmark` job)
  - Performance regression detection  
  - Benchmark execution and artifact upload
  - Runs on main branch only

6. **Security Scan** (`security` job)
  - Gosec security scanner
  - staticcheck static analysis
  - Full codebase security audit

7. **Build Artifacts** (`build` job)
  - Cross-platform binary builds (Linux, Windows, macOS)
  - Multiple architectures (amd64, arm64)
  - Version embedding and size optimization
  - Depends on all other jobs passing

### Guidelines for Adding New CI Tests

#### When to Add New Test Jobs
1. **New Major Features** - Add dedicated test job for complex features
2. **New Dependencies** - Add compatibility tests
3. **Platform-Specific Code** - Add platform-specific test matrices
4. **Security-Critical Features** - Add security-focused test scenarios

#### How to Add New Test Jobs

1. **Create Test Tags**
   ```go
   //go:build feature_name
   // +build feature_name
   
   package mypackage
   ```

2. **Add CI Job to `.github/workflows/main.yml`**
   ```yaml
   feature-name-tests:
     name: Feature Name tests
     runs-on: ubuntu-latest
     steps:
       - name: Set up Go
         uses: actions/setup-go@v5
         with:
           go-version: ${{ env.GO_VERSION }}
       
       - name: Check out code
         uses: actions/checkout@v4
       
       - name: Run feature tests
         run: go test -tags=feature_name -v ./...
   ```

3. **Document Test Purpose**
  - Add clear comments explaining what the test validates
  - Include test scenarios and expected outcomes

#### Build Tag Strategy
Use build tags to categorize tests:

- **Feature Tags**: `concurrent`, `resume`, `error_simulation`
- **Environment Tags**: `integration`, `large_files`, `network_tests`
- **Performance Tags**: `memory_tests`, `leak_tests`, `benchmark_extended`
- **Platform Tags**: `unix`, `windows`, `darwin`

#### Test Data Management
```go
// Use environment variables for test configuration
func getTestBaseURL() string {
    if url := os.Getenv("TEST_HTTP_BASE_URL"); url != "" {
        return url
    }
    return "http://localhost:8080" // default for local testing
}
```

#### Parallel Test Execution
```yaml
strategy:
  matrix:
    test-category: [unit, integration, performance, security]
    go-version: ['1.23', '1.24']
parallel: true
```

### Test Naming Conventions

#### Test Files
- Unit tests: `*_test.go`
- Integration tests: `*_integration_test.go`
- Benchmark tests: `*_bench_test.go`
- Example tests: `example_*_test.go`

#### Test Functions
```go
// Unit test
func TestDownloadBasic(t *testing.T) {}

// Integration test  
func TestDownloadIntegration(t *testing.T) {}

// Benchmark test
func BenchmarkDownloadSpeed(b *testing.B) {}

// Example test
func ExampleDownload() {}
```

### Future CI Enhancements

#### Planned Additions
1. **Phase 5 Tests** - Plugin system and middleware tests
2. **Phase 6 Tests** - Production deployment and monitoring tests
3. **E2E Tests** - Complete user journey testing
4. **Load Tests** - High-volume download scenarios
5. **Chaos Tests** - Network failure simulation

#### CI Performance Optimization
- Use test caching strategies
- Implement parallel test execution
- Add test result artifacts
- Optimize Docker image builds for test services

### Monitoring and Alerts
- Set up alerts for test failures
- Monitor test execution time trends
- Track coverage regression
- Alert on security vulnerability detection

Remember: **Every new feature must include corresponding CI tests before merging to main branch.**

### Package Structure
- `cmd/gdl/` - CLI application
- `gdl.go` - Main public API
- `pkg/` - Public packages
  - `config/` - Configuration management  
  - `errors/` - Error handling and types
  - `progress/` - Progress tracking
  - `types/` - Common types
  - `ui/` - User interface utilities
  - `help/` - Help system
- `internal/` - Private packages
  - `core/` - Core download logic
  - `concurrent/` - Concurrent download management
  - `network/` - Network utilities and diagnostics
  - `storage/` - Storage and disk management
  - `retry/` - Retry logic and strategies
  - `resume/` - Resume functionality
  - `recovery/` - Error recovery
  - `testing/` - Test utilities
- `examples/` - Comprehensive examples
  - `01_basic_download/` - Basic functionality
  - `02_concurrent_download/` - Performance optimization
  - `03_progress_tracking/` - Progress monitoring
  - `04_resume_functionality/` - Resume capabilities
  - `05_error_handling/` - Error handling strategies
  - `library/`, `cli/`, `integration/` - Interface examples
- `docs/` - Documentation

### Commit Message Format
Follow conventional commits:
```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only changes
- `test`: Adding missing tests
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `chore`: Changes to the build process or auxiliary tools

### Dependencies
- Minimize external dependencies
- Prefer standard library over third-party packages
- If a dependency is needed, document why in the code

### Performance Considerations
- Use buffered I/O for file operations
- Implement connection pooling for HTTP clients
- Avoid unnecessary allocations in hot paths
- Use sync.Pool for frequently allocated objects

## Project-Specific Guidelines

### Download Implementation
1. Always support resume capability when possible
2. Implement proper cleanup on cancellation
3. Use context for cancellation propagation
4. Provide detailed progress information

### Error Types
Create specific error types for different failure scenarios:
```go
var (
    ErrInvalidURL      = errors.New("invalid URL")
    ErrFileExists      = errors.New("file already exists")
    ErrInsufficientSpace = errors.New("insufficient disk space")
)
```

### Logging
- Use structured logging (when implemented)
- Log errors with context
- Avoid logging in library code (return errors instead)
- Let the application decide what to log

## Important Project Information

### Module Name
- The Go module name is: `github.com/forest6511/gdl`
- Always use this exact module path in imports and go.mod
- GitHub username: forest6511
- Project name: gdl

### Import Path Examples
```go
import (
    "github.com/forest6511/gdl"
    "github.com/forest6511/gdl/pkg/errors"
    "github.com/forest6511/gdl/pkg/types"
)
```

### Test Quality Gates
- All tests must pass before merging to main
- New code must have >90% test coverage
- No new security vulnerabilities allowed
- Performance regression threshold: <10% slowdown
- All examples must be tested and functional
- README.md and documentation must be kept up-to-date

### Documentation and Examples Requirements

#### Examples Management
- All new features must include comprehensive examples
- Examples should be self-contained and executable
- Each example must include:
  - Clear documentation of demonstrated features
  - Error handling demonstrations
  - Performance considerations
  - Cleanup procedures

#### Documentation Updates Required
When adding new features, always update:
1. **README.md** - Feature matrix, CLI reference, API documentation
2. **docs/** - Detailed technical documentation
3. **examples/** - Practical usage demonstrations
4. **CLAUDE.md** - Development guidelines (this file)

#### Automated Documentation Workflow
The project includes automated workflows to ensure documentation consistency:
- README.md validation
- Example testing and validation
- Documentation link checking
- Feature matrix accuracy verification

## 📋 Comprehensive Extension Rules

When adding new features to gdl, follow these mandatory update rules to ensure consistency and maintainability.

### 🎯 Core Rule: Feature Completeness Matrix

Every new feature must be implemented and documented across ALL applicable layers:

1. **Core Implementation** (`internal/`, `pkg/`)
2. **Public API** (`gdl.go`, `pkg/types/`)
3. **CLI Interface** (`cmd/gdl/`)
4. **Documentation** (README.md, `docs/`)
5. **Examples** (`examples/`)
6. **Tests** (unit, integration, examples)
7. **CI/CD** (workflows, validation)

### 🔄 Mandatory Updates by Category

#### 1. **New Core Feature** (e.g., new download algorithm, protocol support)

**MUST UPDATE:**
- [ ] **Code Implementation**
  - Core logic in `internal/core/` or relevant package
  - Public API functions in `gdl.go`
  - Types and interfaces in `pkg/types/`
  - Error handling in `pkg/errors/`

- [ ] **CLI Integration**
  - Command-line flags in `cmd/gdl/main.go`
  - Help text and usage examples
  - Flag validation and mapping to library options

- [ ] **Documentation**
  - README.md Feature Parity Matrix (both CLI and Library columns)
  - README.md API Documentation section
  - README.md CLI Reference section
  - `docs/` directory with detailed documentation

- [ ] **Examples**
  - New dedicated example if major feature
  - Update existing examples if enhancement
  - CLI examples in `examples/cli/`
  - Integration examples showing both CLI and library usage

- [ ] **Tests**
  - Unit tests for core functionality
  - Integration tests for end-to-end scenarios
  - CLI tests for command-line interface
  - Example compilation and execution tests
  - Benchmark tests if performance-critical

- [ ] **CI/CD**
  - Update workflows if new test categories needed
  - Update linter rules if new patterns introduced

#### 2. **New CLI-Only Feature** (e.g., interactive prompts, output formatting)

**MUST UPDATE:**
- [ ] CLI implementation in `cmd/gdl/`
- [ ] README.md CLI Reference section
- [ ] README.md Feature Parity Matrix (mark Library as ❌ "Not applicable")
- [ ] CLI examples in `examples/cli/`
- [ ] CLI tests and validation
- [ ] Help system updates

#### 3. **New Library-Only Feature** (e.g., callback interfaces, advanced options)

**MUST UPDATE:**
- [ ] Core implementation and public API
- [ ] README.md API Documentation section
- [ ] README.md Feature Parity Matrix (mark CLI as ❌ "Not applicable")
- [ ] Dedicated example in `examples/`
- [ ] Library integration tests
- [ ] Go package documentation (godoc comments)

#### 4. **New Error Type or Handling**

**MUST UPDATE:**
- [ ] Error definitions in `pkg/errors/`
- [ ] Error handling in core components
- [ ] CLI error messages and suggestions
- [ ] Error handling example updates
- [ ] Documentation of error scenarios
- [ ] Test cases for error conditions

#### 5. **Performance Enhancement**

**MUST UPDATE:**
- [ ] Implementation in relevant packages
- [ ] Benchmark tests
- [ ] Performance documentation
- [ ] Configuration options (if applicable)
- [ ] Examples demonstrating optimization
- [ ] CI performance regression tests

#### 6. **New Configuration Option**

**MUST UPDATE:**
- [ ] Types in `pkg/types/types.go`
- [ ] CLI flags in `cmd/gdl/main.go`
- [ ] README.md CLI Reference
- [ ] README.md API Documentation
- [ ] Configuration examples
- [ ] Validation tests
- [ ] Default value documentation

### 🔍 Quality Gates for Extensions

#### Before Committing Any Extension:

1. **All Tests Pass**
   ```bash
   go test ./...
   go test -race ./...
   ```

3. **Linting Passes**
   ```bash
   golangci-lint run
   ```

4. **Examples Compile and Run**
   ```bash
   # For each new/modified example
   cd examples/[example_name] && go run main.go
   ```

5. **CLI Integration Test**
   ```bash
   go build -o gdl ./cmd/gdl/
   ./gdl --help  # Verify help shows new options
   ```

6. **Coverage Maintained**
   ```bash
   go test -cover ./...
   # Must maintain >90% coverage
   ```

### 📚 Documentation Update Checklist

#### README.md Updates (ALWAYS required):
- [ ] Feature description in main Features section
- [ ] API documentation with code examples
- [ ] CLI reference with usage examples  
- [ ] Feature Parity Matrix updated accurately
- [ ] Examples section updated with new examples
- [ ] Installation/usage instructions (if changed)

#### docs/ Directory Updates:
- [ ] Create detailed documentation file for complex features
- [ ] Update existing docs if behavior changes
- [ ] Add troubleshooting guides for new error scenarios
- [ ] Include architecture diagrams if structural changes

#### Examples Updates:
- [ ] Create new numbered example (06_, 07_, etc.) for major features
- [ ] Update existing examples if API changes
- [ ] Ensure all examples are self-contained and executable
- [ ] Include error handling and cleanup in examples
- [ ] Add both CLI and library usage examples

### 🧪 Test Update Requirements

#### Unit Tests:
- [ ] Test all new public functions/methods
- [ ] Test error conditions and edge cases
- [ ] Test concurrency safety (with `-race`)
- [ ] Table-driven tests for multiple scenarios

#### Integration Tests:
- [ ] End-to-end scenarios using real network requests
- [ ] CLI integration tests
- [ ] Cross-platform compatibility tests

#### Example Tests:
- [ ] Compilation tests for all examples
- [ ] Basic execution tests (with timeouts)
- [ ] Documentation accuracy verification

### 🚀 CI/CD Updates

#### When to Update Workflows:
- [ ] New test categories require dedicated CI jobs
- [ ] New build targets or platforms
- [ ] New dependencies or external services
- [ ] Performance-critical features need regression testing

#### Validation Scripts:
- [ ] Add new examples to compilation checks
- [ ] Update feature matrix validation rules
- [ ] Add any new file pattern checks

### 🔧 Configuration Updates

#### .golangci.yml:
- [ ] Add exclusions for new example patterns
- [ ] Update linter rules for new code patterns
- [ ] Adjust complexity thresholds if needed

#### Makefile:
- [ ] Add new build targets for new packages or binaries
- [ ] Update test targets for new test categories
- [ ] Add validation targets for new features
- [ ] Update CI targets for new workflow requirements
- [ ] Add example targets for new example directories
- [ ] Update clean targets for new build artifacts

#### GitHub Workflows:
- [ ] Add new examples to CI validation
- [ ] Update build matrices for new features
- [ ] Add performance benchmarks for critical features

### 📋 Pre-Release Checklist

Before creating a release with new features:

1. **Complete Feature Implementation**
   - [ ] All mandatory updates completed
   - [ ] Feature parity achieved across interfaces
   - [ ] Error handling implemented

2. **Documentation Completeness**
   - [ ] README.md fully updated
   - [ ] All examples working and documented
   - [ ] API documentation complete

3. **Quality Assurance**
   - [ ] All tests pass including race detection
   - [ ] Examples compile and execute successfully
   - [ ] No performance regressions

4. **Cross-Platform Verification**
   - [ ] CI passes on all supported platforms
   - [ ] Examples tested on multiple environments
   - [ ] CLI functionality verified across platforms

### 🎯 Feature-Specific Rules

#### Download Engine Features:
- Must support both CLI and library interfaces
- Must include progress callbacks for library
- Must include visual progress for CLI
- Must support cancellation via context
- Must include comprehensive error handling

#### Network Features:
- Must include timeout configuration
- Must support both HTTP and HTTPS
- Must include retry logic
- Must validate certificates properly
- Must support proxy configuration

#### File System Features:
- Must support resume functionality
- Must validate disk space
- Must handle permissions properly
- Must create parent directories if requested
- Must support overwrite protection

#### User Interface Features:
- CLI features must include help text
- Must support quiet/verbose modes
- Must provide user-friendly error messages
- Must support different output formats where applicable

Remember: **Every extension must enhance BOTH the library API and CLI tool** unless there's a technical reason why one interface cannot support the feature.

## 🛠️ Makefile Maintenance Guidelines

The project uses a comprehensive Makefile for build automation and development workflows. When extending the project, the Makefile must be updated to maintain consistency and automation.

### When to Update Makefile

#### **New Packages or Modules**
- Add build targets for new CLI tools or binaries
- Update test targets to include new package paths
- Add lint targets for new code areas

#### **New Test Categories**
- Add test targets for new test types (e.g., `test-plugin`, `test-security`)
- Update coverage targets to include new packages
- Add benchmark targets for performance-critical components

#### **New Examples**
- Update `examples` target to include new example directories
- Add specific example targets for complex demonstrations
- Update validation targets to test new examples

#### **New Build Artifacts**
- Update `clean` target to remove new build outputs
- Add cross-platform build targets for new binaries
- Update artifact collection for CI/CD

### Makefile Update Patterns

#### **Adding New Build Targets**
```make
# Template for new binary
build-newbinary: ## Build the new binary
	@echo "Building new binary..."
	go build -ldflags="-s -w" -o bin/newbinary ./cmd/newbinary/

# Template for new package
build-plugin: ## Build plugin system
	@echo "Building plugin components..."
	go build -buildmode=plugin -o plugins/ ./pkg/plugins/
```

#### **Adding New Test Targets**
```make
# Template for new test category
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -tags=integration ./...

test-security: ## Run security tests
	@echo "Running security tests..."
	go test -tags=security ./...
```

#### **Adding New Validation Targets**
```make
# Template for new validation
validate-security: ## Validate security configuration
	@echo "Validating security settings..."
	./scripts/validate_security.sh

validate-plugins: ## Validate plugin compatibility
	@echo "Validating plugins..."
	./scripts/validate_plugins.sh
```

#### **Updating CI Targets**
```make
# Template for CI integration
ci-test-full: ## CI full test suite with new categories
	go test -race -tags="integration,security" -coverprofile=coverage.out ./...

ci-build-all: ## CI build all binaries
	$(MAKE) build build-newbinary build-plugin
```

### Makefile Quality Standards

#### **Target Documentation**
- Every target must have a `## Description` comment
- Use consistent formatting: `target: ## Description`
- Group related targets with blank lines

#### **Error Handling**
- Use `@echo "Message..."` for user feedback
- Set `.PHONY:` for all non-file targets
- Use consistent error messages

#### **Cross-Platform Compatibility**
- Test Makefile on Linux, macOS, and Windows (with appropriate tools)
- Use portable shell commands when possible
- Document any platform-specific requirements

### Makefile Update Checklist

When adding new features, update Makefile with:

- [ ] **New build targets** for any new binaries or packages
- [ ] **New test targets** for new test categories or special test requirements
- [ ] **New validation targets** for feature-specific validation
- [ ] **Updated clean targets** to remove new build artifacts
- [ ] **Updated help target** documentation for new targets
- [ ] **Updated CI targets** for new continuous integration requirements
- [ ] **Cross-platform testing** to ensure new targets work on all platforms

### Example Makefile Updates

#### **When Adding Plugin System**
```make
# Add to build section
build-plugins: ## Build all plugins
	@echo "Building plugins..."
	@mkdir -p plugins
	go build -buildmode=plugin -o plugins/auth.so ./pkg/plugins/auth/
	go build -buildmode=plugin -o plugins/transform.so ./pkg/plugins/transform/

# Add to test section  
test-plugins: ## Test plugin system
	@echo "Testing plugins..."
	go test ./pkg/plugins/...

# Add to validation section
validate-plugins: ## Validate plugin compatibility
	@echo "Validating plugins..."
	./scripts/validate_plugins.sh

# Update clean section
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf plugins/
	rm -f coverage.out coverage.html
	go clean -testcache

# Update help and phony declarations
.PHONY: help build build-plugins test test-plugins validate-plugins clean
```

#### **When Adding New Examples**
```make
# Update examples target
examples: ## Test all examples including new ones
	@echo "Testing examples..."
	@for dir in examples/*/; do \
		if [[ -f "$$dir/main.go" ]]; then \
			echo "Testing: $$dir"; \
			cd "$$dir" && timeout 30s go run main.go || echo "Example completed"; \
			cd - > /dev/null; \
		fi; \
	done
	@echo "Testing example plugins..."
	@cd examples/plugin_demo && go run main.go

# Add specific example targets if needed
example-plugin: ## Run plugin example
	@echo "Running plugin example..."
	cd examples/plugin_demo && go run main.go
```

This ensures the Makefile remains a comprehensive and maintainable build automation tool as the project grows.

## 📁 Files Requiring Regular Maintenance

When adding new features or making significant changes, these files must be kept up-to-date:

### Core Documentation Files (CRITICAL)
- **`README.md`** - Primary project documentation
  - Feature descriptions and capabilities
  - API usage examples with current syntax
  - CLI reference with all command options
  - Feature parity matrix (CLI vs Library)
  - Installation and quick start instructions
  - Links to all documentation resources

- **`docs/API_REFERENCE.md`** - Complete library API documentation
  - All public functions with signatures
  - Parameter descriptions and examples
  - Return value documentation
  - Error handling patterns
  - Advanced usage patterns and best practices

- **`docs/CLI_REFERENCE.md`** - Comprehensive CLI documentation
  - All command-line flags and options
  - Usage examples for each feature
  - Configuration options and environment variables
  - Exit codes and error messages
  - Shell integration examples

- **`docs/DIRECTORY_STRUCTURE.md`** - Project organization
  - Complete directory tree with descriptions
  - Package organization and responsibilities
  - File naming conventions
  - Build artifacts and generated files

### GitHub Workflows (CI/CD)
- **`.github/workflows/main.yml`** - Main orchestrator workflow
  - Update when adding new workflow dependencies
  - Modify conditional execution logic for new workflows

- **`.github/workflows/unit-tests.yml`** - Unit test execution
  - Add new test groups when creating new packages
  - Update coverage targets for new modules
  - Modify test matrix for new Go versions

- **`.github/workflows/integration-tests.yml`** - Integration testing
  - Add new integration test scenarios
  - Update example testing when adding new examples
  - Modify test data and endpoints

- **`.github/workflows/lint.yml`** - Code quality checks
  - Update linting rules for new code patterns
  - Add new static analysis tools
  - Modify formatting checks

- **`.github/workflows/cross-platform.yml`** - Platform testing
  - Add new platforms or architectures
  - Update Go version matrix
  - Modify build verification steps

- **`.github/workflows/security.yml`** - Security scanning
  - Update security tools and configurations
  - Add new vulnerability checks
  - Modify scan frequency and targets

- **`.github/workflows/benchmark.yml`** - Performance testing
  - Add new benchmarks for performance-critical features
  - Update comparison logic for PR reviews
  - Modify performance thresholds

- **`.github/workflows/release.yml`** - Release automation
  - Update build targets for new platforms
  - Modify asset collection and distribution
  - Add new release validation steps

### Configuration Files
- **`.golangci.yml`** - Linter configuration
  - Add new linting rules for new code patterns
  - Update exclusions for generated code
  - Modify severity levels and thresholds

- **`Makefile`** - Build automation
  - Add targets for new packages or binaries
  - Update test targets for new test categories
  - Modify clean targets for new artifacts
  - Add validation targets for new features

- **`go.mod`** - Go module dependencies
  - Update when adding new dependencies
  - Maintain minimum Go version compatibility
  - Keep dependencies minimal and up-to-date

### Examples and Validation
- **`examples/README.md`** - Examples documentation
  - Add descriptions for new examples
  - Update running instructions
  - Maintain quick start guide

- **`examples/integration/`** - Integration verification
  - **`library_usage.go`** - Library API examples
  - **`cli_usage.sh`** - CLI usage demonstrations
  - **`feature_parity_test.go`** - Automated parity verification


### Package Documentation
- **`pkg/types/types.go`** - Core type definitions
  - Update godoc comments for all exported types
  - Add examples for complex type usage
  - Maintain backward compatibility documentation

- **`cmd/gdl/main.go`** - CLI implementation
  - Update help text for new flags
  - Maintain flag consistency and validation
  - Add usage examples in comments

### Maintenance Schedule

#### After Adding New Features:
1. Update feature documentation (README.md, API_REFERENCE.md, CLI_REFERENCE.md)
2. Add examples demonstrating the feature
3. Update integration tests and parity verification
4. Modify CI workflows if needed
5. Update validation scripts

#### After Adding New Packages:
1. Update DIRECTORY_STRUCTURE.md
2. Add package documentation (doc.go files)
3. Update Makefile targets
4. Modify CI test groupings
5. Update import examples in documentation

#### After Changing CLI Interface:
1. Update CLI_REFERENCE.md with new flags/options
2. Update README.md CLI examples
3. Modify cli_usage.sh integration examples
4. Update help system and validation
5. Test cross-platform compatibility

#### After Changing API Interface:
1. Update API_REFERENCE.md with new signatures
2. Update README.md library examples  
3. Modify library_usage.go integration examples
4. Update godoc comments
5. Verify backward compatibility

#### Regular Maintenance (Monthly):
1. Review and update dependency versions
2. Update Go version matrix in CI
3. Review and update security scanning rules
4. Validate all documentation links
5. Run full integration test suite
6. Update performance benchmarks

### Critical Maintenance Notes

- **Test all examples after API changes: `make examples`**
- **Verify CI workflows after workflow changes: check GitHub Actions**
- **Maintain feature parity between CLI and library interfaces**
- **Keep documentation examples current with actual API**
- **Update version references when cutting releases**

## 🔄 Local CI Compatibility

To prevent CI failures, always use CI-equivalent commands locally:

### Pre-commit Hook (Automatic)
A pre-commit hook is installed that runs CI-equivalent checks:
```bash
# The hook automatically runs:
# - gofmt -s -w . (format code)
# - go mod tidy (tidy modules) 
# - go vet $(go list ./... | grep -v '/examples/') (vet excluding examples)
# - golangci-lint run (if available)
# - go test -short ./... (quick tests)
```

### Manual CI Checks
```bash
# Full CI equivalent check
make ci-check

# Individual CI equivalent commands
make ci-format      # Format exactly like CI
make ci-vet         # Vet exactly like CI  
make ci-test-core   # Test core library like CI

# Run local CI check script
./scripts/local-ci-check.sh
```

### Why Local and CI Differ
Common causes of local vs CI differences:
1. **Go version differences** (check with `go version`)
2. **golangci-lint version differences** (CI uses latest)
3. **Command scope differences** (CI excludes `/examples/`)
4. **OS/platform differences** (CI runs on Ubuntu Linux)
5. **Execution order differences** (CI runs steps sequentially)

### Best Practices
1. **Always run `make ci-check` before pushing**
2. **Use the pre-commit hook for automatic checking**
3. **Keep golangci-lint updated to match CI**
4. **Test with the exact Go version used in CI (1.24+)**
5. **Run `./scripts/local-ci-check.sh` for detailed feedback**

This comprehensive maintenance approach ensures the project remains well-documented, properly tested, and easy to use as it grows and evolves.
