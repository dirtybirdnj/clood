#!/bin/bash
#
# chain.sh - Multi-model chaining for clood
#
# Uses fast model for initial analysis, then passes to larger model.
# Works with LiteLLM proxy or direct Ollama.
#
# Usage:
#   ./chain.sh "analyze this codebase for bottlenecks"
#   ./chain.sh --fast tinyllama --deep deepseek-coder:6.7b "query"
#   cat file.py | ./chain.sh "review this code"
#
# Environment:
#   OLLAMA_HOST - Ollama endpoint (default: http://localhost:11434)
#   LITELLM_HOST - LiteLLM proxy (default: none, uses Ollama directly)
#

set -e

# Defaults
FAST_MODEL="${FAST_MODEL:-llama3.2:3b}"
DEEP_MODEL="${DEEP_MODEL:-deepseek-coder:6.7b}"
OLLAMA_HOST="${OLLAMA_HOST:-http://localhost:11434}"
LITELLM_HOST="${LITELLM_HOST:-}"

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --fast)
            FAST_MODEL="$2"
            shift 2
            ;;
        --deep)
            DEEP_MODEL="$2"
            shift 2
            ;;
        --litellm)
            LITELLM_HOST="$2"
            shift 2
            ;;
        *)
            QUERY="$1"
            shift
            ;;
    esac
done

# Check for piped input
if [ -t 0 ]; then
    CONTEXT=""
else
    CONTEXT=$(cat)
fi

# Determine API endpoint
if [ -n "$LITELLM_HOST" ]; then
    API_BASE="$LITELLM_HOST"
    API_PATH="/chat/completions"
else
    API_BASE="$OLLAMA_HOST"
    API_PATH="/api/generate"
fi

# Function to query Ollama directly
query_ollama() {
    local model="$1"
    local prompt="$2"

    curl -s "$OLLAMA_HOST/api/generate" \
        -d "{\"model\":\"$model\",\"prompt\":\"$prompt\",\"stream\":false}" \
        | jq -r '.response'
}

# Function to query LiteLLM
query_litellm() {
    local model="$1"
    local prompt="$2"

    curl -s "$LITELLM_HOST/chat/completions" \
        -H "Content-Type: application/json" \
        -d "{\"model\":\"$model\",\"messages\":[{\"role\":\"user\",\"content\":\"$prompt\"}]}" \
        | jq -r '.choices[0].message.content'
}

# Choose query function
if [ -n "$LITELLM_HOST" ]; then
    query() { query_litellm "$@"; }
else
    query() { query_ollama "$@"; }
fi

echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}  CHAIN: $FAST_MODEL → $DEEP_MODEL${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo ""

# Build the fast query prompt
if [ -n "$CONTEXT" ]; then
    FAST_PROMPT="Given this context:

$CONTEXT

Task: $QUERY

Provide a brief summary (3-5 bullet points) of the key information needed to answer this task. Be concise."
else
    FAST_PROMPT="Task: $QUERY

Provide a brief summary (3-5 bullet points) of what information would be needed to complete this task. Be concise."
fi

# Step 1: Fast model
echo -e "${YELLOW}[1/2] Fast analysis with $FAST_MODEL...${NC}"
FAST_START=$(date +%s.%N)

FAST_RESPONSE=$(query "$FAST_MODEL" "$FAST_PROMPT")

FAST_END=$(date +%s.%N)
FAST_TIME=$(echo "$FAST_END - $FAST_START" | bc)

echo -e "${GREEN}Done in ${FAST_TIME}s${NC}"
echo ""
echo "$FAST_RESPONSE"
echo ""

# Step 2: Deep model with context
DEEP_PROMPT="You are a senior developer. Based on this preliminary analysis:

$FAST_RESPONSE

Now provide a detailed response to the original task:
$QUERY

$([ -n "$CONTEXT" ] && echo "Original context was provided - the analysis above summarizes the key points.")

Be thorough and specific."

echo -e "${YELLOW}[2/2] Deep analysis with $DEEP_MODEL...${NC}"
DEEP_START=$(date +%s.%N)

DEEP_RESPONSE=$(query "$DEEP_MODEL" "$DEEP_PROMPT")

DEEP_END=$(date +%s.%N)
DEEP_TIME=$(echo "$DEEP_END - $DEEP_START" | bc)

echo -e "${GREEN}Done in ${DEEP_TIME}s${NC}"
echo ""
echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}  RESULT${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo "$DEEP_RESPONSE"
echo ""
echo -e "${CYAN}───────────────────────────────────────────────────────────${NC}"
echo -e "Fast model: ${FAST_TIME}s | Deep model: ${DEEP_TIME}s | Total: $(echo "$FAST_TIME + $DEEP_TIME" | bc)s"
