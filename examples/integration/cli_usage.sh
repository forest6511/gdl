#!/bin/bash

# Godl CLI Usage Examples
# This script demonstrates all CLI features

set -e

echo "=== Godl CLI Usage Examples ==="
echo ""

# Build the CLI tool first
echo "Building godl CLI..."
go build -o godl ../../cmd/godl/
echo ""

# Example 1: Basic download
echo "1. Basic Download"
echo "-----------------"
./godl -o basic_cli.bin https://httpbin.org/bytes/1024
echo ""

# Example 2: Download with progress bar types
echo "2. Progress Bar Types"
echo "---------------------"
echo "Simple progress:"
./godl --progress-bar simple -o simple_progress.bin https://httpbin.org/bytes/5120

echo "Detailed progress (default):"
./godl --progress-bar detailed -o detailed_progress.bin https://httpbin.org/bytes/5120

echo "JSON progress:"
./godl --progress-bar json -o json_progress.bin https://httpbin.org/bytes/5120 | head -n 5
echo ""

# Example 3: Concurrent download
echo "3. Concurrent Download"
echo "----------------------"
./godl -c 4 --chunk-size 2KB -o concurrent_cli.bin https://httpbin.org/bytes/20480
echo ""

# Example 4: Resume functionality
echo "4. Resume Download"
echo "------------------"
# Start download and interrupt it
timeout 0.1s ./godl --resume -o resume_cli.bin https://httpbin.org/bytes/51200 || true
echo "Download interrupted, resuming..."
./godl --resume -o resume_cli.bin https://httpbin.org/bytes/51200
echo ""

# Example 5: Custom headers
echo "5. Custom Headers"
echo "-----------------"
./godl -H "Authorization: Bearer token123" \
       -H "X-Custom-Header: value" \
       --user-agent "godl-cli-example/1.0" \
       -o headers_cli.json \
       https://httpbin.org/headers
echo ""

# Example 6: Quiet and verbose modes
echo "6. Output Modes"
echo "---------------"
echo "Quiet mode:"
./godl -q -o quiet_cli.bin https://httpbin.org/bytes/1024
echo "Download completed (quiet mode)"

echo "Verbose mode:"
./godl -v -o verbose_cli.bin https://httpbin.org/bytes/1024
echo ""

# Example 7: Force overwrite
echo "7. Force Overwrite"
echo "------------------"
./godl -f -o overwrite_cli.bin https://httpbin.org/bytes/1024
echo "Overwriting existing file..."
./godl -f -o overwrite_cli.bin https://httpbin.org/bytes/2048
echo ""

# Example 8: Retry configuration
echo "8. Retry Configuration"
echo "----------------------"
./godl --retry 5 --retry-delay 2s -o retry_cli.bin https://httpbin.org/status/500 || echo "Failed after retries"
echo ""

# Example 9: Timeout configuration
echo "9. Timeout Configuration"
echo "------------------------"
./godl --timeout 10s -o timeout_cli.bin https://httpbin.org/delay/1
echo ""

# Example 10: Check connectivity and space
echo "10. Pre-download Checks"
echo "-----------------------"
./godl --check-connectivity --check-space -o checks_cli.bin https://httpbin.org/bytes/1024
echo ""

# Example 11: No color output
echo "11. No Color Output"
echo "-------------------"
./godl --no-color -o nocolor_cli.bin https://httpbin.org/bytes/1024
echo ""

# Example 12: Create parent directories
echo "12. Create Directories"
echo "----------------------"
./godl --create-dirs -o nested/dir/structure/file.bin https://httpbin.org/bytes/1024
echo ""

# Example 13: Version and help
echo "13. Version and Help"
echo "--------------------"
./godl --version
echo ""
./godl --help | head -n 20
echo ""

# Cleanup
echo "Cleaning up test files..."
rm -f *.bin *.json
rm -rf nested/
rm -f godl

echo ""
echo "=== All CLI examples completed successfully ===="