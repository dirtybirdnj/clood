#!/bin/bash
#
# detect-hardware.sh - Analyze system and recommend clood configuration
#
# Usage: ./scripts/detect-hardware.sh
#        ./scripts/detect-hardware.sh --json    # Machine-readable output
#

set -e

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

JSON_MODE=false
[ "$1" = "--json" ] && JSON_MODE=true

# ============================================
# Detection Functions
# ============================================

detect_cpu() {
    CPU_MODEL=$(lscpu 2>/dev/null | grep "Model name" | cut -d: -f2 | xargs || echo "Unknown")
    CPU_CORES=$(nproc 2>/dev/null || echo "1")
    CPU_ARCH=$(uname -m)

    # Check for AVX support (important for llama.cpp)
    AVX=$(grep -o 'avx[^ ]*' /proc/cpuinfo 2>/dev/null | sort -u | tr '\n' ' ' || echo "none")

    # Check for Apple Silicon
    if [ "$CPU_ARCH" = "arm64" ] && [ "$(uname -s)" = "Darwin" ]; then
        CPU_TYPE="apple-silicon"
    elif echo "$CPU_MODEL" | grep -qi "intel"; then
        CPU_TYPE="intel"
    elif echo "$CPU_MODEL" | grep -qi "amd"; then
        CPU_TYPE="amd"
    else
        CPU_TYPE="unknown"
    fi
}

detect_ram() {
    if [ "$(uname -s)" = "Darwin" ]; then
        RAM_BYTES=$(sysctl -n hw.memsize 2>/dev/null || echo "0")
        RAM_GB=$((RAM_BYTES / 1024 / 1024 / 1024))
    else
        RAM_GB=$(free -g 2>/dev/null | awk '/Mem:/ {print $2}' || echo "0")
    fi

    # Available RAM (not total)
    if [ "$(uname -s)" = "Darwin" ]; then
        RAM_AVAIL_GB=$RAM_GB  # macOS unified memory
    else
        RAM_AVAIL_GB=$(free -g 2>/dev/null | awk '/Mem:/ {print $7}' || echo "0")
    fi
}

detect_gpu() {
    GPU_TYPE="cpu-only"
    GPU_NAME="None"
    GPU_VRAM_GB="0"
    GPU_BACKEND="cpu"

    if [ "$(uname -s)" = "Darwin" ]; then
        # macOS - check for Apple Silicon GPU
        if system_profiler SPDisplaysDataType 2>/dev/null | grep -qi "apple"; then
            GPU_TYPE="apple-silicon"
            GPU_NAME=$(system_profiler SPDisplaysDataType 2>/dev/null | grep "Chipset Model" | cut -d: -f2 | xargs || echo "Apple GPU")
            GPU_VRAM_GB="unified"
            GPU_BACKEND="metal"
        fi
    else
        # Linux - check for discrete GPU
        if command -v nvidia-smi &>/dev/null; then
            GPU_TYPE="nvidia"
            GPU_NAME=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1 || echo "Unknown NVIDIA")
            GPU_VRAM_MB=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits 2>/dev/null | head -1 || echo "0")
            GPU_VRAM_GB=$((GPU_VRAM_MB / 1024))
            GPU_BACKEND="cuda"
        elif lspci 2>/dev/null | grep -qi "vga.*amd\|3d.*amd\|display.*amd"; then
            GPU_TYPE="amd"
            GPU_NAME=$(lspci 2>/dev/null | grep -i "vga\|3d\|display" | grep -i amd | head -1 | sed 's/.*: //' || echo "Unknown AMD")

            # Try to get VRAM from sysfs
            if [ -f /sys/class/drm/card0/device/mem_info_vram_total ]; then
                GPU_VRAM_BYTES=$(cat /sys/class/drm/card0/device/mem_info_vram_total 2>/dev/null || echo "0")
                GPU_VRAM_GB=$((GPU_VRAM_BYTES / 1024 / 1024 / 1024))
            elif [ -f /sys/class/drm/card1/device/mem_info_vram_total ]; then
                GPU_VRAM_BYTES=$(cat /sys/class/drm/card1/device/mem_info_vram_total 2>/dev/null || echo "0")
                GPU_VRAM_GB=$((GPU_VRAM_BYTES / 1024 / 1024 / 1024))
            else
                # Fallback: estimate from GPU name
                if echo "$GPU_NAME" | grep -qi "590\|580\|570\|Ellesmere"; then
                    GPU_VRAM_GB=8
                elif echo "$GPU_NAME" | grep -qi "6[0-9]00\|RDNA"; then
                    GPU_VRAM_GB=8  # varies by model
                elif echo "$GPU_NAME" | grep -qi "7[0-9]00"; then
                    GPU_VRAM_GB=12  # varies
                fi
            fi

            # Determine backend based on GPU generation
            if echo "$GPU_NAME" | grep -qi "polaris\|rx 5[78]0\|rx 590\|gfx803"; then
                GPU_BACKEND="vulkan"  # ROCm dropped Polaris
            elif echo "$GPU_NAME" | grep -qi "vega\|rx [67][0-9]00\|rdna"; then
                GPU_BACKEND="rocm"
            else
                GPU_BACKEND="vulkan"  # Safe default
            fi
        fi
    fi
}

