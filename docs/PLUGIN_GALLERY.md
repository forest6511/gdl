# Plugin Gallery

A showcase of plugins and extensions for gdl. These examples demonstrate how to extend gdl's functionality for various use cases.

## Official Examples

### Authentication Plugins

#### OAuth2 Authentication
Authenticate downloads using OAuth2 tokens.

**Location:** `examples/plugins/auth/oauth2/`

```go
// Usage
gdl --plugin oauth2-auth https://api.example.com/files/document.pdf
```

**Features:**
- Automatic token refresh
- Multiple OAuth2 providers support
- Secure credential storage

[View Source](../examples/plugins/auth/oauth2/main.go) | [Documentation](PLUGIN_DEVELOPMENT.md#authentication-plugins)

---

### Storage Plugins

#### Google Cloud Storage (GCS)
Download directly to Google Cloud Storage.

**Location:** `examples/plugins/storage/gcs/`

```go
// Usage in code
downloader.UseStorage(gcs.NewStorage(bucketName))
```

**Features:**
- Direct upload to GCS buckets
- Resumable uploads
- Progress tracking

[View Source](../examples/plugins/storage/gcs/main.go) | [Documentation](PLUGIN_DEVELOPMENT.md#storage-plugins)

---

### Transform Plugins

#### Image Optimizer
Optimize images during download.

**Location:** `examples/plugins/transform/image-optimizer/`

```go
// Usage in code
downloader.UseTransform(imageopt.NewOptimizer(quality))
```

**Features:**
- Automatic format detection
- Quality optimization
- Size reduction

[View Source](../examples/plugins/transform/image-optimizer/main.go) | [Documentation](PLUGIN_DEVELOPMENT.md#transform-plugins)

---

## Plugin Types Overview

| Type | Purpose | Example Use Cases |
|------|---------|-------------------|
| **Auth** | Handle authentication | OAuth2, API keys, certificates |
| **Protocol** | Add download protocols | FTP, S3, SFTP, custom protocols |
| **Storage** | Custom storage backends | Cloud storage, databases |
| **Transform** | Process downloaded data | Compression, encryption, conversion |
| **Hook** | Lifecycle events | Logging, analytics, notifications |

## Creating Your Own Plugin

### Quick Start

1. **Choose a plugin type** from the table above
2. **Use the template** from `examples/plugins/`
3. **Implement the interface** for your plugin type
4. **Build and test** your plugin

### Plugin Interface Examples

#### Authentication Plugin

```go
package main

import (
    "context"
    "net/http"
)

type AuthPlugin interface {
    // Name returns the plugin name
    Name() string

    // Authenticate adds authentication to the request
    Authenticate(ctx context.Context, req *http.Request) error

    // Refresh refreshes credentials if needed
    Refresh(ctx context.Context) error
}

// Example implementation
type APIKeyAuth struct {
    apiKey string
}

func (a *APIKeyAuth) Name() string {
    return "api-key-auth"
}

func (a *APIKeyAuth) Authenticate(ctx context.Context, req *http.Request) error {
    req.Header.Set("Authorization", "Bearer "+a.apiKey)
    return nil
}

func (a *APIKeyAuth) Refresh(ctx context.Context) error {
    return nil // API keys don't need refresh
}
```

#### Storage Plugin

```go
package main

import (
    "context"
    "io"
)

type StoragePlugin interface {
    // Name returns the plugin name
    Name() string

    // Write writes data to storage
    Write(ctx context.Context, path string, reader io.Reader) error

    // Exists checks if a file exists
    Exists(ctx context.Context, path string) (bool, error)

    // Size returns the size of existing file (for resume)
    Size(ctx context.Context, path string) (int64, error)
}
```

#### Transform Plugin

```go
package main

import (
    "context"
    "io"
)

type TransformPlugin interface {
    // Name returns the plugin name
    Name() string

    // Transform processes the data stream
    Transform(ctx context.Context, reader io.Reader) (io.Reader, error)

    // ContentType returns the output content type (if changed)
    ContentType() string
}
```

## Community Plugins

> Want to add your plugin here? Submit a PR!

### How to Submit

1. Create your plugin following the [Plugin Development Guide](PLUGIN_DEVELOPMENT.md)
2. Add documentation with usage examples
3. Submit a PR to add it to this gallery

### Submission Requirements

- [ ] Clear documentation with usage examples
- [ ] Working example code
- [ ] Tests for core functionality
- [ ] License compatible with MIT

## Resources

- [Plugin Development Guide](PLUGIN_DEVELOPMENT.md) - Detailed development guide
- [Extending Guide](EXTENDING.md) - Extension points overview
- [API Reference](API_REFERENCE.md) - Core API documentation
- [Examples](../examples/) - Working code examples

## Need Help?

- [GitHub Discussions](https://github.com/forest6511/gdl/discussions) - Ask questions
- [GitHub Issues](https://github.com/forest6511/gdl/issues) - Report issues
