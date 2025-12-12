# Last Session - 2025-12-12

## What We Did

### 1. Fixed Ollama GPU Acceleration (RX 590)
- **Problem**: Ollama was running on CPU only despite ROCm being installed
- **Root cause**: ROCm 6.x dropped gfx803 (Polaris/RX 590) support entirely - no kernels exist
- **Solution**: Switched to Vulkan backend which has broader GPU support via Mesa RADV
- **Result**: GPU now utilized, 46% VRAM usage, ~70 tok/s on Qwen 7B (was ~10 tok/s on CPU)

### 2. Abandoned Open WebUI
- Tool calling is broken, documentation is garbage, version changes break everything
- Decided to focus on **Crush** (charmbracelet) as the local Claude Code alternative

### 3. Configured Crush with MCP Servers
Set up three MCP (Model Context Protocol) servers for tool capabilities:

| Server | Package | Purpose |
|--------|---------|---------|
| filesystem | @modelcontextprotocol/server-filesystem | Read/write ~/Code |
| searxng | @kevinwatt/mcp-server-searxng | Web search via local SearXNG |
| github | any-cli-mcp-server | Full gh CLI access |

### 4. Installed Node.js
- Required for MCP servers (they run via npx)
- `sudo apt install nodejs npm`

### 5. Updated Documentation
- crush.md now has complete MCP configuration guide
- infrastructure/configs/crush/crush.json updated with MCP config

## Final Ollama Config (Tuned)

`/etc/systemd/system/ollama.service.d/override.conf`:
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

**Critical finding:** `GGML_VK_VISIBLE_DEVICES=0` is essential - without it, Ollama splits models across RX 590 AND Intel iGPU, causing 7x slowdown on smaller models.

## Benchmarked Performance (RX 590 + Vulkan)

| Model | Eval Speed | Prompt Speed | Notes |
|-------|------------|--------------|-------|
| TinyLlama | ~150 tok/s | ~200 tok/s | Quick queries |
| Qwen 3B | **64 tok/s** | 222 tok/s | Best balance |
| Qwen 7B | 32 tok/s | 105 tok/s | Complex tasks |
| DeepSeek 6.7B | ~30 tok/s | ~100 tok/s | Coding focus |

**Recommendation:** Use Qwen 3B for most Crush tasks - it's 2x faster than 7B with good quality.

## Crush Config Location

- Global: `~/.config/crush/crush.json`
- Project: `.crush.json` in project root

Current config includes:
- Local Ollama provider with 7 models
- 3 MCP servers (filesystem, searxng, github)

## To Test on Laptop/Mac Mini

1. Install Crush: `brew install charmbracelet/tap/crush` (macOS) or from GitHub releases
2. Install Node.js + npm
3. Install and auth gh CLI: `gh auth login`
4. Copy crush.json from repo, adjust paths:
   - Change `/home/mgilbert/Code` to local path
   - Change SearXNG URL to `http://192.168.4.62:8888` (workstation IP)
   - Change Ollama base_url to `http://192.168.4.62:11434/v1/`

## Features Ready to Test

1. **Read code**: "List files in the clood project"
2. **Web search**: "Search for Python asyncio best practices"
3. **GitHub**: "Show my repos" / "Create a PR for these changes"
4. **Full workflow**: Search → Read code → Make changes → Create PR

## Terminal Improvement (Next)

Installing Kitty terminal for Mac-like copy/paste:
```bash
sudo apt install kitty
```

Config for `~/.config/kitty/kitty.conf`:
```
map ctrl+c copy_or_interrupt
map ctrl+v paste_from_clipboard
```

## Files Changed This Session

- `crush.md` - Added MCP server documentation
- `infrastructure/configs/crush/crush.json` - Added MCP configuration
- `/etc/systemd/system/ollama.service.d/override.conf` - Vulkan backend
- `~/.config/crush/crush.json` - Live config with MCP servers

## Benchmarking Commands

Run these on each machine to compare performance:

```bash
# Quick benchmark - run each model and note the eval rate
ollama run tinyllama "Write hello world in Python" --verbose 2>&1 | grep "eval rate"
ollama run qwen2.5-coder:3b "Write fizzbuzz in Python" --verbose 2>&1 | grep "eval rate"
ollama run qwen2.5-coder:7b "Write a function to reverse a string" --verbose 2>&1 | grep "eval rate"
```

**Record results here:**

| Machine | TinyLlama | Qwen 3B | Qwen 7B |
|---------|-----------|---------|---------|
| ubuntu25 (RX 590) | ~150 tok/s | 64 tok/s | 32 tok/s |
| M4 MacBook Air | | | |
| M4 Mac Mini | | | |

## Vi Cheat Sheet

For editing config files:

| Step | Keys | Action |
|------|------|--------|
| 1 | `gg` | Go to top |
| 2 | `dG` | Delete all content |
| 3 | `i` | Enter insert mode |
| 4 | Paste | |
| 5 | `Esc` | Exit insert mode |
| 6 | `:wq` Enter | Save and quit |

Bail out without saving: `Esc` then `:q!` Enter

## Key Discoveries

1. **ROCm 6.x has no gfx803 support** - HSA_OVERRIDE_GFX_VERSION trick doesn't work when kernels don't exist
2. **Vulkan works great** for older AMD GPUs via Mesa RADV driver
3. **Crush supports MCP** - can extend local models with tools just like Claude Code
4. **SearXNG is ideal** for AI search - self-hosted means traffic looks human (single residential IP)
5. **Disable Intel iGPU** - `GGML_VK_VISIBLE_DEVICES=0` prevents slow multi-GPU splits
6. **Qwen 3B is the sweet spot** - 64 tok/s vs 32 tok/s for 7B, good quality for most tasks
