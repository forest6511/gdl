# GitHub Actions Workflows

This directory contains GitHub Actions workflow configurations for the godl project.

## Workflow Structure

The CI/CD pipeline is split into multiple specialized workflows for better maintainability and parallel execution:

### Core Workflows

1. **main.yml** - Main orchestrator workflow that calls other workflows
   - Triggers: Push to main/develop/feature branches, Pull requests
   - Coordinates all other workflows

2. **unit-tests.yml** - Unit test execution with coverage
   - Runs tests in parallel groups for faster feedback
   - Generates and uploads coverage reports to Codecov
   - Test groups:
     - Core Library
     - Internal Core
     - CLI Commands
     - Storage & Network
     - Retry & Resume

3. **integration-tests.yml** - Integration and end-to-end tests
   - Tests with real HTTP endpoints
   - CLI functionality tests
   - Example program validation

4. **lint.yml** - Code quality and linting checks
   - golangci-lint
   - go vet
   - go mod tidy verification
   - gofmt formatting checks

5. **security.yml** - Security scanning
   - gosec security scanner
   - staticcheck analysis
   - govulncheck for known vulnerabilities
   - Runs weekly and on main/develop branches

6. **cross-platform.yml** - Multi-platform compatibility
   - Tests on Ubuntu, Windows, macOS
   - Multiple Go versions (1.23, 1.24)
   - Binary build verification

7. **benchmark.yml** - Performance benchmarks
   - Runs on main branch and PRs
   - Compares benchmark results for PRs
   - Tracks performance over time

### Specialized Workflows

8. **release.yml** - Release automation
   - Triggers on version tags (v*)
   - Builds binaries for multiple platforms
   - Creates GitHub releases with artifacts


## Workflow Dependencies

```
main.yml
├── quick-checks (inline)
├── lint.yml
├── unit-tests.yml
├── integration-tests.yml
├── security.yml (main/develop only)
├── cross-platform.yml (PR/main only)
└── benchmark.yml (main only)
```

## Triggering Workflows

- **Push to main/develop**: All workflows except cross-platform (main only)
- **Push to feature branches**: Core workflows (lint, unit-tests, integration)
- **Pull requests**: All core workflows + cross-platform
- **Version tags**: Release workflow
- **Schedule**: Security workflow (weekly)
- **Manual**: All workflows support workflow_dispatch

## Optimizations Applied

### Performance Improvements
1. **Built-in Go caching**: Uses `cache: true` in setup-go action
2. **Reduced matrix complexity**: Selective OS/version combinations
3. **Parallel job execution**: All tests run concurrently
4. **Conditional execution**: Expensive jobs only on specific branches

### Security Enhancements
1. **Multiple security tools**: Gosec + staticcheck
2. **Dependency vulnerability scanning**: Via golangci-lint
3. **Token protection**: Optional Codecov token usage

### Reliability Features
1. **Race detection**: All tests run with `-race` flag
2. **Timeout protection**: Prevents hanging builds
3. **Artifact preservation**: Build outputs stored for troubleshooting
4. **Comprehensive error handling**: Multiple failure scenarios covered

## Configuration Files

- `.golangci.yml` - Comprehensive linting configuration
- This workflow file - Complete CI/CD pipeline
- `examples/` - Integration test validation

## Usage

The pipeline runs automatically on:
- **Pull Requests** to `main` and `develop`
- **Pushes** to `main`, `develop`, and `feature/**` branches
- **Tag pushes** (triggers build artifacts)

### Manual Workflow Dispatch
Not currently enabled, but can be added if needed for manual testing.

### Branch Protection
Recommended branch protection rules:
- Require status checks to pass
- Require up-to-date branches
- Require review before merging

## Monitoring and Maintenance

### Regular Updates Required
1. **Go version updates** - Update `GO_VERSION` environment variable
2. **Action version updates** - Keep GitHub Actions up to date
3. **Linter updates** - golangci-lint and staticcheck versions
4. **Security scanner updates** - Gosec tool versions

### Troubleshooting
- **Failed lints**: Check `.golangci.yml` configuration
- **Test failures**: Review test logs for specific failures
- **Build failures**: Verify Go modules and dependencies
- **Security issues**: Address Gosec and staticcheck findings

### Performance Considerations
- Average pipeline duration: ~8-12 minutes
- Parallel execution reduces total time
- Artifact storage: ~50MB per build matrix
- Bandwidth usage: Moderate with caching

## Environment Variables

- `GO_VERSION`: Default Go version for builds (currently 1.24)
- `CODECOV_TOKEN`: Token for Codecov integration (secret)

## Concurrency Control

Each workflow uses concurrency groups to:
- Cancel in-progress runs when new commits are pushed
- Prevent duplicate runs for the same ref
- Save CI resources

## Permissions

Workflows use minimal required permissions:
- `contents: read` - Read repository code
- `pull-requests: write` - Comment on PRs
- `security-events: write` - Upload security scan results
- `packages: write` - Publish packages (release only)

## Maintenance

To modify workflows:
1. Edit the specific workflow file for targeted changes
2. Test changes in a feature branch
3. Monitor the Actions tab for execution results
4. Check job summaries for detailed information
