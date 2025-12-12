# GPU Setup for Ollama

Guide for enabling GPU acceleration with AMD Radeon cards.

## Current Hardware

- **GPU:** AMD Radeon RX 590 (8GB VRAM) - gfx803 (Polaris)
- **CPU:** Intel i7-8086K (6 cores / 12 threads)
- **RAM:** 64GB
- **OS:** Ubuntu 25.04
- **Backend:** Vulkan (via Mesa RADV)

**Important:** The RX 590 uses gfx803 (Polaris) architecture. ROCm 6.x dropped support for gfx803, so **Vulkan is the recommended backend** for this GPU. Vulkan provides excellent performance without the ROCm compatibility issues.

## Check GPU Status

```bash
# See what Ollama is using
ollama ps
# If it shows "100% CPU" - GPU not working

# Check GPU devices exist
ls /dev/kfd /dev/dri/render*
# Should show: /dev/kfd, /dev/dri/renderD128, etc.
```

## Option 1: Vulkan Backend (Recommended for RX 590)

Vulkan via Mesa RADV is the **recommended backend** for Polaris GPUs (RX 570/580/590). It's simpler to set up and more reliable than ROCm for gfx803 architecture.

### Configure Ollama for Vulkan

```bash
# Create override directory
sudo mkdir -p /etc/systemd/system/ollama.service.d

# Create override file
sudo tee /etc/systemd/system/ollama.service.d/override.conf << 'EOF'
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
Environment="OLLAMA_VULKAN=true"
Environment="HIP_VISIBLE_DEVICES="
Environment="GGML_VK_VISIBLE_DEVICES=0"
Environment="OLLAMA_FLASH_ATTENTION=1"
Environment="OLLAMA_KV_CACHE_TYPE=q8_0"
EOF

# Reload and restart
sudo systemctl daemon-reload
sudo systemctl restart ollama
```

**What these do:**
- `OLLAMA_VULKAN=true` - Forces Vulkan backend
- `HIP_VISIBLE_DEVICES=` - Disables ROCm/HIP (empty string)
- `GGML_VK_VISIBLE_DEVICES=0` - Uses first Vulkan GPU (discrete, not iGPU)
- `OLLAMA_FLASH_ATTENTION=1` - Faster attention computation
- `OLLAMA_KV_CACHE_TYPE=q8_0` - Reduces VRAM usage by ~50%

### Verify Vulkan is Working

```bash
# Check GPU utilization during inference
ollama run qwen2.5-coder:7b "hello" &
watch -n1 "cat /sys/class/drm/card*/device/gpu_busy_percent 2>/dev/null"
# Should show >0% when model is running
```

---

## Option 2: ROCm (Deprecated for Polaris)

**Note:** ROCm 6.x dropped official support for gfx803 (Polaris). Use Vulkan instead.

If you still want to try ROCm (not recommended for RX 590):

```bash
# Create override file with HSA override
sudo tee /etc/systemd/system/ollama.service.d/override.conf << 'EOF'
[Service]
Environment="HSA_OVERRIDE_GFX_VERSION=8.0.3"
Environment="HIP_VISIBLE_DEVICES=0"
EOF

sudo systemctl daemon-reload
sudo systemctl restart ollama
```

This is a workaround that may or may not work depending on your system. **Vulkan is more reliable.**

## Monitoring GPU

```bash
# AMD GPU monitor (already installed)
radeontop -c

# Or install nvtop (works with AMD)
sudo apt install nvtop
nvtop

# ROCm stats
rocm-smi
watch -n1 rocm-smi

# Quick memory check
cat /sys/class/drm/card0/device/mem_info_vram_used
cat /sys/class/drm/card0/device/mem_info_vram_total
```

## Optimized Ollama Config

Once GPU is working, you can add more options to the override:

```bash
sudo tee /etc/systemd/system/ollama.service.d/override.conf << 'EOF'
[Service]
Environment="HSA_OVERRIDE_GFX_VERSION=8.0.3"
Environment="HIP_VISIBLE_DEVICES=0"
Environment="OLLAMA_HOST=0.0.0.0:11434"
Environment="OLLAMA_NUM_PARALLEL=4"
Environment="OLLAMA_FLASH_ATTENTION=1"
Environment="OLLAMA_KV_CACHE_TYPE=q8_0"
EOF

sudo systemctl daemon-reload
sudo systemctl restart ollama
```

## Expected Performance (Vulkan on RX 590)

| Model | CPU Only | With Vulkan GPU |
|-------|----------|-----------------|
| tinyllama:latest | ~30 tok/s | ~150-180 tok/s |
| qwen2.5-coder:3b | ~15 tok/s | ~64 tok/s |
| qwen2.5-coder:7b | ~8 tok/s | ~32 tok/s |
| llama3.1:8b | ~7 tok/s | ~30 tok/s |
| qwen2.5-coder:14b | ~3 tok/s | ~15 tok/s (partial GPU) |

## Troubleshooting

### "100% CPU" still showing

```bash
# Check ROCm sees GPU
rocminfo | grep -E "Name:|Marketing Name:|Device Type:"

# Check Ollama logs for GPU detection
journalctl -u ollama -f

# Verify systemd override is applied
systemctl cat ollama | grep HSA

# Test manually with override
export HSA_OVERRIDE_GFX_VERSION=8.0.3
export HIP_VISIBLE_DEVICES=0
ollama serve
```

### Permission denied on /dev/kfd

```bash
sudo usermod -aG render $USER
sudo usermod -aG video $USER
# Then logout/login or reboot
```

### Model too big for VRAM

8GB VRAM fits ~7B models fully. Larger models split between GPU and CPU:

```bash
ollama ps
# Shows "50% GPU/50% CPU" for partial offload
```
