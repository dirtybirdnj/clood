# RTX 2080 Integration Planning

> *Turing tensor dreams*
> *CUDA flows where Vulkan fought*
> *Green joins the garden*

> *For Jon:*
> *Miles don't matter much*
> *When an old friend shares his card*
> *NYC to VT*

**Status:** Hypothetical - documenting options for potential RTX 2080 acquisition

---

## RTX 2080 Specifications

| Spec | Value | vs RX 590 |
|------|-------|-----------|
| Architecture | Turing (TU104) | Polaris 30 |
| VRAM | 8GB GDDR6 | 8GB GDDR5 (same capacity) |
| Memory Bandwidth | 448 GB/s | 256 GB/s (**1.75x faster**) |
| Memory Bus | 256-bit | 256-bit |
| CUDA Cores | 2944 | N/A |
| Tensor Cores | 288 | ❌ None |
| RT Cores | 46 | ❌ None |
| Base Clock | 1515 MHz | 1469 MHz |
| Boost Clock | 1710 MHz | 1545 MHz |
| TDP | 215W | 225W (slightly less) |
| PCIe | 3.0 x16 | 3.0 x16 |
| Power Connectors | 1x 6-pin + 1x 8-pin | 1x 8-pin |
| ML Backend | **CUDA (native)** | Vulkan (workaround) |

### Why RTX 2080 is Better for LLMs

1. **Native CUDA support** - Ollama/llama.cpp optimized for CUDA, Vulkan is a fallback
2. **Tensor Cores** - Hardware acceleration for matrix operations (FP16/INT8)
3. **Higher memory bandwidth** - Faster token generation, especially for larger models
4. **Better software ecosystem** - PyTorch, TensorFlow, etc. all CUDA-first

### Expected Performance Improvement

| Model | RX 590 (Vulkan) | RTX 2080 (CUDA) | Improvement |
|-------|-----------------|-----------------|-------------|
| TinyLlama 1B | 150 tok/s | ~250-300 tok/s | ~1.8x |
| Qwen 2.5 3B | 64 tok/s | ~100-120 tok/s | ~1.7x |
| Qwen 2.5 7B | 32 tok/s | ~50-60 tok/s | ~1.7x |
| Llama 3.1 8B | 30 tok/s | ~45-55 tok/s | ~1.6x |

*Estimates based on memory bandwidth ratio and CUDA optimization gains*

---

## Option 2: Dual GPU in ubuntu25 (Preferred)

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    ubuntu25 (dual GPU)                   │
│                                                          │
│  ┌─────────────────┐         ┌─────────────────┐        │
│  │    RTX 2080     │         │     RX 590      │        │
│  │   (Slot 1, x16) │         │   (Slot 2, x4)  │        │
│  │                 │         │                 │        │
│  │ • LLM inference │         │ • Display out   │        │
│  │ • CUDA primary  │         │ • Image gen?    │        │
│  │ • Ollama target │         │ • Backup/spare  │        │
│  └────────┬────────┘         └────────┬────────┘        │
│           │                           │                  │
│           └───────────┬───────────────┘                  │
│                       │                                  │
│              ┌────────▼────────┐                        │
│              │    i7-8086K     │                        │
│              │   64GB DDR4     │                        │
│              │   PCIe 3.0 x16  │                        │
│              └─────────────────┘                        │
└─────────────────────────────────────────────────────────┘
```

### PCIe Lane Considerations

The i7-8086K has **16 PCIe 3.0 lanes** from the CPU. Typical Z370/Z390 motherboard configurations:

| Slot Config | RTX 2080 | RX 590 | Notes |
|-------------|----------|--------|-------|
| x16 / x0 | x16 | ❌ disabled | Single GPU only |
| x8 / x8 | x8 | x8 | Both at half bandwidth |
| x16 / x4 (chipset) | x16 | x4 (via Z390) | **Optimal** - RTX gets full lanes |

**Recommendation:** Put RTX 2080 in primary x16 slot, RX 590 in secondary slot (runs at x4 via chipset, fine for display/light use).

### Power Requirements

| Component | Power Draw |
|-----------|------------|
| i7-8086K | 95W TDP (140W peak) |
| RTX 2080 | 215W TDP (250W peak) |
| RX 590 | 225W TDP (275W peak) |
| System (RAM, storage, fans) | ~50W |
| **Total Peak** | **~715W** |

**PSU Requirement:** 850W+ recommended (80+ Gold for efficiency)

**ubuntu25 Status:** Has 1000W PSU ✅ - No upgrade needed! Plenty of headroom for dual GPU.

### Ollama Configuration (Dual GPU)

`/etc/systemd/system/ollama.service.d/override.conf`:

```ini
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
Environment="CUDA_VISIBLE_DEVICES=0"
Environment="OLLAMA_FLASH_ATTENTION=1"
Environment="OLLAMA_KV_CACHE_TYPE=q8_0"
Environment="OLLAMA_KEEP_ALIVE=30m"
# No more Vulkan hacks needed!
```

### Migration Steps

```bash
# 1. Document current performance (baseline)
./scripts/benchmark.sh > benchmark-rx590-before.txt

# 2. Backup configs
cp /etc/systemd/system/ollama.service.d/override.conf ~/ollama-vulkan-backup.conf

# 3. Power down, install RTX 2080 in primary slot
#    Move RX 590 to secondary slot (for display)

# 4. Boot, verify both GPUs detected
lspci | grep -i vga
nvidia-smi  # Should show RTX 2080

