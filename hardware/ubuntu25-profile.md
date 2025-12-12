# ubuntu25 Hardware Profile

Complete hardware specification for the primary clood workstation.

---

## System Summary

| Component | Model | Key Specs |
|-----------|-------|-----------|
| CPU | Intel Core i7-8086K | 6C/12T, 4.0GHz base, 5.0GHz turbo |
| Motherboard | Gigabyte Z390 AORUS PRO WIFI-CF | Intel Z390 chipset |
| RAM | DDR4 | 64GB (speed TBD) |
| GPU | AMD Radeon RX 590 | 8GB GDDR5, Polaris (gfx803) |
| Boot Drive | WD_BLACK SN850X 1TB | NVMe Gen4 |
| Storage | Samsung 970 EVO Plus 500GB | NVMe Gen3 |
| Storage | SanDisk SSD PLUS 1TB | SATA SSD |
| Storage | Toshiba 4TB | HDD (HDWR440) |

---

## CPU: Intel Core i7-8086K

**40th Anniversary Limited Edition** - Only 8,086 units made.

| Spec | Value |
|------|-------|
| Codename | Coffee Lake |
| Process | 14nm++ |
| Cores / Threads | 6 / 12 |
| Base Clock | 4.0 GHz |
| Max Turbo (1 core) | 5.0 GHz |
| All-Core Turbo | 4.3 GHz |
| L3 Cache | 12 MB SmartCache |
| TDP | 95W |
| Memory Support | DDR4-2666 (dual channel) |
| PCIe | 3.0, 16 lanes |
| Socket | LGA 1151 |

### Instruction Set Support

| Feature | Status | LLM Relevance |
|---------|--------|---------------|
| SSE4.1/4.2 | ✅ | Basic SIMD |
| AVX | ✅ | Vector math |
| AVX2 | ✅ | **Used by llama.cpp** |
| AVX-512 | ❌ | Would help, not available |
| FMA | ✅ | Fused multiply-add |
| AES-NI | ✅ | Encryption |
| VT-x / VT-d | ✅ | Virtualization |

### Current State

```
Governor: powersave (NEEDS FIX)
Current Freq: ~4.3 GHz (should be 5.0 GHz)
Min Freq: 800 MHz
Max Freq: 5000 MHz
```

### Performance Optimization

See [i7-8086k.md](i7-8086k.md) for detailed tuning guide.

---

## Motherboard: Gigabyte Z390 AORUS PRO WIFI-CF

| Spec | Value |
|------|-------|
| Chipset | Intel Z390 |
| BIOS Vendor | American Megatrends Inc. |
| BIOS Version | F11 |
| BIOS Date | 10/15/2019 |
| Form Factor | ATX |

### Features

- Intel Z390 chipset (Coffee Lake optimized)
- Dual channel DDR4 support up to 4266MHz (OC)
- 2x M.2 slots (NVMe)
- Intel GbE LAN + WiFi
- USB 3.1 Gen2

### BIOS Optimization Potential

- [ ] Check for newer BIOS (F11 is from 2019)
- [ ] XMP profile for RAM
- [ ] Per-core turbo ratios
- [ ] Power limits (PL1/PL2)
- [ ] LLC (Load Line Calibration)

---

## RAM: 64GB DDR4

| Spec | Value |
|------|-------|
| Total | 64 GB |
| Swap | 14 GB |
| Architecture | Dual Channel DDR4 |
| Speed | TBD (check BIOS/dmidecode) |

### Memory Mapping

```
DirectMap4k:    433 MB
DirectMap2M:     14 GB
DirectMap1G:     52 GB
HugePages:       0 (not configured)
```

### Optimization Potential

- [ ] Verify XMP is enabled in BIOS
- [ ] Check if running at DDR4-2666 or higher
- [ ] Consider enabling HugePages for LLM workloads
- [ ] Verify dual-channel is active

---

## GPU: AMD Radeon RX 590

| Spec | Value |
|------|-------|
| Model | Sapphire Radeon RX 590 |
| Architecture | Polaris 30 (gfx803) |
| VRAM | 8 GB GDDR5 |
| Bus | PCIe 3.0 x16 |
| Shader Units | 2304 |
| Memory Bus | 256-bit |
| TDP | 225W |

### Driver Status

