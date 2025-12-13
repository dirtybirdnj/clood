# Last Session - 2025-12-12 (Evening)

## Context: Pre-BIOS Benchmark Baseline

You're about to reboot ubuntu25 for BIOS tuning. Here's the baseline to compare against.

---

## Pre-BIOS Benchmark Results (SAVE THESE!)

| Model | Avg tok/s | Best Test |
|-------|-----------|-----------|
| tinyllama | 189.6 | simple_math (200 tok/s) |
| llama3.2:3b | 64.6 | simple_math (75 tok/s) |
| deepseek-coder:6.7b | 42.8 | simple_math (43.75 tok/s) |
| mistral:7b | 32.5 | simple_math (35 tok/s) |

**System state before BIOS changes:**
- CPU Governor: `powersave`
- CPU Frequency: 4390 MHz
- RAM: 57GB available

---

## What We Did This Session

1. **Tested ollama-dashboard.sh** - tmux dashboard working perfectly
2. **Added temperature monitoring** - new sensors pane in dashboard
3. **Installed lm-sensors** - `sudo apt install lm-sensors && sudo sensors-detect`
4. **Loaded sensor modules** - `sudo modprobe coretemp it87`
5. **Ran full benchmarks** - baseline captured above

---

## BIOS Settings to Configure

Enter BIOS: Press `DEL` during POST

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

## After Reboot - Verify & Benchmark

```bash
# 1. Check CPU governor (should still be powersave, we'll fix it)
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

# 2. Set to performance mode
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# 3. Verify frequencies are higher
watch -n1 "cat /sys/devices/system/cpu/cpu*/cpufreq/scaling_cur_freq | sort -u"

# 4. Check temps are okay
sensors

# 5. Start the dashboard to monitor during benchmark
~/Code/clood/scripts/ollama-dashboard.sh

# 6. Run full benchmark (in another terminal or after Ctrl+b d)
~/Code/clood/scripts/benchmark.sh --all
```

---

## Tmux Cheatsheet (You Asked!)

| Keys | Action |
|------|--------|
| `Ctrl+b` then `d` | **Detach** - exit but keep running |
| `Ctrl+b` then `arrows` | Move between panes |
| `Ctrl+b` then `z` | Zoom pane (toggle fullscreen) |
| `q` | Exit btop/nvtop |

Reattach: `tmux attach -t ollama-dash`

---

## Expected Improvements After BIOS Tuning

With C-states disabled, XMP enabled, and performance governor:
- tinyllama: 189 → 220+ tok/s
- llama3.2:3b: 64 → 75+ tok/s
- deepseek-coder:6.7b: 42 → 50+ tok/s
- mistral:7b: 32 → 40+ tok/s

---

## NOT Touching (Per Your Request)
- Voltage manipulation (Vcore, VCCIO, VCCSA)
- CPU voltage override
- AVX offset

Good luck in the BIOS!
