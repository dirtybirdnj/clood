# Surgical Finish - Tasks for Future Agents

```
Docs reviewed with care
IPs and specs now align
One machine remains
```

*Created: 2025-12-12 during documentation audit*

---

## Completed This Session

- [x] Fixed IP addresses across all docs (ubuntu25 = 192.168.4.63)
- [x] Fixed MacBook Air specs (M4 32GB, IP 192.168.4.47)
- [x] Updated GPU-SETUP.md to recommend Vulkan over ROCm
- [x] Removed stale Open WebUI references
- [x] Updated network tables in all docs

---

## Remaining Task: Mac Mini Profile

The Mac Mini specs are marked as "TBD" across the docs. A future agent should gather this info.

### Mac Mini Details Needed

| Field | Command to Run | Current Value |
|-------|----------------|---------------|
| IP Address | (known) | 192.168.4.41 |
| Chip | `system_profiler SPHardwareDataType \| grep Chip` | TBD |
| Memory | `system_profiler SPHardwareDataType \| grep Memory` | TBD |
| GPU Cores | `system_profiler SPHardwareDataType \| grep "Total Number of Cores"` | TBD |
| Storage | `system_profiler SPStorageDataType \| grep "Media Name"` | TBD |
| Metal Support | `system_profiler SPDisplaysDataType \| grep Metal` | TBD |

### How to Gather

SSH into the Mac Mini and run:

```bash
ssh 192.168.4.41

# Full hardware profile
system_profiler SPHardwareDataType SPDisplaysDataType

# Or one-liner for key stats
echo "=== Mac Mini Specs ===" && \
  system_profiler SPHardwareDataType | grep -E "Model Name|Chip|Memory|Total Number of Cores" && \
  system_profiler SPDisplaysDataType | grep -E "Metal"
```

### Files to Update After Gathering

Once you have the specs, update these files:

1. `README.md` - Hardware table
2. `SERVER-GARDEN.md` - Hardware Specs table and ASCII diagram
3. `model-comparison.md` - Server Garden table
4. `WEEKEND-SURVIVAL.md` - Network Reference table (already has correct IP)
5. `ollama-tuning.md` - M4 Mac Mini section (if different from Air)

### Template for Updates

Replace `TBD` with actual values:

```markdown
| **Mac Mini** | 192.168.4.41 | M4 [X]-core GPU | [X]GB unified | M4 [X]-core CPU | [X]TB SSD |
```

---

## Other Cleanup Suggestions

### Consider Removing

| File | Reason | Action |
|------|--------|--------|
| `LAST_SESSION.md` | Ephemeral session notes | Add to .gitignore or delete |
| `GOLDEN-PATH.md` | Stale session-specific content | Update or archive |

### Consider Merging

| Files | Into | Reason |
|-------|------|--------|
| `MULTI-MACHINE.md` + `SERVER-GARDEN.md` | Single doc | Heavy overlap |
| `TIERED-MODELS.md` content | `model-comparison.md` | Duplicates model guidance |

---

## BIOS Tuning Status

The BIOS tuning for ubuntu25 is documented in `hardware/BIOS-TUNING.md`.

**Status:** User was in BIOS when this session paused. After BIOS changes:

1. Reboot and verify temps: `sensors | grep -E 'Core|Package'`
2. Verify CPU frequency: `cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq`
3. Run benchmark: `ollama run qwen2.5-coder:7b "Hello" --verbose 2>&1 | grep "eval rate"`
4. Record results in `ollama-tuning.md` benchmark table

---

## Quick Verification Commands

After any doc changes, verify consistency:

```bash
# Check all IP references are consistent
grep -r "192.168.4" *.md | grep -v surgical-finish

# Check for stale Open WebUI references
grep -r "Open WebUI\|:3000" *.md

# Check for TBD markers that need filling
grep -r "TBD" *.md
```
