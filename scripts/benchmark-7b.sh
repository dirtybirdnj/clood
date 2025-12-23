#!/bin/bash
# Benchmark: Ollama vs llama.cpp on ubuntu25 (7B models)
# Tests qwen2.5-coder:7b on both backends with RX 590 Vulkan

set -e

OLLAMA_URL="http://192.168.4.64:11434"
LLAMACPP_URL="http://192.168.4.64:8080"
PROMPT="Write a Python function that implements binary search on a sorted list. Include docstring and type hints."
RUNS=3

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Backend Benchmark: Ollama vs llama.cpp (7B Models)           ${NC}"
echo -e "${BLUE}  ubuntu25 - RX 590 8GB - Vulkan                               ${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo "Ollama model:    qwen2.5-coder:7b"
echo "llama.cpp model: qwen2.5-coder-7b-instruct-q4_k_m.gguf"
echo "Prompt: ${PROMPT:0:60}..."
echo "Runs: $RUNS"
echo ""

# Warm up
echo -e "${YELLOW}Warming up models...${NC}"
curl -s "$OLLAMA_URL/api/generate" -d '{"model":"qwen2.5-coder:7b","prompt":"hi","stream":false}' > /dev/null
echo "  Ollama:    warmed"
curl -s "$LLAMACPP_URL/v1/chat/completions" -H "Content-Type: application/json" \
    -d '{"model":"qwen","messages":[{"role":"user","content":"hi"}],"max_tokens":10}' > /dev/null
echo "  llama.cpp: warmed"
echo ""

benchmark_ollama() {
    local start=$(python3 -c 'import time; print(time.time())')
    local result=$(curl -s "$OLLAMA_URL/api/generate" \
        -d "{\"model\":\"qwen2.5-coder:7b\",\"prompt\":\"$PROMPT\",\"stream\":false}")
    local end=$(python3 -c 'import time; print(time.time())')

    local eval_count=$(echo "$result" | jq -r '.eval_count // 0')
    local eval_duration=$(echo "$result" | jq -r '.eval_duration // 1')
    local tok_per_sec=$(python3 -c "print(f'{$eval_count / ($eval_duration / 1e9):.2f}')" 2>/dev/null || echo "0")
    local wall_time=$(python3 -c "print(f'{$end - $start:.2f}')")
    echo "$eval_count $tok_per_sec $wall_time"
}

benchmark_llamacpp() {
    local start=$(python3 -c 'import time; print(time.time())')
    local result=$(curl -s "$LLAMACPP_URL/v1/chat/completions" \
        -H "Content-Type: application/json" \
        -d "{\"model\":\"qwen\",\"messages\":[{\"role\":\"user\",\"content\":\"$PROMPT\"}],\"max_tokens\":300}")
    local end=$(python3 -c 'import time; print(time.time())')

    local tokens=$(echo "$result" | jq -r '.usage.completion_tokens // 0')
    local wall_time=$(python3 -c "print(f'{$end - $start:.2f}')")
    local tok_per_sec=$(python3 -c "print(f'{$tokens / ($end - $start):.2f}')" 2>/dev/null || echo "0")
    echo "$tokens $tok_per_sec $wall_time"
}

echo -e "${GREEN}Running Ollama benchmarks (qwen2.5-coder:7b)...${NC}"
ollama_results=()
for i in $(seq 1 $RUNS); do
    result=$(benchmark_ollama)
    ollama_results+=("$result")
    tokens=$(echo $result | cut -d' ' -f1)
    tps=$(echo $result | cut -d' ' -f2)
    wall=$(echo $result | cut -d' ' -f3)
    echo "  Run $i: ${tokens} tokens, ${tps} tok/s, ${wall}s"
done

echo ""
echo -e "${GREEN}Running llama.cpp benchmarks (qwen2.5-coder-7b)...${NC}"
llamacpp_results=()
for i in $(seq 1 $RUNS); do
    result=$(benchmark_llamacpp)
    llamacpp_results+=("$result")
    tokens=$(echo $result | cut -d' ' -f1)
    tps=$(echo $result | cut -d' ' -f2)
    wall=$(echo $result | cut -d' ' -f3)
    echo "  Run $i: ${tokens} tokens, ${tps} tok/s, ${wall}s"
done

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  RESULTS SUMMARY (7B Models on RX 590)                        ${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

ollama_tps_sum=0
for r in "${ollama_results[@]}"; do
    tps=$(echo $r | cut -d' ' -f2)
    ollama_tps_sum=$(python3 -c "print($ollama_tps_sum + $tps)")
done
ollama_avg=$(python3 -c "print(f'{$ollama_tps_sum / $RUNS:.2f}')")

llamacpp_tps_sum=0
for r in "${llamacpp_results[@]}"; do
    tps=$(echo $r | cut -d' ' -f2)
    llamacpp_tps_sum=$(python3 -c "print($llamacpp_tps_sum + $tps)")
done
llamacpp_avg=$(python3 -c "print(f'{$llamacpp_tps_sum / $RUNS:.2f}')")

printf "  %-15s %10s tok/s\n" "Ollama:" "$ollama_avg"
printf "  %-15s %10s tok/s\n" "llama.cpp:" "$llamacpp_avg"
echo ""

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
echo "Both using: Vulkan on Radeon RX 590 (8GB VRAM)"

# Save results to ollama-tuning.md if it exists
TUNING_FILE="/Users/mgilbert/Code/clood/ollama-tuning.md"
if [ -f "$TUNING_FILE" ]; then
    # Check if Backend Comparison section exists
    if ! grep -q "## Backend Comparison: Ollama vs llama.cpp" "$TUNING_FILE"; then
        cat >> "$TUNING_FILE" << 'SECTION'

---

## Backend Comparison: Ollama vs llama.cpp

Both Ollama and llama.cpp can run GGUF models. This section compares performance on the same hardware.

### ubuntu25 (RX 590 8GB Vulkan)

| Model | Ollama | llama.cpp | Winner |
|-------|--------|-----------|--------|
SECTION
    fi

    # Append new benchmark result
    DATE=$(date +%Y-%m-%d)
    echo "| Qwen 7B Q4_K_M | ${ollama_avg} tok/s | ${llamacpp_avg} tok/s | $(python3 -c "print('llama.cpp' if $llamacpp_avg > $ollama_avg else 'Ollama')") | <!-- $DATE -->" >> "$TUNING_FILE"
    echo ""
    echo -e "${GREEN}Results saved to ollama-tuning.md${NC}"
fi
