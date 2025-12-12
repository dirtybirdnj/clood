# Tmux Workstation Display Wishlist

A dream layout for monitoring Ollama and system performance on ubuntu25.

---

## Current Layout (Implemented!)

```
┌─────────────────────────────┬─────────────────────────────┐
│          btop               │          nvtop              │
│   CPU/RAM/Disk/Network      │     GPU/VRAM utilization    │
├─────────────────────────────┼─────────────────────────────┤
│  journalctl -u ollama -f    │   sensors (temps/fans)      │
│  (token flow, errors)       │   (CPU/GPU temperatures)    │
├─────────────────────────────┴─────────────────────────────┤
│  watch -n1 'curl -s localhost:11434/api/ps | jq'         │
│          (loaded models, VRAM per model)                  │
└───────────────────────────────────────────────────────────┘
```

**Status:** ✅ Implemented in `scripts/ollama-dashboard.sh`

---

## TUI Tools to Install

| Tool | Purpose | Install Command |
|------|---------|-----------------|
| **btop** | CPU/RAM/Disk/Network (already have) | `apt install btop` |
| **nvtop** | GPU monitoring (already have) | `apt install nvtop` |
| **s-tui** | CPU freq/temp/power + stress mode | `pip install s-tui` |
| **glances** | All-in-one + web UI option | `pip install glances` |
| **below** | Facebook's cgroup monitor | `apt install below` |
| **dstat** | Real-time columnar stats | `apt install dstat` |

---

## Temperature Monitoring

### Required Package

```bash
sudo apt install lm-sensors
sudo sensors-detect  # Run once to detect sensors, say YES to all
```

### What Can Be Monitored

| Component | Tool | Command |
|-----------|------|---------|
| CPU temp | lm-sensors | `sensors` |
| CPU per-core | lm-sensors | `sensors coretemp-*` |
| GPU temp (AMD) | nvtop, radeontop | `cat /sys/class/drm/card*/device/hwmon/hwmon*/temp1_input` |
| Motherboard/Chipset | lm-sensors | `sensors` (shows all detected) |
| NVMe/SSD | smartctl | `sudo smartctl -a /dev/nvme0n1 \| grep Temp` |
| RAM | Usually not exposed | (rarely available on consumer boards) |

### Quick Temperature Commands

```bash
# All sensors at once
sensors

# Watch temps live (updates every 2 sec)
watch -n 2 sensors

# Just CPU package temp
sensors | grep -E "Package|Core"

# GPU temp (AMD)
cat /sys/class/drm/card1/device/hwmon/hwmon*/temp1_input | awk '{print $1/1000"°C"}'

# NVMe temp
sudo nvme smart-log /dev/nvme0n1 | grep temperature
```

### TUI Tools with Temperature

| Tool | Shows Temps? | Notes |
|------|--------------|-------|
| **btop** | ✅ CPU temps | Built-in, shows per-core |
| **nvtop** | ✅ GPU temp | Shows AMD/NVIDIA temps |
| **s-tui** | ✅ CPU temp + freq | Best for CPU stress testing |
| **glances** | ✅ All sensors | Most comprehensive |
| **sensors** | ✅ Everything | CLI, use with `watch` |

### Adding Temps to Dashboard

For a temp-focused dashboard pane, add:

```bash
# In ollama-dashboard.sh, add a pane with:
watch -n 2 'sensors | grep -E "Package|Core|temp1|fan"'
```

### Temperature Thresholds (i7-8086K)

| State | Temp | Action |
|-------|------|--------|
| Idle | 30-40°C | Normal |
| Light load | 40-60°C | Normal |
| Heavy load | 60-80°C | Normal, monitor |
| Throttling | 80-100°C | CPU slows down to protect itself |
| Critical | >100°C | Auto-shutdown |

### GPU Temperature (RX 590)

| State | Temp | Notes |
|-------|------|-------|
| Idle | 35-45°C | Normal |
| Load | 60-75°C | Normal for Vulkan inference |
| Hot | 75-85°C | Consider fan curve adjustment |
| Throttle | >85°C | GPU will clock down |

---

## LLM-Specific Watch Commands

```bash
# Live Ollama logs (most useful for debugging)
journalctl -u ollama -f

# Model loading status with VRAM
watch -n 1 'curl -s http://localhost:11434/api/ps | jq'

# Thread count of running Ollama process
watch -n 1 'cat /proc/$(pgrep -x ollama)/status | grep Threads'

# Quick token rate check
ollama run tinyllama "hi" --verbose 2>&1 | tail -5
```

---

## Tmux Navigation Cheatsheet

All tmux commands start with `Ctrl+b` (the prefix), then release and press the next key:

