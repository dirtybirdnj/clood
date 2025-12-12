# BIOS Tuning Guide - Z390 AORUS PRO WIFI

```
Silicon sleeps light
Tweaker menu holds the keys
Wake the beast within

Thermal guardian
Eighty degrees is the wall
Respect the limit

Six cores united
Power limits cast aside
Golden path awaits
```

**Board:** Gigabyte Z390 AORUS PRO WIFI-CF (BIOS F11)
**CPU:** Intel i7-8086K (6C/12T, 4.0 GHz base, 5.0 GHz turbo)
**Cooling:** Stock/Modest (conservative settings recommended)

---

## Thermal Safety Zones

Before touching anything, understand your limits:

| Zone | Temperature | Status | Action |
|------|-------------|--------|--------|
| **GREEN** | < 70°C | Optimal | All good |
| **YELLOW** | 70-80°C | Warm | Monitor closely |
| **ORANGE** | 80-85°C | Hot | Consider backing off |
| **RED** | 85-95°C | Danger | Reduce clocks/voltage |
| **CRITICAL** | > 95°C | Emergency | Revert to defaults immediately |

**TjMax (Thermal Junction Max):** 100°C - CPU will throttle or shutdown

### Monitor Command (Post-Reboot)
```bash
watch -n1 "sensors | grep -E 'Core|Package'"
```

---

## BIOS Navigation

**Enter BIOS:** Press `DEL` during POST

**Main Menu:**
- Favorites
- **Tweaker** ← This is where we work
- Settings
- System Info
- Boot
- Save & Exit

---

## Settings Checklist

Work through these in order. **Save and test after each major section.**

### Section 1: XMP (Memory Profile)

**Path:** `Tweaker` → `Advanced Memory Settings`

| Setting | Value | Risk Level |
|---------|-------|------------|
| Extreme Memory Profile (X.M.P.) | **Profile 1** | LOW |

This enables your RAM's rated speed. Almost zero risk.

---

### Section 2: Turbo Boost (Unlock Full Speed)

**Path:** `Tweaker` → `Advanced Frequency Settings` → `Advanced CPU Core Settings`

| Setting | Value | Risk Level |
|---------|-------|------------|
| Intel Turbo Boost | **Enabled** | LOW |
| Turbo Boost Short Power Max | **Disabled** | LOW |
| Turbo Boost Power Max | **Disabled** | LOW |

This removes Intel's artificial power limits. The CPU can now sustain turbo speeds.

---

### Section 3: C-States (Disable Sleep States)

**Path:** `Tweaker` → `Advanced Frequency Settings` → `Advanced CPU Core Settings`

| Setting | Value | Risk Level |
|---------|-------|------------|
| CPU Enhanced Halt (C1E) | **Disabled** | LOW |
| C3 State Support | **Disabled** | LOW |
| C6/C7 State Support | **Disabled** | LOW |
| Package C State Limit | **C0/C1** | LOW |

Prevents CPU from entering deep sleep. Faster wake = faster inference.

---

### Section 4: SpeedStep (Disable Frequency Scaling)

**Path:** `Tweaker` → `Advanced Frequency Settings` → `Advanced CPU Core Settings`

| Setting | Value | Risk Level |
|---------|-------|------------|
| Intel Speed Shift Technology | **Disabled** | LOW |
| Enhanced Intel SpeedStep (EIST) | **Disabled** | LOW |

CPU stays at high frequency instead of ramping up/down.

---

### Section 5: CPU Clock Ratio (The Big One)

**Path:** `Tweaker` → `Advanced Frequency Settings`

#### CONSERVATIVE (Recommended for Modest Cooling)

| Setting | Value | Frequency | Risk Level |
|---------|-------|-----------|------------|
| CPU Clock Ratio | **48** | 4.8 GHz | MEDIUM |
| Ring Ratio | **43** | 4.3 GHz | LOW |

#### AGGRESSIVE (Only with Good Cooling)

