# Ollama Performance Tuning Guide

This guide covers performance optimization for Ollama across multiple machines and provides a framework for benchmarking and aggregating results.

---

## Quick Start: Pull Models

**IMPORTANT:** Run these commands manually in your terminal. Do not run via Claude CLI.

### Tool-Capable Models (Priority)

```bash
# Best for MCP/tool calling
ollama pull llama3-groq-tool-use:8b

# Latest generation with tools
ollama pull qwen3:8b

# Fast + tools
ollama pull llama3.2:3b
```

### Coding Models

```bash
# Already installed on ubuntu25
ollama pull qwen2.5-coder:3b
ollama pull qwen2.5-coder:7b
ollama pull deepseek-coder:6.7b
ollama pull llama3.1:8b
```

### Verify Installation

```bash
ollama list
```

---

## Environment Variables Reference

### Core Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_HOST` | `127.0.0.1:11434` | Bind address. Use `0.0.0.0:11434` for network access |
| `OLLAMA_MODELS` | `~/.ollama/models` | Model storage location |
| `OLLAMA_KEEP_ALIVE` | `5m` | How long to keep model loaded after last request |
| `OLLAMA_NUM_PARALLEL` | `1-4` | Max parallel requests per model |
| `OLLAMA_MAX_LOADED_MODELS` | `1` | Max models loaded simultaneously |

### Performance Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_FLASH_ATTENTION` | `false` | Enable flash attention (faster, less VRAM) |
| `OLLAMA_KV_CACHE_TYPE` | `f16` | KV cache quantization: `f16`, `q8_0`, `q4_0` |
| `OLLAMA_NUM_GPU` | auto | Number of GPU layers to offload |
| `OLLAMA_GPU_OVERHEAD` | `0` | Reserved VRAM in bytes |

### GPU Selection

| Variable | Description |
|----------|-------------|
| `CUDA_VISIBLE_DEVICES` | NVIDIA GPU selection (e.g., `0` or `0,1`) |
| `ROCR_VISIBLE_DEVICES` | AMD ROCm GPU selection |
| `GGML_VK_VISIBLE_DEVICES` | Vulkan GPU selection (e.g., `0`) |
| `HIP_VISIBLE_DEVICES` | AMD HIP GPU selection |
| `OLLAMA_VULKAN` | `true` to force Vulkan backend |

---

## Machine-Specific Configurations

### ubuntu25 (AMD RX 590 + Vulkan)

**Hardware:** RX 590 8GB VRAM, Intel i7, 64GB RAM

**Config:** `/etc/systemd/system/ollama.service.d/override.conf`

```ini
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
Environment="OLLAMA_VULKAN=true"
Environment="HIP_VISIBLE_DEVICES="
Environment="OLLAMA_FLASH_ATTENTION=1"
Environment="OLLAMA_KV_CACHE_TYPE=q8_0"
Environment="OLLAMA_NUM_PARALLEL=1"
Environment="OLLAMA_KEEP_ALIVE=30m"
Environment="GGML_VK_VISIBLE_DEVICES=0"
```

**Critical:** `GGML_VK_VISIBLE_DEVICES=0` disables Intel iGPU, preventing slow multi-GPU splits.

**Apply changes:**
```bash
sudo systemctl daemon-reload
sudo systemctl restart ollama
```

### M4 MacBook Air / M4 Mac Mini (Apple Silicon)

**Hardware:** M4 chip with unified memory, Metal acceleration

**Config:** Ollama uses Metal by default on macOS. No special config needed.

**Optional tuning:** Create `~/.ollama/config.json`:
```json
{
  "gpu": true,
  "num_gpu": 999
}
```

Or set environment variables in `~/.zshrc`:
```bash
export OLLAMA_KEEP_ALIVE=30m
export OLLAMA_NUM_PARALLEL=2
```

### NVIDIA GPU Systems

**Config:** `/etc/systemd/system/ollama.service.d/override.conf`

