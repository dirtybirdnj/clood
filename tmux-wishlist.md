# Tmux Workstation Display Wishlist

A dream layout for monitoring Ollama and system performance on ubuntu25.

---

## Ideal Layout

```
┌─────────────────────────────┬─────────────────────────────┐
│          btop               │          nvtop              │
│   CPU/RAM/Disk/Network      │     GPU/VRAM utilization    │
│                             │                             │
├─────────────────────────────┴─────────────────────────────┤
│                  journalctl -u ollama -f                  │
│          (live token flow, layer loading, errors)         │
├───────────────────────────────────────────────────────────┤
│  watch -n1 'curl -s localhost:11434/api/ps | jq'         │
│          (loaded models, VRAM per model)                  │
└───────────────────────────────────────────────────────────┘
```

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

## Tmux Session Script (Future)

```bash
#!/bin/bash
# ollama-dashboard.sh - launch monitoring dashboard

SESSION="ollama-dash"

tmux new-session -d -s $SESSION -n 'dashboard'

# Top left: btop
tmux send-keys 'btop' C-m

# Top right: nvtop
tmux split-window -h
tmux send-keys 'nvtop' C-m

# Bottom: Ollama logs
tmux split-window -v -t 0
tmux send-keys 'journalctl -u ollama -f' C-m

# Bottom right: model watcher
tmux split-window -v -t 1
tmux send-keys "watch -n 1 'curl -s http://localhost:11434/api/ps | jq'" C-m

tmux attach -t $SESSION
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

### Option 1: Boot to Multi-User Target (No GUI)

```bash
# Set default to text mode (no desktop)
sudo systemctl set-default multi-user.target

# To temporarily boot with GUI (one-time)
sudo systemctl start gdm

# To revert to GUI boot permanently
sudo systemctl set-default graphical.target
```

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
