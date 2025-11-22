# Compatibility Policy

This document describes gdl's versioning policy, API stability guarantees, and deprecation process.

## Semantic Versioning

gdl follows [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH
```

- **MAJOR**: Breaking changes (API incompatibility)
- **MINOR**: New features (backwards compatible)
- **PATCH**: Bug fixes (backwards compatible)

## Current Version Status

| Version | Status | Support |
|---------|--------|---------|
| v1.x.x | **Stable** | Full support |
| v0.x.x | Development | Limited support |

## API Stability Guarantees

### Stable APIs (v1.0+)

Once gdl reaches v1.0, the following guarantees apply:

#### Public API (pkg/)
- **Full Compatibility**: No breaking changes within a major version
- **Location**: `github.com/forest6511/gdl` and `github.com/forest6511/gdl/pkg/*`
- **Includes**:
  - All exported functions, types, and constants
  - Function signatures and return types
  - Struct field types and tags
  - Interface definitions

#### CLI Interface
- **Command Compatibility**: Commands and flags remain stable
- **Output Format**: Default output format preserved
- **Exit Codes**: Exit code meanings unchanged

### Internal APIs (internal/)

- **No Guarantees**: May change at any time
- **Location**: `github.com/forest6511/gdl/internal/*`
- **Warning**: Do not import from `internal/` packages

### Experimental Features

Features marked as experimental:
- May change or be removed in minor versions
- Are clearly marked in documentation
- Should not be used in production

## Breaking Changes

### What Constitutes a Breaking Change

- Removing or renaming exported functions, types, or constants
- Changing function signatures (parameters, return types)
- Changing struct field types
- Changing interface definitions
- Changing default behavior significantly
- Removing CLI commands or flags
- Changing exit codes

### What Is NOT a Breaking Change

- Adding new functions, types, or constants
- Adding new optional parameters (via options structs)
- Adding new struct fields
- Adding new CLI commands or flags
- Bug fixes (even if they change incorrect behavior)
- Performance improvements
- Internal implementation changes

## Deprecation Process

### Timeline

1. **Announcement** (Minor version N): Feature marked deprecated
2. **Warning Period** (Versions N to N+2): Deprecation warnings in logs
3. **Removal** (Major version N+1): Feature removed

### Deprecation Markers

```go
// Deprecated: Use NewFunction instead. Will be removed in v2.0.
func OldFunction() {}
```

### Finding Deprecated APIs

```bash
# Check for deprecation warnings in your code
go doc -all github.com/forest6511/gdl | grep -i deprecated
```

## Communication

### Release Notes

All changes are documented in:
- [CHANGELOG.md](../CHANGELOG.md) - Version history
- [GitHub Releases](https://github.com/forest6511/gdl/releases) - Release notes

### Breaking Change Announcements

Breaking changes are announced:
- At least one minor version in advance
- In the CHANGELOG with migration guidance
- Via GitHub release notes

### Migration Guides

Major version upgrades include migration guides:
- Step-by-step upgrade instructions
- Code examples for API changes
- Automated migration tools when possible

## Support Policy

### Active Support

| Version Type | Support Duration |
|--------------|------------------|
| Latest major | Full support |
| Previous major | Security fixes for 12 months |
| Older versions | Community support only |

### What "Support" Means

- **Full Support**: Bug fixes, security patches, new features
- **Security Fixes**: Critical security patches only
- **Community Support**: Issues addressed by community, no official patches

## Go Version Compatibility

### Minimum Go Version

- gdl supports the **two most recent Go releases**
- Current minimum: Go 1.24

### Go Version Policy

- New Go version support added in minor releases
- Go version requirements changed only in minor releases
- Deprecation announced one minor version before removal

## Platform Support

### Supported Platforms

| OS | Architecture | Support Level |
|----|--------------|---------------|
| Linux | amd64 | Full |
| Linux | arm64 | Full |
| Linux | arm | Full |
| macOS | amd64 | Full |
| macOS | arm64 | Full |
| Windows | amd64 | Full |

### Platform-Specific Features

Some features are platform-specific:
- **Zero-copy I/O**: Linux only (uses sendfile)
- **Plugin system**: Linux and macOS only (Go limitation)

## Best Practices for Users

### Stay Updated

```bash
# Check for updates
go list -m -u github.com/forest6511/gdl

# Update to latest
go get -u github.com/forest6511/gdl
```

### Pin Dependencies

```go
// go.mod
require github.com/forest6511/gdl v1.4.0
```

### Watch for Deprecations

```bash
# Build with deprecation warnings
go build -gcflags="-d=checkptr" ./...
```

### Test Before Upgrading

```bash
# Test with new version before committing
go get github.com/forest6511/gdl@latest
go test ./...
```

## Questions?

- [GitHub Discussions](https://github.com/forest6511/gdl/discussions) - Ask questions
- [GitHub Issues](https://github.com/forest6511/gdl/issues) - Report issues
