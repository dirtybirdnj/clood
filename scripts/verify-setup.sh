#!/bin/bash
#
# clood verification script
# Tests that all infrastructure components are working
#
# Usage: ./scripts/verify-setup.sh
#

# Source common configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/clood-common.sh"

PASS="${GREEN}[PASS]${NC}"
FAIL="${RED}[FAIL]${NC}"
WARN="${YELLOW}[WARN]${NC}"
INFO="${BLUE}[INFO]${NC}"

echo ""
echo "=========================================="
echo "       clood Verification Script"
echo "=========================================="
echo ""

ERRORS=0
WARNINGS=0

# ==============================================================================
# CORE SERVICES
# ==============================================================================

# 1. Check GPU Ollama service
echo -e "${INFO} Checking GPU Ollama (Tier 2)..."
if curl -s "$GPU_OLLAMA_URL/api/tags" > /dev/null 2>&1; then
    echo -e "  ${PASS} GPU Ollama responding on :$GPU_OLLAMA_PORT"
else
    # Try systemd on Linux
    if systemctl is-active --quiet ollama 2>/dev/null; then
        echo -e "  ${WARN} Ollama service running but API not responding"
        ((WARNINGS++))
    else
        echo -e "  ${FAIL} GPU Ollama not responding on :$GPU_OLLAMA_PORT"
        echo "       Fix: ollama serve (or systemctl start ollama)"
        ((ERRORS++))
    fi
fi

# 2. Check CPU Ollama (Split Brain)
echo -e "${INFO} Checking CPU Ollama (Tier 1)..."
if curl -s "$CPU_OLLAMA_URL/api/tags" > /dev/null 2>&1; then
    echo -e "  ${PASS} CPU Ollama responding on :$CPU_OLLAMA_PORT"
else
    echo -e "  ${WARN} CPU Ollama not running (optional for Split Brain)"
    echo "       Start: ./scripts/start-cpu-services.sh"
    ((WARNINGS++))
fi

# 3. Check Qdrant (Vector DB)
echo -e "${INFO} Checking Qdrant (Vector DB)..."
if curl -s "$VECTOR_DB_URL/collections" > /dev/null 2>&1; then
    echo -e "  ${PASS} Qdrant responding on :$VECTOR_DB_PORT"
else
    echo -e "  ${WARN} Qdrant not running (optional for RAG)"
    echo "       Start: ./scripts/start-cpu-services.sh"
    ((WARNINGS++))
fi

# ==============================================================================
# MODELS
# ==============================================================================

# 4. List GPU models
echo -e "${INFO} Checking GPU models..."
GPU_MODELS=$(OLLAMA_HOST="$GPU_OLLAMA_HOST" ollama list 2>/dev/null | tail -n +2 | wc -l)
if [ "$GPU_MODELS" -gt 0 ]; then
    echo -e "  ${PASS} Found $GPU_MODELS GPU model(s):"
    OLLAMA_HOST="$GPU_OLLAMA_HOST" ollama list 2>/dev/null | tail -n +2 | head -5 | while read line; do
        echo "       - $line"
    done
else
    echo -e "  ${WARN} No GPU models found"
    echo "       Fix: ollama pull llama3-groq-tool-use:8b"
    ((WARNINGS++))
fi

# 5. Check for tool-capable model
echo -e "${INFO} Checking for tool-capable model..."
if OLLAMA_HOST="$GPU_OLLAMA_HOST" ollama list 2>/dev/null | grep -qE "llama3-groq-tool-use|qwen|llama3.2"; then
    echo -e "  ${PASS} Tool-capable model available"
else
    echo -e "  ${WARN} No tool-capable model found"
    echo "       Recommended: ollama pull llama3-groq-tool-use:8b"
    ((WARNINGS++))
fi

# 6. Check CPU models (if CPU Ollama running)
if curl -s "$CPU_OLLAMA_URL/api/tags" > /dev/null 2>&1; then
    echo -e "${INFO} Checking CPU models..."
    for model in "$MODEL_ROUTER" "$MODEL_EMBED"; do
        if OLLAMA_HOST="$CPU_OLLAMA_HOST" ollama list 2>/dev/null | grep -q "$model"; then
            echo -e "  ${PASS} $model available"
        else
            echo -e "  ${WARN} $model not pulled on CPU tier"
            ((WARNINGS++))
        fi
    done
fi

# ==============================================================================
# EXTERNAL SERVICES
# ==============================================================================

# 7. Check SearXNG
echo -e "${INFO} Checking SearXNG..."
if curl -s "http://localhost:8888/search?q=test&format=json" | grep -q '"results"' 2>/dev/null; then
    echo -e "  ${PASS} SearXNG responding on :8888"
