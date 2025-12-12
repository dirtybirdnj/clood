# ubuntu25 Performance Optimization Guide

```
Old silicon dreams
Tokens flow through copper veins
Speed from careful hands

Kernel whispers low
Swappiness falls to zero
Memory stays close

No queue for the drive
NVMe speaks directly now
Latency dissolves

Six cores stand alone
Threads pinned to silicon beds
No wandering tasks

Huge pages unfold
Two megabytes at a time
TLB finds its peace

XMP awakens
DDR4 runs at full stride
BIOS knows the way

Twenty-seven percent
Wrung from aging copper paths
Clood runs swift tonight
```

Deep optimization techniques for maximizing LLM inference performance beyond basic settings.

**Hardware:** i7-8086K (6C/12T, AVX2) + RX 590 8GB (Vulkan) + 64GB DDR4 + NVMe

---

## Quick Wins (Apply Now)

These require no reboot and provide immediate gains:

```bash
# 1. CPU Governor (10-30% improvement)
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# 2. Transparent Huge Pages
echo always | sudo tee /sys/kernel/mm/transparent_hugepage/enabled
echo madvise | sudo tee /sys/kernel/mm/transparent_hugepage/defrag

# 3. Reduce swappiness
sudo sysctl -w vm.swappiness=1

# 4. NVMe scheduler
echo none | sudo tee /sys/block/nvme0n1/queue/scheduler
echo none | sudo tee /sys/block/nvme1n1/queue/scheduler

# 5. Verify
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq
```

---

## Ollama Configuration

Update `/etc/systemd/system/ollama.service.d/override.conf`:

```ini
[Service]
# Network
Environment="OLLAMA_HOST=0.0.0.0:11434"

# GPU (Vulkan for RX 590)
Environment="OLLAMA_VULKAN=true"
Environment="HIP_VISIBLE_DEVICES="
Environment="GGML_VK_VISIBLE_DEVICES=0"
Environment="GGML_VK_DEVICE0_MEMORY=7000000000"

# Performance
Environment="OLLAMA_FLASH_ATTENTION=1"
Environment="OLLAMA_KV_CACHE_TYPE=q8_0"
Environment="OLLAMA_NUM_THREAD=6"
Environment="OLLAMA_NUM_BATCH=512"
Environment="OLLAMA_NUM_CTX=8192"

# Concurrency
Environment="OLLAMA_NUM_PARALLEL=1"
Environment="OLLAMA_MAX_LOADED_MODELS=1"
Environment="OLLAMA_KEEP_ALIVE=15m"

# Model storage
Environment="OLLAMA_MODELS=/home/ollama-models"

# CPU Affinity (physical cores only)
CPUAffinity=0 1 2 3 4 5

# Memory
LimitMEMLOCK=infinity
```

Apply:
```bash
sudo systemctl daemon-reload
sudo systemctl restart ollama
```

---

## Kernel Parameters

### Create `/etc/sysctl.d/99-llm-inference.conf`:

```ini
# Memory Management
vm.swappiness = 1
vm.dirty_ratio = 10
vm.dirty_background_ratio = 3
vm.dirty_expire_centisecs = 300
vm.dirty_writeback_centisecs = 100
vm.vfs_cache_pressure = 50
vm.overcommit_memory = 1
vm.overcommit_ratio = 100

# Memory mapping
vm.max_map_count = 262144

# Network (for remote Ollama)
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.ipv4.tcp_rmem = 4096 87380 67108864
net.ipv4.tcp_wmem = 4096 65536 67108864

# Scheduler tuning
kernel.sched_migration_cost_ns = 5000000
```

Apply:
```bash
sudo sysctl -p /etc/sysctl.d/99-llm-inference.conf
```

---

## I/O Scheduler

### Create `/etc/udev/rules.d/60-ioschedulers.rules`:

```bash
# NVMe: no scheduler (lowest latency)
ACTION=="add|change", KERNEL=="nvme[0-9]n[0-9]", ATTR{queue/scheduler}="none"

# SATA SSD: mq-deadline
ACTION=="add|change", KERNEL=="sd[a-z]", ATTR{queue/rotational}=="0", ATTR{queue/scheduler}="mq-deadline"

# HDD: mq-deadline
ACTION=="add|change", KERNEL=="sd[a-z]", ATTR{queue/rotational}=="1", ATTR{queue/scheduler}="mq-deadline"
```

Apply:
```bash
sudo udevadm control --reload-rules
sudo udevadm trigger --type=devices --action=change
```

---

## CPU Governor Service

### Create `/etc/systemd/system/cpu-performance.service`:

```ini
[Unit]
Description=Set CPU governor to performance
After=multi-user.target

[Service]
Type=oneshot
ExecStart=/bin/sh -c "echo performance | tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor"
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
```

Enable:
```bash
sudo systemctl daemon-reload
sudo systemctl enable cpu-performance.service
sudo systemctl start cpu-performance.service
```

---

## Boot Parameters (Requires Reboot)

Edit `/etc/default/grub`, add to `GRUB_CMDLINE_LINUX_DEFAULT`:

```bash
# Performance governor at boot
intel_pstate=active

# Transparent huge pages
transparent_hugepage=always

# Optional: Disable security mitigations (5-15% faster, less secure)
# mitigations=off
```

Apply:
```bash
sudo update-grub
sudo reboot
```

