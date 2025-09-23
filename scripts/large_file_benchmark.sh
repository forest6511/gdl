#!/bin/bash
# Large file benchmark script for gdl vs curl vs wget
# Tests with 100MB, 500MB, and 1GB files

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configurations
ITERATIONS=3
OUTPUT_DIR="benchmark_results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULT_FILE="${OUTPUT_DIR}/large_files_${TIMESTAMP}.txt"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Large test files - using reliable CDN sources
declare -a TEST_FILES=(
    "100MB|https://speed.hetzner.de/100MB.bin"
    "200MB|https://ash-speed.hetzner.com/200MB.bin"
    "500MB|https://ash-speed.hetzner.com/500MB.bin"
    "1GB|https://ash-speed.hetzner.com/1GB.bin"
)

# Function to measure download time with timeout
measure_download() {
    local tool=$1
    local url=$2
    local output=$3
    local start_time
    local end_time
    local duration
    local timeout_duration=180  # 3 minutes timeout for large files

    case $tool in
        "gdl")
            start_time=$(date +%s.%N)
            timeout $timeout_duration ./gdl -q -o "$output" "$url" 2>/dev/null || true
            end_time=$(date +%s.%N)
            ;;
        "gdl_4")
            start_time=$(date +%s.%N)
            timeout $timeout_duration ./gdl -q --concurrent 4 -o "$output" "$url" 2>/dev/null || true
            end_time=$(date +%s.%N)
            ;;
        "gdl_8")
            start_time=$(date +%s.%N)
            timeout $timeout_duration ./gdl -q --concurrent 8 -o "$output" "$url" 2>/dev/null || true
            end_time=$(date +%s.%N)
            ;;
        "gdl_16")
            start_time=$(date +%s.%N)
            timeout $timeout_duration ./gdl -q --concurrent 16 -o "$output" "$url" 2>/dev/null || true
            end_time=$(date +%s.%N)
            ;;
        "curl")
            start_time=$(date +%s.%N)
            timeout $timeout_duration curl -s -L -o "$output" "$url" 2>/dev/null || true
            end_time=$(date +%s.%N)
            ;;
        "wget")
            start_time=$(date +%s.%N)
            timeout $timeout_duration wget -q -O "$output" "$url" 2>/dev/null || true
            end_time=$(date +%s.%N)
            ;;
    esac

    # Check if file was downloaded
    if [ -f "$output" ]; then
        duration=$(echo "$end_time - $start_time" | bc)
        local size=$(stat -f%z "$output" 2>/dev/null || stat -c%s "$output" 2>/dev/null || echo 0)

        # Calculate speed in MB/s
        if [ "$size" -gt 0 ] && [ $(echo "$duration > 0" | bc) -eq 1 ]; then
            local speed=$(echo "scale=2; $size / 1048576 / $duration" | bc)
            echo "$duration|$speed"
        else
            echo "999|0"  # Failed download
        fi
    else
        echo "999|0"  # Failed download
    fi

    # Clean up
    rm -f "$output"
}

# Print header
echo "======================================"
echo "Large File Performance Test"
echo "Date: $(date)"
echo "======================================"
echo ""

# Save to file
{
    echo "Large File Performance Report"
    echo "Generated: $(date)"
    echo "Iterations per test: $ITERATIONS"
    echo ""
} > "$RESULT_FILE"

# Check tools availability
echo -e "${BLUE}Checking tools...${NC}"
if [ ! -f "./gdl" ]; then
    echo -e "${YELLOW}Building gdl...${NC}"
    go build -o gdl ./cmd/gdl/
fi

command -v curl >/dev/null 2>&1 || { echo -e "${RED}curl is required but not installed.${NC}" >&2; exit 1; }
command -v wget >/dev/null 2>&1 || { echo -e "${RED}wget is required but not installed.${NC}" >&2; exit 1; }

echo -e "${GREEN}All tools available!${NC}\n"

