#!/bin/bash
# Performance regression check script for CI

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔍 Running performance regression check..."

# Thresholds (in percentage)
REGRESSION_THRESHOLD=10  # Fail if performance degrades by more than 10%
WARNING_THRESHOLD=5      # Warn if performance degrades by more than 5%

# Benchmark configuration - CI uses faster settings
if [ "${CI}" = "true" ]; then
    BENCH_TIME="100ms"
    BENCH_COUNT="1"
    BENCH_TIMEOUT="30s"
    BENCH_PACKAGES="./internal/core/..."
else
    # Local development can use more thorough testing
    BENCH_TIME="${BENCH_TIME:-500ms}"
    BENCH_COUNT="${BENCH_COUNT:-1}"
    BENCH_TIMEOUT="${BENCH_TIMEOUT:-1m}"
    BENCH_PACKAGES="${BENCH_PACKAGES:-./internal/core/...}"
fi

# Run benchmarks and save results
echo "Running benchmarks..."
echo "Settings: benchtime=$BENCH_TIME count=$BENCH_COUNT timeout=$BENCH_TIMEOUT packages=$BENCH_PACKAGES"
go test -bench=BenchmarkDownload -benchmem -benchtime=$BENCH_TIME -count=$BENCH_COUNT -timeout=$BENCH_TIMEOUT $BENCH_PACKAGES > current-bench.txt 2>&1

# Install benchstat if not available
if ! command -v benchstat &> /dev/null; then
    echo "📦 Installing benchstat..."
    go install golang.org/x/perf/cmd/benchstat@latest
    # Add GOPATH/bin to PATH if needed
    GOPATH_BIN=$(go env GOPATH)/bin
    export PATH=$PATH:$GOPATH_BIN
fi

# If we have a base benchmark file, compare
if [ -f "base-bench.txt" ]; then
    echo "📊 Comparing benchmarks..."

    # Run benchstat and capture output
    COMPARISON=$(benchstat base-bench.txt current-bench.txt 2>&1) || true

    echo "$COMPARISON"

    # Check for significant regressions
    if echo "$COMPARISON" | grep -E "\+[0-9]+\.[0-9]+%" > /dev/null; then
        # Extract percentage changes
        CHANGES=$(echo "$COMPARISON" | grep -E "\+[0-9]+\.[0-9]+%" | sed -E 's/.*\+([0-9]+\.[0-9]+)%.*/\1/')

        MAX_REGRESSION=0
        for change in $CHANGES; do
            # Convert to integer for comparison (bash doesn't do float comparison well)
            change_int=$(echo "$change" | cut -d. -f1)
            if [ "$change_int" -gt "$MAX_REGRESSION" ]; then
                MAX_REGRESSION=$change_int
            fi
        done

        if [ "$MAX_REGRESSION" -ge "$REGRESSION_THRESHOLD" ]; then
            echo -e "${RED}❌ Performance regression detected: +${MAX_REGRESSION}%${NC}"
            echo -e "${RED}   Threshold: ${REGRESSION_THRESHOLD}%${NC}"
            exit 1
        elif [ "$MAX_REGRESSION" -ge "$WARNING_THRESHOLD" ]; then
            echo -e "${YELLOW}⚠️ Minor performance regression: +${MAX_REGRESSION}%${NC}"
            echo -e "${YELLOW}   Warning threshold: ${WARNING_THRESHOLD}%${NC}"
        fi
    fi

    # Check for improvements
    if echo "$COMPARISON" | grep -E "\-[0-9]+\.[0-9]+%" > /dev/null; then
        echo -e "${GREEN}✅ Performance improvements detected!${NC}"
    fi
else
    echo "ℹ️ No base benchmark found, skipping comparison"
fi

echo "✅ Performance check completed"

# Save current as new baseline for next run (optional)
# cp current-bench.txt base-bench.txt