detect_os() {
    OS_NAME=$(uname -s)
    if [ "$OS_NAME" = "Darwin" ]; then
        OS_PRETTY="macOS $(sw_vers -productVersion 2>/dev/null || echo "")"
        OS_TYPE="macos"
    elif [ -f /etc/os-release ]; then
        OS_PRETTY=$(grep PRETTY_NAME /etc/os-release | cut -d'"' -f2)
        OS_TYPE="linux"
    else
        OS_PRETTY="Unknown"
        OS_TYPE="unknown"
    fi

    # Check for GUI
    if [ "$OS_TYPE" = "linux" ]; then
        if systemctl is-active --quiet gdm3 2>/dev/null || \
           systemctl is-active --quiet lightdm 2>/dev/null || \
           systemctl is-active --quiet sddm 2>/dev/null; then
            HAS_GUI="yes"
        else
            HAS_GUI="no"
        fi
    else
        HAS_GUI="yes"  # macOS always has GUI
    fi
}

# ============================================
# Recommendation Logic
# ============================================

recommend_models() {
    EFFECTIVE_VRAM=$GPU_VRAM_GB

    # For Apple Silicon, use total RAM as "VRAM" (unified memory)
    if [ "$GPU_TYPE" = "apple-silicon" ]; then
        EFFECTIVE_VRAM=$RAM_GB
    fi

    # Model tier based on effective VRAM
    if [ "$EFFECTIVE_VRAM" -ge 24 ]; then
        MODEL_TIER="heavy"
        RECOMMENDED_MODELS="deepseek-coder:33b qwen2.5-coder:32b llama3.1:70b"
        MAX_MODEL_SIZE="70B"
    elif [ "$EFFECTIVE_VRAM" -ge 16 ]; then
        MODEL_TIER="large"
        RECOMMENDED_MODELS="qwen2.5-coder:14b codellama:13b deepseek-coder:6.7b"
        MAX_MODEL_SIZE="14B"
    elif [ "$EFFECTIVE_VRAM" -ge 8 ]; then
        MODEL_TIER="medium"
        RECOMMENDED_MODELS="deepseek-coder:6.7b mistral:7b llama3-groq-tool-use:8b llama3.2:3b"
        MAX_MODEL_SIZE="8B"
    elif [ "$EFFECTIVE_VRAM" -ge 4 ]; then
        MODEL_TIER="light"
        RECOMMENDED_MODELS="llama3.2:3b phi-3:mini tinyllama"
        MAX_MODEL_SIZE="3B"
    else
        MODEL_TIER="minimal"
        RECOMMENDED_MODELS="tinyllama phi-3:mini"
        MAX_MODEL_SIZE="1B"
    fi
}

recommend_config() {
    CONFIG_HINTS=""

    case $GPU_BACKEND in
        cuda)
            CONFIG_HINTS="NVIDIA GPU detected. Use default Ollama (CUDA auto-enabled)."
            ;;
        rocm)
            CONFIG_HINTS="AMD GPU with ROCm support. Set HSA_OVERRIDE_GFX_VERSION if needed."
            ;;
        vulkan)
            CONFIG_HINTS="AMD Polaris GPU. Set OLLAMA_VULKAN=true and GGML_VK_VISIBLE_DEVICES=0"
            ;;
        metal)
            CONFIG_HINTS="Apple Silicon. Use default Ollama (Metal auto-enabled)."
            ;;
        cpu)
            CONFIG_HINTS="No GPU acceleration. Set OLLAMA_NUM_THREAD=$CPU_CORES"
            ;;
    esac

    # GUI warning
    if [ "$HAS_GUI" = "yes" ] && [ "$OS_TYPE" = "linux" ]; then
        CONFIG_HINTS="$CONFIG_HINTS Consider disabling GUI for better performance (saves ~2-3GB RAM)."
    fi
}

