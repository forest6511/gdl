# Scenario-Based Guides

Real-world usage patterns for gdl. Each guide focuses on a specific use case with working examples.

## Quick Navigation

| Scenario | Description | Difficulty |
|----------|-------------|------------|
| [CDN Large File Delivery](cdn-delivery.md) | Download large files from CDNs efficiently | Beginner |
| [CI/CD Artifact Fetching](ci-artifacts.md) | Integrate gdl into CI/CD pipelines | Intermediate |
| [IoT Delta Updates](iot-updates.md) | Bandwidth-efficient updates for IoT devices | Intermediate |
| [Batch Downloads](batch-downloads.md) | Download multiple files efficiently | Beginner |
| [API Integration](api-integration.md) | Use gdl as a library in your Go application | Intermediate |

## Getting Started

Before diving into scenarios, make sure you have gdl installed:

```bash
# Install via Go (recommended)
go install github.com/forest6511/gdl/cmd/gdl@latest

# Or via Homebrew (macOS/Linux)
brew install forest6511/tap/gdl

# Verify installation
gdl --version
```

## Common Patterns

### Basic Download
```bash
gdl https://example.com/file.zip
```

### Resume Interrupted Download
```bash
gdl --resume https://example.com/large-file.zip
```

### Rate-Limited Download
```bash
gdl --max-rate 1MB/s https://example.com/file.zip
```

### Download with Custom Output
```bash
gdl -o ~/Downloads/myfile.zip https://example.com/file.zip
```

## Need Help?

- [CLI Reference](../CLI_REFERENCE.md) - Complete command reference
- [API Reference](../API_REFERENCE.md) - Library documentation
- [GitHub Discussions](https://github.com/forest6511/gdl/discussions) - Ask questions
