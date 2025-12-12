# OS Strategy: The Golden Idol

The clood repo should be like Indiana Jones swapping the idol for the sandbag - the disk can disappear, and on a fresh machine, clood should be able to:

1. **Detect hardware** (CPU, GPU, RAM)
2. **Recommend optimal configuration**
3. **Bootstrap the environment**
4. **Pull models appropriate for the hardware**

---

## Current State: Ubuntu 25 Desktop

**What You're Running:**
- Ubuntu 25 Desktop (GNOME)
- ~4GB RAM overhead from GUI
- Vulkan via Mesa RADV
- Ollama with models on /home

**The Problem:**
- GNOME consumes RAM/CPU that could go to inference
- Desktop packages you don't need
- Non-reproducible (manual configuration)

---

## Distro Comparison for LLM Inference

| Distro | RAM Overhead | Mesa Version | Reproducible | Learning Curve | Best For |
|--------|--------------|--------------|--------------|----------------|----------|
| **Ubuntu Desktop** | ~4GB (GNOME) | 24.0 | ❌ | Low | Current setup |
| **Ubuntu Server** | ~512MB | 24.0 | ❌ | Low | Quick win |
| **Debian Minimal** | ~256MB | 23.x | ❌ | Low | Stability |
| **Arch Linux** | ~256MB | 25.x+ | ❌ | High | Bleeding edge |
| **NixOS** | ~256MB | 25.x | ✅ | Very High | Golden idol |
| **Fedora Server** | ~512MB | 25.x | ❌ | Medium | Balance |

### Why Mesa Version Matters (RX 590)

Your RX 590 uses **Vulkan via RADV** (not ROCm). Newer Mesa = better Vulkan performance:

- Mesa 24.0 (Ubuntu 24.04): Good
- Mesa 24.2 (Ubuntu 25.04): Better
- Mesa 25.x (Arch/Fedora): Best

---

## The Options

### Option 1: Quick Win - Strip Ubuntu Desktop

Keep Ubuntu, remove GUI overhead:

```bash
# Switch to multi-user target (no GUI)
sudo systemctl set-default multi-user.target

# Disable GNOME services
sudo systemctl disable gdm3

# Reboot into CLI
sudo reboot

# To get GUI back temporarily:
sudo systemctl start gdm3
```

**Saves:** ~2-3GB RAM
**Effort:** 5 minutes
**Downside:** Still have desktop packages installed

### Option 2: Ubuntu Server Fresh Install

Reinstall with Ubuntu Server 24.04 LTS:

```bash
# After install, just add what you need:
sudo apt update
sudo apt install -y curl git docker.io mesa-vulkan-drivers

# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh
```

**Saves:** ~3-4GB RAM
**Effort:** 1-2 hours
**Downside:** Manual configuration, not reproducible

### Option 3: Arch Linux (Maximum Performance)

Minimal install, only what you need:

```bash
# Base install + essentials only
pacman -S base linux linux-firmware
pacman -S mesa vulkan-radeon libva-mesa-driver
pacman -S ollama docker git

# Latest Mesa = best Vulkan performance
# Rolling release = always up to date
```

**Saves:** ~3.5GB RAM
**Effort:** 4-8 hours (initial), ongoing maintenance
**Downside:** Requires Linux expertise, can break on updates

### Option 4: NixOS (The Golden Idol) ⭐

**This is the answer for reproducibility.**

Your entire system becomes a single config file:

```nix
# configuration.nix - THE ENTIRE SYSTEM
{ config, pkgs, ... }:

{
  # Boot
  boot.loader.systemd-boot.enable = true;

  # No GUI - maximum performance
  services.xserver.enable = false;

  # GPU - Vulkan for RX 590
  hardware.graphics = {
    enable = true;
    extraPackages = with pkgs; [
      amdvlk
      rocmPackages.clr.icd
    ];
  };

  # Ollama - declarative!
  services.ollama = {
    enable = true;
    acceleration = "rocm";  # or use Vulkan
    loadModels = [
      "tinyllama"
      "llama3.2:3b"
      "deepseek-coder:6.7b"
      "mistral:7b"
    ];
    environmentVariables = {
      OLLAMA_HOST = "0.0.0.0:11434";
    };
  };

  # Docker for SearXNG
  virtualisation.docker.enable = true;

  # Essential packages
  environment.systemPackages = with pkgs; [
    git
    curl
    jq
    htop
    nvtop  # GPU monitoring
  ];

  # Network
  networking.firewall.allowedTCPPorts = [ 11434 8888 4000 ];

  # User
  users.users.mgilbert = {
    isNormalUser = true;
    extraGroups = [ "wheel" "docker" ];
  };
}
```

**The Magic:**
```bash
# On ANY machine with NixOS:
git clone https://github.com/dirtybirdnj/clood
cd clood/nixos
sudo nixos-rebuild switch --flake .#ubuntu25

# Entire system is now configured identically
# Models auto-download on first boot
```

**Saves:** ~3.5GB RAM
**Effort:** 8-16 hours (initial learning)
**Upside:** TRUE REPRODUCIBILITY - the golden idol
**Downside:** Steep learning curve, different mental model

---

## Recommendation

### Short Term (This Week)
1. **Strip the GUI from Ubuntu 25:**
   ```bash
   sudo systemctl set-default multi-user.target
   sudo reboot
   ```
2. Gain ~2-3GB RAM immediately
3. SSH in from your Mac for GUI needs

