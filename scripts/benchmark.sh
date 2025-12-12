#!/bin/bash
#
# benchmark.sh - Benchmark Ollama model performance for code analysis
#
# Usage: ./scripts/benchmark.sh
#        ./scripts/benchmark.sh --model deepseek-coder:6.7b
#        ./scripts/benchmark.sh --all   # Test all models
#

set -e

# Defaults
OLLAMA_HOST="${OLLAMA_HOST:-http://localhost:11434}"
DEFAULT_MODELS="tinyllama llama3.2:3b deepseek-coder:6.7b mistral:7b"
TEST_MODEL=""
TEST_ALL=false

# Test prompts (increasing complexity)
PROMPTS=(
    "What is 2+2?"
    "Write a Python function that checks if a number is prime"
    "Analyze this code and explain what it does:\n\ndef quicksort(arr):\n    if len(arr) <= 1:\n        return arr\n    pivot = arr[len(arr) // 2]\n    left = [x for x in arr if x < pivot]\n    middle = [x for x in arr if x == pivot]\n    right = [x for x in arr if x > pivot]\n    return quicksort(left) + middle + quicksort(right)"
    "Write a REST API endpoint in Python using FastAPI that accepts a JSON payload with 'code' field, analyzes it for security vulnerabilities, and returns a report"
)

PROMPT_NAMES=(
    "simple_math"
    "write_function"
    "analyze_code"
    "complex_task"
)

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --model)
            TEST_MODEL="$2"
            shift 2
            ;;
        --all)
            TEST_ALL=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

# Get system info
get_system_info() {
    echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  System Info${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"

    # CPU Governor
    GOV=$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || echo "unknown")
    echo "CPU Governor: $GOV"

    # Current CPU freq
    FREQ=$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq 2>/dev/null || echo "0")
    FREQ_MHZ=$((FREQ / 1000))
    echo "CPU Frequency: ${FREQ_MHZ} MHz"

    # Available RAM
    RAM_AVAIL=$(free -g 2>/dev/null | awk '/Mem:/ {print $7}' || echo "?")
    echo "Available RAM: ${RAM_AVAIL}GB"

    # GPU busy (if available)
    GPU_BUSY=$(cat /sys/class/drm/card0/device/gpu_busy_percent 2>/dev/null || echo "N/A")
    echo "GPU Busy: ${GPU_BUSY}%"

    echo ""
}

# Benchmark a single model with a single prompt
benchmark_single() {
    local model="$1"
    local prompt="$2"
    local prompt_name="$3"

    # Warm up the model (load into memory)
    curl -s "$OLLAMA_HOST/api/generate" \
        -d "{\"model\":\"$model\",\"prompt\":\"hello\",\"stream\":false}" > /dev/null 2>&1

    # Time the actual request
    START=$(date +%s.%N)

    RESPONSE=$(curl -s "$OLLAMA_HOST/api/generate" \
        -d "{\"model\":\"$model\",\"prompt\":\"$prompt\",\"stream\":false}")

    END=$(date +%s.%N)

    # Parse response
    EVAL_COUNT=$(echo "$RESPONSE" | jq -r '.eval_count // 0')
    EVAL_DURATION=$(echo "$RESPONSE" | jq -r '.eval_duration // 0')
    TOTAL_DURATION=$(echo "$RESPONSE" | jq -r '.total_duration // 0')

    # Calculate tokens/sec
    if [ "$EVAL_DURATION" -gt 0 ]; then
        TOKENS_PER_SEC=$(echo "scale=2; $EVAL_COUNT / ($EVAL_DURATION / 1000000000)" | bc)
    else
        TOKENS_PER_SEC="0"
    fi

    # Wall clock time
    WALL_TIME=$(echo "$END - $START" | bc)

    echo "$model,$prompt_name,$EVAL_COUNT,$TOKENS_PER_SEC,$WALL_TIME"
}

# Run full benchmark
run_benchmark() {
    local models="$1"

    echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  Ollama Code Analysis Benchmark${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo ""

    get_system_info

    # Results file
    RESULTS_FILE="/tmp/benchmark_$(date +%Y%m%d_%H%M%S).csv"
    echo "model,test,tokens,tok_per_sec,wall_time" > "$RESULTS_FILE"

    for model in $models; do
        echo -e "${YELLOW}Testing: $model${NC}"

        # Check if model exists
        if ! ollama list 2>/dev/null | grep -q "^$model"; then
            echo -e "  ${RED}Model not found, skipping${NC}"
            continue
        fi

        for i in "${!PROMPTS[@]}"; do
            prompt="${PROMPTS[$i]}"
            prompt_name="${PROMPT_NAMES[$i]}"

            echo -n "  $prompt_name: "

            result=$(benchmark_single "$model" "$prompt" "$prompt_name")
            echo "$result" >> "$RESULTS_FILE"

            # Parse and display
            tokens=$(echo "$result" | cut -d, -f3)
            tps=$(echo "$result" | cut -d, -f4)
            wall=$(echo "$result" | cut -d, -f5)

            echo -e "${GREEN}${tokens} tokens @ ${tps} tok/s (${wall}s)${NC}"
        done
        echo ""
    done

    echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  Summary${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo "Results saved to: $RESULTS_FILE"
    echo ""

    # Display summary table
    echo "Model                     | Avg tok/s | Best Test"
    echo "--------------------------|-----------|----------"

    for model in $models; do
        if grep -q "^$model," "$RESULTS_FILE"; then
            avg_tps=$(grep "^$model," "$RESULTS_FILE" | awk -F, '{sum+=$4; count++} END {printf "%.1f", sum/count}')
            best=$(grep "^$model," "$RESULTS_FILE" | sort -t, -k4 -rn | head -1 | cut -d, -f2)
            printf "%-25s | %9s | %s\n" "$model" "$avg_tps" "$best"
        fi
    done

    echo ""
    echo "To compare before/after CPU tuning, save this file and run again."
}

# Main
if [ -n "$TEST_MODEL" ]; then
    run_benchmark "$TEST_MODEL"
elif [ "$TEST_ALL" = true ]; then
    run_benchmark "$DEFAULT_MODELS"
else
    # Quick test with fastest model
    echo "Quick benchmark with tinyllama (use --all for full test)"
    echo ""
    run_benchmark "tinyllama"
fi
