package core

import (
	"sync"
	"testing"
)

func TestBufferPool_Get(t *testing.T) {
	pool := NewBufferPool()

	tests := []struct {
		size     int64
		expected int
	}{
		{1024, 8 * 1024},                   // Small
		{8 * 1024, 8 * 1024},               // Small boundary
		{32 * 1024, 64 * 1024},             // Medium
		{512 * 1024, 1024 * 1024},          // Large
		{2 * 1024 * 1024, 4 * 1024 * 1024}, // Huge
	}

	for _, tt := range tests {
		buf := pool.Get(tt.size)
		if cap(buf) != tt.expected {
			t.Errorf("Get(%d) returned buffer with cap %d, expected %d",
				tt.size, cap(buf), tt.expected)
		}
	}
}

func TestBufferPool_PutAndReuse(t *testing.T) {
	pool := NewBufferPool()

	// Get a buffer
	buf1 := pool.Get(1024)

	// Mark it with test data
	buf1[0] = 42

	// Return it to the pool
	pool.Put(buf1)

	// Verify buffer was cleared
	if buf1[0] != 0 {
		t.Error("Buffer not cleared after Put")
	}

	// Get another buffer of the same size
	buf2 := pool.Get(1024)

	// Should be cleared
	if buf2[0] != 0 {
		t.Error("Reused buffer not cleared")
	}
}

func TestBufferPool_GetSized(t *testing.T) {
	pool := NewBufferPool()

	tests := []struct {
		name    string
		minSize int
		minCap  int
		maxLen  int
	}{
		{"tiny buffer", 1024, 8 * 1024, 8 * 1024},
		{"small buffer", 4 * 1024, 8 * 1024, 8 * 1024},
		{"small boundary", 8 * 1024, 8 * 1024, 8 * 1024},
		{"medium buffer", 32 * 1024, 64 * 1024, 32 * 1024},
		{"medium boundary", 64 * 1024, 64 * 1024, 64 * 1024},
		{"large buffer", 512 * 1024, 1024 * 1024, 512 * 1024},
		{"large boundary", 1024 * 1024, 1024 * 1024, 1024 * 1024},
		{"huge buffer", 2 * 1024 * 1024, 4 * 1024 * 1024, 2 * 1024 * 1024},
		{"huge boundary", 4 * 1024 * 1024, 4 * 1024 * 1024, 4 * 1024 * 1024},
		{"custom size", 8 * 1024 * 1024, 8 * 1024 * 1024, 8 * 1024 * 1024}, // Larger than huge pool
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := pool.GetSized(tt.minSize)

			// GetSized returns slice with appropriate capacity and length up to the max of the pool size
			if len(buf) > tt.maxLen {
				t.Errorf("GetSized(%d) returned buffer with len %d, expected at most %d",
					tt.minSize, len(buf), tt.maxLen)
			}

			if len(buf) < tt.minSize && len(buf) != cap(buf) {
				t.Errorf("GetSized(%d) returned buffer with len %d, expected at least %d",
					tt.minSize, len(buf), tt.minSize)
			}

			if cap(buf) < tt.minCap {
				t.Errorf("GetSized(%d) returned buffer with cap %d, expected at least %d",
					tt.minSize, cap(buf), tt.minCap)
			}
		})
	}
}

func TestBufferPool_GetForFileSize(t *testing.T) {
	pool := NewBufferPool()

	tests := []struct {
		fileSize int64
		expected int
	}{
		{50 * 1024, 8 * 1024},                // 50KB -> 8KB buffer
		{5 * 1024 * 1024, 64 * 1024},         // 5MB -> 64KB buffer
		{50 * 1024 * 1024, 1024 * 1024},      // 50MB -> 1MB buffer
		{500 * 1024 * 1024, 4 * 1024 * 1024}, // 500MB -> 4MB buffer
	}

	for _, tt := range tests {
		buf := pool.GetForFileSize(tt.fileSize)
		if cap(buf) != tt.expected {
			t.Errorf("GetForFileSize(%d) returned buffer with cap %d, expected %d",
				tt.fileSize, cap(buf), tt.expected)
		}
	}
}

func TestPooledBuffer(t *testing.T) {
	// Test basic operations
	pb := NewPooledBuffer(1024)

	buf := pb.Bytes()
	if cap(buf) != 8*1024 {
		t.Errorf("Expected 8KB buffer, got %d", cap(buf))
	}

	// Test resize smaller
	pb.Resize(512)
	if len(pb.Bytes()) != 512 {
		t.Errorf("Expected buffer size 512, got %d", len(pb.Bytes()))
	}

	// Test resize larger
	pb.Resize(100 * 1024) // Should get new 1MB buffer
	if cap(pb.Bytes()) < 100*1024 {
		t.Errorf("Buffer too small after resize: %d", cap(pb.Bytes()))
	}

	// Release
	pb.Release()
	if pb.buf != nil {
		t.Error("Buffer not nil after release")
	}
}

func TestBufferPool_Concurrent(t *testing.T) {
	pool := NewBufferPool()
	var wg sync.WaitGroup

	// Run multiple goroutines getting and putting buffers
	// Use smaller iteration counts and sizes to avoid timeout
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			for j := 0; j < 10; j++ {
				// Use smaller buffer sizes: 64KB, 128KB, 256KB, 512KB
				size := int64((idx%4 + 1) * 64 * 1024)
				buf := pool.Get(size)

				// Use buffer
				if len(buf) > 0 {
					buf[0] = byte(j)
				}

				// Return to pool
				pool.Put(buf)
			}
		}(i)
	}

	wg.Wait()
}

func BenchmarkBufferPool_GetPut(b *testing.B) {
	pool := NewBufferPool()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(64 * 1024)
			// Simulate some work
			buf[0] = 1
			buf[len(buf)-1] = 2
			pool.Put(buf)
		}
	})
}

func BenchmarkBufferPool_vs_Make(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		pool := NewBufferPool()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := pool.Get(64 * 1024)
			// Use buffer
			_ = buf[0]
			pool.Put(buf)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := make([]byte, 64*1024)
			// Use buffer
			_ = buf[0]
			// No pool, just let GC handle it
		}
	})
}
