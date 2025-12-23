# Model Comparison Guide

A comprehensive guide to model selection and workload distribution across the clood server garden.

---

## The Server Garden

| Machine | GPU/Chip | VRAM/Memory | Strengths | Weaknesses |
|---------|----------|-------------|-----------|------------|
| **ubuntu25** | RX 590 8GB (Vulkan) | 64GB RAM | Dedicated VRAM, image gen, large models | Older GPU arch |
| **MacBook Air** | M4 10-core | 32GB unified | Fast prompts, portable, cool/quiet | Shared memory bandwidth |
| **Mac Mini** | M4 | TBD | Always-on server, compact | Specs TBD |

---

## Workload Routing Matrix

| Task | Best Machine | Model | Why |
|------|--------------|-------|-----|
| **Quick coding questions** | MacBook Air | qwen2.5-coder:3b | 46 tok/s, fast prompt processing |
| **Complex code generation** | ubuntu25 | qwen2.5-coder:7b | 32 tok/s, better quality |
| **Tool calling / MCP** | ubuntu25 | llama3-groq-tool-use:8b | Best tool support |
| **Image generation** | ubuntu25 | Stable Diffusion | Dedicated 8GB VRAM |
| **Vision / image analysis** | ubuntu25 | llama3.2-vision:11b | Needs VRAM headroom |
| **Embeddings / RAG** | Any | nomic-embed-text | Tiny, runs anywhere |
| **Large context (32K+)** | MacBook Air | Any 7B | 32GB unified handles it |
| **Background/batch jobs** | Mac Mini | Any | Always-on, won't interrupt work |

---

## Benchmark Results (December 2025)

| Machine | TinyLlama 1B | Qwen 3B | Qwen 7B | Llama 8B |
|---------|--------------|---------|---------|----------|
| ubuntu25 (RX 590) | **150 tok/s** | **64 tok/s** | 32 tok/s | 30 tok/s |
| MacBook Air (M4 32GB) | 123 tok/s | 46 tok/s | - | - |
| Mac Mini (M4 16GB) | 115 tok/s | 44 tok/s | - | - |

**Key insight:** RX 590 with dedicated VRAM beats Apple Silicon on eval rate, but M4 has faster prompt processing (760 tok/s vs ~200 tok/s for cached prompts).

---

## Currently Installed Models

| Model | Params | Size | VRAM | Tool Support | Speed | Status |
|-------|--------|------|------|--------------|-------|--------|
| `tinyllama:latest` | 1.1B | 637MB | ~1GB | ❌ no | ~150 tok/s | ✅ installed |
| `qwen2.5-coder:3b` | 3B | 1.9GB | ~2.5GB | ⚠️ partial | ~64 tok/s | ✅ installed |
| `deepseek-coder:6.7b` | 6.7B | 3.8GB | ~5GB | ⚠️ partial | ~35 tok/s | ✅ installed |
| `qwen2.5-coder:7b` | 7B | 4.7GB | ~6GB | ⚠️ partial | ~32 tok/s | ✅ installed |
| `llama3.1:8b` | 8B | 4.9GB | ~6.5GB | ✅ yes | ~30 tok/s | ✅ installed |
| `nomic-embed-text:latest` | - | 274MB | ~0.5GB | N/A | N/A | ✅ installed (embeddings) |

---

## Models by Category

### Comfortable Fit (< 5GB VRAM) - Fast

| Model | Params | Size | VRAM | Tool Support | Speed | Install |
|-------|--------|------|------|--------------|-------|---------|
| `tinyllama:latest` | 1.1B | 637MB | ~1GB | ❌ | ~150 tok/s | ✅ installed |
| `phi3:mini` | 3.8B | 2.2GB | ~3GB | ❌ | ~55 tok/s | `ollama pull phi3:mini` |
| `qwen2.5-coder:3b` | 3B | 1.9GB | ~2.5GB | ⚠️ partial | ~64 tok/s | ✅ installed |
| `llama3.2:3b` | 3B | 2.0GB | ~2.5GB | ✅ **yes** | ~60 tok/s | `ollama pull llama3.2:3b` |
| `qwen3:4b` | 4B | 2.6GB | ~3.5GB | ✅ **yes** | ~50 tok/s | `ollama pull qwen3:4b` |
| `gemma2:2b` | 2B | 1.6GB | ~2GB | ❌ | ~70 tok/s | `ollama pull gemma2:2b` |
| `codegemma:2b` | 2B | 1.6GB | ~2GB | ❌ | ~70 tok/s | `ollama pull codegemma:2b` |
| `starcoder2:3b` | 3B | 1.7GB | ~2.5GB | ❌ | ~60 tok/s | `ollama pull starcoder2:3b` |

### Good Fit (5-7GB VRAM) - Balanced

