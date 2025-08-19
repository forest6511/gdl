# Directory Structure

Complete project organization and file structure for the gdl project.

## Project Layout

```
gdl/
├── .github/                    # GitHub configuration and workflows
│   ├── ISSUE_TEMPLATE/         # Issue templates
│   │   ├── bug_report.md       # Bug report template
│   │   ├── feature_request.md  # Feature request template
│   │   └── config.yml          # Issue template configuration
│   ├── pull_request_template.md # Pull request template
│   ├── labels.json             # Repository labels configuration
│   ├── act_secrets             # Template for act local testing secrets
│   └── workflows/              # CI/CD workflow definitions
│       ├── main.yml            # Main orchestrator workflow
│       ├── unit-tests.yml      # Unit test workflow
│       ├── integration-tests.yml # Integration test workflow
│       ├── lint.yml            # Code quality checks
│       ├── security.yml        # Security scanning
│       ├── cross-platform.yml  # Multi-platform testing
│       ├── benchmark.yml       # Performance benchmarks
│       ├── release.yml         # Standard release automation
│       ├── release-goreleaser.yml # Advanced release automation
│       └── README.md           # Workflow documentation
│
├── cmd/                        # Command-line applications
│   └── gdl/                   # Main CLI tool
│       ├── main.go             # CLI entry point
│       └── main_test.go        # CLI tests
│
├── docs/                       # Documentation
│   ├── API_REFERENCE.md        # Library API documentation
│   ├── CLI_REFERENCE.md        # CLI usage documentation
│   ├── DIRECTORY_STRUCTURE.md  # This file
│   ├── EXTENDING.md            # Extension points and customization
│   ├── MAINTENANCE.md          # Development procedures
│   ├── PLUGIN_DEVELOPMENT.md   # Plugin development guide
│   ├── CI_WORKFLOW.md          # CI/CD pipeline documentation
│   ├── ACT_TESTING.md          # Local GitHub Actions testing guide
│   ├── RELEASE_SETUP.md        # Release management documentation
│   ├── cli/                    # CLI-specific docs
│   │   └── examples.md         # CLI usage examples
│   └── errors/                 # Error handling documentation
│       └── README.md           # Error types and handling
│
├── examples/                   # Usage examples
│   ├── 01_basic_download/      # Basic download examples
│   │   └── main.go
│   ├── 02_concurrent_download/ # Concurrent download examples
│   │   └── main.go
│   ├── 03_progress_tracking/   # Progress tracking examples
│   │   └── main.go
│   ├── 04_resume_functionality/# Resume download examples
│   │   └── main.go
│   ├── 05_error_handling/      # Error handling examples
│   │   └── main.go
│   ├── cli/                    # CLI usage examples
│   │   ├── basic_cli_examples.sh
│   │   └── advanced_cli_examples.sh
│   ├── extensions/             # Extension examples
│   │   ├── README.md           # Extension examples documentation
│   │   ├── database-protocol/  # Custom database protocol handler
│   │   ├── distributed-storage/ # Distributed storage backend
│   │   ├── analytics-events/   # Analytics event handler
│   │   ├── rate-limiting/      # Rate limiting middleware
│   │   ├── progress-formatters/ # Custom progress formatters
│   │   └── ml-optimizer/       # ML-based download optimizer
│   ├── integration/            # Integration examples
│   │   ├── feature_demo.go     # Feature demonstration
│   │   └── cli_usage.sh        # CLI usage examples
│   ├── library_api/            # Library API examples
│   │   └── main.go             # Complete library usage examples
│   ├── parity_verification/    # Feature parity verification
│   │   └── main.go             # Automated CLI vs Library testing
│   ├── plugins/                # Plugin examples
│   │   ├── README.md           # Plugin examples documentation
│   │   ├── auth/               # Authentication plugins
│   │   │   ├── simple-auth/    # Simple API key auth plugin
│   │   │   ├── oauth2/         # OAuth2 auth plugin
│   │   │   └── jwt/            # JWT auth plugin
│   │   ├── protocol/           # Protocol plugins
│   │   │   ├── s3/             # S3 protocol plugin
│   │   │   ├── ftp/            # FTP protocol plugin
│   │   │   └── custom-api/     # Custom API protocol plugin
│   │   ├── storage/            # Storage plugins
│   │   │   ├── database/       # Database storage plugin
│   │   │   ├── redis/          # Redis cache plugin
│   │   │   └── s3-storage/     # S3 storage plugin
│   │   ├── transform/          # Transform plugins
│   │   │   ├── compression/    # Compression plugin
│   │   │   ├── encryption/     # Encryption plugin
│   │   │   └── image-optimizer/ # Image optimization plugin
│   │   └── hooks/              # Hook plugins
│   │       ├── logging/        # Advanced logging plugin
│   │       ├── webhook/        # Webhook plugin
│   │       └── metrics/        # Metrics plugin
│   ├── library/                # Library-specific examples
│   │   ├── basic_usage.go
│   │   └── advanced_usage.go
│   └── README.md               # Examples documentation
│
├── internal/                   # Private packages (not exported)
│   ├── core/                   # Core download engine
│   │   ├── downloader.go       # Main downloader implementation
│   │   └── downloader_test.go  # Downloader tests
│   ├── concurrent/             # Concurrent download management
│   │   ├── manager.go          # Concurrency manager
│   │   └── manager_test.go     # Manager tests
│   ├── network/                # Network utilities
│   │   ├── client.go           # HTTP client wrapper
│   │   ├── diagnostics.go      # Network diagnostics
│   │   └── network_test.go     # Network tests
│   ├── storage/                # Storage management
│   │   ├── disk.go             # Disk operations
│   │   ├── space.go            # Space checking
│   │   └── storage_test.go     # Storage tests
│   ├── retry/                  # Retry logic
│   │   ├── strategy.go         # Retry strategies
│   │   └── retry_test.go       # Retry tests
│   ├── resume/                 # Resume functionality
│   │   ├── handler.go          # Resume handler
│   │   └── resume_test.go      # Resume tests
│   ├── recovery/               # Error recovery
│   │   ├── recovery.go         # Recovery mechanisms
│   │   └── recovery_test.go    # Recovery tests
│   └── testing/                # Test utilities
│       ├── helpers.go          # Test helper functions
│       └── mock.go             # Mock implementations
│
├── pkg/                        # Public packages (exported)
│   ├── config/                 # Configuration management
│   │   ├── config.go           # Config structures
│   │   └── config_test.go      # Config tests
│   ├── errors/                 # Error handling
│   │   ├── errors.go           # Error types
│   │   ├── codes.go            # Error codes
│   │   └── errors_test.go      # Error tests
│   ├── events/                 # Event system
│   │   ├── events.go           # Event types and emitter
│   │   └── events_test.go      # Event tests
│   ├── middleware/             # Middleware system
│   │   ├── middleware.go       # Middleware interfaces
│   │   ├── chain.go            # Middleware chain
│   │   └── middleware_test.go  # Middleware tests
│   ├── plugin/                 # Plugin system
│   │   ├── plugin.go           # Plugin interfaces
│   │   ├── manager.go          # Plugin manager
│   │   ├── loader.go           # Plugin loader
│   │   ├── security.go         # Security validator
│   │   ├── hooks.go            # Plugin hooks system
│   │   └── plugin_test.go      # Plugin tests
│   ├── progress/               # Progress tracking
│   │   ├── progress.go         # Progress interface
│   │   ├── callback.go         # Progress callbacks
│   │   └── progress_test.go    # Progress tests
│   ├── protocols/              # Protocol registry
│   │   ├── registry.go         # Protocol registry
│   │   ├── handlers.go         # Protocol handlers
│   │   └── protocols_test.go   # Protocol tests
│   ├── storage/                # Storage management
│   │   ├── manager.go          # Storage manager
│   │   ├── backends.go         # Storage backends
│   │   └── storage_test.go     # Storage tests
│   ├── types/                  # Common types
│   │   ├── types.go            # Type definitions
│   │   └── types_test.go       # Type tests
│   ├── ui/                     # User interface utilities
│   │   ├── terminal.go         # Terminal utilities
│   │   ├── prompts.go          # Interactive prompts
│   │   └── ui_test.go          # UI tests
│   ├── validation/             # Input validation
│   │   ├── validation.go       # Validation functions
│   │   └── validation_test.go  # Validation tests
│   └── help/                   # Help system
│       ├── help.go             # Help text generation
│       └── help_test.go        # Help tests
│
├── docker/                     # Docker configuration
│   ├── Dockerfile              # Multi-stage Docker build
│   ├── .dockerignore           # Docker build context exclusions
│   └── README.md               # Docker usage guide
├── scripts/                    # Development scripts
│   ├── local-ci-check.sh       # Local CI compatibility check
│   ├── setup-git-hooks.sh      # Git hooks setup script
│   ├── prepare-release.sh      # Release preparation script
│   ├── update-changelog.sh     # Symlink to prepare-release.sh
│   ├── update-homebrew.sh      # Homebrew formula updater
│   └── sync-labels.sh          # GitHub labels synchronizer
├── .claude/                    # Claude Code configuration
│   └── CLAUDE.md               # Claude Code instructions
├── .actrc                      # Act (local GitHub Actions) configuration
├── .editorconfig               # Editor configuration
├── .gitignore                  # Git ignore patterns
├── .gitmessage                 # Git commit message template
├── .golangci.yml               # Golangci-lint configuration
├── .goreleaser.yml             # GoReleaser configuration
├── CHANGELOG.md                # Project changelog
├── CONTRIBUTING.md             # Contribution guidelines
├── LICENSE                     # MIT License
├── README.md                   # Project documentation
├── go.mod                      # Go module definition
├── go.sum                      # Go module checksums
├── gdl.go                     # Main library API
├── gdl_test.go                # Library API tests
└── Makefile                    # Build and development tasks
```

