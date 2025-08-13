# Extending godl

This guide covers advanced extension points and customization options for the godl download tool.

## Table of Contents

1. [Overview](#overview)
2. [Extension Points](#extension-points)
3. [Custom Protocol Handlers](#custom-protocol-handlers)
4. [Storage Backend Implementation](#storage-backend-implementation)
5. [Middleware Development](#middleware-development)
6. [Event System](#event-system)
7. [Configuration Extensions](#configuration-extensions)
8. [UI Customization](#ui-customization)
9. [Performance Optimization](#performance-optimization)
10. [Advanced Examples](#advanced-examples)

## Overview

godl provides multiple extension points to customize and enhance its functionality:

```
┌─────────────────────────────────────────────────────────────┐
│                      Extension Layer                       │
├─────────────────────────────────────────────────────────────┤
│  Protocols │ Storage │ Middleware │ UI │ Events │ Config   │
├─────────────────────────────────────────────────────────────┤
│                      godl Core Engine                      │
├─────────────────────────────────────────────────────────────┤
│                   Plugin System                            │
└─────────────────────────────────────────────────────────────┘
```

## Extension Points

### Core Extension Points

1. **Protocol Layer**: Add support for new protocols
2. **Storage Layer**: Implement custom storage backends
3. **Middleware Layer**: Add request/response processing
4. **Event Layer**: Hook into download lifecycle events
5. **UI Layer**: Customize user interface components
6. **Configuration Layer**: Add custom configuration sources

### Plugin Integration

All extensions can be packaged as plugins or integrated directly into the core:

```go
// Direct integration
downloader.AddProtocolHandler("custom", customHandler)

// Plugin integration
downloader.UsePlugin(customProtocolPlugin)
```

## Custom Protocol Handlers

### Protocol Interface

Implement the `ProtocolHandler` interface to support custom protocols:

```go
type ProtocolHandler interface {
    // Download content from the protocol source
    Download(ctx context.Context, destination io.Writer) error
    
    // Get file information without downloading
    GetInfo(ctx context.Context) (*FileInfo, error)
    
    // Check if protocol supports resume
    SupportsResume() bool
    
    // Check if protocol supports range requests
    SupportsRangeRequests() bool
    
    // Get download stream (for streaming downloads)
    GetStream(ctx context.Context) (io.ReadCloser, error)
}
```

### Example: Custom Database Protocol

```go
// DatabaseProtocolHandler handles database:// URLs
type DatabaseProtocolHandler struct {
    connectionString string
    query           string
    db              *sql.DB
}

func NewDatabaseHandler(url *url.URL) (*DatabaseProtocolHandler, error) {
    // Parse database://host/database?query=SELECT...
    host := url.Host
    database := strings.TrimPrefix(url.Path, "/")
    query := url.Query().Get("query")
    
    if query == "" {
        return nil, fmt.Errorf("query parameter is required")
    }
    
    connectionString := fmt.Sprintf("host=%s dbname=%s", host, database)
    
    return &DatabaseProtocolHandler{
        connectionString: connectionString,
        query:           query,
    }, nil
}

func (h *DatabaseProtocolHandler) Download(ctx context.Context, destination io.Writer) error {
    if h.db == nil {
        db, err := sql.Open("postgres", h.connectionString)
        if err != nil {
            return fmt.Errorf("failed to connect to database: %w", err)
        }
        h.db = db
    }
    
    rows, err := h.db.QueryContext(ctx, h.query)
    if err != nil {
        return fmt.Errorf("query execution failed: %w", err)
    }
    defer rows.Close()
    
    // Convert query results to JSON
    encoder := json.NewEncoder(destination)
    
    // Get column names
    columns, err := rows.Columns()
    if err != nil {
        return err
    }
    
    for rows.Next() {
        // Create a slice of interface{} to hold the values
        values := make([]interface{}, len(columns))
        scanArgs := make([]interface{}, len(values))
        for i := range values {
            scanArgs[i] = &values[i]
        }
        
        if err := rows.Scan(scanArgs...); err != nil {
            return err
        }
        
        // Create a map for this row
        rowMap := make(map[string]interface{})
        for i, col := range columns {
            val := values[i]
            if b, ok := val.([]byte); ok {
                val = string(b)
            }
            rowMap[col] = val
        }
        
        if err := encoder.Encode(rowMap); err != nil {
            return err
        }
    }
    
    return rows.Err()
}

func (h *DatabaseProtocolHandler) GetInfo(ctx context.Context) (*FileInfo, error) {
    return &FileInfo{
        URL:         h.connectionString,
        Size:        -1, // Unknown size for database queries
        ContentType: "application/json",
        SupportsResume: false,
    }, nil
}

func (h *DatabaseProtocolHandler) SupportsResume() bool {
    return false // Database queries don't support resume
}

func (h *DatabaseProtocolHandler) SupportsRangeRequests() bool {
    return false
}

func (h *DatabaseProtocolHandler) GetStream(ctx context.Context) (io.ReadCloser, error) {
    r, w := io.Pipe()
    
    go func() {
        defer w.Close()
        if err := h.Download(ctx, w); err != nil {
            w.CloseWithError(err)
        }
    }()
    
    return r, nil
}
```

### Protocol Registration

```go
// Register protocol handler
func init() {
    protocols.Register("database", func(url *url.URL) (protocols.Handler, error) {
        return NewDatabaseHandler(url)
    })
}

// Or register via plugin
type DatabasePlugin struct{}

func (p *DatabasePlugin) SupportsScheme(scheme string) bool {
    return scheme == "database"
}

func (p *DatabasePlugin) CreateHandler(ctx context.Context, url *url.URL) (ProtocolHandler, error) {
    return NewDatabaseHandler(url)
}
```

## Storage Backend Implementation

### Storage Interface

Implement custom storage backends:

```go
type StorageBackend interface {
    // Store data with the given key
    Store(ctx context.Context, key string, data io.Reader, metadata *StorageMetadata) error
    
    // Retrieve data by key
    Retrieve(ctx context.Context, key string) (io.ReadCloser, *StorageMetadata, error)
    
    // Delete data by key
    Delete(ctx context.Context, key string) error
    
    // Check if key exists
    Exists(ctx context.Context, key string) (bool, error)
    
    // List keys with optional prefix
    List(ctx context.Context, prefix string, limit int) (*StorageList, error)
    
    // Get storage statistics
    GetStats(ctx context.Context) (*StorageStats, error)
    
    // Close and cleanup
    Close() error
}
```

### Example: Distributed Storage Backend

```go
// DistributedStorageBackend implements distributed storage using multiple backends
type DistributedStorageBackend struct {
    backends    []StorageBackend
    replication int
    hasher      hash.Hash
    mu          sync.RWMutex
}

func NewDistributedStorage(backends []StorageBackend, replication int) *DistributedStorageBackend {
    return &DistributedStorageBackend{
        backends:    backends,
        replication: replication,
        hasher:      sha256.New(),
    }
}

func (d *DistributedStorageBackend) Store(ctx context.Context, key string, data io.Reader, metadata *StorageMetadata) error {
    // Buffer the data for multiple writes
    var buf bytes.Buffer
    teeReader := io.TeeReader(data, &buf)
    
    // Calculate hash for consistent distribution
    _, err := io.Copy(d.hasher, teeReader)
    if err != nil {
        return err
    }
    
    hashSum := d.hasher.Sum(nil)
    d.hasher.Reset()
    
    // Select backends based on hash
    selectedBackends := d.selectBackends(hashSum)
    
    // Store to multiple backends for replication
    var wg sync.WaitGroup
    errCh := make(chan error, len(selectedBackends))
    
    for _, backend := range selectedBackends {
        wg.Add(1)
        go func(b StorageBackend) {
            defer wg.Done()
            
            reader := bytes.NewReader(buf.Bytes())
            if err := b.Store(ctx, key, reader, metadata); err != nil {
                errCh <- err
            }
        }(backend)
    }
    
    wg.Wait()
    close(errCh)
    
    // Check for errors
    var errors []error
    for err := range errCh {
        errors = append(errors, err)
    }
    
    // Allow some failures based on replication factor
    maxFailures := len(selectedBackends) - (d.replication/2 + 1)
    if len(errors) > maxFailures {
        return fmt.Errorf("too many storage failures: %d/%d", len(errors), len(selectedBackends))
    }
    
    return nil
}

func (d *DistributedStorageBackend) Retrieve(ctx context.Context, key string) (io.ReadCloser, *StorageMetadata, error) {
    // Try backends in order until successful
    hashSum := d.hashKey(key)
    backends := d.selectBackends(hashSum)
    
    for _, backend := range backends {
        reader, metadata, err := backend.Retrieve(ctx, key)
        if err == nil {
            return reader, metadata, nil
        }
        
        // Log error but continue trying other backends
        log.Printf("Failed to retrieve from backend: %v", err)
    }
    
    return nil, nil, fmt.Errorf("key not found in any backend: %s", key)
}

func (d *DistributedStorageBackend) selectBackends(hashSum []byte) []StorageBackend {
    // Use consistent hashing to select backends
    startIndex := int(binary.BigEndian.Uint32(hashSum)) % len(d.backends)
    
    selected := make([]StorageBackend, 0, d.replication)
    for i := 0; i < d.replication && i < len(d.backends); i++ {
        index := (startIndex + i) % len(d.backends)
        selected = append(selected, d.backends[index])
    }
    
    return selected
}

func (d *DistributedStorageBackend) hashKey(key string) []byte {
    d.hasher.Write([]byte(key))
    sum := d.hasher.Sum(nil)
    d.hasher.Reset()
    return sum
}
```

### Storage Plugin Integration

```go
type DistributedStoragePlugin struct {
    backend *DistributedStorageBackend
}

func (p *DistributedStoragePlugin) Name() string {
    return "distributed-storage"
}

func (p *DistributedStoragePlugin) Version() string {
    return "1.0.0"
}

func (p *DistributedStoragePlugin) Init(config map[string]interface{}) error {
    // Parse configuration for backend endpoints
    backends := make([]StorageBackend, 0)
    
    if endpoints, ok := config["endpoints"].([]interface{}); ok {
        for _, endpoint := range endpoints {
            if ep, ok := endpoint.(string); ok {
                backend, err := storage.NewBackend(ep)
                if err != nil {
                    return err
                }
                backends = append(backends, backend)
            }
        }
    }
    
    replication := 3 // default
    if rep, ok := config["replication"].(int); ok {
        replication = rep
    }
    
    p.backend = NewDistributedStorage(backends, replication)
    return nil
}
```

## Middleware Development

### Middleware Interface

Create request/response processing middleware:

```go
type Middleware interface {
    // Process request before download
    ProcessRequest(ctx context.Context, req *DownloadRequest) (*DownloadRequest, error)
    
    // Process response after download
    ProcessResponse(ctx context.Context, resp *DownloadResponse) (*DownloadResponse, error)
    
    // Handle errors
    HandleError(ctx context.Context, err error) error
}
```

### Example: Rate Limiting Middleware

```go
// RateLimitingMiddleware implements request rate limiting
type RateLimitingMiddleware struct {
    limiter    *rate.Limiter
    maxBurst   int
    refillRate rate.Limit
}

func NewRateLimitingMiddleware(requestsPerSecond float64, maxBurst int) *RateLimitingMiddleware {
    return &RateLimitingMiddleware{
        limiter:    rate.NewLimiter(rate.Limit(requestsPerSecond), maxBurst),
        maxBurst:   maxBurst,
        refillRate: rate.Limit(requestsPerSecond),
    }
}

func (m *RateLimitingMiddleware) ProcessRequest(ctx context.Context, req *DownloadRequest) (*DownloadRequest, error) {
    // Wait for rate limiter
    err := m.limiter.Wait(ctx)
    if err != nil {
        return nil, fmt.Errorf("rate limiting failed: %w", err)
    }
    
    // Add rate limiting headers to request
    if req.Headers == nil {
        req.Headers = make(map[string]string)
    }
    
    req.Headers["X-Rate-Limit"] = fmt.Sprintf("%.2f", float64(m.refillRate))
    req.Headers["X-Rate-Limit-Burst"] = fmt.Sprintf("%d", m.maxBurst)
    
    return req, nil
}

func (m *RateLimitingMiddleware) ProcessResponse(ctx context.Context, resp *DownloadResponse) (*DownloadResponse, error) {
    // Add rate limit info to response
    resp.Metadata["rate_limited"] = true
    resp.Metadata["remaining_tokens"] = m.limiter.Tokens()
    
    return resp, nil
}

func (m *RateLimitingMiddleware) HandleError(ctx context.Context, err error) error {
    // Check if error is due to rate limiting
    if errors.Is(err, context.DeadlineExceeded) {
        return fmt.Errorf("request timeout - possibly due to rate limiting: %w", err)
    }
    
    return err
}
```

### Middleware Chain

```go
// MiddlewareChain manages multiple middleware
type MiddlewareChain struct {
    middlewares []Middleware
}

func NewMiddlewareChain() *MiddlewareChain {
    return &MiddlewareChain{
        middlewares: make([]Middleware, 0),
    }
}

func (c *MiddlewareChain) Add(middleware Middleware) {
    c.middlewares = append(c.middlewares, middleware)
}

func (c *MiddlewareChain) ProcessRequest(ctx context.Context, req *DownloadRequest) (*DownloadRequest, error) {
    for _, middleware := range c.middlewares {
        var err error
        req, err = middleware.ProcessRequest(ctx, req)
        if err != nil {
            return nil, err
        }
    }
    return req, nil
}

func (c *MiddlewareChain) ProcessResponse(ctx context.Context, resp *DownloadResponse) (*DownloadResponse, error) {
    // Process in reverse order for response
    for i := len(c.middlewares) - 1; i >= 0; i-- {
        var err error
        resp, err = c.middlewares[i].ProcessResponse(ctx, resp)
        if err != nil {
            return nil, err
        }
    }
    return resp, nil
}
```

## Event System

### Event Interface

Hook into download lifecycle events:

```go
type EventHandler interface {
    // Handle download events
    HandleEvent(ctx context.Context, event *Event) error
    
    // Get supported event types
    SupportedEvents() []EventType
}

type Event struct {
    Type      EventType
    Timestamp time.Time
    Data      map[string]interface{}
    Context   context.Context
}

type EventType string

const (
    EventDownloadStart    EventType = "download.start"
    EventDownloadProgress EventType = "download.progress"
    EventDownloadComplete EventType = "download.complete"
    EventDownloadError    EventType = "download.error"
    EventChunkStart       EventType = "chunk.start"
    EventChunkComplete    EventType = "chunk.complete"
)
```

### Example: Analytics Event Handler

```go
// AnalyticsEventHandler collects download analytics
type AnalyticsEventHandler struct {
    collector AnalyticsCollector
    buffer    []*AnalyticsEvent
    mu        sync.Mutex
}

type AnalyticsEvent struct {
    EventType    string            `json:"event_type"`
    Timestamp    time.Time         `json:"timestamp"`
    URL          string            `json:"url"`
    FileSize     int64             `json:"file_size,omitempty"`
    Duration     time.Duration     `json:"duration,omitempty"`
    Speed        int64             `json:"speed,omitempty"`
    ErrorMessage string            `json:"error_message,omitempty"`
    Metadata     map[string]string `json:"metadata,omitempty"`
}

func (h *AnalyticsEventHandler) HandleEvent(ctx context.Context, event *Event) error {
    analyticsEvent := &AnalyticsEvent{
        EventType: string(event.Type),
        Timestamp: event.Timestamp,
    }
    
    // Extract common data
    if url, ok := event.Data["url"].(string); ok {
        analyticsEvent.URL = url
    }
    
    if fileSize, ok := event.Data["file_size"].(int64); ok {
        analyticsEvent.FileSize = fileSize
    }
    
    // Handle specific event types
    switch event.Type {
    case EventDownloadComplete:
        if duration, ok := event.Data["duration"].(time.Duration); ok {
            analyticsEvent.Duration = duration
        }
        if speed, ok := event.Data["speed"].(int64); ok {
            analyticsEvent.Speed = speed
        }
        
    case EventDownloadError:
        if err, ok := event.Data["error"].(error); ok {
            analyticsEvent.ErrorMessage = err.Error()
        }
    }
    
    // Add to buffer
    h.mu.Lock()
    h.buffer = append(h.buffer, analyticsEvent)
    
    // Flush buffer if it's full
    if len(h.buffer) >= 100 {
        events := h.buffer
        h.buffer = nil
        h.mu.Unlock()
        
        // Send events asynchronously
        go h.flushEvents(events)
    } else {
        h.mu.Unlock()
    }
    
    return nil
}

func (h *AnalyticsEventHandler) SupportedEvents() []EventType {
    return []EventType{
        EventDownloadStart,
        EventDownloadProgress,
        EventDownloadComplete,
        EventDownloadError,
    }
}

func (h *AnalyticsEventHandler) flushEvents(events []*AnalyticsEvent) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := h.collector.SendBatch(ctx, events); err != nil {
        log.Printf("Failed to send analytics events: %v", err)
    }
}
```

### Event Emitter Integration

```go
// Integrate with godl's event system
func (d *Downloader) setupEventHandlers() {
    // Add analytics handler
    analyticsHandler := NewAnalyticsEventHandler(analyticsCollector)
    d.eventEmitter.AddHandler(analyticsHandler)
    
    // Add logging handler
    loggingHandler := NewLoggingEventHandler(logger)
    d.eventEmitter.AddHandler(loggingHandler)
    
    // Add metrics handler
    metricsHandler := NewMetricsEventHandler(metricsClient)
    d.eventEmitter.AddHandler(metricsHandler)
}
```

## Configuration Extensions

### Custom Configuration Sources

Extend configuration loading:

```go
type ConfigurationSource interface {
    // Load configuration from source
    LoadConfig(ctx context.Context) (*Configuration, error)
    
    // Watch for configuration changes
    Watch(ctx context.Context, callback func(*Configuration)) error
    
    // Get source name
    Name() string
}

// Example: Consul configuration source
type ConsulConfigSource struct {
    client *consul.Client
    key    string
}

func (c *ConsulConfigSource) LoadConfig(ctx context.Context) (*Configuration, error) {
    kv, _, err := c.client.KV().Get(c.key, nil)
    if err != nil {
        return nil, err
    }
    
    if kv == nil {
        return nil, fmt.Errorf("configuration key not found: %s", c.key)
    }
    
    var config Configuration
    err = json.Unmarshal(kv.Value, &config)
    if err != nil {
        return nil, fmt.Errorf("failed to parse configuration: %w", err)
    }
    
    return &config, nil
}

func (c *ConsulConfigSource) Watch(ctx context.Context, callback func(*Configuration)) error {
    plan, err := watch.Parse(map[string]interface{}{
        "type": "key",
        "key":  c.key,
    })
    if err != nil {
        return err
    }
    
    plan.Handler = func(idx uint64, raw interface{}) {
        if kv, ok := raw.(*consul.KVPair); ok && kv != nil {
            var config Configuration
            if err := json.Unmarshal(kv.Value, &config); err == nil {
                callback(&config)
            }
        }
    }
    
    return plan.Run(c.client.Address())
}
```

## UI Customization

### Custom Formatters

Create custom output formatters:

```go
type OutputFormatter interface {
    // Format progress information
    FormatProgress(progress *Progress) string
    
    // Format download completion
    FormatCompletion(stats *DownloadStats) string
    
    // Format errors
    FormatError(err error) string
    
    // Format file information
    FormatFileInfo(info *FileInfo) string
}

// Example: JSON formatter
type JSONFormatter struct {
    pretty bool
}

func (f *JSONFormatter) FormatProgress(progress *Progress) string {
    data := map[string]interface{}{
        "type":             "progress",
        "bytes_downloaded": progress.BytesDownloaded,
        "total_size":       progress.TotalSize,
        "percentage":       progress.Percentage,
        "speed":           progress.Speed,
        "time_elapsed":    progress.TimeElapsed.Seconds(),
        "time_remaining":  progress.TimeRemaining.Seconds(),
    }
    
    return f.marshal(data)
}

func (f *JSONFormatter) FormatCompletion(stats *DownloadStats) string {
    data := map[string]interface{}{
        "type":            "completion",
        "bytes_downloaded": stats.BytesDownloaded,
        "duration":        stats.Duration.Seconds(),
        "average_speed":   stats.AverageSpeed,
        "retries":         stats.Retries,
    }
    
    return f.marshal(data)
}

func (f *JSONFormatter) marshal(data interface{}) string {
    var result []byte
    var err error
    
    if f.pretty {
        result, err = json.MarshalIndent(data, "", "  ")
    } else {
        result, err = json.Marshal(data)
    }
    
    if err != nil {
        return fmt.Sprintf(`{"error": "marshalling failed: %s"}`, err)
    }
    
    return string(result)
}
```

### Progress Bar Customization

```go
type ProgressBar interface {
    // Update progress
    Update(current, total int64, speed int64)
    
    // Finish progress display
    Finish()
    
    // Set custom template
    SetTemplate(template string)
}

// Custom progress bar with emojis
type EmojiProgressBar struct {
    width    int
    template string
}

func (p *EmojiProgressBar) Update(current, total int64, speed int64) {
    percentage := float64(current) / float64(total) * 100
    
    // Create emoji progress bar
    filled := int(percentage / 100.0 * float64(p.width))
    bar := strings.Repeat("🟩", filled) + strings.Repeat("⬜", p.width-filled)
    
    // Format with template
    output := fmt.Sprintf("%s %.1f%% %s %s/s 🚀", 
        bar, 
        percentage,
        formatBytes(current),
        formatBytes(speed))
    
    fmt.Printf("\r%s", output)
}
```

## Performance Optimization

### Custom Download Strategies

Implement optimized download strategies:

```go
type DownloadStrategy interface {
    // Plan the download strategy
    Plan(ctx context.Context, info *FileInfo) (*DownloadPlan, error)
    
    // Execute the download
    Execute(ctx context.Context, plan *DownloadPlan, destination io.Writer) error
    
    // Get strategy name
    Name() string
}

// Adaptive chunking strategy
type AdaptiveChunkingStrategy struct {
    initialChunkSize int64
    maxChunkSize     int64
    minChunkSize     int64
    speedThreshold   int64
}

func (s *AdaptiveChunkingStrategy) Plan(ctx context.Context, info *FileInfo) (*DownloadPlan, error) {
    chunks := make([]*ChunkPlan, 0)
    
    if !info.SupportsRangeRequests {
        // Single chunk download
        chunks = append(chunks, &ChunkPlan{
            Start: 0,
            End:   info.Size - 1,
            Size:  info.Size,
        })
    } else {
        // Calculate optimal chunk count based on file size
        chunkSize := s.calculateOptimalChunkSize(info.Size)
        
        for start := int64(0); start < info.Size; start += chunkSize {
            end := start + chunkSize - 1
            if end >= info.Size {
                end = info.Size - 1
            }
            
            chunks = append(chunks, &ChunkPlan{
                Start: start,
                End:   end,
                Size:  end - start + 1,
            })
        }
    }
    
    return &DownloadPlan{
        URL:        info.URL,
        TotalSize:  info.Size,
        Chunks:     chunks,
        Strategy:   s.Name(),
        Concurrent: len(chunks) > 1,
    }, nil
}

func (s *AdaptiveChunkingStrategy) calculateOptimalChunkSize(fileSize int64) int64 {
    // Adaptive algorithm based on file size
    if fileSize < 10*1024*1024 { // < 10MB
        return s.minChunkSize
    } else if fileSize > 1024*1024*1024 { // > 1GB
        return s.maxChunkSize
    } else {
        // Scale chunk size with file size
        ratio := float64(fileSize) / float64(1024*1024*1024)
        chunkSize := int64(float64(s.initialChunkSize) * (1 + ratio))
        
        if chunkSize > s.maxChunkSize {
            return s.maxChunkSize
        }
        if chunkSize < s.minChunkSize {
            return s.minChunkSize
        }
        
        return chunkSize
    }
}
```

## Advanced Examples

### Complete Custom Protocol Implementation

See the database protocol example in `examples/extensions/database-protocol/`.

### Distributed Storage Plugin

See the distributed storage implementation in `examples/extensions/distributed-storage/`.

### Machine Learning Download Optimizer

```go
// MLDownloadOptimizer uses machine learning to optimize download parameters
type MLDownloadOptimizer struct {
    model      *MLModel
    predictor  *PerformancePredictor
    history    *DownloadHistory
}

func (opt *MLDownloadOptimizer) OptimizeDownload(ctx context.Context, url string) (*OptimizationResult, error) {
    // Extract features from URL and historical data
    features := opt.extractFeatures(url)
    
    // Predict optimal parameters using ML model
    prediction, err := opt.model.Predict(features)
    if err != nil {
        return nil, err
    }
    
    return &OptimizationResult{
        ChunkSize:      prediction.OptimalChunkSize,
        Concurrency:    prediction.OptimalConcurrency,
        RetryStrategy:  prediction.OptimalRetryStrategy,
        Confidence:     prediction.Confidence,
    }, nil
}
```

### Custom Authentication Chain

```go
// AuthenticationChain handles multiple authentication methods
type AuthenticationChain struct {
    authenticators []Authenticator
    fallback       Authenticator
}

func (chain *AuthenticationChain) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
    var lastError error
    
    // Try each authenticator in order
    for _, auth := range chain.authenticators {
        resp, err := auth.Authenticate(ctx, req)
        if err == nil {
            return resp, nil
        }
        
        lastError = err
        
        // Check if we should continue or abort
        if isAuthError(err) && !isContinuableError(err) {
            break
        }
    }
    
    // Try fallback if available
    if chain.fallback != nil {
        return chain.fallback.Authenticate(ctx, req)
    }
    
    return nil, fmt.Errorf("all authentication methods failed, last error: %w", lastError)
}
```

## Integration Testing

Test your extensions:

```go
func TestCustomExtension_Integration(t *testing.T) {
    // Setup test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("test content"))
    }))
    defer server.Close()
    
    // Create downloader with custom extensions
    downloader := godl.NewDownloader()
    
    // Add custom protocol handler
    customHandler := &CustomProtocolHandler{}
    downloader.AddProtocolHandler("custom", customHandler)
    
    // Add custom middleware
    rateLimiter := NewRateLimitingMiddleware(10.0, 5)
    downloader.AddMiddleware(rateLimiter)
    
    // Test download
    ctx := context.Background()
    err := downloader.Download(ctx, "custom://test", "/tmp/test", nil)
    require.NoError(t, err)
}
```

## Resources

- [Plugin Development Guide](./PLUGIN_DEVELOPMENT.md)
- [API Reference](./API_REFERENCE.md)
- [Extension Examples](../examples/extensions/)
- [Performance Guide](./PERFORMANCE.md)

For plugin-specific development, see the [Plugin Development Guide](./PLUGIN_DEVELOPMENT.md).