| Model | Params | Size | VRAM | Tool Support | Speed | Install |
|-------|--------|------|------|--------------|-------|---------|
| `deepseek-coder:6.7b` | 6.7B | 3.8GB | ~5GB | ⚠️ partial | ~35 tok/s | ✅ installed |
| `qwen2.5-coder:7b` | 7B | 4.7GB | ~6GB | ⚠️ partial | ~32 tok/s | ✅ installed |
| `llama3.1:8b` | 8B | 4.9GB | ~6.5GB | ✅ **yes** | ~30 tok/s | ✅ installed |
| `qwen3:8b` | 8B | 4.9GB | ~6.5GB | ✅ **yes** | ~30 tok/s | `ollama pull qwen3:8b` |
| `llama3-groq-tool-use:8b` | 8B | 4.7GB | ~6GB | ✅ **best** | ~30 tok/s | `ollama pull llama3-groq-tool-use:8b` |
| `codellama:7b` | 7B | 3.8GB | ~5GB | ❌ | ~35 tok/s | `ollama pull codellama:7b` |
| `mistral:7b` | 7B | 4.1GB | ~5.5GB | ✅ yes | ~32 tok/s | `ollama pull mistral:7b` |
| `gemma2:9b` | 9B | 5.4GB | ~7GB | ❌ | ~28 tok/s | `ollama pull gemma2:9b` |
| `codegemma:7b` | 7B | 5.0GB | ~6.5GB | ❌ | ~30 tok/s | `ollama pull codegemma:7b` |
| `deepseek-coder-v2:16b` | 16B | 8.9GB | ~7GB* | ⚠️ partial | ~25 tok/s | `ollama pull deepseek-coder-v2:16b` |

*MoE architecture - uses less VRAM than parameter count suggests

### Tight Fit (7-8GB VRAM) - May Spill to RAM

| Model | Params | Size | VRAM | Tool Support | Speed | Install |
|-------|--------|------|------|--------------|-------|---------|
| `mistral-nemo:12b` | 12B | 7.1GB | ~8GB | ✅ yes | ~22 tok/s | `ollama pull mistral-nemo` |
| `phi3:medium` | 14B | 7.9GB | ~9GB | ❌ | ~18 tok/s | `ollama pull phi3:medium` |
| `wizardcoder:13b` | 13B | 7.3GB | ~8.5GB | ❌ | ~20 tok/s | `ollama pull wizardcoder:13b` |
| `codellama:13b` | 13B | 7.4GB | ~8.5GB | ❌ | ~18 tok/s | `ollama pull codellama:13b` |

### Too Large (Will Spill to RAM - Slow)

| Model | Params | Size | VRAM | Tool Support | Speed | Install |
|-------|--------|------|------|--------------|-------|---------|
| `qwen2.5-coder:14b` | 14B | 9.0GB | ~10GB | ⚠️ partial | ~15 tok/s* | `ollama pull qwen2.5-coder:14b` |
| `qwen3:14b` | 14B | 9.0GB | ~10GB | ✅ yes | ~15 tok/s* | `ollama pull qwen3:14b` |
| `llama3.1:70b` | 70B | 40GB | ~45GB | ✅ yes | ~3 tok/s* | `ollama pull llama3.1:70b` |
| `deepseek-coder:33b` | 33B | 19GB | ~22GB | ⚠️ partial | ~5 tok/s* | `ollama pull deepseek-coder:33b` |
| `qwen2.5-coder:32b` | 32B | 18GB | ~20GB | ⚠️ partial | ~5 tok/s* | `ollama pull qwen2.5-coder:32b` |
| `codellama:34b` | 34B | 19GB | ~22GB | ❌ | ~5 tok/s* | `ollama pull codellama:34b` |

*Will use system RAM, significantly slower

---

## Tool Support Legend

| Symbol | Meaning |
|--------|---------|
| ✅ **yes** | Officially supported by Ollama for tool calling |
| ✅ **best** | Specifically trained for tool/function calling |
| ⚠️ partial | Has tool template but inconsistent results |
| ❌ no | No tool calling support |

---

## Recommended Models for Tool Calling / MCP

Priority order for clood + MCP:

1. **`llama3-groq-tool-use:8b`** - Purpose-built for tool calling
2. **`qwen3:8b`** - Latest generation, good tool support
3. **`llama3.1:8b`** - Officially supported, already installed
4. **`llama3.2:3b`** - Fast, smaller, has tool support
5. **`mistral:7b`** - Good balance

```bash
# Pull recommended tool-calling models
ollama pull llama3-groq-tool-use:8b
ollama pull qwen3:8b
ollama pull llama3.2:3b
```

---

## Recommended Models for Coding (No Tools)

If tool calling isn't needed:

1. **`qwen2.5-coder:7b`** - Best code quality at this size
2. **`deepseek-coder:6.7b`** - Strong coding focus
3. **`codellama:7b`** - Meta's code-focused model
4. **`starcoder2:3b`** - Fast, code-specific

---

## Quantization Reference

Same model, different quantizations:

| Quantization | Size Multiplier | Quality | Use Case |
|--------------|-----------------|---------|----------|
| Q4_K_M | 0.5x | Good | **Default, recommended** |
| Q5_K_M | 0.6x | Better | Slightly better quality |
| Q6_K | 0.7x | Great | Quality-focused |
| Q8_0 | 1.0x | Excellent | Best quality if fits |
| FP16 | 2.0x | Original | Only if VRAM allows |

