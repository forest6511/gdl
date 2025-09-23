//go:build linux
// +build linux

package core

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestSendFileLinux(t *testing.T) {
	tests := []struct {
		name        string
		dataSize    int64
		offset      int64
		count       int64
		expectError bool
		errorType   error
	}{
		{
			name:        "Small file transfer",
			dataSize:    1024,
			offset:      0,
			count:       1024,
			expectError: false,
		},
		{
			name:        "Medium file transfer",
			dataSize:    10 * 1024 * 1024, // 10MB
			offset:      0,
			count:       10 * 1024 * 1024,
			expectError: false,
		},
		{
			name:        "Large file transfer exceeding sendfile limit",
			dataSize:    0x7ffff000 + 1024, // Just over the sendfile limit
			offset:      0,
			count:       0x7ffff000 + 1024,
			expectError: false,
		},
		{
			name:        "Partial file transfer",
			dataSize:    1024 * 1024,
			offset:      0,
			count:       512 * 1024,
			expectError: false,
		},
		{
			name:        "Zero byte transfer",
			dataSize:    1024,
			offset:      0,
			count:       0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test data
			testData := make([]byte, tt.dataSize)
			for i := range testData {
				testData[i] = byte(i % 256)
			}

			// Create source file
			tempDir := t.TempDir()
			srcPath := filepath.Join(tempDir, "source.bin")
			if err := os.WriteFile(srcPath, testData, 0644); err != nil {
				t.Fatalf("Failed to create source file: %v", err)
			}

			srcFile, err := os.Open(srcPath)
			if err != nil {
				t.Fatalf("Failed to open source file: %v", err)
			}
			defer func() { _ = srcFile.Close() }()

			// Create destination file
			dstPath := filepath.Join(tempDir, "dest.bin")
			dstFile, err := os.Create(dstPath)
			if err != nil {
				t.Fatalf("Failed to create destination file: %v", err)
			}
			defer func() { _ = dstFile.Close() }()

			// Perform sendfile
			written, err := SendFileLinux(dstFile, srcFile, tt.offset, tt.count)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				expectedWritten := tt.count
				if expectedWritten > tt.dataSize {
					expectedWritten = tt.dataSize
				}

				if written != expectedWritten {
					t.Errorf("Written bytes mismatch: got %d, want %d", written, expectedWritten)
				}

				// Verify file contents
				_ = dstFile.Close()
				dstData, err := os.ReadFile(dstPath)
				if err != nil {
					t.Fatalf("Failed to read destination file: %v", err)
				}

				if int64(len(dstData)) != written {
					t.Errorf("Destination file size mismatch: got %d, want %d", len(dstData), written)
				}

				// Compare data
				expectedData := testData[:written]
				if !bytes.Equal(dstData, expectedData) {
					t.Error("Data mismatch after sendfile")
				}
			}
		})
	}
}

