#!/bin/bash
# Real-world environment testing script

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Output directory
OUTPUT_DIR="benchmark_results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULT_FILE="${OUTPUT_DIR}/real_world_${TIMESTAMP}.txt"

mkdir -p "$OUTPUT_DIR"

echo "======================================"
echo "Real-World Environment Testing"
echo "Date: $(date)"
echo "======================================"

# Test scenarios
declare -a TESTS=(
    "CDN|CloudFlare|https://cdnjs.cloudflare.com/ajax/libs/jquery/3.6.0/jquery.min.js"
    "CDN|jsDelivr|https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css"
    "GitHub|Release|https://github.com/git/git/archive/refs/tags/v2.42.0.tar.gz"
    "Package|NPM|https://registry.npmjs.org/react/-/react-18.2.0.tgz"
    "Media|Image|https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=1920"
)

# Function to test download
test_download() {
    local category=$1
    local provider=$2
    local url=$3

    echo -e "${BLUE}Testing: $category - $provider${NC}"
    echo "URL: $url"

    # gdl test
    echo -n "gdl: "
    gdl_start=$(date +%s.%N)
    ./gdl -q -o test_gdl.tmp "$url" 2>/dev/null
    gdl_end=$(date +%s.%N)
    gdl_time=$(echo "$gdl_end - $gdl_start" | bc)
    gdl_size=$(stat -f%z test_gdl.tmp 2>/dev/null || stat -c%s test_gdl.tmp 2>/dev/null || echo 0)

    # curl test
    echo -n "curl: "
    curl_start=$(date +%s.%N)
    curl -s -L -o test_curl.tmp "$url" 2>/dev/null
    curl_end=$(date +%s.%N)
    curl_time=$(echo "$curl_end - $curl_start" | bc)

    # Calculate performance
    if [ $(echo "$curl_time > 0" | bc) -eq 1 ]; then
        perf=$(echo "scale=1; ($curl_time / $gdl_time) * 100" | bc)
    else
        perf="N/A"
    fi

    # Calculate speed
    if [ "$gdl_size" -gt 0 ] && [ $(echo "$gdl_time > 0" | bc) -eq 1 ]; then
        speed=$(echo "scale=2; $gdl_size / 1048576 / $gdl_time" | bc)
    else
        speed="0"
    fi

    printf "gdl: %.2fs (%.2f MB/s), curl: %.2fs, Performance: %s%%\n" "$gdl_time" "$speed" "$curl_time" "$perf"

    # Save to file
    {
        echo "Test: $category - $provider"
        echo "URL: $url"
        printf "gdl: %.2fs (%.2f MB/s)\n" "$gdl_time" "$speed"
        printf "curl: %.2fs\n" "$curl_time"
        printf "Performance: %s%% of curl speed\n" "$perf"
        echo "---"
    } >> "$RESULT_FILE"

    # Cleanup
    rm -f test_gdl.tmp test_curl.tmp
    echo ""
}

# Run tests
for test_config in "${TESTS[@]}"; do
    IFS='|' read -r category provider url <<< "$test_config"
    test_download "$category" "$provider" "$url"
done

# Network latency test
echo -e "${BLUE}Testing Network Latency Impact${NC}"
echo "Testing with servers at different geographic locations..."

# Geographic diversity tests
declare -a GEO_TESTS=(
    "US-West|https://speed.hetzner.de/100MB.bin"
    "Europe|https://fsn1-speed.hetzner.com/100MB.bin"
    "Asia|https://ash-speed.hetzner.com/100MB.bin"
)

for test_config in "${GEO_TESTS[@]}"; do
    IFS='|' read -r location url <<< "$test_config"
    echo "Location: $location"

    # Just test first 10MB to save bandwidth
    echo -n "Testing first 10MB... "

    gdl_start=$(date +%s.%N)
    timeout 10 ./gdl -q -o test.tmp "$url" 2>/dev/null || true
    gdl_end=$(date +%s.%N)
    gdl_time=$(echo "$gdl_end - $gdl_start" | bc)

    curl_start=$(date +%s.%N)
    timeout 10 curl -s -L -o test.tmp "$url" 2>/dev/null || true
    curl_end=$(date +%s.%N)
    curl_time=$(echo "$curl_end - $curl_start" | bc)

    printf "gdl: %.2fs, curl: %.2fs\n" "$gdl_time" "$curl_time"
    rm -f test.tmp
done

echo ""
echo -e "${GREEN}Testing Complete!${NC}"
echo "Results saved to: $RESULT_FILE"

# Summary
echo ""
echo "Summary:"
grep -E "Performance:|MB/s" "$RESULT_FILE" | tail -10