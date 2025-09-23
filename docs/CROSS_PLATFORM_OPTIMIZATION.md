# Cross-Platform Optimization Guide

## Overview

`gdl` implements sophisticated platform-specific optimizations to maximize download performance across different operating systems and architectures. The system automatically detects the runtime platform and applies optimal settings for buffer sizes, concurrency, connection pooling, and I/O strategies.

## Platform Detection

The platform detector analyzes:
- Operating System (Linux, macOS, Windows, etc.)
- Architecture (x86_64, ARM32, ARM64, etc.)
- CPU count and capabilities
- System grade (server vs. desktop vs. embedded)

## Platform-Specific Optimizations

### Linux
- **Buffer Size**: 512KB (optimal for kernel buffer management)
- **Concurrency**: NumCPU × 8 (leveraging epoll efficiency)
- **Max Connections**: 200 (high connection handling capability)
- **Special Features**:
  - Sendfile support for zero-copy transfers
  - TCP_NODELAY for low latency
  - TCP_QUICKACK for faster acknowledgments
  - TCP_CORK for packet batching
  - Automatic ulimit adjustment

### macOS (Darwin)
- **Intel Mac**:
  - Buffer Size: 256KB
  - Concurrency: NumCPU × 4
  - Max Connections: 150
- **Apple Silicon (ARM64)**:
  - Buffer Size: 128KB (optimized for unified memory)
  - Concurrency: NumCPU × 2-6 (based on server grade)
  - Max Connections: 50-150
- **Special Features**:
  - SO_REUSEPORT for better load distribution
  - Sendfile support
  - Optimized for macOS networking stack

### Windows
- **Buffer Size**: 128KB (balanced for IOCP)
- **Concurrency**: NumCPU × 2
- **Max Connections**: 100
- **Special Features**:
  - Windows auto-tuning integration
  - Conservative settings for stability
  - No sendfile (TransmitFile not exposed in Go)

### ARM Architecture
- **ARM32** (Embedded/IoT):
  - Buffer Size: 32KB (memory-constrained)
  - Concurrency: NumCPU
  - Max Connections: 30
  - Zero-copy disabled for memory efficiency
- **ARM64 Mobile**:
  - Buffer Size: 128KB
  - Concurrency: NumCPU × 2
  - Max Connections: 50
- **ARM64 Server** (AWS Graviton, Ampere):
  - Buffer Size: 128KB
  - Concurrency: NumCPU × 6
  - Max Connections: 150

## Adaptive Strategies

### File Size Optimization
The system adjusts chunk sizes based on file size:
- **< 1MB**: Small chunks (1/4 platform buffer)
- **1-10MB**: Medium chunks (1/2 platform buffer)
- **10-100MB**: Standard chunks (full platform buffer)
- **> 100MB**: Large chunks (2× platform buffer)

### Zero-Copy Thresholds
Platform-specific thresholds for zero-copy I/O:
- **Linux**: Files > 1MB
- **macOS**: Files > 5MB
- **Windows**: Not supported
- **ARM**: Disabled on 32-bit, enabled on 64-bit

### Low-End Hardware Detection
Systems with ≤2 CPUs receive special treatment:
- Reduced concurrency (max 4)
- Limited connections (max 20)
- Smaller buffers (32KB)
- Conservative resource usage

## Performance Benchmarks

### Platform Comparison (100MB file)
| Platform | Buffer Size | Speed vs. curl | CPU Usage |
|----------|------------|----------------|-----------|
| Linux x64 | 512KB | 115% | 8% |
| macOS Intel | 256KB | 110% | 10% |
| macOS ARM64 | 128KB | 108% | 9% |
| Windows | 128KB | 105% | 12% |
| Linux ARM64 | 128KB | 107% | 11% |

### Small File Performance (< 1MB)
- Lightweight mode enabled
- Minimal overhead
- 60-90% of curl speed
- Single connection optimization

### Large File Performance (> 100MB)
- Zero-copy enabled (Linux/macOS)
- Buffer pooling active
- 20-30% CPU reduction
- 110-120% of curl speed

## Configuration Override

While platform detection is automatic, you can override settings:

```go
options := &types.DownloadOptions{
    ChunkSize:      1024 * 1024,  // Force 1MB chunks
    MaxConcurrency: 16,            // Force 16 connections
}
```

## Testing

### Unit Tests
```bash
go test ./internal/core -run TestPlatform
```

### Cross-Platform CI
The project includes comprehensive CI workflows:
- Standard platforms (Linux, macOS, Windows)
- ARM cross-compilation
- ARM64 QEMU emulation
- Platform-specific optimization validation

### Manual Testing
```bash
# Test platform detection
go run cmd/gdl/main.go --debug-platform

# Benchmark on current platform
go test -bench=BenchmarkPlatform ./internal/core
```

## Best Practices

1. **Trust the Defaults**: Platform detection provides optimal settings for most cases
2. **Monitor Performance**: Use `--verbose` to see platform optimizations in action
3. **Test on Target**: Always test on your deployment platform
4. **Resource Constraints**: Consider container/VM resource limits
5. **Network Conditions**: Platform optimizations assume good network connectivity

## Troubleshooting

### Performance Issues
1. Check platform detection: `gdl --debug-platform`
2. Verify resource limits: `ulimit -n` (Linux/macOS)
3. Monitor CPU/memory usage during downloads
4. Try manual optimization overrides

### ARM-Specific Issues
- Ensure proper ARM variant detection (ARMv7 vs ARM64)
- Check memory constraints on embedded systems
- Verify kernel support for advanced TCP options

### Windows-Specific Issues
- Check Windows Defender exclusions
- Verify Windows Firewall settings
- Ensure TCP auto-tuning is enabled

## Future Enhancements

- [ ] RISC-V architecture support
- [ ] FreeBSD/OpenBSD optimizations
- [ ] Dynamic adaptation based on runtime performance
- [ ] Network condition detection and adaptation
- [ ] Container-aware optimizations (Docker, K8s)