**Rule of thumb:** Larger model + more quantization often beats smaller model + less quantization.

Example: `qwen3:14b-q4` may outperform `qwen3:8b-q8` if both fit in VRAM.

---

## Benchmarking Commands

Test any model:

```bash
# Quick speed test
ollama run MODEL_NAME "Write fizzbuzz in Python" --verbose 2>&1 | grep "eval rate"

# Tool calling test (in clood)
# Ask: "List the files in the current directory"
# Watch if it calls MCP or hallucinates JSON
```

---

## Context Length vs VRAM

More context = more VRAM. For 8GB GPU with Q4 models:

| Model Size | Max Context (approx) |
|------------|---------------------|
| 3B | ~32K tokens |
| 7-8B | ~16K tokens |
| 13-14B | ~4-8K tokens |

Use `OLLAMA_KV_CACHE_TYPE=q8_0` to extend context by ~50%.

---

## Machine-Specific Notes

### ubuntu25 (RX 590 8GB + Vulkan)
- Best: 7-8B models at Q4
- `GGML_VK_VISIBLE_DEVICES=0` required (disable Intel iGPU)
- Sweet spot: `qwen2.5-coder:3b` (64 tok/s) or `llama3.1:8b` (30 tok/s)
- **Image gen capable:** Can run Stable Diffusion, ComfyUI, AUTOMATIC1111

### M4 Mac Mini (16GB unified)
- Metal acceleration, 10-core GPU
- **Benchmarked (2025-12-12):** TinyLlama ~115 tok/s, Qwen 3B ~44 tok/s
- Slower eval rate than RX 590 but faster prompt processing
- Can run 14B+ models in unified memory without spillover
- Sweet spot: `qwen2.5-coder:3b` (44 tok/s) for quick tasks
- Good for: always-on server, background jobs

### M4 MacBook Air (32GB unified)
- **Benchmarked (2025-12-12):** TinyLlama ~123 tok/s, Qwen 3B ~46 tok/s
- Slightly faster than Mac Mini (more unified memory bandwidth)
- Best prompt processing: 760 tok/s (hot cache)
- Can handle 14B+ models comfortably with 32GB
- Good for: primary coding workstation, portable development
- Image gen: Use Draw Things or DiffusionBee (native macOS apps)

---

## Image Generation Options

### ubuntu25 (Recommended for quality/speed)

| Tool | VRAM | Speed | Notes |
|------|------|-------|-------|
| **ComfyUI** | 4-6GB | ~30s/img | Node-based, most flexible |
| **AUTOMATIC1111** | 4-6GB | ~30s/img | Feature-rich WebUI |
| **Fooocus** | 4GB | ~45s/img | Simplest, Midjourney-like |

```bash
# Docker install for AUTOMATIC1111
docker run -d --name sd-webui \
  -p 7860:7860 \
  -v ~/sd-models:/models \
  --device /dev/dri \
  ghcr.io/automattic1111/stable-diffusion-webui
```

### Apple Silicon (MacBook Air / Mac Mini)

| Tool | Memory | Speed | Notes |
|------|--------|-------|-------|
| **Draw Things** | 4-8GB | ~15s/img | Native macOS, very fast on M4 |
| **DiffusionBee** | 4-8GB | ~20s/img | Simple UI, good for beginners |
| **MLX Stable Diffusion** | 4-8GB | ~10s/img | Apple's optimized library |

**Note:** M4 Macs are actually faster for image gen than RX 590 due to Metal optimization!

---

## Recommended Model Priority

### For Coding Agents (Claude comparison)

```bash
# On ubuntu25 - pull these in order:
ollama pull qwen2.5-coder:7b         # Best coding quality
ollama pull llama3-groq-tool-use:8b  # Tool calling champion
ollama pull deepseek-coder:6.7b      # Alternative coding focus
ollama pull codellama:13b            # If you need bigger
```

### For Tool Calling / MCP

```bash
ollama pull llama3-groq-tool-use:8b  # #1 choice
ollama pull qwen3:8b                 # Good alternative
ollama pull llama3.2:3b              # Fast + tools
```

### For Vision

```bash
ollama pull llama3.2-vision:11b      # Can see and describe images
ollama pull llava:13b                # Alternative vision model
```

### For Embeddings/RAG

```bash
ollama pull nomic-embed-text         # Only 274MB, essential
```

---

## Coding Agent Comparison Setup

To compare local models vs Claude:

| Agent Tool | Works With | Best For |
|------------|------------|----------|
| **Aider** | Ollama, OpenAI, Claude | Git-aware coding, file editing |
| **clood + MCP** | Ollama | Tool calling, web search |
| **Continue.dev** | Ollama, any API | IDE integration (VSCode/JetBrains) |
| **Open Interpreter** | Ollama, OpenAI | Code execution, system control |

```bash
# Aider setup (recommended for Claude comparison)
pip install aider-chat
cd ~/Code/your-project
aider --model ollama/qwen2.5-coder:7b
```

Compare same tasks between:
1. `aider --model ollama/qwen2.5-coder:7b` (local)
2. `aider --model claude-3-5-sonnet` (Claude)
3. `claude` (Claude Code CLI)
