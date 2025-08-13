package concurrent

type ChunkInfo struct {
	Index      int
	Start      int64
	End        int64
	Downloaded int64
	Complete   bool
}

type Chunker struct {
	fileSize   int64
	chunkCount int
	chunks     []*ChunkInfo
}

const (
	minChunkSize = 1024 * 1024 // 1MB minimum chunk size
	maxChunks    = 32          // Maximum number of chunks
)

// NewChunker creates a new chunker for the given file size.
func NewChunker(fileSize int64) *Chunker {
	c := &Chunker{
		fileSize: fileSize,
	}
	c.chunkCount = c.CalculateOptimalChunks()
	c.SplitIntoChunks()

	return c
}

// CalculateOptimalChunks determines the optimal number of chunks based on file size.
func (c *Chunker) CalculateOptimalChunks() int {
	if c.fileSize <= 0 {
		return 1
	}

	// If file is smaller than minimum chunk size, use single chunk
	if c.fileSize <= minChunkSize {
		return 1
	}

	// Calculate chunks based on file size
	// Additional heuristics based on file size ranges
	var optimalChunks int
	switch {
	case c.fileSize < 10*minChunkSize: // < 10MB
		optimalChunks = 2
	case c.fileSize < 50*minChunkSize: // < 50MB
		optimalChunks = 4
	case c.fileSize < 100*minChunkSize: // < 100MB
		optimalChunks = 8
	case c.fileSize < 500*minChunkSize: // < 500MB
		optimalChunks = 16
	default:
		// For very large files, calculate based on chunk size but cap at maxChunks
		optimalChunks = int(c.fileSize / minChunkSize)
		if optimalChunks > maxChunks {
			optimalChunks = maxChunks
		}
		if optimalChunks < 1 {
			optimalChunks = 1
		}
	}

	// Ensure we don't exceed max chunks
	if optimalChunks > maxChunks {
		optimalChunks = maxChunks
	}

	return optimalChunks
}

// GetChunks returns the list of chunks.
func (c *Chunker) GetChunks() []*ChunkInfo {
	return c.chunks
}

// SplitIntoChunks divides the file into chunks based on the calculated chunk count.
func (c *Chunker) SplitIntoChunks() {
	if c.fileSize <= 0 || c.chunkCount <= 0 {
		c.chunks = []*ChunkInfo{{
			Index:      0,
			Start:      0,
			End:        c.fileSize - 1,
			Downloaded: 0,
			Complete:   false,
		}}

		return
	}

	chunkSize := c.fileSize / int64(c.chunkCount)
	remainder := c.fileSize % int64(c.chunkCount)

	c.chunks = make([]*ChunkInfo, c.chunkCount)

	var currentStart int64 = 0

	for i := 0; i < c.chunkCount; i++ {
		size := chunkSize
		// Distribute remainder bytes across first chunks
		if int64(i) < remainder {
			size++
		}

		c.chunks[i] = &ChunkInfo{
			Index:      i,
			Start:      currentStart,
			End:        currentStart + size - 1,
			Downloaded: 0,
			Complete:   false,
		}

		currentStart += size
	}

	// Ensure the last chunk ends at the correct position
	if c.chunkCount > 0 {
		c.chunks[c.chunkCount-1].End = c.fileSize - 1
	}
}
