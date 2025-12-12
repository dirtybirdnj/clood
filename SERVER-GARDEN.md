# The Server Garden

Three machines working together for local AI workloads.

```
┌─────────────────────────────────────────────────────────────┐
│                    THE SERVER GARDEN                         │
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐   │
│  │   ubuntu25   │    │ MacBook Air  │    │  Mac Mini    │   │
│  │   RX 590     │    │   M4 32GB    │    │   M4 (TBD)   │   │
│  │              │    │              │    │              │   │
│  │ • Image gen  │    │ • Primary    │    │ • Always-on  │   │
│  │ • Big models │    │ • Portable   │    │ • Background │   │
│  │ • Tool call  │    │ • Fast cache │    │ • Overflow   │   │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘   │
│         │                   │                   │            │
│         └───────────────────┴───────────────────┘            │
│                    192.168.4.x LAN                           │
└─────────────────────────────────────────────────────────────┘
```

---

## Hardware Specs

| Machine | IP | GPU/Chip | VRAM/Memory | CPU | Storage |
|---------|-----|----------|-------------|-----|---------|
| **ubuntu25** | 192.168.4.63 | RX 590 8GB (Vulkan) | 64GB DDR4 | i7-8086K 6c/12t | 1TB NVMe + 4TB HDD |
| **MacBook Air** | 192.168.4.47 | M4 10-core GPU | 32GB unified | M4 10-core CPU | 2TB SSD |
| **Mac Mini** | 192.168.4.41 | M4 (TBD) | TBD | M4 (TBD) | TBD |

---

## Workload Routing

| Task | Best Machine | Model | Speed |
|------|--------------|-------|-------|
| Quick coding questions | MacBook Air | qwen2.5-coder:3b | 46 tok/s |
| Complex code generation | ubuntu25 | qwen2.5-coder:7b | 32 tok/s |
| Tool calling / MCP | ubuntu25 | llama3-groq-tool-use:8b | 30 tok/s |
| Image generation | ubuntu25 or Mac | SD / Draw Things | 15-30s/img |
| Vision / image analysis | ubuntu25 | llama3.2-vision:11b | ~20 tok/s |
| Embeddings / RAG | Any | nomic-embed-text | instant |
| Large context (32K+) | MacBook Air | Any 7B | varies |
| Background/batch jobs | Mac Mini | Any | varies |

---

## Benchmark Results (December 2025)

| Machine | TinyLlama 1B | Qwen 3B | Qwen 7B | Prompt (hot) |
|---------|--------------|---------|---------|--------------|
| ubuntu25 (RX 590) | **150 tok/s** | **64 tok/s** | 32 tok/s | ~200 tok/s |
| MacBook Air (M4) | 123 tok/s | 46 tok/s | - | **760 tok/s** |
| Mac Mini (M4) | 115 tok/s | 44 tok/s | - | ~500 tok/s |

**Key insight:** RX 590 wins on eval rate (dedicated VRAM), M4 wins on prompt processing (fast cache).

---

## Network Setup

| Machine | IP | Ollama Port | Other Services |
|---------|-----|-------------|----------------|
| ubuntu25 | 192.168.4.63 | 11434 | SearXNG :8888 |
| MacBook Air | 192.168.4.47 | 11434 (local) | - |
| Mac Mini | 192.168.4.41 | 11434 (local) | - |

**SSH:** Use `ssh ubuntu25` (configured in `~/.ssh/config`)

---

## Image Generation

### ubuntu25 (RX 590)
- ComfyUI, AUTOMATIC1111, Fooocus
- ~30 seconds per image (SD 1.5)
- Best for: batch generation, SDXL

### Apple Silicon (M4 Macs)
- Draw Things (App Store, free)
- ~15 seconds per image
- Best for: quick iterations, portable use

---

## Model Distribution

### On ubuntu25 (pull here first, rsync to Macs)
```bash
ollama pull qwen2.5-coder:7b         # Complex coding
ollama pull llama3-groq-tool-use:8b  # Tool calling
ollama pull deepseek-coder:6.7b      # Alt coding
ollama pull llama3.2-vision:11b      # Vision
ollama pull nomic-embed-text         # Embeddings
```

### Sync to Macs
```bash
rsync -av --progress ubuntu25:/home/ollama-models/ ~/.ollama/models/
```

---

## Use Cases

### "I need to write code quickly"
→ MacBook Air + qwen2.5-coder:3b (46 tok/s, portable)

### "I need high-quality code generation"
→ ubuntu25 + qwen2.5-coder:7b (better output, 32 tok/s)

### "I need an agent with tools"
→ ubuntu25 + llama3-groq-tool-use:8b + Crush/MCP

### "I need to generate images"
→ MacBook Air + Draw Things (fastest)
→ ubuntu25 + ComfyUI (most flexible)

### "I need to run jobs overnight"
→ Mac Mini (always-on, quiet)

---

## Status

- [x] ubuntu25: Ollama + Vulkan configured
- [x] MacBook Air: Ollama installed, SSH to ubuntu25
- [ ] Mac Mini: Needs Ollama setup
- [ ] Multi-machine load balancing (future)
- [ ] Shared model storage (future)
