# Model Comparison Guide

Hardware reference: **RX 590 (8GB VRAM)** / **64GB System RAM**

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

Priority order for Crush + MCP:

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

# Tool calling test (in Crush)
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

### M4 MacBook Air / M4 Mac Mini
- Metal acceleration, unified memory
- Can run larger models (14B+) efficiently
- Test and record speeds in LAST_SESSION.md