## Configuration Files

### Development Tools
- **.actrc**: Configuration for act (local GitHub Actions testing)
- **.editorconfig**: Editor formatting and behavior configuration
- **.gitmessage**: Commit message template for Conventional Commits
- **.gitignore**: Files and directories excluded from Git
- **.golangci.yml**: Comprehensive linting rules and configuration
- **.goreleaser.yml**: GoReleaser configuration for advanced releases

### Build & Release
- **CHANGELOG.md**: Version history and release notes
- **Makefile**: Build automation and development tasks
- **go.mod/go.sum**: Go module dependencies and checksums

### GitHub Configuration
- **.github/labels.json**: Repository labels for issue/PR management
- **.github/act_secrets**: Template for local testing secrets
- **GitHub Templates**: Issue/PR templates for consistent reporting

### Docker
- **docker/Dockerfile**: Multi-stage container build definition
- **docker/.dockerignore**: Build context exclusions for smaller images
- **docker/README.md**: Container usage and deployment guide

### Scripts
- **scripts/prepare-release.sh**: Automated release preparation
- **scripts/setup-git-hooks.sh**: Git hooks installation
- **scripts/sync-labels.sh**: GitHub labels synchronization
- **scripts/update-homebrew.sh**: Homebrew formula maintenance
- **scripts/local-ci-check.sh**: Local CI compatibility validation

