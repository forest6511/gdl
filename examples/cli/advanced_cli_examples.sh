#!/bin/bash

# Advanced CLI Usage Examples for godl
# This script demonstrates all the advanced command-line interface features

echo "=== Advanced CLI Usage Examples ==="

# Build the godl binary first
echo "Building godl..."
go build -o godl ./cmd/godl

echo ""
echo "1. Download with Custom Headers:"
echo "Command: ./godl -H 'Authorization: Bearer token123' -H 'X-Custom-Header: MyValue' -o headers_test.json https://httpbin.org/headers"
./godl -H 'Authorization: Bearer token123' -H 'X-Custom-Header: MyValue' -o headers_test.json https://httpbin.org/headers
echo "✓ Custom headers download completed"
echo ""

echo "2. Concurrent Download with Multiple Connections:"
echo "Command: ./godl --concurrent 8 -o concurrent_test.bin https://httpbin.org/bytes/16384"
./godl --concurrent 8 -o concurrent_test.bin https://httpbin.org/bytes/16384
echo "✓ Concurrent download completed"
echo ""

echo "3. Download with Custom Chunk Size:"
echo "Command: ./godl --concurrent 4 --chunk-size 2KB -o chunks_test.bin https://httpbin.org/bytes/8192"
./godl --concurrent 4 --chunk-size 2KB -o chunks_test.bin https://httpbin.org/bytes/8192
echo "✓ Custom chunk size download completed"
echo ""

echo "4. Download with Bandwidth Throttling:"
echo "Command: ./godl --max-rate 50KB/s -o throttled_test.bin https://httpbin.org/bytes/8192"
./godl --max-rate 50KB/s -o throttled_test.bin https://httpbin.org/bytes/8192
echo "✓ Bandwidth throttled download completed"
echo ""

echo "5. Download with Bandwidth + Concurrency:"
echo "Command: ./godl --concurrent 4 --max-rate 100KB/s -o throttled_concurrent.bin https://httpbin.org/bytes/16384"
./godl --concurrent 4 --max-rate 100KB/s -o throttled_concurrent.bin https://httpbin.org/bytes/16384
echo "✓ Throttled concurrent download completed"
echo ""

echo "6. Force Single-threaded Download:"
echo "Command: ./godl --no-concurrent -o single_thread.bin https://httpbin.org/bytes/4096"
./godl --no-concurrent -o single_thread.bin https://httpbin.org/bytes/4096
echo "✓ Single-threaded download completed"
echo ""

echo "7. Download with Retry Configuration:"
echo "Command: ./godl --retry 5 --retry-delay 2s -o retry_test.json https://httpbin.org/json"
./godl --retry 5 --retry-delay 2s -o retry_test.json https://httpbin.org/json
echo "✓ Retry configuration download completed"
echo ""

echo "8. Download with Maximum Redirects:"
echo "Command: ./godl --max-redirects 5 -o redirect_test.json https://httpbin.org/redirect/2"
./godl --max-redirects 5 -o redirect_test.json https://httpbin.org/redirect/2
echo "✓ Maximum redirects download completed"
echo ""

echo "9. Download with Different Progress Bar Types:"
echo "Command: ./godl --progress-bar simple -o progress_simple.bin https://httpbin.org/bytes/8192"
./godl --progress-bar simple -o progress_simple.bin https://httpbin.org/bytes/8192
echo ""
echo "Command: ./godl --progress-bar detailed -o progress_detailed.bin https://httpbin.org/bytes/8192"
./godl --progress-bar detailed -o progress_detailed.bin https://httpbin.org/bytes/8192
echo ""
echo "Command: ./godl --progress-bar json -o progress_json.bin https://httpbin.org/bytes/4096"
./godl --progress-bar json -o progress_json.bin https://httpbin.org/bytes/4096
echo "✓ Different progress bar types completed"
echo ""

echo "8. Download with Language Settings:"
echo "Command: ./godl --language ja --verbose -o language_ja.json https://httpbin.org/json"
./godl --language ja --verbose -o language_ja.json https://httpbin.org/json
echo ""

echo "9. Download with No Color Output:"
echo "Command: ./godl --no-color --verbose -o no_color.json https://httpbin.org/json"
./godl --no-color --verbose -o no_color.json https://httpbin.org/json
echo ""

echo "10. Download with Pre-flight Checks:"
echo "Command: ./godl --check-connectivity --check-space -o preflight.json https://httpbin.org/json"
./godl --check-connectivity --check-space -o preflight.json https://httpbin.org/json
echo "✓ Pre-flight checks download completed"
echo ""

echo "11. Insecure Download (Skip SSL Verification):"
echo "Command: ./godl --insecure -o insecure_test.json https://httpbin.org/json"
./godl --insecure -o insecure_test.json https://httpbin.org/json
echo "✓ Insecure download completed"
echo ""

echo "12. Download with Output Format Control:"
echo "Command: ./godl --output-format json --verbose -o format_test.json https://httpbin.org/json"
./godl --output-format json --verbose -o format_test.json https://httpbin.org/json
echo ""

echo "13. Continue Partial Downloads:"
echo "Command: ./godl --continue-partial --resume -o partial_test.bin https://httpbin.org/bytes/8192"
./godl --continue-partial --resume -o partial_test.bin https://httpbin.org/bytes/8192
echo "✓ Continue partial download completed"
echo ""

echo "14. Comprehensive Download with All Features:"
echo "Command: ./godl --concurrent 6 --chunk-size 1MB --retry 3 --retry-delay 1s --max-redirects 10 \\"
echo "         -H 'Accept: application/json' -H 'X-Test: Comprehensive' --verbose --create-dirs \\"
echo "         --force --resume --check-space --progress-bar detailed \\"
echo "         -o downloads/comprehensive/test.json https://httpbin.org/json"
./godl --concurrent 6 --chunk-size 1MB --retry 3 --retry-delay 1s --max-redirects 10 \
       -H 'Accept: application/json' -H 'X-Test: Comprehensive' --verbose --create-dirs \
       --force --resume --check-space --progress-bar detailed \
       -o downloads/comprehensive/test.json https://httpbin.org/json
echo "✓ Comprehensive download with all features completed"
echo ""

echo "15. Short Flag Combinations:"
echo "Command: ./godl -fv -o short_flags.json https://httpbin.org/json"
./godl -fv -o short_flags.json https://httpbin.org/json
echo ""
echo "Command: ./godl -c 8 -H 'User-Agent: ShortFlags/1.0' -o short_concurrent.bin https://httpbin.org/bytes/4096"
./godl -c 8 -H 'User-Agent: ShortFlags/1.0' -o short_concurrent.bin https://httpbin.org/bytes/4096
echo "✓ Short flag combinations completed"
echo ""

echo "=== Advanced CLI Examples Completed ==="
echo "Files created:"
find . -name "*.json" -o -name "*.bin" | grep -E "(headers|concurrent|chunks|single|retry|redirect|progress|language|color|preflight|insecure|format|partial|comprehensive|short)" | head -15
echo "..."
echo ""