| Keys | Action |
|------|--------|
| `Ctrl+b` then `d` | **Detach** - exit tmux but keep session running |
| `Ctrl+b` then `arrow keys` | Move between panes |
| `Ctrl+b` then `z` | **Zoom** - make current pane fullscreen (toggle) |
| `Ctrl+b` then `x` | Kill current pane (with confirmation) |
| `Ctrl+b` then `[` | Scroll mode (use arrows/PgUp/PgDn, `q` to exit) |
| `Ctrl+b` then `?` | Show all keybindings |

**Quick tips:**
- `tmux attach -t ollama-dash` - reattach to detached session
- `tmux kill-session -t ollama-dash` - destroy session completely
- Inside btop/nvtop: press `q` to quit that tool

---

## Dashboard Script

See `scripts/ollama-dashboard.sh` for the implementation.

```bash
# Start the dashboard
./scripts/ollama-dashboard.sh

# Reattach to existing session
tmux attach -t ollama-dash

# Kill the session
./scripts/ollama-dashboard.sh -k
```

---

## Nice-to-Haves

- [ ] Custom ollama stats TUI (Rust? Go?)
- [ ] Token-per-second graph over time
- [ ] Alert when VRAM > 90%
- [ ] Integration with Grafana for historical data
- [ ] Prometheus exporter for Ollama metrics

---

## Headless Boot Mode (Reduce GUI Overhead)

Goal: Boot ubuntu25 into text mode with tmux dashboard auto-starting, no GUI rendering overhead.

### Understanding Systemd Targets

Ubuntu uses **systemd targets** to determine boot mode:

| Target | What It Does | GPU Usage |
|--------|--------------|-----------|
| `graphical.target` | Full desktop (GNOME/KDE) | ~200-500MB VRAM for compositing |
| `multi-user.target` | Text mode, no GUI, all services | 0 - GPU free for Ollama |

### Option 1: Boot to Multi-User Target (No GUI)

```bash
# Set default to text mode (no desktop)
sudo systemctl set-default multi-user.target

# To temporarily boot with GUI (one-time)
sudo systemctl start gdm

# To revert to GUI boot permanently
sudo systemctl set-default graphical.target
```

### Switching Without Reboot

```bash
# Drop from desktop to CLI (kills GNOME, frees GPU)
sudo systemctl isolate multi-user.target

# Start desktop from CLI
sudo systemctl isolate graphical.target
# or just: sudo systemctl start gdm
```

### One-Time Boot Mode from GRUB

1. Hold `Shift` during boot → GRUB menu appears
2. Press `e` to edit the selected entry
3. Find the line starting with `linux`
4. Add `3` or `systemd.unit=multi-user.target` at the end
5. Press `Ctrl+X` to boot with that option

### Recommended Workflow

Keep `graphical.target` as default (for when you're at the workstation):

1. **For max performance (remote):**
   - SSH in from another machine
   - Run `sudo systemctl isolate multi-user.target`
   - Run `~/bin/ollama-dashboard.sh`
   - GPU fully dedicated to Ollama

2. **For local work:**
   - Use desktop normally
   - Open terminal, run `~/bin/ollama-dashboard.sh`
   - Slight overhead from compositor but convenient

### Option 2: Auto-Start Tmux Dashboard on Login

Create `~/.bash_profile` or add to `~/.bashrc`:

```bash
# Auto-start ollama dashboard if not already in tmux
if [[ -z "$TMUX" ]] && [[ "$TERM" != "screen" ]]; then
    if tmux has-session -t ollama-dash 2>/dev/null; then
        tmux attach -t ollama-dash
    else
        ~/bin/ollama-dashboard.sh
    fi
fi
```

### Option 3: Systemd User Service

Create `~/.config/systemd/user/ollama-dashboard.service`:

```ini
[Unit]
Description=Ollama Monitoring Dashboard
After=network.target

[Service]
Type=forking
ExecStart=/usr/bin/tmux new-session -d -s ollama-dash '~/.local/bin/ollama-dashboard.sh'
ExecStop=/usr/bin/tmux kill-session -t ollama-dash

[Install]
WantedBy=default.target
```

Enable with:
```bash
systemctl --user enable ollama-dashboard
systemctl --user start ollama-dashboard
```

### Option 4: Getty Autologin + Tmux

Edit `/etc/systemd/system/getty@tty1.service.d/autologin.conf`:

```ini
[Service]
ExecStart=
ExecStart=-/sbin/agetty --autologin mgilbert --noclear %I $TERM
```

Combined with Option 2, this auto-logs in and starts the dashboard.

### Performance Notes

- GUI (GNOME/KDE) uses ~200-500MB RAM and some GPU for compositing
- Text mode frees GPU entirely for Ollama
- SSH access still works regardless of boot mode
- Can always `sudo systemctl start gdm` if you need GUI temporarily

---

*Workstation dreams for ubuntu25 - i7-8086K + RX 590*
