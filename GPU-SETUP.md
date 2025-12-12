# GPU Setup for Ollama

Guide for enabling GPU acceleration with AMD Radeon cards.

## Current Hardware

- **GPU:** AMD Radeon RX 570/580 (8GB VRAM)
- **CPU:** Intel i7-8086K (6 cores / 12 threads)
- **RAM:** 64GB
- **OS:** Ubuntu 25.10

## Check GPU Status

```bash
# See what Ollama is using
ollama ps
# If it shows "100% CPU" - GPU not working

# Check GPU devices exist
ls /dev/kfd /dev/dri/render*
# Should show: /dev/kfd, /dev/dri/renderD128, etc.
```

## Option 1: Native ROCm Install

ROCm is AMD's GPU compute platform (like CUDA for NVIDIA).

### Install ROCm

```bash
# Download AMD installer (use noble for Ubuntu 24.04+)
wget https://repo.radeon.com/amdgpu-install/6.3.1/ubuntu/noble/amdgpu-install_6.3.60301-1_all.deb
sudo dpkg -i amdgpu-install_6.3.60301-1_all.deb
sudo apt update

# Install ROCm (--no-dkms if kernel module issues)
sudo amdgpu-install --usecase=rocm --no-dkms

# Add user to required groups
sudo usermod -aG render $USER
sudo usermod -aG video $USER

# Reboot
sudo reboot
```

### Verify ROCm

```bash
rocminfo | head -30
# Should show your GPU

rocm-smi
# Shows GPU stats
```

### For Older GPUs (RX 580, etc.)

If Ollama doesn't detect the GPU, force compatibility:

```bash
# Add to /etc/systemd/system/ollama.service.d/override.conf
sudo systemctl edit ollama
```

Add:
```ini
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
Environment="HSA_OVERRIDE_GFX_VERSION=9.0.0"
```

Then:
```bash
sudo systemctl daemon-reload
sudo systemctl restart ollama
```

## Option 2: Docker with ROCm

Cleaner approach - runs Ollama in a container with ROCm support.

### Stop native Ollama

```bash
sudo systemctl stop ollama
sudo systemctl disable ollama
```

### Run Ollama ROCm container

```bash
docker run -d \
  --name ollama-rocm \
  --device=/dev/kfd \
  --device=/dev/dri \
  --group-add video \
  -p 11434:11434 \
  -v ollama-data:/root/.ollama \
  --restart unless-stopped \
  ollama/ollama:rocm
```

### Pull models in container

```bash
docker exec ollama-rocm ollama pull qwen2.5-coder:7b
docker exec ollama-rocm ollama pull deepseek-coder:6.7b
```

### Check GPU usage

```bash
docker exec ollama-rocm ollama ps
# Should show GPU % instead of CPU
```

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

Once GPU is working, update systemd config:

```bash
sudo systemctl edit ollama
```

```ini
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
Environment="HSA_OVERRIDE_GFX_VERSION=9.0.0"
Environment="OLLAMA_NUM_PARALLEL=4"
Environment="OLLAMA_FLASH_ATTENTION=1"
Environment="OLLAMA_KV_CACHE_TYPE=q8_0"
```

```bash
sudo systemctl daemon-reload
sudo systemctl restart ollama
```

## Expected Performance

| Model | CPU Only | With GPU |
|-------|----------|----------|
| qwen2.5-coder:3b | ~15 tok/s | ~40 tok/s |
| qwen2.5-coder:7b | ~8 tok/s | ~25 tok/s |
| deepseek-coder:6.7b | ~7 tok/s | ~20 tok/s |
| qwen2.5-coder:14b | ~3 tok/s | ~10 tok/s (partial GPU) |

## Troubleshooting

### "100% CPU" still showing

```bash
# Check ROCm sees GPU
rocminfo | grep -i "name"

# Check Ollama logs
journalctl -u ollama -f

# Try HSA override
export HSA_OVERRIDE_GFX_VERSION=9.0.0
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
