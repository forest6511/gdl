#!/bin/bash

# Godl CLI Usage Examples
# This script demonstrates all CLI features

set -e

echo "=== Godl CLI Usage Examples ==="
echo ""

# Build the CLI tool first
echo "Building gdl CLI..."
go build -o gdl ../../cmd/gdl/
echo ""

# Example 1: Basic download
echo "1. Basic Download"
echo "-----------------"
./gdl -o basic_cli.bin https://httpbin.org/bytes/1024
echo ""

# Example 2: Download with progress bar types
echo "2. Progress Bar Types"
echo "---------------------"
echo "Simple progress:"
./gdl --progress-bar simple -o simple_progress.bin https://httpbin.org/bytes/5120

echo "Detailed progress (default):"
./gdl --progress-bar detailed -o detailed_progress.bin https://httpbin.org/bytes/5120

echo "JSON progress:"
./gdl --progress-bar json -o json_progress.bin https://httpbin.org/bytes/5120 | head -n 5
echo ""

# Example 3: Concurrent download
echo "3. Concurrent Download"
echo "----------------------"
./gdl -c 4 --chunk-size 2KB -o concurrent_cli.bin https://httpbin.org/bytes/20480
echo ""

# Example 4: Resume functionality
echo "4. Resume Download"
echo "------------------"
# Start download and interrupt it
timeout 0.1s ./gdl --resume -o resume_cli.bin https://httpbin.org/bytes/51200 || true
echo "Download interrupted, resuming..."
./gdl --resume -o resume_cli.bin https://httpbin.org/bytes/51200
echo ""

# Example 5: Custom headers
echo "5. Custom Headers"
echo "-----------------"
./gdl -H "Authorization: Bearer token123" \
       -H "X-Custom-Header: value" \
       --user-agent "gdl-cli-example/1.0" \
       -o headers_cli.json \
       https://httpbin.org/headers
echo ""

# Example 6: Quiet and verbose modes
echo "6. Output Modes"
echo "---------------"
echo "Quiet mode:"
./gdl -q -o quiet_cli.bin https://httpbin.org/bytes/1024
echo "Download completed (quiet mode)"

echo "Verbose mode:"
./gdl -v -o verbose_cli.bin https://httpbin.org/bytes/1024
echo ""

# Example 7: Force overwrite
echo "7. Force Overwrite"
echo "------------------"
./gdl -f -o overwrite_cli.bin https://httpbin.org/bytes/1024
echo "Overwriting existing file..."
./gdl -f -o overwrite_cli.bin https://httpbin.org/bytes/2048
echo ""

# Example 8: Retry configuration
echo "8. Retry Configuration"
echo "----------------------"
./gdl --retry 5 --retry-delay 2s -o retry_cli.bin https://httpbin.org/status/500 || echo "Failed after retries"
echo ""

# Example 9: Timeout configuration
echo "9. Timeout Configuration"
echo "------------------------"
./gdl --timeout 10s -o timeout_cli.bin https://httpbin.org/delay/1
echo ""

# Example 10: Check connectivity and space
echo "10. Pre-download Checks"
echo "-----------------------"
./gdl --check-connectivity --check-space -o checks_cli.bin https://httpbin.org/bytes/1024
echo ""

# Example 11: No color output
echo "11. No Color Output"
echo "-------------------"
./gdl --no-color -o nocolor_cli.bin https://httpbin.org/bytes/1024
echo ""

# Example 12: Create parent directories
echo "12. Create Directories"
echo "----------------------"
./gdl --create-dirs -o nested/dir/structure/file.bin https://httpbin.org/bytes/1024
echo ""

# Example 13: Version and help
echo "13. Version and Help"
echo "--------------------"
./gdl --version
echo ""
./gdl --help | head -n 20
echo ""

# Cleanup
echo "Cleaning up test files..."
rm -f *.bin *.json
rm -rf nested/
rm -f gdl

echo ""
echo "=== All CLI examples completed successfully ===="