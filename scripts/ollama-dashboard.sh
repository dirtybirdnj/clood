#!/bin/bash
# ollama-dashboard.sh - Tmux monitoring dashboard for Ollama workstation
#
# Layout:
# ┌─────────────────────────────┬─────────────────────────────┐
# │          btop               │          nvtop              │
# │   CPU/RAM/Disk/Network      │     GPU/VRAM utilization    │
# ├─────────────────────────────┴─────────────────────────────┤
# │                  journalctl -u ollama -f                  │
# ├───────────────────────────────────────────────────────────┤
# │  watch -n1 'curl -s localhost:11434/api/ps | jq'         │
# └───────────────────────────────────────────────────────────┘
#
# Usage:
#   ./ollama-dashboard.sh          # Start new session
#   tmux attach -t ollama-dash     # Reattach to existing
#
# Install location on ubuntu25:
#   cp scripts/ollama-dashboard.sh ~/bin/
#   chmod +x ~/bin/ollama-dashboard.sh

SESSION="ollama-dash"

# Kill existing session if requested
if [[ "$1" == "-k" ]] || [[ "$1" == "--kill" ]]; then
    tmux kill-session -t $SESSION 2>/dev/null
    echo "Killed session: $SESSION"
    exit 0
fi

# Check if session already exists
if tmux has-session -t $SESSION 2>/dev/null; then
    echo "Session '$SESSION' already exists. Attaching..."
    tmux attach -t $SESSION
    exit 0
fi

# Create new session with btop in first pane
tmux new-session -d -s $SESSION -n 'dashboard'
tmux send-keys 'btop' C-m

# Top right: nvtop (GPU monitoring)
tmux split-window -h
tmux send-keys 'nvtop' C-m

# Bottom left: Ollama logs
tmux select-pane -t 0
tmux split-window -v
tmux send-keys 'journalctl -u ollama -f' C-m

# Bottom right: Model status watcher
tmux select-pane -t 2
tmux split-window -v
tmux send-keys "watch -n 1 'curl -s http://localhost:11434/api/ps | jq'" C-m

# Select the btop pane as default
tmux select-pane -t 0

# Attach to the session
tmux attach -t $SESSION