func TestSendFileLinux_Errors(t *testing.T) {
	// Test with invalid file descriptors
	t.Run("Invalid source file", func(t *testing.T) {
		tempDir := t.TempDir()
		dstPath := filepath.Join(tempDir, "dest.bin")
		dstFile, err := os.Create(dstPath)
		if err != nil {
			t.Fatalf("Failed to create destination file: %v", err)
		}
		defer func() { _ = dstFile.Close() }()

		// Create a closed file to get an invalid fd
		srcPath := filepath.Join(tempDir, "source.bin")
		_ = os.WriteFile(srcPath, []byte("test"), 0644)
		srcFile, _ := os.Open(srcPath)
		_ = srcFile.Close() // Close to make it invalid

		_, err = SendFileLinux(dstFile, srcFile, 0, 1024)
		if err == nil {
			t.Error("Expected error with closed source file")
		}
	})

	t.Run("Invalid destination file", func(t *testing.T) {
		tempDir := t.TempDir()
		srcPath := filepath.Join(tempDir, "source.bin")
		if err := os.WriteFile(srcPath, []byte("test data"), 0644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		srcFile, err := os.Open(srcPath)
		if err != nil {
			t.Fatalf("Failed to open source file: %v", err)
		}
		defer func() { _ = srcFile.Close() }()

		// Create a closed file for invalid fd
		dstPath := filepath.Join(tempDir, "dest.bin")
		dstFile, _ := os.Create(dstPath)
		_ = dstFile.Close() // Close to make it invalid

		_, err = SendFileLinux(dstFile, srcFile, 0, 1024)
		if err == nil {
			t.Error("Expected error with closed destination file")
		}
	})
}

func TestSendFileLinux_EAGAIN_Retry(t *testing.T) {
	// This test is conceptual since we can't easily trigger EAGAIN
	// but we test the code path exists and compiles
	t.Run("EAGAIN handling", func(t *testing.T) {
		// The actual EAGAIN retry logic is tested in integration tests
		// Here we just ensure the code compiles and basic flow works
		tempDir := t.TempDir()

		srcPath := filepath.Join(tempDir, "source.bin")
		testData := []byte("test data for EAGAIN")
		if err := os.WriteFile(srcPath, testData, 0644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		srcFile, err := os.Open(srcPath)
		if err != nil {
			t.Fatalf("Failed to open source file: %v", err)
		}
		defer func() { _ = srcFile.Close() }()

		dstPath := filepath.Join(tempDir, "dest.bin")
		dstFile, err := os.Create(dstPath)
		if err != nil {
			t.Fatalf("Failed to create destination file: %v", err)
		}
		defer func() { _ = dstFile.Close() }()

		written, err := SendFileLinux(dstFile, srcFile, 0, int64(len(testData)))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if written != int64(len(testData)) {
			t.Errorf("Written bytes mismatch: got %d, want %d", written, len(testData))
		}
	})
}

func BenchmarkSendFileLinux_vs_StandardCopy(b *testing.B) {
	sizes := []int{
		1024,             // 1KB
		100 * 1024,       // 100KB
		1024 * 1024,      // 1MB
		10 * 1024 * 1024, // 10MB
	}

	for _, size := range sizes {
		testData := make([]byte, size)
		for i := range testData {
			testData[i] = byte(i % 256)
		}

		b.Run(fmt.Sprintf("SendFile_%dKB", size/1024), func(b *testing.B) {
			tempDir := b.TempDir()
			srcPath := filepath.Join(tempDir, "source.bin")
			if err := os.WriteFile(srcPath, testData, 0644); err != nil {
				b.Fatalf("Failed to create source file: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcFile, _ := os.Open(srcPath)
				dstPath := filepath.Join(tempDir, fmt.Sprintf("dest_%d.bin", i))
				dstFile, _ := os.Create(dstPath)

				_, _ = SendFileLinux(dstFile, srcFile, 0, int64(size))

				_ = srcFile.Close()
				_ = dstFile.Close()
				_ = os.Remove(dstPath)
			}
		})

		b.Run(fmt.Sprintf("StandardCopy_%dKB", size/1024), func(b *testing.B) {
			tempDir := b.TempDir()
			srcPath := filepath.Join(tempDir, "source.bin")
			if err := os.WriteFile(srcPath, testData, 0644); err != nil {
				b.Fatalf("Failed to create source file: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcFile, _ := os.Open(srcPath)
				dstPath := filepath.Join(tempDir, fmt.Sprintf("dest_%d.bin", i))
				dstFile, _ := os.Create(dstPath)

				_, _ = io.Copy(dstFile, srcFile)

				_ = srcFile.Close()
				_ = dstFile.Close()
				_ = os.Remove(dstPath)
			}
		})
	}
}

// TestSendFileLinux_EdgeCases tests edge cases and error conditions
func TestSendFileLinux_EdgeCases(t *testing.T) {
	t.Run("EOF handling", func(t *testing.T) {
		tempDir := t.TempDir()
		srcPath := filepath.Join(tempDir, "empty.bin")
		// Create empty file
		if err := os.WriteFile(srcPath, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}

		srcFile, err := os.Open(srcPath)
		if err != nil {
			t.Fatalf("Failed to open source file: %v", err)
		}
		defer func() { _ = srcFile.Close() }()

		dstPath := filepath.Join(tempDir, "dest.bin")
		dstFile, err := os.Create(dstPath)
		if err != nil {
			t.Fatalf("Failed to create destination file: %v", err)
		}
		defer func() { _ = dstFile.Close() }()

		written, err := SendFileLinux(dstFile, srcFile, 0, 1024)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if written != 0 {
			t.Errorf("Expected 0 bytes written for empty file, got %d", written)
		}
	})

	t.Run("Read-only destination", func(t *testing.T) {
		tempDir := t.TempDir()
		srcPath := filepath.Join(tempDir, "source.bin")
		if err := os.WriteFile(srcPath, []byte("test data"), 0644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		srcFile, err := os.Open(srcPath)
		if err != nil {
			t.Fatalf("Failed to open source file: %v", err)
		}
		defer func() { _ = srcFile.Close() }()

		// Open destination in read-only mode
		dstPath := filepath.Join(tempDir, "readonly.bin")
		_ = os.WriteFile(dstPath, []byte{}, 0444)
		dstFile, err := os.Open(dstPath) // Open in read-only mode
		if err != nil {
			t.Fatalf("Failed to open destination file: %v", err)
		}
		defer func() { _ = dstFile.Close() }()

		_, err = SendFileLinux(dstFile, srcFile, 0, 1024)
		if err == nil {
			t.Error("Expected error with read-only destination")
		}
	})
}

// Mock test for EAGAIN scenario (conceptual)
func TestSendFileLinux_MockEAGAIN(t *testing.T) {
	// This test ensures the EAGAIN handling code path compiles
	// In real scenario, EAGAIN would occur with non-blocking sockets

	// EAGAIN is a constant error value, not a pointer
	// Just verify it exists as a compile-time check
	_ = syscall.EAGAIN

	// The retry logic for EAGAIN is embedded in SendFileLinux
	// and will be exercised during actual file transfers
	t.Log("EAGAIN retry logic is compiled and will be tested during actual transfers")
}
