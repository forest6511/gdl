package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/forest6511/gdl/internal/core"
	"github.com/forest6511/gdl/pkg/types"
)

func BenchmarkDownloadProfile(b *testing.B) {
	// Create test data
	testData := make([]byte, 10*1024*1024) // 10MB
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10485760")
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	// Create CPU profile
	cpuFile, err := os.Create("cpu.prof")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = cpuFile.Close() }()

	// Start CPU profiling
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		b.Fatal(err)
	}
	defer pprof.StopCPUProfile()

	// Create memory profile file
	memFile, err := os.Create("mem.prof")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = memFile.Close() }()

	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		downloader := core.NewDownloader()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		tempFile, _ := os.CreateTemp("", "benchmark-*.tmp")
		_ = tempFile.Close()
		defer func() { _ = os.Remove(tempFile.Name()) }()

		options := &types.DownloadOptions{
			MaxConcurrency: 4,
			ChunkSize:      128 * 1024,
		}

		_, _ = downloader.Download(ctx, server.URL, tempFile.Name(), options)
		cancel()
	}

	// Write memory profile
	if err := pprof.WriteHeapProfile(memFile); err != nil {
		b.Fatal(err)
	}
}

func TestProfileAnalysis(t *testing.T) {
	// This test is for manual profiling
	t.Skip("Skipping profile analysis - run manually with: go test -run TestProfileAnalysis -cpuprofile=cpu.prof -memprofile=mem.prof")

	// After running:
	// go tool pprof cpu.prof
	// go tool pprof mem.prof
	// Or for web view:
	// go tool pprof -http=:8080 cpu.prof
}
