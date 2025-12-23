#!/bin/bash
# Benchmark: Ollama vs llama.cpp on ubuntu25
# Tests the same model (TinyLlama) on both backends

set -e

HOST="ubuntu25"
OLLAMA_URL="http://192.168.4.64:11434"
LLAMACPP_URL="http://192.168.4.64:8080"
PROMPT="Write a short story about a robot learning to garden. Include dialogue and be creative."
RUNS=3

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Backend Benchmark: Ollama vs llama.cpp on ubuntu25 (RX 590)  ${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo "Model: TinyLlama 1.1B"
echo "Prompt: ${PROMPT:0:60}..."
echo "Runs: $RUNS"
echo ""

# Check both are online
echo -e "${YELLOW}Checking backends...${NC}"
if ! curl -s "$OLLAMA_URL/api/tags" > /dev/null 2>&1; then
    echo "ERROR: Ollama not responding at $OLLAMA_URL"
    exit 1
fi
echo "  Ollama:    online"

if ! curl -s "$LLAMACPP_URL/health" > /dev/null 2>&1; then
    echo "ERROR: llama.cpp not responding at $LLAMACPP_URL"
    exit 1
fi
echo "  llama.cpp: online"
echo ""

# Warm up both (first inference is always slow due to model loading)
echo -e "${YELLOW}Warming up models...${NC}"
curl -s "$OLLAMA_URL/api/generate" -d '{"model":"tinyllama","prompt":"hi","stream":false}' > /dev/null
echo "  Ollama:    warmed"
curl -s "$LLAMACPP_URL/v1/chat/completions" -H "Content-Type: application/json" \
    -d '{"model":"tinyllama","messages":[{"role":"user","content":"hi"}],"max_tokens":10}' > /dev/null
echo "  llama.cpp: warmed"
echo ""

# Benchmark function for Ollama
benchmark_ollama() {
    local start=$(python3 -c 'import time; print(time.time())')

    local result=$(curl -s "$OLLAMA_URL/api/generate" \
        -d "{\"model\":\"tinyllama\",\"prompt\":\"$PROMPT\",\"stream\":false}")

    local end=$(python3 -c 'import time; print(time.time())')

    local eval_count=$(echo "$result" | jq -r '.eval_count // 0')
    local eval_duration=$(echo "$result" | jq -r '.eval_duration // 1')
    local total_duration=$(echo "$result" | jq -r '.total_duration // 1')

    # Ollama reports duration in nanoseconds
    local tok_per_sec=$(python3 -c "print(f'{$eval_count / ($eval_duration / 1e9):.2f}')" 2>/dev/null || echo "0")
    local wall_time=$(python3 -c "print(f'{$end - $start:.2f}')")

    echo "$eval_count $tok_per_sec $wall_time"
}

# Benchmark function for llama.cpp
benchmark_llamacpp() {
    local start=$(python3 -c 'import time; print(time.time())')

    local result=$(curl -s "$LLAMACPP_URL/v1/chat/completions" \
        -H "Content-Type: application/json" \
        -d "{\"model\":\"tinyllama\",\"messages\":[{\"role\":\"user\",\"content\":\"$PROMPT\"}],\"max_tokens\":256}")

    local end=$(python3 -c 'import time; print(time.time())')

    local tokens=$(echo "$result" | jq -r '.usage.completion_tokens // 0')
    local wall_time=$(python3 -c "print(f'{$end - $start:.2f}')")
    local tok_per_sec=$(python3 -c "print(f'{$tokens / ($end - $start):.2f}')" 2>/dev/null || echo "0")

    echo "$tokens $tok_per_sec $wall_time"
}

# Run benchmarks
echo -e "${GREEN}Running Ollama benchmarks...${NC}"
ollama_results=()
for i in $(seq 1 $RUNS); do
    result=$(benchmark_ollama)
    ollama_results+=("$result")
    tokens=$(echo $result | cut -d' ' -f1)
    tps=$(echo $result | cut -d' ' -f2)
    wall=$(echo $result | cut -d' ' -f3)
    echo "  Run $i: ${tokens} tokens, ${tps} tok/s, ${wall}s wall time"
done

echo ""
echo -e "${GREEN}Running llama.cpp benchmarks...${NC}"
llamacpp_results=()
for i in $(seq 1 $RUNS); do
    result=$(benchmark_llamacpp)
    llamacpp_results+=("$result")
    tokens=$(echo $result | cut -d' ' -f1)
    tps=$(echo $result | cut -d' ' -f2)
    wall=$(echo $result | cut -d' ' -f3)
    echo "  Run $i: ${tokens} tokens, ${tps} tok/s, ${wall}s wall time"
done

# Calculate averages
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  RESULTS SUMMARY                                              ${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

# Ollama average
ollama_tps_sum=0
for r in "${ollama_results[@]}"; do
    tps=$(echo $r | cut -d' ' -f2)
    ollama_tps_sum=$(python3 -c "print($ollama_tps_sum + $tps)")
done
ollama_avg=$(python3 -c "print(f'{$ollama_tps_sum / $RUNS:.2f}')")

# llama.cpp average
llamacpp_tps_sum=0
for r in "${llamacpp_results[@]}"; do
    tps=$(echo $r | cut -d' ' -f2)
    llamacpp_tps_sum=$(python3 -c "print($llamacpp_tps_sum + $tps)")
done
llamacpp_avg=$(python3 -c "print(f'{$llamacpp_tps_sum / $RUNS:.2f}')")

printf "  %-15s %10s tok/s\n" "Ollama:" "$ollama_avg"
printf "  %-15s %10s tok/s\n" "llama.cpp:" "$llamacpp_avg"
echo ""

# Winner
winner_diff=$(python3 -c "
o = $ollama_avg
l = $llamacpp_avg
if o > l:
    print(f'Ollama is {((o-l)/l)*100:.1f}% faster')
elif l > o:
    print(f'llama.cpp is {((l-o)/o)*100:.1f}% faster')
else:
    print('Dead heat!')
")
echo -e "  ${YELLOW}$winner_diff${NC}"
echo ""
echo "Note: Both using Vulkan on RX 590"

# Save results to ollama-tuning.md if it exists
TUNING_FILE="/Users/mgilbert/Code/clood/ollama-tuning.md"
if [ -f "$TUNING_FILE" ]; then
    # Check if Backend Comparison section exists, if not it will be created by benchmark-7b.sh
    if grep -q "## Backend Comparison: Ollama vs llama.cpp" "$TUNING_FILE"; then
        DATE=$(date +%Y-%m-%d)
        echo "| TinyLlama 1.1B Q4_K_M | ${ollama_avg} tok/s | ${llamacpp_avg} tok/s | $(python3 -c "print('llama.cpp' if $llamacpp_avg > $ollama_avg else 'Ollama')") | <!-- $DATE -->" >> "$TUNING_FILE"
        echo ""
        echo -e "${GREEN}Results saved to ollama-tuning.md${NC}"
    fi
fi
