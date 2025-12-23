# Troubleshooting Guide

Centralized troubleshooting for common clood issues.

---

## Quick Diagnostics

Run the verification script first:
```bash
./scripts/verify-setup.sh
```

---

## Ollama Issues

### Ollama service not starting

```bash
# Check status
sudo systemctl status ollama

# View logs
sudo journalctl -u ollama -n 50 --no-pager

# Common fix: restart
sudo systemctl restart ollama
```

### "No models found" / Models not visible

**Cause:** OLLAMA_MODELS path mismatch

```bash
# Check where Ollama is looking
cat /etc/systemd/system/ollama.service.d/override.conf | grep OLLAMA_MODELS

# Check where models actually are
ls -la /home/ollama-models/
ls -la ~/.ollama/models/

# Fix: Ensure path matches and permissions are correct
sudo chown -R ollama:ollama /home/ollama-models
sudo systemctl restart ollama
```

### GPU not being used

**Symptoms:** Slow inference, high CPU usage

```bash
# Check Ollama is configured for Vulkan (RX 590)
grep -E "VULKAN|VK|HIP" /etc/systemd/system/ollama.service.d/override.conf

# Should see:
# Environment="OLLAMA_VULKAN=true"
# Environment="GGML_VK_VISIBLE_DEVICES=0"

# NOT:
# Environment="HSA_OVERRIDE_GFX_VERSION=9.0.0"  (ROCm - doesn't work with gfx803)
```

### Model download stuck/slow

```bash
# Check disk space
df -h /home

# Check network
ping -c 3 ollama.ai

# Retry with verbose
OLLAMA_DEBUG=1 ollama pull modelname
```

---

## SearXNG Issues

### SearXNG not responding on :8888

```bash
# Check container status
docker ps | grep searxng

# Start if not running
cd ~/Code/clood/infrastructure
docker compose up -d searxng

# Check logs
docker logs searxng --tail 50

# Test directly
curl "http://localhost:8888/search?q=test&format=json" | head
```

### SearXNG returning empty results

**Cause:** Search engines may be rate-limited or blocked

```bash
# Check settings
cat infrastructure/configs/searxng/settings.yml | grep -A5 "engines"

# Test specific engine
curl "http://localhost:8888/search?q=test&format=json&engines=duckduckgo"
```

---

## Performance Issues

### Slow inference

**Check CPU governor:**
```bash
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

# Fix: Set to performance
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Make persistent: See hardware/OPTIMIZATION-GUIDE.md
```

**Check GPU is being used:**
```bash
# During inference, run in another terminal:
watch -n1 "cat /sys/class/drm/card*/device/gpu_busy_percent 2>/dev/null"
# Should show >0% when model is running
```

**Check model size vs VRAM:**
```bash
# RX 590 has 8GB VRAM
# Safe models: 7B-8B (Q4 quant) or smaller
# Marginal: 13B (may offload to CPU)
# Too large: 33B+ (will be CPU-only)
```

### High memory usage

```bash
# Check memory
free -h

# Check what Ollama is using
ps aux | grep ollama

# Ollama keeps models loaded - unload if needed:
curl -X DELETE http://localhost:11434/api/generate -d '{"model": "modelname"}'
```

---

## Network Issues

### Can't access Ollama from other machines

```bash
# Check Ollama is bound to all interfaces
grep OLLAMA_HOST /etc/systemd/system/ollama.service.d/override.conf
# Should be: OLLAMA_HOST=0.0.0.0:11434

# Check firewall
sudo ufw status
sudo ufw allow 11434

# Test from remote machine
curl http://192.168.4.63:11434/api/tags
```

### SSH connection refused

```bash
# Check SSH service
sudo systemctl status ssh

# Check SSH config allows key auth
grep PubkeyAuthentication /etc/ssh/sshd_config

# Verify key permissions
ls -la ~/.ssh/
# Private key should be 600, .ssh directory should be 700
```

---

## Docker Issues

### Containers not starting

```bash
# Check Docker service
sudo systemctl status docker

# Check compose file syntax
cd ~/Code/clood/infrastructure
docker compose config

# Start with logs
docker compose up
```

### Disk space issues

```bash
# Check disk usage
df -h

# Clean Docker
docker system prune -a

# Clean Ollama old models
ollama rm unused-model-name
```

---

## Common Error Messages

| Error | Cause | Fix |
|-------|-------|-----|
| `model 'X' not found` | Model not pulled or wrong name | `ollama pull X` with exact tag |
| `connection refused :11434` | Ollama not running | `sudo systemctl start ollama` |
| `no space left on device` | Disk full | Move models to /home or clean up |
| `GPU not found` | Driver issue or wrong backend | Check Vulkan config, not ROCm |
| `permission denied` | Wrong ownership | `sudo chown -R ollama:ollama /path` |
| `CUDA error` | Wrong GPU backend | Use Vulkan for RX 590, not CUDA/ROCm |

---

## Getting Help

1. Run `clood doctor` for diagnostics
2. Check this doc for your specific error
3. Review component-specific docs:
   - Ollama: `ollama-tuning.md`
   - GPU: `GPU-SETUP.md`, `hardware/rx590.md`
   - clood: `clood-cli/docs/USAGE_GUIDE.md`
4. Check logs: `sudo journalctl -u ollama -f`
