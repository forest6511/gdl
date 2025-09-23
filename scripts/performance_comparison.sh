#!/bin/bash
# Performance comparison script for gdl vs curl vs wget

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
RESULT_FILE="${OUTPUT_DIR}/comparison_${TIMESTAMP}.txt"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Test URLs with different file sizes
declare -a TEST_FILES=(
    "1KB|https://raw.githubusercontent.com/git/git/master/.gitignore"
    "10KB|https://raw.githubusercontent.com/torvalds/linux/master/CREDITS"
    "100KB|https://raw.githubusercontent.com/torvalds/linux/master/MAINTAINERS"
    "500KB|https://raw.githubusercontent.com/torvalds/linux/master/Documentation/admin-guide/kernel-parameters.txt"
    "1MB|https://github.com/git/git/archive/refs/tags/v2.30.0.tar.gz"
    "10MB|https://github.com/git/git/archive/refs/tags/v2.42.0.tar.gz"
)

# Function to measure download time
measure_download() {
    local tool=$1
    local url=$2
    local output=$3
    local start_time
    local end_time
    local duration

    case $tool in
        "gdl")
            start_time=$(date +%s.%N)
            ./gdl -q -o "$output" "$url" 2>/dev/null
            end_time=$(date +%s.%N)
            ;;
        "gdl_single")
            start_time=$(date +%s.%N)
            ./gdl -q --concurrent 1 -o "$output" "$url" 2>/dev/null
            end_time=$(date +%s.%N)
            ;;
        "curl")
            start_time=$(date +%s.%N)
            curl -s -L -o "$output" "$url" 2>/dev/null
            end_time=$(date +%s.%N)
            ;;
        "wget")
            start_time=$(date +%s.%N)
            wget -q -O "$output" "$url" 2>/dev/null
            end_time=$(date +%s.%N)
            ;;
    esac

    duration=$(echo "$end_time - $start_time" | bc)
    echo "$duration"

    # Clean up
    rm -f "$output"
}

# Print header
echo "======================================"
echo "Performance Comparison Test"
echo "Date: $(date)"
echo "======================================"
echo ""

# Save to file
{
    echo "Performance Comparison Report"
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
    declare -a gdl_single_times=()
    declare -a curl_times=()
    declare -a wget_times=()

    # Run multiple iterations
    for i in $(seq 1 $ITERATIONS); do
        echo -n "  Iteration $i/$ITERATIONS: "

        # Test gdl (smart defaults)
        echo -n "gdl..."
        time_gdl=$(measure_download "gdl" "$url" "test_gdl.tmp")
        gdl_times+=($time_gdl)

        # Test gdl (single connection)
        echo -n "gdl(1)..."
        time_gdl_single=$(measure_download "gdl_single" "$url" "test_gdl_single.tmp")
        gdl_single_times+=($time_gdl_single)

        # Test curl
        echo -n "curl..."
        time_curl=$(measure_download "curl" "$url" "test_curl.tmp")
        curl_times+=($time_curl)

        # Test wget
        echo -n "wget..."
        time_wget=$(measure_download "wget" "$url" "test_wget.tmp")
        wget_times+=($time_wget)

        echo "done"
    done

    # Calculate averages
    avg_gdl=$(echo "${gdl_times[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')
    avg_gdl_single=$(echo "${gdl_single_times[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')
    avg_curl=$(echo "${curl_times[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')
    avg_wget=$(echo "${wget_times[@]}" | awk '{sum=0; for(i=1;i<=NF;i++)sum+=$i; print sum/NF}')

    # Calculate relative performance
    curl_baseline=100
    gdl_perf=$(echo "scale=1; ($avg_curl / $avg_gdl) * 100" | bc)
    gdl_single_perf=$(echo "scale=1; ($avg_curl / $avg_gdl_single) * 100" | bc)
    wget_perf=$(echo "scale=1; ($avg_curl / $avg_wget) * 100" | bc)

    # Print results
    echo ""
    echo "  Results for $size:"
    echo "  ----------------"
    printf "  %-15s: %6.3fs (%.0f%% vs curl)\n" "gdl (smart)" "$avg_gdl" "$gdl_perf"
    printf "  %-15s: %6.3fs (%.0f%% vs curl)\n" "gdl (single)" "$avg_gdl_single" "$gdl_single_perf"
    printf "  %-15s: %6.3fs (baseline)\n" "curl" "$avg_curl"
    printf "  %-15s: %6.3fs (%.0f%% vs curl)\n" "wget" "$avg_wget" "$wget_perf"
    echo ""

    # Save to file
    {
        echo "Test: $size file"
        echo "URL: $url"
        echo "Results (average of $ITERATIONS iterations):"
        printf "  %-15s: %6.3fs (%.0f%% vs curl)\n" "gdl (smart)" "$avg_gdl" "$gdl_perf"
        printf "  %-15s: %6.3fs (%.0f%% vs curl)\n" "gdl (single)" "$avg_gdl_single" "$gdl_single_perf"
        printf "  %-15s: %6.3fs (baseline)\n" "curl" "$avg_curl"
        printf "  %-15s: %6.3fs (%.0f%% vs curl)\n" "wget" "$avg_wget" "$wget_perf"
        echo ""
        echo "Raw data:"
        echo "  gdl (smart): ${gdl_times[@]}"
        echo "  gdl (single): ${gdl_single_times[@]}"
        echo "  curl: ${curl_times[@]}"
        echo "  wget: ${wget_times[@]}"
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
echo "Summary:"
echo "--------"
cat "$RESULT_FILE" | grep -E "Test:|gdl \(smart\)|curl" | sed 's/^/  /'

echo ""
echo -e "${BLUE}Performance Analysis:${NC}"
echo "- Values >100% mean faster than curl"
echo "- Values <100% mean slower than curl"
echo ""