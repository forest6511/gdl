# VSCode Configuration for gdl

This directory contains VSCode workspace configuration for the gdl project.

## Files

### `settings.json`
- **Go language server settings**: Enables gopls for intelligent code completion
- **Linting**: Configured to use golangci-lint on save
- **Formatting**: Auto-format on save with gofmt -s
- **Testing**: Race detection enabled by default
- **File exclusions**: Hides build artifacts and temporary files
- **Language-specific settings**: 
  - Go: tabs, 4-space tab size
  - JSON/YAML: 2-space indentation
  - Markdown: 2-space, word wrap enabled
  - Makefile: tabs only

### `extensions.json`
Recommended extensions for Go development:
- **golang.go**: Official Go extension with gopls
- **ms-vscode.makefile-tools**: Makefile support
- **redhat.vscode-yaml**: YAML language support
- **davidanson.vscode-markdownlint**: Markdown linting
- **github.vscode-github-actions**: GitHub Actions workflow support
- **editorconfig.editorconfig**: EditorConfig support
- **streetsidesoftware.code-spell-checker**: Spell checking
- **timonwong.shellcheck**: Shell script linting
- **ms-vscode.vscode-docker**: Docker support

### `tasks.json`
Pre-configured tasks for common development operations:
- **go: build**: Build the CLI binary
- **go: test**: Run all tests
- **go: test with race**: Run tests with race detection
- **go: test coverage**: Generate coverage report
- **golangci-lint**: Run linter
- **make: ci-check**: Full CI validation locally
- **make: examples**: Test all examples
- **act: test main workflow**: Test GitHub Actions locally
- **docker: build**: Build Docker image
- **release: prepare**: Prepare a new release

### `launch.json`
Debug configurations:
- **Launch CLI**: Debug the CLI tool with arguments
- **Debug Current Test**: Debug specific test function
- **Debug All Tests in Package**: Debug all tests in current package
- **Debug Example**: Debug example programs
- **Attach to Process**: Attach debugger to running process

## Usage

### Quick Development
1. Open project in VSCode
2. Install recommended extensions when prompted
3. Use `Ctrl+Shift+P` (Cmd+Shift+P on macOS) → "Tasks: Run Task"
4. Select desired task (build, test, lint, etc.)

### Testing
- **F5**: Launch CLI with default args
- **Ctrl+F5**: Run without debugging
- Use "Debug Current Test" to debug specific test functions

### Building
- **Ctrl+Shift+P** → "Tasks: Run Task" → "go: build"
- Or use terminal: `make build`

### CI Validation
- **Ctrl+Shift+P** → "Tasks: Run Task" → "make: ci-check"
- Runs the same checks as GitHub Actions locally

## Settings Alignment

The VSCode settings are aligned with:
- **`.editorconfig`**: Consistent indentation and formatting
- **`.golangci.yml`**: Same linting rules as CI
- **`Makefile`**: Same build commands and targets
- **GitHub Actions**: Same Go version and tools

## Troubleshooting

### Go Extension Issues
1. **Ctrl+Shift+P** → "Go: Install/Update Tools"
2. Restart VSCode
3. Check Go version: `go version` (should be 1.23+)

### Linting Issues
1. Install golangci-lint: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
2. Check PATH includes `$GOPATH/bin`
3. Restart VSCode

### Test Debugging
1. Set breakpoints in test files
2. Use "Debug Current Test" configuration
3. Enter test function name when prompted

### Performance
- Large projects: Adjust `go.toolsManagement.autoUpdate` to `false`
- Disable features in `settings.json` if experiencing slowdowns
- Use `files.exclude` to hide unnecessary directories
