#!/bin/bash
# Ollama Performance Tuning Script
#
# Run this on your Ollama host to optimize for code analysis

echo "=== Ollama Performance Tuning ==="
echo ""

# Check current settings
echo "Current running models:"
ollama ps
echo ""

# Recommended environment variables
cat << 'EOF'
Add these to your shell profile (~/.bashrc or ~/.zshrc):

# Enable Flash Attention (reduces VRAM, increases speed)
export OLLAMA_FLASH_ATTENTION=1

# KV cache quantization (reduces memory usage)
export OLLAMA_KV_CACHE_TYPE="q8_0"

# For servers with limited RAM, restrict parallel requests
# export OLLAMA_NUM_PARALLEL=1
# export OLLAMA_MAX_LOADED_MODELS=1

# For debugging
# export OLLAMA_DEBUG=1
EOF

echo ""
echo "=== Creating Optimized Modelfiles ==="

# Create directory for modelfiles
mkdir -p ~/.ollama/modelfiles

# Optimized qwen2.5-coder for code analysis
cat > ~/.ollama/modelfiles/qwen-coder-analysis << 'EOF'
FROM qwen2.5-coder:7b

# Context window - essential for code analysis
PARAMETER num_ctx 16384

# Allow longer responses
PARAMETER num_predict 4096

# Temperature for analysis (lower = more deterministic)
PARAMETER temperature 0.2

# Stop generating at these tokens
PARAMETER stop <|im_end|>
PARAMETER stop <|endoftext|>

SYSTEM """You are a code analysis assistant. When reviewing code:
1. Be specific - reference line numbers and function names
2. Focus on bugs, edge cases, and security issues
3. Provide actionable suggestions
4. If the code looks good, say so briefly
"""
EOF

echo "Created: ~/.ollama/modelfiles/qwen-coder-analysis"

# Optimized model for reasoning/review
cat > ~/.ollama/modelfiles/llama-analysis << 'EOF'
FROM llama3.1:8b

# Large context for full file analysis
PARAMETER num_ctx 16384

# Allow longer responses
PARAMETER num_predict 4096

# Lower temperature for analysis
PARAMETER temperature 0.3

SYSTEM """You are a senior software engineer reviewing code.
Focus on:
- Potential bugs and edge cases
- Security vulnerabilities
- Performance issues
- Code clarity and maintainability
Be specific and reference line numbers when possible.
"""
EOF

echo "Created: ~/.ollama/modelfiles/llama-analysis"

echo ""
echo "=== Creating Optimized Models ==="
echo ""
echo "To create the optimized models, run:"
echo ""
echo "  ollama create qwen-coder-analysis -f ~/.ollama/modelfiles/qwen-coder-analysis"
echo "  ollama create llama-analysis -f ~/.ollama/modelfiles/llama-analysis"
echo ""
echo "Then use them with:"
echo ""
echo "  clood analyze file.go --model qwen-coder-analysis"
echo "  clood ask 'review this code' --model llama-analysis"
echo ""
echo "=== Hardware Check ==="
echo ""
echo "GPU Memory:"
if command -v nvidia-smi &> /dev/null; then
    nvidia-smi --query-gpu=memory.used,memory.total --format=csv
else
    echo "No NVIDIA GPU found (or nvidia-smi not installed)"
fi
echo ""
echo "System RAM:"
free -h | head -2
echo ""
echo "=== Recommended Models by Hardware ==="
echo ""
echo "8GB VRAM:   qwen2.5-coder:3b, tinyllama (small context only)"
echo "16GB VRAM:  qwen2.5-coder:7b, llama3.1:8b"
echo "24GB VRAM:  qwen2.5-coder:14b, deepseek-r1:14b"
echo "32GB+ VRAM: qwen2.5-coder:32b, deepseek-r1:32b"
echo ""