# Run tests for each file size
for test_config in "${TEST_FILES[@]}"; do
    IFS='|' read -r size url <<< "$test_config"

    echo -e "${BLUE}Testing with $size file...${NC}"
    echo "URL: $url"
    echo ""

    # Results storage
    declare -a gdl_times=()
    declare -a gdl_speeds=()
    declare -a gdl4_times=()
    declare -a gdl4_speeds=()
    declare -a gdl8_times=()
    declare -a gdl8_speeds=()
    declare -a gdl16_times=()
    declare -a gdl16_speeds=()
    declare -a curl_times=()
    declare -a curl_speeds=()
    declare -a wget_times=()
    declare -a wget_speeds=()

    # Run multiple iterations
    for i in $(seq 1 $ITERATIONS); do
        echo -n "  Iteration $i/$ITERATIONS: "

        # Test gdl (smart defaults)
        echo -n "gdl..."
        result=$(measure_download "gdl" "$url" "test_gdl.tmp")
        IFS='|' read -r time speed <<< "$result"
        gdl_times+=($time)
        gdl_speeds+=($speed)

        # Test gdl with 4 connections
        echo -n "gdl(4)..."
        result=$(measure_download "gdl_4" "$url" "test_gdl4.tmp")
        IFS='|' read -r time speed <<< "$result"
        gdl4_times+=($time)
        gdl4_speeds+=($speed)

        # Test gdl with 8 connections
        echo -n "gdl(8)..."
        result=$(measure_download "gdl_8" "$url" "test_gdl8.tmp")
        IFS='|' read -r time speed <<< "$result"
        gdl8_times+=($time)
        gdl8_speeds+=($speed)

        # Test gdl with 16 connections
        echo -n "gdl(16)..."
        result=$(measure_download "gdl_16" "$url" "test_gdl16.tmp")
        IFS='|' read -r time speed <<< "$result"
        gdl16_times+=($time)
        gdl16_speeds+=($speed)

        # Test curl
        echo -n "curl..."
        result=$(measure_download "curl" "$url" "test_curl.tmp")
        IFS='|' read -r time speed <<< "$result"
        curl_times+=($time)
        curl_speeds+=($speed)

        # Test wget
        echo -n "wget..."
        result=$(measure_download "wget" "$url" "test_wget.tmp")
        IFS='|' read -r time speed <<< "$result"
        wget_times+=($time)
        wget_speeds+=($speed)

        echo "done"
    done

    # Calculate averages
    avg_gdl=$(echo "${gdl_times[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')
    avg_gdl_speed=$(echo "${gdl_speeds[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')

    avg_gdl4=$(echo "${gdl4_times[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')
    avg_gdl4_speed=$(echo "${gdl4_speeds[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')

    avg_gdl8=$(echo "${gdl8_times[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')
    avg_gdl8_speed=$(echo "${gdl8_speeds[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')

    avg_gdl16=$(echo "${gdl16_times[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')
    avg_gdl16_speed=$(echo "${gdl16_speeds[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')

    avg_curl=$(echo "${curl_times[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')
    avg_curl_speed=$(echo "${curl_speeds[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')

    avg_wget=$(echo "${wget_times[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')
    avg_wget_speed=$(echo "${wget_speeds[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')

    # Calculate relative performance
    if [ $(echo "$avg_curl > 0" | bc) -eq 1 ]; then
        curl_baseline=100
        gdl_perf=$(echo "scale=1; ($avg_curl / $avg_gdl) * 100" | bc 2>/dev/null || echo "N/A")
        gdl4_perf=$(echo "scale=1; ($avg_curl / $avg_gdl4) * 100" | bc 2>/dev/null || echo "N/A")
        gdl8_perf=$(echo "scale=1; ($avg_curl / $avg_gdl8) * 100" | bc 2>/dev/null || echo "N/A")
        gdl16_perf=$(echo "scale=1; ($avg_curl / $avg_gdl16) * 100" | bc 2>/dev/null || echo "N/A")
        wget_perf=$(echo "scale=1; ($avg_curl / $avg_wget) * 100" | bc 2>/dev/null || echo "N/A")
    else
        curl_baseline=100
        gdl_perf="N/A"
        gdl4_perf="N/A"
        gdl8_perf="N/A"
        gdl16_perf="N/A"
        wget_perf="N/A"
    fi

    # Print results
    echo ""
    echo "  Results for $size:"
    echo "  ----------------"
    printf "  %-20s: %6.2fs (%6.2f MB/s) - %s%% vs curl\n" "gdl (smart)" "$avg_gdl" "$avg_gdl_speed" "$gdl_perf"
    printf "  %-20s: %6.2fs (%6.2f MB/s) - %s%% vs curl\n" "gdl (4 connections)" "$avg_gdl4" "$avg_gdl4_speed" "$gdl4_perf"
    printf "  %-20s: %6.2fs (%6.2f MB/s) - %s%% vs curl\n" "gdl (8 connections)" "$avg_gdl8" "$avg_gdl8_speed" "$gdl8_perf"
    printf "  %-20s: %6.2fs (%6.2f MB/s) - %s%% vs curl\n" "gdl (16 connections)" "$avg_gdl16" "$avg_gdl16_speed" "$gdl16_perf"
    printf "  %-20s: %6.2fs (%6.2f MB/s) - baseline\n" "curl" "$avg_curl" "$avg_curl_speed"
    printf "  %-20s: %6.2fs (%6.2f MB/s) - %s%% vs curl\n" "wget" "$avg_wget" "$avg_wget_speed" "$wget_perf"
    echo ""

    # Save to file
    {
        echo "Test: $size file"
        echo "URL: $url"
        echo "Results (average of $ITERATIONS iterations):"
        printf "  %-20s: %6.2fs (%6.2f MB/s) - %s%% vs curl\n" "gdl (smart)" "$avg_gdl" "$avg_gdl_speed" "$gdl_perf"
        printf "  %-20s: %6.2fs (%6.2f MB/s) - %s%% vs curl\n" "gdl (4 connections)" "$avg_gdl4" "$avg_gdl4_speed" "$gdl4_perf"
        printf "  %-20s: %6.2fs (%6.2f MB/s) - %s%% vs curl\n" "gdl (8 connections)" "$avg_gdl8" "$avg_gdl8_speed" "$gdl8_perf"
        printf "  %-20s: %6.2fs (%6.2f MB/s) - %s%% vs curl\n" "gdl (16 connections)" "$avg_gdl16" "$avg_gdl16_speed" "$gdl16_perf"
        printf "  %-20s: %6.2fs (%6.2f MB/s) - baseline\n" "curl" "$avg_curl" "$avg_curl_speed"
        printf "  %-20s: %6.2fs (%6.2f MB/s) - %s%% vs curl\n" "wget" "$avg_wget" "$avg_wget_speed" "$wget_perf"
        echo "----------------------------------------"
        echo ""
    } >> "$RESULT_FILE"
done

# Summary
echo "======================================"
echo -e "${GREEN}Test completed!${NC}"
echo "Results saved to: $RESULT_FILE"
echo ""

# Display summary
echo "Performance Summary:"
echo "-------------------"
grep -E "gdl \(smart\)|curl|MB/s" "$RESULT_FILE" | tail -20

echo ""
echo -e "${BLUE}Analysis:${NC}"
echo "- Higher MB/s = Better throughput"
echo "- Lower time = Faster download"
echo "- Values >100% mean faster than curl"
echo ""