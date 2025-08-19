#!/bin/bash

# Basic CLI Usage Examples for gdl
# This script demonstrates all the basic command-line interface features

echo "=== Basic CLI Usage Examples ==="

# Build the gdl binary first
echo "Building gdl..."
go build -o gdl ./cmd/gdl

echo ""
echo "1. Simple Download:"
echo "Command: ./gdl https://httpbin.org/bytes/1024"
./gdl -o simple.bin https://httpbin.org/bytes/1024
echo "✓ Simple download completed"
echo ""

echo "2. Download with Custom Output Filename:"
echo "Command: ./gdl -o custom_name.json https://httpbin.org/json"
./gdl -o custom_name.json https://httpbin.org/json
echo "✓ Download with custom filename completed"
echo ""

echo "3. Download with Force Overwrite:"
echo "Command: ./gdl --force -o overwrite_test.json https://httpbin.org/json"
./gdl --force -o overwrite_test.json https://httpbin.org/json
echo "✓ Force overwrite download completed"
echo ""

echo "4. Download with Directory Creation:"
echo "Command: ./gdl --create-dirs -o downloads/subdir/test.json https://httpbin.org/json"
./gdl --create-dirs -o downloads/subdir/test.json https://httpbin.org/json
echo "✓ Download with directory creation completed"
echo ""

echo "5. Quiet Mode Download:"
echo "Command: ./gdl --quiet -o quiet_test.bin https://httpbin.org/bytes/2048"
./gdl --quiet -o quiet_test.bin https://httpbin.org/bytes/2048
echo "✓ Quiet mode download completed"
echo ""

echo "6. Verbose Mode Download:"
echo "Command: ./gdl --verbose -o verbose_test.json https://httpbin.org/json"
./gdl --verbose -o verbose_test.json https://httpbin.org/json
echo ""

echo "7. Download with Custom User-Agent:"
echo "Command: ./gdl --user-agent 'MyApp/1.0' -o useragent_test.json https://httpbin.org/user-agent"
./gdl --user-agent 'MyApp/1.0' -o useragent_test.json https://httpbin.org/user-agent
echo "✓ Custom user-agent download completed"
echo ""

echo "8. Download with Timeout:"
echo "Command: ./gdl --timeout 10s -o timeout_test.json https://httpbin.org/delay/1"
./gdl --timeout 10s -o timeout_test.json https://httpbin.org/delay/1
echo "✓ Timeout download completed"
echo ""

echo "9. Resume Download (will demonstrate resume capability):"
echo "Command: ./gdl --resume -o resume_test.bin https://httpbin.org/bytes/4096"
./gdl --resume -o resume_test.bin https://httpbin.org/bytes/4096
echo "✓ Resume download completed"
echo ""

echo "10. Version and Help:"
echo "Command: ./gdl --version"
./gdl --version
echo ""
echo "Command: ./gdl --help | head -10"
./gdl --help | head -10
echo "..."
echo ""

echo "=== Basic CLI Examples Completed ==="
echo "Files created:"
ls -la *.json *.bin downloads/ 2>/dev/null | head -10 || echo "Some files may not exist"
echo ""