```ini
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
Environment="OLLAMA_FLASH_ATTENTION=1"
Environment="OLLAMA_KV_CACHE_TYPE=q8_0"
Environment="CUDA_VISIBLE_DEVICES=0"
```

---

## Benchmarking Framework

### Single Model Benchmark

Run this on each machine to test a specific model:

```bash
MODEL="qwen2.5-coder:3b"
echo "=== Benchmarking $MODEL ==="
ollama run $MODEL "Write a Python function to calculate fibonacci numbers" --verbose 2>&1 | tail -10
```

### Full Benchmark Script

Save as `benchmark.sh` and run on each machine:

```bash
#!/bin/bash

HOSTNAME=$(hostname)
DATE=$(date +%Y-%m-%d)
OUTPUT="benchmark-${HOSTNAME}-${DATE}.txt"

echo "=== Ollama Benchmark ===" > $OUTPUT
echo "Host: $HOSTNAME" >> $OUTPUT
echo "Date: $DATE" >> $OUTPUT
echo "" >> $OUTPUT

# System info
echo "=== System Info ===" >> $OUTPUT
uname -a >> $OUTPUT
echo "" >> $OUTPUT

# GPU info
echo "=== GPU Info ===" >> $OUTPUT
if command -v nvidia-smi &> /dev/null; then
    nvidia-smi --query-gpu=name,memory.total --format=csv >> $OUTPUT
elif command -v rocm-smi &> /dev/null; then
    rocm-smi --showproductname >> $OUTPUT
elif [[ $(uname) == "Darwin" ]]; then
    system_profiler SPDisplaysDataType | grep -E "Chipset|VRAM|Metal" >> $OUTPUT
fi
echo "" >> $OUTPUT

# Models to test
MODELS=(
    "tinyllama:latest"
    "qwen2.5-coder:3b"
    "llama3.1:8b"
    "qwen2.5-coder:7b"
)

PROMPT="Write a Python function to reverse a string"

echo "=== Benchmark Results ===" >> $OUTPUT
for MODEL in "${MODELS[@]}"; do
    echo "Testing: $MODEL" >> $OUTPUT

    # Check if model exists
    if ollama list | grep -q "$MODEL"; then
        RESULT=$(ollama run "$MODEL" "$PROMPT" --verbose 2>&1 | grep "eval rate")
        echo "$MODEL: $RESULT" >> $OUTPUT
    else
        echo "$MODEL: NOT INSTALLED" >> $OUTPUT
    fi
    echo "" >> $OUTPUT
done

echo "Benchmark saved to $OUTPUT"
cat $OUTPUT
```

### Quick Benchmark Commands

Copy-paste these to test individual models:

```bash
# TinyLlama (baseline)
ollama run tinyllama "Write hello world in Python" --verbose 2>&1 | grep "eval rate"

# Qwen 3B
ollama run qwen2.5-coder:3b "Write fizzbuzz in Python" --verbose 2>&1 | grep "eval rate"

# Llama 3.1 8B
ollama run llama3.1:8b "Write a function to reverse a string" --verbose 2>&1 | grep "eval rate"

# Qwen 7B
ollama run qwen2.5-coder:7b "Write a binary search function" --verbose 2>&1 | grep "eval rate"

# Tool-use model
ollama run llama3-groq-tool-use:8b "List files in current directory" --verbose 2>&1 | grep "eval rate"
```

---

## Benchmark Results Table

Fill in after running benchmarks on each machine:

| Machine | GPU | TinyLlama | Qwen 3B | Llama 8B | Qwen 7B | Notes |
|---------|-----|-----------|---------|----------|---------|-------|
| ubuntu25 | RX 590 (Vulkan) | ~150 tok/s | 64 tok/s | 30 tok/s | 32 tok/s | `GGML_VK_VISIBLE_DEVICES=0` |
| M4 MacBook Air | M4 (Metal) | | | | | |
| M4 Mac Mini | M4 (Metal) | | | | | |