# ============================================
# Run Detection
# ============================================

detect_cpu
detect_ram
detect_gpu
detect_os
recommend_models
recommend_config

# ============================================
# Output
# ============================================

if [ "$JSON_MODE" = true ]; then
    cat << EOF
{
  "cpu": {
    "model": "$CPU_MODEL",
    "cores": $CPU_CORES,
    "type": "$CPU_TYPE",
    "arch": "$CPU_ARCH",
    "avx": "$AVX"
  },
  "ram": {
    "total_gb": $RAM_GB,
    "available_gb": $RAM_AVAIL_GB
  },
  "gpu": {
    "type": "$GPU_TYPE",
    "name": "$GPU_NAME",
    "vram_gb": "$GPU_VRAM_GB",
    "backend": "$GPU_BACKEND"
  },
  "os": {
    "name": "$OS_PRETTY",
    "type": "$OS_TYPE",
    "has_gui": "$HAS_GUI"
  },
  "recommendations": {
    "model_tier": "$MODEL_TIER",
    "max_model_size": "$MAX_MODEL_SIZE",
    "models": "$RECOMMENDED_MODELS",
    "config_hints": "$CONFIG_HINTS"
  }
}
EOF
else
    echo ""
    echo -e "${BOLD}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}║           clood Hardware Detection Report                    ║${NC}"
    echo -e "${BOLD}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    echo -e "${CYAN}┌─ CPU ─────────────────────────────────────────────────────────┐${NC}"
    echo -e "│ Model:  $CPU_MODEL"
    echo -e "│ Cores:  $CPU_CORES"
    echo -e "│ Arch:   $CPU_ARCH"
    echo -e "│ AVX:    $AVX"
    echo -e "${CYAN}└───────────────────────────────────────────────────────────────┘${NC}"
    echo ""

    echo -e "${CYAN}┌─ Memory ──────────────────────────────────────────────────────┐${NC}"
    echo -e "│ Total:     ${RAM_GB}GB"
    echo -e "│ Available: ${RAM_AVAIL_GB}GB"
    echo -e "${CYAN}└───────────────────────────────────────────────────────────────┘${NC}"
    echo ""

    echo -e "${CYAN}┌─ GPU ────────────────────────────────────────────────────────┐${NC}"
    echo -e "│ Type:    $GPU_TYPE"
    echo -e "│ Name:    $GPU_NAME"
    echo -e "│ VRAM:    ${GPU_VRAM_GB}GB"
    echo -e "│ Backend: $GPU_BACKEND"
    echo -e "${CYAN}└───────────────────────────────────────────────────────────────┘${NC}"
    echo ""

    echo -e "${CYAN}┌─ OS ─────────────────────────────────────────────────────────┐${NC}"
    echo -e "│ OS:      $OS_PRETTY"
    echo -e "│ Has GUI: $HAS_GUI"
    echo -e "${CYAN}└───────────────────────────────────────────────────────────────┘${NC}"
    echo ""

    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                    RECOMMENDATIONS                           ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${YELLOW}Model Tier:${NC} $MODEL_TIER (max ~${MAX_MODEL_SIZE} parameters)"
    echo ""
    echo -e "${YELLOW}Recommended Models:${NC}"
    for model in $RECOMMENDED_MODELS; do
        echo "  • $model"
    done
    echo ""
    echo -e "${YELLOW}Configuration:${NC}"
    echo "  $CONFIG_HINTS"
    echo ""

    # Quick start commands
    echo -e "${CYAN}┌─ Quick Start ────────────────────────────────────────────────┐${NC}"
    echo "│"
    echo "│ # Install recommended models:"
    for model in $RECOMMENDED_MODELS; do
        echo "│ ollama pull $model"
    done
    echo "│"
    if [ "$GPU_BACKEND" = "vulkan" ]; then
        echo "│ # GPU config (add to ollama service):"
        echo "│ Environment=\"OLLAMA_VULKAN=true\""
        echo "│ Environment=\"GGML_VK_VISIBLE_DEVICES=0\""
    fi
    echo "│"
    echo -e "${CYAN}└───────────────────────────────────────────────────────────────┘${NC}"
fi
