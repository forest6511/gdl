#!/bin/bash

# Basic CLI Usage Examples for godl
# This script demonstrates all the basic command-line interface features

echo "=== Basic CLI Usage Examples ==="

# Build the godl binary first
echo "Building godl..."
go build -o godl ./cmd/godl

echo ""
echo "1. Simple Download:"
echo "Command: ./godl https://httpbin.org/bytes/1024"
./godl -o simple.bin https://httpbin.org/bytes/1024
echo "✓ Simple download completed"
echo ""

echo "2. Download with Custom Output Filename:"
echo "Command: ./godl -o custom_name.json https://httpbin.org/json"
./godl -o custom_name.json https://httpbin.org/json
echo "✓ Download with custom filename completed"
echo ""

echo "3. Download with Force Overwrite:"
echo "Command: ./godl --force -o overwrite_test.json https://httpbin.org/json"
./godl --force -o overwrite_test.json https://httpbin.org/json
echo "✓ Force overwrite download completed"
echo ""

echo "4. Download with Directory Creation:"
echo "Command: ./godl --create-dirs -o downloads/subdir/test.json https://httpbin.org/json"
./godl --create-dirs -o downloads/subdir/test.json https://httpbin.org/json
echo "✓ Download with directory creation completed"
echo ""

echo "5. Quiet Mode Download:"
echo "Command: ./godl --quiet -o quiet_test.bin https://httpbin.org/bytes/2048"
./godl --quiet -o quiet_test.bin https://httpbin.org/bytes/2048
echo "✓ Quiet mode download completed"
echo ""

echo "6. Verbose Mode Download:"
echo "Command: ./godl --verbose -o verbose_test.json https://httpbin.org/json"
./godl --verbose -o verbose_test.json https://httpbin.org/json
echo ""

echo "7. Download with Custom User-Agent:"
echo "Command: ./godl --user-agent 'MyApp/1.0' -o useragent_test.json https://httpbin.org/user-agent"
./godl --user-agent 'MyApp/1.0' -o useragent_test.json https://httpbin.org/user-agent
echo "✓ Custom user-agent download completed"
echo ""

echo "8. Download with Timeout:"
echo "Command: ./godl --timeout 10s -o timeout_test.json https://httpbin.org/delay/1"
./godl --timeout 10s -o timeout_test.json https://httpbin.org/delay/1
echo "✓ Timeout download completed"
echo ""

echo "9. Resume Download (will demonstrate resume capability):"
echo "Command: ./godl --resume -o resume_test.bin https://httpbin.org/bytes/4096"
./godl --resume -o resume_test.bin https://httpbin.org/bytes/4096
echo "✓ Resume download completed"
echo ""

echo "10. Version and Help:"
echo "Command: ./godl --version"
./godl --version
echo ""
echo "Command: ./godl --help | head -10"
./godl --help | head -10
echo "..."
echo ""

echo "=== Basic CLI Examples Completed ==="
echo "Files created:"
ls -la *.json *.bin downloads/ 2>/dev/null | head -10 || echo "Some files may not exist"
echo ""