```
Kernel Driver: amdgpu
Ollama Backend: Vulkan (ROCm 6.x dropped gfx803)
```

### Current Configuration

```ini
OLLAMA_VULKAN=true
GGML_VK_VISIBLE_DEVICES=0
HIP_VISIBLE_DEVICES=  (disabled)
```

See [rx590.md](rx590.md) for GPU tuning details.

---

## Storage

### NVMe Drives

| Device | Model | Size | Role |
|--------|-------|------|------|
| nvme0n1 | WD_BLACK SN850X | 1 TB | Boot (/, /home) |
| nvme1n1 | Samsung 970 EVO Plus | 500 GB | Available |

### SATA Drives

| Device | Model | Size | Type |
|--------|-------|------|------|
| sda | SanDisk SSD PLUS | 1 TB | SSD |
| sdb | Toshiba HDWR440 | 4 TB | HDD |

### Partition Layout (nvme0n1)

```
nvme0n1p1   779 GB   /home
nvme0n1p2   489 MB   /boot/efi
nvme0n1p3    92 GB   /
```

**Note:** Root partition is undersized. See [REPARTITIONING.md](../infrastructure/REPARTITIONING.md).

---

## Network

### Interfaces

| Interface | Type | Status | Speed |
|-----------|------|--------|-------|
| eno1 | Intel GbE | DOWN (no cable) | 1000 Mbit/s capable |
| wlp4s0 | Intel WiFi (AX200?) | UP | RX: 390 Mbit/s, TX: 527 Mbit/s |

### Current Connection (WiFi)

```
SSID: thatwifi
Frequency: 5580 MHz (5GHz band)
Signal: -64 dBm (decent)
Link Speed: ~400-500 Mbit/s
```

### Bottleneck Analysis

**Primary limitation: WiFi instead of Ethernet**

- Gigabit ethernet = 1000 Mbit/s, stable, low latency
- Current WiFi = ~450 Mbit/s average, variable latency
- **Fix: Plug in ethernet cable to eno1**

### TCP Buffer Settings

Current (default):
```
net.core.rmem_max = 212992 (208 KB)
net.core.wmem_max = 212992 (208 KB)
```

Recommended for bulk transfers:
```bash
sudo sysctl -w net.core.rmem_max=134217728
sudo sysctl -w net.core.wmem_max=134217728
sudo sysctl -w net.ipv4.tcp_rmem="4096 87380 134217728"
sudo sysctl -w net.ipv4.tcp_wmem="4096 65536 134217728"
```

### Optimized rsync Command

```bash
rsync -avz --compress-level=1 --progress --inplace \
  -e "ssh -T -c aes128-gcm@openssh.com -o Compression=no" \
  source/ destination/
```

---

## Optimization Checklist

### Immediate (Software)

- [ ] Set CPU governor to `performance`
- [ ] Verify Ollama using Vulkan GPU
- [ ] Set OLLAMA_NUM_THREAD=6 for CPU workloads

### BIOS (Requires Reboot)

- [ ] Update BIOS if newer version available
- [ ] Enable XMP for RAM
- [ ] Check/increase power limits (PL1/PL2)
- [ ] Verify all-core turbo settings

### Advanced (Research Needed)

- [ ] HugePages for LLM memory allocation
- [ ] CPU affinity/pinning for Ollama
- [ ] I/O scheduler tuning for NVMe
- [ ] NUMA awareness (if applicable)
- [ ] Kernel parameters for performance

---

## Benchmarking Commands

```bash
# CPU frequency check
watch -n1 "cat /sys/devices/system/cpu/cpu*/cpufreq/scaling_cur_freq | sort -u"

# GPU utilization (requires radeontop)
radeontop

# Memory bandwidth (requires mbw)
mbw 1024

# Ollama inference benchmark
ollama run qwen2.5-coder:7b "Write hello world" --verbose 2>&1 | grep "eval rate"
```

---

## References

- [Intel Ark: i7-8086K](https://ark.intel.com/content/www/us/en/ark/products/148263/intel-core-i78086k-processor-12m-cache-up-to-5-00-ghz.html)
- [Gigabyte Z390 AORUS PRO WIFI](https://www.gigabyte.com/Motherboard/Z390-AORUS-PRO-WIFI-rev-10)
- [AMD Radeon RX 590 Specs](https://www.amd.com/en/products/graphics/amd-radeon-rx-590)
