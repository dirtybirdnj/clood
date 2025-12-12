# Last Session - 2025-12-12 (Late Night)

## Context Dump: BIOS Optimization for i7-8086K

### Your Motherboard (Confirmed in Repo)

**Gigabyte Z390 AORUS PRO WIFI-CF** (not 380 - it's Z390)
- BIOS Version: F11 (dated 10/15/2019)
- Source: `hardware/ubuntu25-profile.md`

### BIOS Settings We Were About to Configure

Enter BIOS: Press `DEL` during POST

---

#### 1. Intel Turbo Boost
**Path:** `M.I.T.` → `Advanced Frequency Settings` → `Advanced CPU Core Settings`

| Setting | Recommended |
|---------|-------------|
| Intel Turbo Boost | **Enabled** |
| Turbo Boost Short Power Max | **Disabled** (removes PL2 time limit) |
| Turbo Boost Power Max | **Disabled** (removes PL1 power limit) |

---

#### 2. C-States (Disable for Performance)
**Path:** `M.I.T.` → `Advanced Frequency Settings` → `Advanced CPU Core Settings`

| Setting | Recommended |
|---------|-------------|
| CPU Enhanced Halt (C1E) | **Disabled** |
| C3 State Support | **Disabled** |
| C6/C7 State Support | **Disabled** |
| Package C State Limit | **C0/C1** |

---

#### 3. SpeedStep (EIST)
**Path:** `M.I.T.` → `Advanced Frequency Settings` → `Advanced CPU Core Settings`

| Setting | Recommended |
|---------|-------------|
| Intel Speed Shift Technology | **Disabled** |
| Enhanced Intel SpeedStep (EIST) | **Disabled** |

---

#### 4. Per-Core Turbo Ratios
**Path:** `M.I.T.` → `Advanced Frequency Settings` → `Advanced CPU Core Settings` → `CPU Clock Ratio`

| Cores Active | Ratio | Frequency |
|--------------|-------|-----------|
| 1-Core | 50 | 5.0 GHz |
| 2-Core | 50 | 5.0 GHz |
| 3-Core | 49 | 4.9 GHz |
| 4-Core | 48 | 4.8 GHz |
| 5-Core | 47 | 4.7 GHz |
| 6-Core | 47 | 4.7 GHz |

Or just set **CPU Clock Ratio: 48** (all-core 4.8GHz) as safe starting point.

---

#### 5. LLC (Load Line Calibration)
**Path:** `M.I.T.` → `Advanced Voltage Settings` → `CPU Core Voltage Control`

| Setting | Recommended |
|---------|-------------|
| CPU Vcore Loadline Calibration | **Turbo** (Level 5 of 7) |

Scale: Low → Normal → Standard → High → **Turbo** → Extreme → Ultra Extreme

---

#### 6. CPU Core Ratio
**Path:** `M.I.T.` → `Advanced Frequency Settings`

| Setting | Safe | Aggressive |
|---------|------|------------|
| CPU Clock Ratio | **48** (4.8GHz) | **50** (5.0GHz) |
| Ring Ratio | **43** | **47** |

---

### Also Enable: XMP
**Path:** `M.I.T.` → `Advanced Memory Settings`

| Setting | Value |
|---------|-------|
| Extreme Memory Profile (X.M.P.) | **Profile 1** |

---

### Quick Reference - Full Settings List

```
M.I.T. → Advanced Frequency Settings:
  CPU Clock Ratio: 48 (safe) or 50 (aggressive)
  Ring Ratio: 43

M.I.T. → Advanced Frequency Settings → Advanced CPU Core Settings:
  Intel Turbo Boost: Enabled
  Turbo Boost Short Power Max: Disabled
  Turbo Boost Power Max: Disabled
  CPU Enhanced Halt (C1E): Disabled
  C3/C6/C7 State Support: Disabled
  Package C State Limit: C0/C1
  Intel Speed Shift Technology: Disabled
  Enhanced Intel SpeedStep (EIST): Disabled

M.I.T. → Advanced Voltage Settings → CPU Core Voltage Control:
  CPU Vcore Loadline Calibration: Turbo

M.I.T. → Advanced Memory Settings:
  X.M.P.: Profile 1
```

---

### NOT Touching (Per Your Request)
- Voltage manipulation (Vcore, VCCIO, VCCSA)
- CPU voltage override
- AVX offset

---

### What Already Exists in Repo
- `hardware/OPTIMIZATION-GUIDE.md` - Software/kernel optimizations (done)
- `hardware/ubuntu25-profile.md` - Full hardware profile
- `hardware/i7-8086k.md` - CPU specs and haiku

### Pending
- Create dedicated BIOS optimization doc? (you asked, I said yes)
- After BIOS changes: re-benchmark to measure improvement

---

## Previous Session Summary (Earlier Today)

- Mac Mini M4 benchmarking (~126 tok/s TinyLlama, ~44 tok/s Qwen 3B)
- RTX 2080 integration planning doc
- WEEKEND-SURVIVAL.md for Claude-free coding
- fun/bonsai.py stubbed for dogfooding
- Session 3 haikus (Frankenstein/dogfooding)

---

## Quick Commands Post-Reboot

```bash
# Verify CPU frequency (should be higher after BIOS changes)
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq

# Watch frequencies in real-time
watch -n1 "cat /sys/devices/system/cpu/cpu*/cpufreq/scaling_cur_freq | sort -u"

# Set governor to performance (if not already)
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Quick Ollama test
ollama run qwen2.5-coder:7b "Hello" --verbose 2>&1 | grep "eval rate"
```

---

## Issue That Caused Restart

Mouse scroll wheel injecting garbage into terminal (both macOS Terminal and Kitty affected).
