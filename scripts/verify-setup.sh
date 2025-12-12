#!/bin/bash
#
# clood verification script
# Tests that all infrastructure components are working
#
# Usage: ./scripts/verify-setup.sh
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# 1. Check Ollama service
echo -e "${INFO} Checking Ollama service..."
if systemctl is-active --quiet ollama 2>/dev/null; then
    echo -e "  ${PASS} Ollama service is running"
else
    echo -e "  ${FAIL} Ollama service is not running"
    echo "       Fix: sudo systemctl start ollama"
    ((ERRORS++))
fi

# 2. Check Ollama API
echo -e "${INFO} Checking Ollama API..."
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo -e "  ${PASS} Ollama API responding on :11434"
else
    echo -e "  ${FAIL} Ollama API not responding"
    echo "       Check: curl http://localhost:11434/api/tags"
    ((ERRORS++))
fi

# 3. List available models
echo -e "${INFO} Checking available models..."
MODELS=$(ollama list 2>/dev/null | tail -n +2 | wc -l)
if [ "$MODELS" -gt 0 ]; then
    echo -e "  ${PASS} Found $MODELS model(s):"
    ollama list 2>/dev/null | tail -n +2 | while read line; do
        echo "       - $line"
    done
else
    echo -e "  ${WARN} No models found"
    echo "       Fix: ollama pull llama3-groq-tool-use:8b"
    ((WARNINGS++))
fi

# 4. Check for tool-capable model
echo -e "${INFO} Checking for tool-capable model..."
if ollama list 2>/dev/null | grep -q "llama3-groq-tool-use\|qwen3\|llama3.2"; then
    echo -e "  ${PASS} Tool-capable model available"
else
    echo -e "  ${WARN} No tool-capable model found"
    echo "       Recommended: ollama pull llama3-groq-tool-use:8b"
    ((WARNINGS++))
fi

# 5. Check SearXNG
echo -e "${INFO} Checking SearXNG..."
if curl -s "http://localhost:8888/search?q=test&format=json" | grep -q '"results"' 2>/dev/null; then
    echo -e "  ${PASS} SearXNG responding on :8888"
else
    echo -e "  ${WARN} SearXNG not responding"
    echo "       Fix: cd infrastructure && docker compose up -d searxng"
    ((WARNINGS++))
fi

# 6. Check GPU (Vulkan)
echo -e "${INFO} Checking GPU acceleration..."
if command -v vulkaninfo &> /dev/null; then
    GPU_NAME=$(vulkaninfo 2>/dev/null | grep "deviceName" | head -1 | cut -d= -f2 | xargs)
    if [ -n "$GPU_NAME" ]; then
        echo -e "  ${PASS} Vulkan GPU: $GPU_NAME"
    else
        echo -e "  ${WARN} Vulkan available but no GPU detected"
        ((WARNINGS++))
    fi
else
    echo -e "  ${WARN} vulkaninfo not installed (apt install vulkan-tools)"
    ((WARNINGS++))
fi

# 7. Check CPU governor
echo -e "${INFO} Checking CPU governor..."
GOVERNOR=$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || echo "unknown")
if [ "$GOVERNOR" = "performance" ]; then
    echo -e "  ${PASS} CPU governor: performance"
else
    echo -e "  ${WARN} CPU governor: $GOVERNOR (not performance)"
    echo "       Fix: echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor"
    ((WARNINGS++))
fi

# 8. Check Crush CLI
echo -e "${INFO} Checking Crush CLI..."
if command -v crush &> /dev/null; then
    echo -e "  ${PASS} Crush CLI installed"
else
    echo -e "  ${WARN} Crush CLI not found"
    echo "       Install: See infrastructure/CRUSH-INSTALL.md"
    ((WARNINGS++))
fi

# 9. Check Crush config
echo -e "${INFO} Checking Crush configuration..."
if [ -f ~/.config/crush/crush.json ]; then
    echo -e "  ${PASS} Crush config exists at ~/.config/crush/crush.json"

    # Check MCP servers configured
    if grep -q "mcp_servers" ~/.config/crush/crush.json 2>/dev/null; then
        echo -e "  ${PASS} MCP servers configured in crush.json"
    else
        echo -e "  ${WARN} No MCP servers in crush.json"
        ((WARNINGS++))
    fi
else
    echo -e "  ${WARN} No Crush config found"
    echo "       Copy template: cp infrastructure/configs/crush/crush.json ~/.config/crush/"
    ((WARNINGS++))
fi

# 10. Check disk space
echo -e "${INFO} Checking disk space..."
ROOT_USED=$(df / | tail -1 | awk '{print $5}' | tr -d '%')
HOME_USED=$(df /home 2>/dev/null | tail -1 | awk '{print $5}' | tr -d '%' || echo "0")

if [ "$ROOT_USED" -lt 90 ]; then
    echo -e "  ${PASS} Root partition: ${ROOT_USED}% used"
else
    echo -e "  ${WARN} Root partition: ${ROOT_USED}% used (low space!)"
    ((WARNINGS++))
fi

if [ "$HOME_USED" != "0" ] && [ "$HOME_USED" -lt 90 ]; then
    echo -e "  ${PASS} Home partition: ${HOME_USED}% used"
elif [ "$HOME_USED" != "0" ]; then
    echo -e "  ${WARN} Home partition: ${HOME_USED}% used (low space!)"
    ((WARNINGS++))
fi

# Summary
echo ""
echo "=========================================="
echo "              Summary"
echo "=========================================="
echo ""

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}All checks passed!${NC}"
    echo ""
    echo "Ready to use. Test MCP in Crush with:"
    echo "  crush"
    echo "  > List files in /home/mgilbert/Code/clood"
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

# MCP Test Commands
echo "=========================================="
echo "         Manual MCP Tests"
echo "=========================================="
echo ""
echo "In Crush, test these commands:"
echo ""
echo "1. Filesystem MCP:"
echo "   > List files in /home/mgilbert/Code/clood"
echo ""
echo "2. SearXNG MCP:"
echo "   > Search the web for \"ollama vulkan performance\""
echo ""
echo "3. GitHub MCP:"
echo "   > Show recent commits in this repo"
echo ""