| Setting | Value | Frequency | Risk Level |
|---------|-------|-----------|------------|
| CPU Clock Ratio | **50** | 5.0 GHz | HIGH |
| Ring Ratio | **47** | 4.7 GHz | MEDIUM |

**With modest cooling: STICK TO CONSERVATIVE.** The i7-8086K runs hot.

---

### Section 6: Load Line Calibration (LLC)

**Path:** `Tweaker` → `Advanced Voltage Settings` → `CPU Core Voltage Control`

| Setting | Value | Risk Level |
|---------|-------|------------|
| CPU Vcore Loadline Calibration | **Turbo** | MEDIUM |

Scale: Low → Normal → Standard → High → **Turbo** → Extreme → Ultra Extreme

LLC prevents voltage droop under load. Turbo is safe; don't go higher without better cooling.

---

## Quick Reference Card

Print this or keep it on your phone:

```
TWEAKER → Advanced Memory Settings:
  X.M.P.: Profile 1

TWEAKER → Advanced Frequency Settings:
  CPU Clock Ratio: 48 (conservative) or 50 (aggressive)
  Ring Ratio: 43

TWEAKER → Advanced Frequency Settings → Advanced CPU Core Settings:
  Intel Turbo Boost: Enabled
  Turbo Boost Short Power Max: Disabled
  Turbo Boost Power Max: Disabled
  CPU Enhanced Halt (C1E): Disabled
  C3/C6/C7 State Support: Disabled
  Package C State Limit: C0/C1
  Intel Speed Shift Technology: Disabled
  Enhanced Intel SpeedStep (EIST): Disabled

TWEAKER → Advanced Voltage Settings → CPU Core Voltage Control:
  CPU Vcore Loadline Calibration: Turbo
```

---

## What We're NOT Touching

These require more expertise and better cooling:

- Vcore (CPU voltage) - Auto is fine
- VCCIO / VCCSA voltages
- AVX offset
- Anything that says "override"

---

## Post-Reboot Verification

After saving BIOS and booting into Ubuntu:

```bash
# Check CPU frequency (should show ~4800000 for 4.8GHz)
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq

# Watch frequencies real-time
watch -n1 "cat /sys/devices/system/cpu/cpu*/cpufreq/scaling_cur_freq | sort -u"

# Check temperatures (IMPORTANT!)
watch -n1 "sensors | grep -E 'Core|Package'"

# Set governor to performance
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Quick benchmark
ollama run qwen2.5-coder:7b "Hello" --verbose 2>&1 | grep "eval rate"
```

---

## If Things Go Wrong

### System Won't Boot
1. Power off completely (hold power button 10 seconds)
2. Clear CMOS: Remove battery for 30 seconds OR use CLR_CMOS jumper
3. BIOS resets to defaults
4. Start over with conservative settings

### Temperatures Too High
1. Reduce CPU Clock Ratio from 48 to 46 (4.6 GHz)
2. Change LLC from Turbo to High
3. Re-enable C-states if needed for thermal headroom

### System Unstable
1. Could be memory - try disabling XMP first
2. Could be CPU - reduce clock ratio
3. Could be LLC - try one level lower

---

## Expected Results

With conservative settings (4.8 GHz all-core, LLC Turbo, XMP enabled):

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Base Clock | 4.0 GHz | 4.8 GHz | +20% |
| Turbo Sustained | Limited | Unlimited | +++ |
| Memory Speed | 2133 MHz | Rated (3200?) | +50%? |
| Inference Speed | Baseline | +15-25% | TBD |

---

## The Golden Path Checklist

- [ ] XMP enabled (memory at rated speed)
- [ ] Turbo Boost enabled, power limits disabled
- [ ] C-states disabled
- [ ] SpeedStep/Speed Shift disabled
- [ ] CPU ratio set to 48 (4.8 GHz)
- [ ] Ring ratio set to 43
- [ ] LLC set to Turbo
- [ ] Temperatures verified < 80°C under load
- [ ] Benchmark run, results recorded

---

*Last updated: 2025-12-12*
