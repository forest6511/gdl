package core

import (
	"sync"
)

// BufferPool manages reusable byte buffers to reduce memory allocations
type BufferPool struct {
	small  *sync.Pool // 8KB buffers for small files
	medium *sync.Pool // 64KB buffers for medium files
	large  *sync.Pool // 1MB buffers for large files
	huge   *sync.Pool // 4MB buffers for huge files
}

// NewBufferPool creates a new buffer pool with various sizes
func NewBufferPool() *BufferPool {
	return &BufferPool{
		small: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 8*1024) // 8KB
				return &buf
			},
		},
		medium: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 64*1024) // 64KB
				return &buf
			},
		},
		large: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 1024*1024) // 1MB
				return &buf
			},
		},
		huge: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 4*1024*1024) // 4MB
				return &buf
			},
		},
	}
}

// Get returns a buffer appropriate for the given size
func (bp *BufferPool) Get(size int64) []byte {
	var pool *sync.Pool

	switch {
	case size <= 8*1024: // <= 8KB
		pool = bp.small
	case size <= 64*1024: // <= 64KB
		pool = bp.medium
	case size <= 1024*1024: // <= 1MB
		pool = bp.large
	default: // > 1MB
		pool = bp.huge
	}

	bufPtr := pool.Get().(*[]byte)
	return *bufPtr
}

// GetSized returns a buffer of at least the requested size
func (bp *BufferPool) GetSized(minSize int) []byte {
	// Select the appropriate pool based on minimum size
	switch {
	case minSize <= 8*1024:
		return bp.Get(int64(minSize))
	case minSize <= 64*1024:
		bufPtr := bp.medium.Get().(*[]byte)
		return (*bufPtr)[:minSize]
	case minSize <= 1024*1024:
		bufPtr := bp.large.Get().(*[]byte)
		return (*bufPtr)[:minSize]
	default:
		bufPtr := bp.huge.Get().(*[]byte)
		if minSize > len(*bufPtr) {
			// Need a custom size buffer
			buf := make([]byte, minSize)
			return buf
		}
		return (*bufPtr)[:minSize]
	}
}

// Put returns a buffer to the pool for reuse
func (bp *BufferPool) Put(buf []byte) {
	// Clear sensitive data
	for i := range buf {
		buf[i] = 0
	}

	size := cap(buf)
	switch size {
	case 8 * 1024:
		bp.small.Put(&buf)
	case 64 * 1024:
		bp.medium.Put(&buf)
	case 1024 * 1024:
		bp.large.Put(&buf)
	case 4 * 1024 * 1024:
		bp.huge.Put(&buf)
	default:
		// Don't pool custom-sized buffers
	}
}

// GetForFileSize returns an appropriate buffer based on file size
func (bp *BufferPool) GetForFileSize(fileSize int64) []byte {
	// Use smaller buffers for smaller files to reduce memory waste
	switch {
	case fileSize < 100*1024: // < 100KB
		return bp.Get(8 * 1024) // Use 8KB buffer
	case fileSize < 10*1024*1024: // < 10MB
		return bp.Get(64 * 1024) // Use 64KB buffer
	case fileSize < 100*1024*1024: // < 100MB
		return bp.Get(1024 * 1024) // Use 1MB buffer
	default:
		return bp.Get(4 * 1024 * 1024) // Use 4MB buffer
	}
}

// GlobalBufferPool is a shared buffer pool for the entire application
var GlobalBufferPool = NewBufferPool()

// PooledBuffer provides a convenient wrapper for pooled buffers
type PooledBuffer struct {
	buf  []byte
	pool *BufferPool
}

// NewPooledBuffer gets a buffer from the pool
func NewPooledBuffer(size int64) *PooledBuffer {
	return &PooledBuffer{
		buf:  GlobalBufferPool.Get(size),
		pool: GlobalBufferPool,
	}
}

// Bytes returns the underlying byte slice
func (pb *PooledBuffer) Bytes() []byte {
	return pb.buf
}

// Release returns the buffer to the pool
func (pb *PooledBuffer) Release() {
	if pb.buf != nil && pb.pool != nil {
		pb.pool.Put(pb.buf)
		pb.buf = nil
	}
}

// Resize adjusts the buffer size if needed
func (pb *PooledBuffer) Resize(newSize int) {
	if newSize > cap(pb.buf) {
		// Release old buffer
		pb.Release()
		// Get new buffer
		pb.buf = pb.pool.GetSized(newSize)
	} else {
		// Reuse existing buffer
		pb.buf = pb.buf[:newSize]
	}
}