## Package Organization

### Public API (`/`)
- **gdl.go**: Main library entry point providing the public API
- **gdl_test.go**: Public API tests

### Command Line (`cmd/gdl/`)
- **main.go**: CLI application with all command-line features
- **main_test.go**: CLI functionality tests

### Internal Packages (`internal/`)
Private implementation details not exposed to library users:

- **core/**: Core download engine and orchestration
- **concurrent/**: Manages parallel downloads and chunk distribution
- **network/**: HTTP client, connectivity checks, and diagnostics
- **storage/**: File system operations and disk space management
- **retry/**: Retry strategies and backoff algorithms
- **resume/**: Download resume and partial file handling
- **recovery/**: Error recovery mechanisms
- **testing/**: Shared test utilities and mocks

### Public Packages (`pkg/`)
Exported packages that can be used by library consumers:

- **config/**: Configuration structures and validation
- **errors/**: Error types, codes, and handling
- **events/**: Event system for download lifecycle events
- **middleware/**: Middleware system for request/response processing
- **plugin/**: Plugin system interfaces and management
- **progress/**: Progress tracking interfaces and callbacks
- **protocols/**: Protocol registry and handler management
- **storage/**: Storage backend management
- **types/**: Common type definitions
- **ui/**: Terminal utilities and interactive features
- **validation/**: Input validation utilities
- **help/**: Help text generation and formatting

## File Naming Conventions

- **`*_test.go`**: Test files for the corresponding source file
- **`doc.go`**: Package documentation (if separate from main file)
- **`example_*.go`**: Example code for documentation
- **`mock_*.go`**: Mock implementations for testing
- **`benchmark_*.go`**: Benchmark tests

## Build Artifacts

The following are generated during build/test:

```
gdl                    # CLI binary (from go build)
coverage.out            # Test coverage report
coverage.html           # HTML coverage report
benchmark.txt           # Benchmark results
*.test                  # Test binaries
```

## Configuration Files

- **`.gitignore`**: Git ignore patterns
- **`.golangci.yml`**: Linter configuration
- **`go.mod`/`go.sum`**: Go module files
- **`.claude/CLAUDE.md`**: Claude Code development instructions

## Scripts

- **`scripts/local-ci-check.sh`**: Local CI compatibility validation script
- **`Makefile`**: Build automation and development tasks


## GitHub Workflows

Located in `.github/workflows/`:

- **`main.yml`**: Orchestrates all CI/CD workflows
- **`unit-tests.yml`**: Runs unit tests with coverage
- **`integration-tests.yml`**: Runs integration tests
- **`lint.yml`**: Code quality and linting
- **`security.yml`**: Security scanning
- **`cross-platform.yml`**: Multi-platform testing
- **`benchmark.yml`**: Performance benchmarking
- **`release.yml`**: Automated releases

## Module Structure

The project follows Go module best practices:

- Single module at root (`github.com/forest6511/gdl`)
- Public API at module root
- Internal packages hidden from consumers
- Versioned releases following semantic versioning