# 5. Install NVIDIA drivers (if not present)
sudo apt install nvidia-driver-545  # or latest

# 6. Update Ollama config (remove Vulkan, add CUDA)
sudo nano /etc/systemd/system/ollama.service.d/override.conf

# 7. Restart and benchmark
sudo systemctl daemon-reload
sudo systemctl restart ollama
./scripts/benchmark.sh > benchmark-rtx2080-after.txt

# 8. Compare results
diff benchmark-rx590-before.txt benchmark-rtx2080-after.txt
```

---

## Option 3: Dedicated RTX 2080 Machine (Parts List)

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      THE SERVER GARDEN v2                        │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │   ubuntu25   │  │   rtx-node   │  │  Mac Mini    │           │
│  │   RX 590     │  │   RTX 2080   │  │   M4 16GB    │           │
│  │              │  │              │  │              │           │
│  │ • Tools/MCP  │  │ • Fast infer │  │ • Always-on  │           │
│  │ • Image gen  │  │ • Primary LLM│  │ • Overflow   │           │
│  │ • SearXNG    │  │ • CUDA speed │  │ • Background │           │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘           │
│         │                 │                 │                    │
│         └─────────────────┼─────────────────┘                    │
│                           │                                      │
│                  ┌────────▼────────┐                            │
│                  │    LiteLLM      │                            │
│                  │  Load Balancer  │                            │
│                  └─────────────────┘                            │
└─────────────────────────────────────────────────────────────────┘
```

### Hypothetical Parts List (Budget Build)

Goal: Minimum viable machine to run RTX 2080 for LLM inference

| Component | Model | Price (USD) | Notes |
|-----------|-------|-------------|-------|
| **CPU** | Intel i5-12400F | ~$120 | 6c/12t, no iGPU needed (GPU does display) |
| **Motherboard** | B660M (micro-ATX) | ~$90 | PCIe 4.0 x16, good enough |
| **RAM** | 32GB DDR4-3200 (2x16GB) | ~$65 | Minimum for model spillover |
| **Storage** | 500GB NVMe SSD | ~$40 | OS + models (can network mount more) |
| **PSU** | 650W 80+ Bronze | ~$60 | Plenty for single GPU |
| **Case** | Basic micro-ATX | ~$45 | Airflow matters, nothing fancy |
| **GPU** | RTX 2080 (gifted) | $0 | The whole point! |
| | | | |
| **Total** | | **~$420** | Without GPU |

### Alternative: Used Parts (Even Cheaper)

| Component | Model | Price (USD) | Notes |
|-----------|-------|-------------|-------|
| **CPU** | i7-4770/4790 | ~$40 | Old but 4c/8t, enough for inference |
| **Motherboard** | Used LGA 1150 | ~$50 | Check PCIe x16 slot |
| **RAM** | 32GB DDR3 (4x8GB) | ~$40 | DDR3 is cheap now |
| **Storage** | 256GB SATA SSD | ~$20 | Bare minimum |
| **PSU** | Used 600W | ~$30 | Check age/condition |
| **Case** | Used/spare | ~$20 | Anything with airflow |
| **GPU** | RTX 2080 (gifted) | $0 | |
| | | | |
| **Total** | | **~$200** | Basement budget |

### Performance Comparison

| Machine | GPU | Expected Qwen 7B | Role |
|---------|-----|------------------|------|
| ubuntu25 | RX 590 | 32 tok/s | Tools, image gen, backup |
| rtx-node | RTX 2080 | ~55 tok/s | Primary fast inference |
| Mac Mini | M4 | ~35 tok/s* | Always-on, background |

*Estimated, needs 7B model benchmark

### Network Configuration

```yaml
# LiteLLM config addition for rtx-node
model_list:
  - model_name: "fast-coder"
    litellm_params:
      model: "ollama/qwen2.5-coder:7b"
      api_base: "http://rtx-node:11434"  # New machine

  - model_name: "tool-model"
    litellm_params:
      model: "ollama/llama3-groq-tool-use:8b"
      api_base: "http://ubuntu25:11434"  # Existing
```

### Pros/Cons of Option 3

**Pros:**
- Redundancy (machine failure doesn't stop work)
- Parallel inference (multiple requests simultaneously)
- Dedicated workloads (tools vs speed)
- RX 590 continues working as-is
- Can experiment without disrupting ubuntu25

**Cons:**
- More money (~$200-420)
- More power consumption (~150-200W idle)
- More complexity (networking, LiteLLM routing)
- More physical space
- Another machine to maintain

---

## Decision Matrix

| Factor | Option 2 (Dual GPU) | Option 3 (New Machine) |
|--------|---------------------|------------------------|
| Cost | $0-130 (maybe PSU) | $200-420 |
| Complexity | Medium | High |
| Performance | 1.7x improvement | Same + parallel |
| Redundancy | Low (single machine) | High (two machines) |
| Power | +215W | +350W (whole system) |
| Space | None | Need desk/shelf |
| Maintenance | Same as now | 2x machines |

**Recommendation:** Start with **Option 2** (dual GPU in ubuntu25). If you later need parallel inference or redundancy, the RX 590 can be moved to a new budget build.

---

## References

- [RTX 2080 Specifications](https://www.nvidia.com/en-us/geforce/20-series/)
- [Ollama CUDA Guide](https://github.com/ollama/ollama/blob/main/docs/gpu.md)
- [llama.cpp CUDA Backend](https://github.com/ggerganov/llama.cpp/blob/master/docs/build.md#cuda)
- [PCPartPicker](https://pcpartpicker.com) - For current pricing