---

## Multi-Machine Architecture

### Local Network Setup

```
┌─────────────────────────────────────────────────────────────┐
│                    Home Network                              │
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   ubuntu25   │    │ MacBook Air  │    │  Mac Mini    │  │
│  │   RX 590     │    │     M4       │    │     M4       │  │
│  │  :11434      │    │   :11434     │    │   :11434     │  │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘  │
│         │                   │                   │           │
│         └───────────────────┼───────────────────┘           │
│                             │                               │
│                    ┌────────▼────────┐                      │
│                    │    SearXNG      │                      │
│                    │  ubuntu25:8888  │                      │
│                    └─────────────────┘                      │
└─────────────────────────────────────────────────────────────┘
```

### Accessing Remote Ollama

From any machine, access ubuntu25's Ollama:
```bash
curl http://192.168.4.62:11434/v1/models
```

Crush config for remote Ollama:
```json
{
  "providers": {
    "ollama-remote": {
      "name": "ubuntu25 Ollama",
      "base_url": "http://192.168.4.62:11434/v1/",
      "type": "openai",
      "api_key": "ollama"
    }
  }
}
```

### Load Balancing Strategy

| Task Type | Recommended Machine | Model |
|-----------|---------------------|-------|
| Quick queries | Local (any) | TinyLlama, Qwen 3B |
| Code generation | ubuntu25 or Mac | Qwen 7B, DeepSeek 6.7B |
| Tool calling | Local with tools | llama3-groq-tool-use:8b |
| Large context | M4 Mac (unified memory) | Qwen 14B+ |

---

## Troubleshooting

### Model Running on CPU Instead of GPU

```bash
# Check Ollama logs
journalctl -u ollama -f

# Look for "offloaded X/Y layers to GPU"
# If 0 layers offloaded, GPU not being used
```

### Multiple GPUs Causing Slowdown

On systems with iGPU + discrete GPU:
```bash
# Disable iGPU for Ollama
Environment="GGML_VK_VISIBLE_DEVICES=0"  # Use only first GPU
```

### Out of VRAM

```bash
# Use smaller quantization
ollama run model:q4_0

# Or reduce context
OLLAMA_NUM_CTX=2048 ollama run model
```

### Model Won't Load

```bash
# Check disk space
df -h ~/.ollama

# Remove and re-pull
ollama rm model_name
ollama pull model_name
```

---

## KV Cache Quantization Impact

| Setting | VRAM Usage | Quality | Use Case |
|---------|------------|---------|----------|
| `f16` (default) | 100% | Best | Quality-critical |
| `q8_0` | ~50% | Excellent | **Recommended** |
| `q4_0` | ~25% | Good | VRAM-constrained |

Set via:
```bash
Environment="OLLAMA_KV_CACHE_TYPE=q8_0"
```

---

## Context Length vs VRAM

Approximate context limits for 8GB VRAM with Q4 quantization:

| Model Size | Max Context |
|------------|-------------|
| 3B | ~32K tokens |
| 7-8B | ~16K tokens |
| 13-14B | ~4-8K tokens |
| 32B+ | ~2K tokens (CPU spillover) |

---

## Useful Commands

```bash
# Show loaded models
curl http://localhost:11434/api/ps

# Unload all models (free VRAM)
curl http://localhost:11434/api/generate -d '{"model": "", "keep_alive": 0}'

# Show model details
ollama show model_name

# Check Ollama version
ollama --version

# View logs (Linux)
journalctl -u ollama -f

# View logs (macOS)
tail -f ~/.ollama/logs/server.log
```

---

## References

- [Ollama Documentation](https://docs.ollama.com)
- [Ollama GitHub](https://github.com/ollama/ollama)
- [VRAM Calculator](https://localllm.in/blog/interactive-vram-calculator)
- [Model Library](https://ollama.com/library)