---

## HugePages (Optional, Advanced)

For heavy LLM workloads, HugePages reduce TLB misses:

```bash
# Allocate 16GB of HugePages (8192 x 2MB)
echo "vm.nr_hugepages = 8192" | sudo tee -a /etc/sysctl.d/99-llm-inference.conf
echo "vm.hugetlb_shm_group = 1000" | sudo tee -a /etc/sysctl.d/99-llm-inference.conf
sudo sysctl -p /etc/sysctl.d/99-llm-inference.conf

# Verify
cat /proc/meminfo | grep Huge
```

---

## Model-Specific Optimizations

### Context Length by Model (8GB VRAM)

| Model | Recommended num_ctx | Notes |
|-------|---------------------|-------|
| TinyLlama 1B | 2048 | Full GPU |
| Qwen 3B | 16384 | Full GPU |
| Qwen/Llama 7-8B | 8192 | Full GPU |
| 13-14B | 4096 | Partial GPU offload |
| 30B+ | 2048 | Heavy CPU spillover |

### Create Optimized Modelfiles

```bash
# High-context 3B model
cat > /tmp/Modelfile.3b-maxctx << 'EOF'
FROM qwen2.5-coder:3b
PARAMETER num_ctx 16384
PARAMETER num_batch 512
PARAMETER num_gpu 999
EOF
ollama create qwen2.5-coder:3b-max -f /tmp/Modelfile.3b-maxctx

# Optimized 7B model
cat > /tmp/Modelfile.7b-opt << 'EOF'
FROM qwen2.5-coder:7b
PARAMETER num_ctx 8192
PARAMETER num_batch 512
PARAMETER num_gpu 999
EOF
ollama create qwen2.5-coder:7b-opt -f /tmp/Modelfile.7b-opt
```

---

## Monitoring

### VRAM Usage

```bash
# Create monitoring script
cat > ~/bin/gpu-watch << 'EOF'
#!/bin/bash
watch -n1 'echo "=== RX 590 Status ===" && \
  echo "VRAM Used: $(numfmt --to=iec < /sys/class/drm/card1/device/mem_info_vram_used)" && \
  echo "VRAM Total: $(numfmt --to=iec < /sys/class/drm/card1/device/mem_info_vram_total)" && \
  echo "GPU Load: $(cat /sys/class/drm/card1/device/gpu_busy_percent 2>/dev/null || echo N/A)%"'
EOF
chmod +x ~/bin/gpu-watch
```

### CPU Frequency

```bash
watch -n1 "cat /sys/devices/system/cpu/cpu*/cpufreq/scaling_cur_freq | sort -u | head -1 | awk '{printf \"CPU: %.2f GHz\n\", \$1/1000000}'"
```

### Temperature

```bash
sudo apt install lm-sensors
sudo sensors-detect
watch -n1 sensors
```

---

## Benchmarking

```bash
# Quick benchmark
ollama run qwen2.5-coder:7b "Write a Python fibonacci function" --verbose 2>&1 | grep "eval rate"

# Full benchmark script
cat > ~/bin/bench-llm << 'EOF'
#!/bin/bash
MODEL=${1:-qwen2.5-coder:7b}
echo "=== Benchmarking $MODEL ==="
echo "CPU Governor: $(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor)"
echo "CPU Freq: $(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq | awk '{printf "%.2f GHz", $1/1000000}')"
echo ""
ollama run "$MODEL" "Write a Python function to calculate factorial using recursion and iteration. Include docstrings and type hints." --verbose 2>&1 | tail -5
EOF
chmod +x ~/bin/bench-llm
```

---

## Expected Performance

After full optimization:

| Model | Before | After | Improvement |
|-------|--------|-------|-------------|
| TinyLlama 1B | ~150 tok/s | ~170 tok/s | +13% |
| Qwen 3B | ~64 tok/s | ~75 tok/s | +17% |
| Qwen 7B | ~32 tok/s | ~40 tok/s | +25% |
| Llama 8B | ~30 tok/s | ~38 tok/s | +27% |

---

## Checklist

### Immediate (no reboot)
- [ ] CPU governor → performance
- [ ] Enable THP
- [ ] Set vm.swappiness=1
- [ ] NVMe scheduler → none
- [ ] Update Ollama override.conf
- [ ] Restart Ollama

### Persistent (survives reboot)
- [ ] Create cpu-performance.service
- [ ] Create /etc/sysctl.d/99-llm-inference.conf
- [ ] Create /etc/udev/rules.d/60-ioschedulers.rules
- [ ] Update GRUB parameters

### Optional (advanced)
- [ ] Configure HugePages
- [ ] CPU isolation (isolcpus)
- [ ] Disable mitigations (security tradeoff)
- [ ] Update BIOS (XMP, power limits)

---

## References

- [llama.cpp Optimization Guide](https://github.com/ggml-org/llama.cpp)
- [Ollama Performance Tuning](https://github.com/ollama/ollama/blob/main/docs/faq.md)
- [Linux Kernel VM Documentation](https://docs.kernel.org/admin-guide/sysctl/vm.html)
- [Intel i7-8086K Specifications](https://ark.intel.com/content/www/us/en/ark/products/148263/)
- [AMD Radeon RX 590 Specifications](https://www.amd.com/en/products/graphics/amd-radeon-rx-590)
