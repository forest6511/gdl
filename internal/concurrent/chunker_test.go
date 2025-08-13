package concurrent

import (
	"testing"
)

func TestNewChunker(t *testing.T) {
	tests := []struct {
		name         string
		fileSize     int64
		wantChunks   int
		wantChunkLen int
	}{
		{
			name:         "Empty file",
			fileSize:     0,
			wantChunks:   1,
			wantChunkLen: 1,
		},
		{
			name:         "Small file (< 1MB)",
			fileSize:     512 * 1024, // 512KB
			wantChunks:   1,
			wantChunkLen: 1,
		},
		{
			name:         "Exactly 1MB",
			fileSize:     1024 * 1024,
			wantChunks:   1,
			wantChunkLen: 1,
		},
		{
			name:         "5MB file",
			fileSize:     5 * 1024 * 1024,
			wantChunks:   2,
			wantChunkLen: 2,
		},
		{
			name:         "25MB file",
			fileSize:     25 * 1024 * 1024,
			wantChunks:   4,
			wantChunkLen: 4,
		},
		{
			name:         "75MB file",
			fileSize:     75 * 1024 * 1024,
			wantChunks:   8,
			wantChunkLen: 8,
		},
		{
			name:         "200MB file",
			fileSize:     200 * 1024 * 1024,
			wantChunks:   16,
			wantChunkLen: 16,
		},
		{
			name:         "1GB file",
			fileSize:     1024 * 1024 * 1024,
			wantChunks:   32,
			wantChunkLen: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunker := NewChunker(tt.fileSize)

			if chunker.fileSize != tt.fileSize {
				t.Errorf("fileSize = %d, want %d", chunker.fileSize, tt.fileSize)
			}

			if chunker.chunkCount != tt.wantChunks {
				t.Errorf("chunkCount = %d, want %d", chunker.chunkCount, tt.wantChunks)
			}

			if len(chunker.chunks) != tt.wantChunkLen {
				t.Errorf("chunks length = %d, want %d", len(chunker.chunks), tt.wantChunkLen)
			}
		})
	}
}

func TestCalculateOptimalChunks(t *testing.T) {
	tests := []struct {
		name       string
		fileSize   int64
		wantChunks int
	}{
		{
			name:       "Negative file size",
			fileSize:   -100,
			wantChunks: 1,
		},
		{
			name:       "Zero file size",
			fileSize:   0,
			wantChunks: 1,
		},
		{
			name:       "Very small file",
			fileSize:   100,
			wantChunks: 1,
		},
		{
			name:       "8MB file",
			fileSize:   8 * 1024 * 1024,
			wantChunks: 2,
		},
		{
			name:       "45MB file",
			fileSize:   45 * 1024 * 1024,
			wantChunks: 4,
		},
		{
			name:       "90MB file",
			fileSize:   90 * 1024 * 1024,
			wantChunks: 8,
		},
		{
			name:       "450MB file",
			fileSize:   450 * 1024 * 1024,
			wantChunks: 16,
		},
		{
			name:       "2GB file",
			fileSize:   2 * 1024 * 1024 * 1024,
			wantChunks: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunker := &Chunker{fileSize: tt.fileSize}
			got := chunker.CalculateOptimalChunks()

			if got != tt.wantChunks {
				t.Errorf("CalculateOptimalChunks() = %d, want %d", got, tt.wantChunks)
			}

			// Verify constraints
			if got < 1 {
				t.Errorf("chunks should be at least 1, got %d", got)
			}

			if got > maxChunks {
				t.Errorf("chunks should not exceed %d, got %d", maxChunks, got)
			}
		})
	}
}

func TestSplitIntoChunks(t *testing.T) {
	tests := []struct {
		name       string
		fileSize   int64
		chunkCount int
		validate   func(t *testing.T, chunks []*ChunkInfo)
	}{
		{
			name:       "Single chunk",
			fileSize:   1000,
			chunkCount: 1,
			validate: func(t *testing.T, chunks []*ChunkInfo) {
				if len(chunks) != 1 {
					t.Errorf("expected 1 chunk, got %d", len(chunks))
				}
				if chunks[0].Start != 0 || chunks[0].End != 999 {
					t.Errorf("chunk range = %d-%d, want 0-999", chunks[0].Start, chunks[0].End)
				}
			},
		},
		{
			name:       "Even split",
			fileSize:   1000,
			chunkCount: 4,
			validate: func(t *testing.T, chunks []*ChunkInfo) {
				if len(chunks) != 4 {
					t.Errorf("expected 4 chunks, got %d", len(chunks))
				}

				// Check each chunk size
				expectedSize := int64(250)
				for i, chunk := range chunks {
					size := chunk.End - chunk.Start + 1
					if size != expectedSize {
						t.Errorf("chunk %d size = %d, want %d", i, size, expectedSize)
					}
				}

				// Verify continuity
				for i := 1; i < len(chunks); i++ {
					if chunks[i].Start != chunks[i-1].End+1 {
						t.Errorf("discontinuity between chunk %d and %d", i-1, i)
					}
				}
			},
		},
		{
			name:       "Uneven split with remainder",
			fileSize:   1003,
			chunkCount: 4,
			validate: func(t *testing.T, chunks []*ChunkInfo) {
				if len(chunks) != 4 {
					t.Errorf("expected 4 chunks, got %d", len(chunks))
				}

				// First 3 chunks should have 251 bytes, last one 250
				sizes := []int64{251, 251, 251, 250}
				for i, chunk := range chunks {
					size := chunk.End - chunk.Start + 1
					if size != sizes[i] {
						t.Errorf("chunk %d size = %d, want %d", i, size, sizes[i])
					}
				}

				// Verify total coverage
				if chunks[0].Start != 0 {
					t.Errorf("first chunk should start at 0")
				}
				if chunks[len(chunks)-1].End != 1002 {
					t.Errorf("last chunk should end at %d, got %d", 1002, chunks[len(chunks)-1].End)
				}
			},
		},
		{
			name:       "Zero file size",
			fileSize:   0,
			chunkCount: 4,
			validate: func(t *testing.T, chunks []*ChunkInfo) {
				if len(chunks) != 1 {
					t.Errorf("expected 1 chunk for zero size, got %d", len(chunks))
				}
				if chunks[0].End != -1 {
					t.Errorf("expected End=-1 for zero size, got %d", chunks[0].End)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunker := &Chunker{
				fileSize:   tt.fileSize,
				chunkCount: tt.chunkCount,
			}
			chunker.SplitIntoChunks()

			tt.validate(t, chunker.chunks)

			// Common validations
			for i, chunk := range chunker.chunks {
				if chunk.Index != i {
					t.Errorf("chunk index = %d, want %d", chunk.Index, i)
				}

				if chunk.Downloaded != 0 {
					t.Errorf("initial downloaded should be 0, got %d", chunk.Downloaded)
				}

				if chunk.Complete {
					t.Errorf("initial complete should be false")
				}
			}
		})
	}
}

func TestGetChunks(t *testing.T) {
	chunker := NewChunker(10 * 1024 * 1024) // 10MB
	chunks := chunker.GetChunks()

	if chunks == nil {
		t.Fatal("GetChunks() returned nil")
	}

	if len(chunks) != chunker.chunkCount {
		t.Errorf("GetChunks() returned %d chunks, want %d", len(chunks), chunker.chunkCount)
	}

	// Verify chunks are the same reference
	if &chunks[0] != &chunker.chunks[0] {
		t.Error("GetChunks() should return the same chunk references")
	}
}
