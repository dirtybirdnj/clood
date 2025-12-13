#!/bin/bash
# ==============================================================================
# CLOOD COMMON - Shared Configuration for Split Brain Architecture
# ==============================================================================
# Source this file in other scripts: source "$(dirname "$0")/clood-common.sh"
#
# The Split Brain
# Two minds work as one
# CPU routes, GPU thinks deep
# Wisdom flows between
# ==============================================================================

# Detect script directory for relative paths
CLOOD_ROOT="${CLOOD_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
CLOOD_SCRIPTS="$CLOOD_ROOT/scripts"

# ==============================================================================
# TIER CONFIGURATION
# ==============================================================================

# Tier 1: CPU Brain (fast routing, embeddings, simple tasks)
CPU_OLLAMA_PORT="${CPU_OLLAMA_PORT:-11435}"
CPU_OLLAMA_HOST="127.0.0.1:$CPU_OLLAMA_PORT"
CPU_OLLAMA_URL="http://$CPU_OLLAMA_HOST"

# Tier 2: GPU Brain (complex reasoning, coding, analysis)
GPU_OLLAMA_PORT="${GPU_OLLAMA_PORT:-11434}"
GPU_OLLAMA_HOST="${GPU_OLLAMA_HOST:-127.0.0.1:$GPU_OLLAMA_PORT}"
GPU_OLLAMA_URL="http://$GPU_OLLAMA_HOST"

# Sidecar Services
VECTOR_DB_PORT="${VECTOR_DB_PORT:-6333}"
VECTOR_DB_URL="http://localhost:$VECTOR_DB_PORT"
LITELLM_PORT="${LITELLM_PORT:-4000}"
LITELLM_URL="http://localhost:$LITELLM_PORT"

# ==============================================================================
# MODEL CONFIGURATION
# ==============================================================================

# CPU Models (lightweight, fast)
MODEL_ROUTER="${MODEL_ROUTER:-tinyllama}"
MODEL_EMBED="${MODEL_EMBED:-nomic-embed-text}"
MODEL_SUMMARIZE="${MODEL_SUMMARIZE:-qwen2.5:0.5b}"

# GPU Models (heavyweight, smart)
MODEL_SMART="${MODEL_SMART:-llama3-groq-tool-use:8b}"
MODEL_CODER="${MODEL_CODER:-qwen2.5-coder:7b}"
MODEL_REASON="${MODEL_REASON:-deepseek-r1:8b}"

# ==============================================================================
# COLORS & OUTPUT
# ==============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ==============================================================================
# HELPER FUNCTIONS
# ==============================================================================

# Log with tier indicator
log_tier() {
    local tier="$1"
    local msg="$2"
    case "$tier" in
        cpu|1)
            echo -e "${CYAN}[CPU]${NC} $msg"
            ;;
        gpu|2)
            echo -e "${PURPLE}[GPU]${NC} $msg"
            ;;
        info)
            echo -e "${BLUE}[INFO]${NC} $msg"
            ;;
        ok|success)
            echo -e "${GREEN}[OK]${NC} $msg"
            ;;
        warn)
            echo -e "${YELLOW}[WARN]${NC} $msg"
            ;;
        error|err)
            echo -e "${RED}[ERROR]${NC} $msg" >&2
            ;;
        *)
            echo "$msg"
            ;;
    esac
}

# Check if a service is responding
check_service() {
    local url="$1"
    local timeout="${2:-5}"
    curl -s --max-time "$timeout" "$url" > /dev/null 2>&1
}

# Wait for service with timeout
wait_for_service() {
    local url="$1"
    local name="$2"
    local max_wait="${3:-30}"
    local elapsed=0

    echo -n "Waiting for $name..."
    while ! check_service "$url" 2; do
        sleep 1
        elapsed=$((elapsed + 1))
        echo -n "."
        if [ "$elapsed" -ge "$max_wait" ]; then
            echo -e " ${RED}TIMEOUT${NC}"
            return 1
        fi
    done
    echo -e " ${GREEN}OK${NC}"
    return 0
}

# Query Ollama API
ollama_query() {
    local host="$1"
    local model="$2"
    local prompt="$3"

    curl -s "http://$host/api/generate" \
        -d "{\"model\": \"$model\", \"prompt\": \"$prompt\", \"stream\": false}" \
        | jq -r '.response // .error // "No response"'
}

# Check if running on macOS or Linux (for sed compatibility)
is_macos() {
    [[ "$(uname)" == "Darwin" ]]
}

# Portable sed in-place edit
sed_inplace() {
    if is_macos; then
        sed -i '' "$@"
    else
        sed -i "$@"
    fi
}

# ==============================================================================
# ROUTING LOGIC
# ==============================================================================

# Classify query complexity (returns: simple, moderate, complex)
classify_query() {
    local query="$1"
    local word_count=$(echo "$query" | wc -w | tr -d ' ')
    local has_code_keywords=$(echo "$query" | grep -iE 'refactor|implement|debug|analyze|optimize|architecture|design' || true)

    if [ "$word_count" -lt 10 ] && [ -z "$has_code_keywords" ]; then
        echo "simple"
    elif [ "$word_count" -lt 30 ] && [ -z "$has_code_keywords" ]; then
        echo "moderate"
    else
        echo "complex"
    fi
}

# Select appropriate tier based on query
select_tier() {
    local query="$1"
    local complexity=$(classify_query "$query")

    case "$complexity" in
        simple)
            echo "cpu"
            ;;
        moderate|complex)
            echo "gpu"
            ;;
    esac
}

# ==============================================================================
# INITIALIZATION
# ==============================================================================

# Only print if sourced interactively
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "CLOOD Common Configuration"
    echo "=========================="
    echo "CPU Brain: $CPU_OLLAMA_URL"
    echo "GPU Brain: $GPU_OLLAMA_URL"
    echo "Vector DB: $VECTOR_DB_URL"
    echo ""
    echo "CPU Models: $MODEL_ROUTER, $MODEL_EMBED, $MODEL_SUMMARIZE"
    echo "GPU Models: $MODEL_SMART, $MODEL_CODER, $MODEL_REASON"
fi
