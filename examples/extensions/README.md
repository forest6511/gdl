# Extension Examples

This directory contains examples of how to extend godl's functionality through various extension points.

## Available Examples

### Database Protocol Handler
- **Location**: `database-protocol/`
- **Description**: Custom protocol handler for `database://` URLs that exports SQL query results as JSON
- **Features**: SQL query execution, JSON streaming, connection management

### Distributed Storage Backend
- **Location**: `distributed-storage/`
- **Description**: Distributed storage backend using multiple storage nodes with replication
- **Features**: Consistent hashing, replication, fault tolerance

### Analytics Event Handler
- **Location**: `analytics-events/`
- **Description**: Event handler that collects download analytics and sends them to analytics services
- **Features**: Event buffering, batch sending, metrics collection

### Rate Limiting Middleware
- **Location**: `rate-limiting/`
- **Description**: Middleware for rate limiting download requests
- **Features**: Token bucket algorithm, configurable limits, request queuing

### Custom Progress Formatters
- **Location**: `progress-formatters/`
- **Description**: Custom progress display formatters including JSON, emoji, and terminal formats
- **Features**: Multiple output formats, customizable templates

### Machine Learning Optimizer
- **Location**: `ml-optimizer/`
- **Description**: ML-based download optimization using historical data
- **Features**: Parameter prediction, performance optimization, learning from history

## Getting Started

Each example includes:
- Complete implementation code
- Build instructions
- Usage examples
- Integration tests
- Documentation

### Building Extensions

```bash
cd examples/extensions/database-protocol
go build -buildmode=plugin -o database.so
```

### Testing Extensions

```bash
cd examples/extensions/database-protocol
go test ./...
```

### Integration with godl

```go
// Load custom extension
handler := NewDatabaseHandler(url)
downloader.RegisterProtocol(handler)

// Use with custom URL
err := downloader.Download(ctx, "database://localhost/mydb?query=SELECT * FROM users", "users.json", nil)
```

## Extension Development Guidelines

1. **Follow Interface Contracts**: Implement all required methods completely
2. **Handle Errors Gracefully**: Provide meaningful error messages and recovery options
3. **Resource Management**: Properly clean up resources in Close() methods
4. **Thread Safety**: Ensure concurrent access safety where applicable
5. **Testing**: Include comprehensive unit and integration tests
6. **Documentation**: Provide clear usage examples and API documentation

## Contributing

When adding new extension examples:

1. Create a new subdirectory
2. Include complete implementation
3. Add comprehensive tests
4. Update this README
5. Follow the existing patterns

For more information on extending godl, see the [Extension Guide](../../docs/EXTENDING.md).