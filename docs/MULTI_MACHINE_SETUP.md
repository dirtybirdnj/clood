# Multi-Machine Setup Guide

*Getting clood running across your Server Garden*

---

## Quick Start

```bash
# 1. On each machine: Install and start Ollama
curl -fsSL https://ollama.ai/install.sh | sh
OLLAMA_HOST=0.0.0.0:11434 ollama serve

# 2. On your driver machine: Discover hosts
clood discover

# 3. Copy the suggested config to ~/.config/clood/config.yaml

# 4. Verify everything works
clood hosts
clood preflight
```

---

## Prerequisites

Each machine in your garden needs:
- Ollama installed and running
- Network access from your driver machine
- Port 11434 open (or your chosen port)

---

## Step 1: Install Ollama on Each Host

### Linux (Ubuntu/Debian)
```bash
curl -fsSL https://ollama.ai/install.sh | sh
```

### macOS
```bash
brew install ollama
```

### Windows
Download from https://ollama.ai/download

---

## Step 2: Configure Network Binding

By default, Ollama only listens on localhost. To accept remote connections:

### Linux (Systemd)
```bash
# Edit the service file
sudo systemctl edit ollama

# Add these lines:
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"

# Restart
sudo systemctl restart ollama
```

### macOS (launchd)
```bash
# Set environment and restart
launchctl setenv OLLAMA_HOST 0.0.0.0:11434
brew services restart ollama
```

### Windows
Set environment variable `OLLAMA_HOST=0.0.0.0:11434` in System Properties.

---

## Step 3: Open Firewall

### Linux (UFW)
```bash
sudo ufw allow 11434/tcp
```

### macOS
System Preferences → Security & Privacy → Firewall → Allow ollama

### Windows
Windows Defender Firewall → Allow an app → Add ollama.exe

---

## Step 4: Discover Your Garden

From your driver machine:

```bash
clood discover
```

Output:
```
Network Discovery

  Local IP: 192.168.4.47
  Scanning: 192.168.4.0/24
  Port: 11434

Found 2 Ollama Instance(s)

  ● 192.168.4.41:11434 (mac-mini)
    Version: 0.13.2
    Latency: 16ms
    Models: mistral:7b, qwen2.5-coder:14b, ...

  ● 192.168.4.64:11434 (ubuntu25)
    Version: 0.13.2
    Latency: 33ms
    Models: deepseek-r1:14b, llama3.1:8b, ...

Suggested Config
(copy this to ~/.config/clood/config.yaml)
```

---

## Step 5: Configure clood

Create `~/.config/clood/config.yaml`:

```yaml
hosts:
  - name: mac-mini       # Friendly name
    url: http://192.168.4.41:11434
    priority: 1          # Lower = higher priority
    enabled: true

  - name: ubuntu25
    url: http://192.168.4.64:11434
    priority: 1
    enabled: true

  - name: localhost
    url: http://localhost:11434
    priority: 2
    enabled: true

tiers:
  fast:
    model: qwen2.5-coder:3b    # Quick responses
  deep:
    model: qwen2.5-coder:7b    # Deeper analysis
```

---

## Step 6: Pull Models

Pull models appropriate for each machine's hardware:

```bash
# On your high-VRAM machine (GPU)
ollama pull deepseek-r1:14b
ollama pull llama3.1:8b
ollama pull qwen2.5-coder:7b

# On your Mac (Apple Silicon)
ollama pull qwen2.5-coder:14b
ollama pull mistral:7b

# On any machine (lightweight)
ollama pull qwen2.5-coder:3b
ollama pull tinyllama
```

---

## Step 7: Verify Setup

```bash
# Check all hosts are online
clood hosts

# Full health check
clood health

# Check available models
clood models

# Check storage usage
clood models --storage

# Run a test query
clood ask "What is 2+2?"

# Run a catfight across hosts
clood catfight --hosts "ubuntu25,mac-mini" "Write hello world in Go"
```

---

## Tier Strategy

Match models to hardware:

| Hardware | Recommended Models | Role |
|----------|-------------------|------|
| GPU (12GB+ VRAM) | deepseek-r1:14b, llama3.1:8b | Heavy analysis |
| Apple Silicon (M1/M2/M4) | qwen2.5-coder:7b-14b, mistral:7b | All-around |
| CPU Only | qwen2.5-coder:3b, tinyllama | Quick tasks |

---

## Troubleshooting

### Host shows offline
```bash
# Check if Ollama is running on the remote host
ssh user@hostname "curl http://localhost:11434/api/version"

# Check network binding
ssh user@hostname "curl http://0.0.0.0:11434/api/version"

# Check from your machine
curl http://<host-ip>:11434/api/version
```

### Connection refused
- Ollama not running: `ollama serve`
- Not bound to network: Set `OLLAMA_HOST=0.0.0.0:11434`
- Firewall blocking: Open port 11434

### Slow responses
- Check `clood hosts` for latency
- Try closer host: Adjust priorities in config
- Model too large: Use smaller model for quick tasks

### Model not found
```bash
# Check what models are available on each host
clood models

# Pull missing model on specific host
ssh user@hostname "ollama pull qwen2.5-coder:3b"
```

---

## Example Garden Setup

```
┌─────────────────────────────────────────────────────────────────┐
│                    THE SERVER GARDEN                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  MacBook (Driver)                                               │
│  ├── clood CLI                                                  │
│  └── Coordinates the garden                                     │
│           │                                                     │
│           ├───────────────┬───────────────┐                     │
│           │               │               │                     │
│           ▼               ▼               ▼                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  ubuntu25   │  │  mac-mini   │  │  localhost  │             │
│  │  (Iron Keep)│  │ (Sentinel)  │  │  (Jade)     │             │
│  │             │  │             │  │             │             │
│  │  RX 590 GPU │  │  M4 Apple   │  │  CPU Only   │             │
│  │  10 models  │  │  14 models  │  │  2 models   │             │
│  │  Priority 1 │  │  Priority 1 │  │  Priority 2 │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Commands Reference

| Command | Purpose |
|---------|---------|
| `clood discover` | Find Ollama instances on network |
| `clood hosts` | Show host status and latency |
| `clood models` | List all available models |
| `clood models --storage` | Show disk usage by host |
| `clood health` | Full system health check |
| `clood preflight` | Quick pre-task check |
| `clood ask -H ubuntu25 "prompt"` | Query specific host |
| `clood catfight --hosts "a,b"` | Compare across hosts |

---

*The garden grows,*
*Multiple trees bear fruit—*
*Distributed strength.*