else
    echo -e "  ${WARN} SearXNG not responding"
    echo "       Fix: cd infrastructure && docker compose up -d searxng"
    ((WARNINGS++))
fi

# 8. Check LiteLLM
echo -e "${INFO} Checking LiteLLM proxy..."
if curl -s "$LITELLM_URL/health" > /dev/null 2>&1; then
    echo -e "  ${PASS} LiteLLM responding on :$LITELLM_PORT"
else
    echo -e "  ${WARN} LiteLLM not running"
    echo "       Start: ./scripts/start-litellm.sh"
    ((WARNINGS++))
fi

# ==============================================================================
# TOOLS
# ==============================================================================

# 9. Check mods CLI
echo -e "${INFO} Checking mods CLI..."
if command -v mods &> /dev/null; then
    echo -e "  ${PASS} mods CLI installed"
else
    echo -e "  ${WARN} mods CLI not found"
    echo "       Install: brew install charmbracelet/tap/mods"
    ((WARNINGS++))
fi

# 10. Check clood CLI
echo -e "${INFO} Checking clood CLI..."
if command -v clood &> /dev/null || [ -x "$HOME/Code/clood/clood-cli/clood" ]; then
    echo -e "  ${PASS} clood CLI available"
else
    echo -e "  ${WARN} clood CLI not found"
    echo "       Build: cd clood-cli && go build -o clood ./cmd/clood"
    ((WARNINGS++))
fi

# ==============================================================================
# HARDWARE
# ==============================================================================

# 11. Check GPU (platform-dependent)
echo -e "${INFO} Checking GPU acceleration..."
if [[ "$(uname)" == "Darwin" ]]; then
    # macOS - check for Metal
    GPU_INFO=$(system_profiler SPDisplaysDataType 2>/dev/null | grep "Chipset Model" | head -1 | cut -d: -f2 | xargs)
    if [ -n "$GPU_INFO" ]; then
        echo -e "  ${PASS} GPU: $GPU_INFO (Metal)"
    else
        echo -e "  ${WARN} Could not detect GPU"
        ((WARNINGS++))
    fi
else
    # Linux - check Vulkan
    if command -v vulkaninfo &> /dev/null; then
        GPU_NAME=$(vulkaninfo 2>/dev/null | grep "deviceName" | head -1 | cut -d= -f2 | xargs)
        if [ -n "$GPU_NAME" ]; then
            echo -e "  ${PASS} Vulkan GPU: $GPU_NAME"
        else
            echo -e "  ${WARN} Vulkan available but no GPU detected"
            ((WARNINGS++))
        fi
    else
        echo -e "  ${WARN} vulkaninfo not installed"
        ((WARNINGS++))
    fi
fi

# 12. Check disk space
echo -e "${INFO} Checking disk space..."
if [[ "$(uname)" == "Darwin" ]]; then
    DISK_USED=$(df -h / | tail -1 | awk '{print $5}' | tr -d '%')
else
    DISK_USED=$(df / | tail -1 | awk '{print $5}' | tr -d '%')
fi

if [ "$DISK_USED" -lt 90 ]; then
    echo -e "  ${PASS} Disk usage: ${DISK_USED}%"
else
    echo -e "  ${WARN} Disk usage: ${DISK_USED}% (low space!)"
    ((WARNINGS++))
fi

# ==============================================================================
# SUMMARY
# ==============================================================================

echo ""
echo "=========================================="
echo "              Summary"
echo "=========================================="
echo ""

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}All checks passed!${NC}"
    echo ""
    echo "Ready to use. Test with:"
    echo "  ./scripts/clood-flow.sh 'Hello, what can you do?'"
    echo ""
elif [ $ERRORS -eq 0 ]; then
    echo -e "${YELLOW}Passed with $WARNINGS warning(s)${NC}"
    echo "Core functionality works. Review warnings above."
    echo ""
else
    echo -e "${RED}$ERRORS error(s), $WARNINGS warning(s)${NC}"
    echo "Fix errors above before proceeding."
    echo ""
fi

# Quick test suggestions
echo "=========================================="
echo "           Quick Tests"
echo "=========================================="
echo ""
echo "1. Test Split Brain routing:"
echo "   ./scripts/clood-flow.sh -v 'What is 2+2?'"
echo ""
echo "2. Force GPU tier:"
echo "   ./scripts/clood-flow.sh -t gpu 'Explain recursion'"
echo ""
echo "3. Start CPU services (for full Split Brain):"
echo "   ./scripts/start-cpu-services.sh"
echo ""