### Medium Term (This Month)
1. **Install Ubuntu Server 24.04 LTS** on a spare drive/partition
2. Document the exact setup steps in clood
3. Test performance difference

### Long Term (The Vision)
1. **Learn NixOS fundamentals**
2. **Create clood NixOS flake** that defines the entire environment
3. **Hardware-specific profiles** that auto-detect and configure
4. **One command** to go from bare metal to fully configured

---

## The Golden Idol Architecture

```
clood/
├── nixos/                          # NixOS configurations
│   ├── flake.nix                   # Entry point
│   ├── modules/
│   │   ├── ollama.nix              # Ollama service config
│   │   ├── searxng.nix             # SearXNG container
│   │   └── litellm.nix             # LiteLLM proxy
│   └── hosts/
│       ├── ubuntu25.nix            # RX 590, i7-8086K config
│       ├── macbook-air.nix         # M4 16GB config
│       └── mac-mini.nix            # M4 24GB config
│
├── scripts/
│   ├── detect-hardware.sh          # Identify CPU/GPU/RAM
│   ├── recommend-models.sh         # Suggest models for hardware
│   └── bootstrap.sh                # Full setup from scratch
│
└── hardware/
    ├── profiles/
    │   ├── amd-polaris.nix         # RX 580/590 config
    │   ├── amd-rdna2.nix           # RX 6000 series
    │   ├── nvidia-ampere.nix       # RTX 30 series
    │   └── apple-silicon.nix       # M1/M2/M3/M4
    └── detection/
        └── gpu-identify.sh         # Auto-detect GPU family
```

---

## Bootstrap Script Concept

```bash
#!/bin/bash
# clood bootstrap - detect hardware, recommend config

echo "=== clood Hardware Detection ==="

# CPU
CPU=$(lscpu | grep "Model name" | cut -d: -f2 | xargs)
CORES=$(nproc)
echo "CPU: $CPU ($CORES cores)"

# RAM
RAM_GB=$(free -g | awk '/Mem:/ {print $2}')
echo "RAM: ${RAM_GB}GB"

# GPU
if lspci | grep -i "vga\|3d" | grep -qi nvidia; then
    GPU_TYPE="nvidia"
    GPU_NAME=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null || echo "Unknown NVIDIA")
    VRAM=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits 2>/dev/null || echo "?")
elif lspci | grep -i "vga\|3d" | grep -qi amd; then
    GPU_TYPE="amd"
    GPU_NAME=$(lspci | grep -i "vga\|3d" | grep -i amd | cut -d: -f3 | xargs)
    # AMD VRAM detection is trickier
    VRAM=$(cat /sys/class/drm/card0/device/mem_info_vram_total 2>/dev/null | awk '{print $1/1024/1024/1024}' || echo "?")
else
    GPU_TYPE="cpu-only"
    GPU_NAME="None detected"
    VRAM="0"
fi
echo "GPU: $GPU_NAME (${VRAM}GB VRAM) [$GPU_TYPE]"

echo ""
echo "=== Recommended Configuration ==="

# Model recommendations based on VRAM
if [ "$VRAM" -ge 24 ]; then
    echo "Tier: HEAVY - Can run 33B+ models"
    echo "Recommended: deepseek-coder:33b, qwen2.5-coder:32b"
elif [ "$VRAM" -ge 16 ]; then
    echo "Tier: LARGE - Can run 14B-20B models"
    echo "Recommended: qwen2.5-coder:14b, codellama:13b"
elif [ "$VRAM" -ge 8 ]; then
    echo "Tier: MEDIUM - Can run 7B-8B models"
    echo "Recommended: deepseek-coder:6.7b, mistral:7b, llama3-groq-tool-use:8b"
elif [ "$VRAM" -ge 4 ]; then
    echo "Tier: LIGHT - Can run 3B models"
    echo "Recommended: llama3.2:3b, phi-3:mini"
else
    echo "Tier: CPU-ONLY - Limited to small models"
    echo "Recommended: tinyllama, phi-3:mini"
fi

# Backend recommendation
echo ""
echo "=== Backend Configuration ==="
case $GPU_TYPE in
    nvidia)
        echo "Use: CUDA backend"
        echo "Install: NVIDIA drivers + CUDA toolkit"
        ;;
    amd)
        if echo "$GPU_NAME" | grep -qi "polaris\|rx 5[78]0\|rx 590"; then
            echo "Use: Vulkan backend (ROCm dropped gfx803 support)"
            echo "Set: OLLAMA_VULKAN=true"
        else
            echo "Use: ROCm backend"
            echo "Install: ROCm 6.x"
        fi
        ;;
    *)
        echo "Use: CPU backend with AVX2"
        echo "Set: OLLAMA_NUM_THREAD=$CORES"
        ;;
esac
```

---

## Next Steps

1. **Immediate:** Strip GUI from current Ubuntu
2. **This week:** Create `scripts/detect-hardware.sh`
3. **This month:** Test Ubuntu Server on spare partition
4. **Future:** Build NixOS flake for full reproducibility

---

## Resources

- [NixOS Ollama Wiki](https://wiki.nixos.org/wiki/Ollama)
- [Arch Linux AI](https://www.talentelgia.com/blog/top-5-linux-distro-for-ai/)
- [Ubuntu Server vs Desktop](https://www.hostinger.com/tutorials/ubuntu-server-vs-desktop)
- [LLM Linux Server Build](https://linuxblog.io/build-llm-linux-server-on-budget/)
