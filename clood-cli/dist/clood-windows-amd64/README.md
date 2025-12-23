# clood for Windows

Lightning in a Bottle - Local LLM Infrastructure

## Quick Start

```powershell
# 1. Install Ollama from https://ollama.ai/download

# 2. Start Ollama (runs in system tray)

# 3. Pull at least one model
ollama pull qwen2.5-coder:7b

# 4. Run setup wizard
.\clood.exe setup

# 5. Verify installation
.\clood.exe doctor
```

## Single-Machine vs Multi-Machine

**Single Machine (Most Users)**
- Just you and your GPU
- Only `localhost` in config
- All commands work locally
- This is the default setup

**Multi-Machine (Server Garden)**
- Multiple networked machines
- Distributed model inference
- Requires additional configuration
- See full docs for details

## NVIDIA GPU Detection

clood automatically detects your GPU via `nvidia-smi`. Ensure you have:
- NVIDIA GeForce/RTX drivers installed
- `nvidia-smi` in your PATH (usually installed with drivers)

Check GPU detection:
```powershell
.\clood.exe system
```

## Configuration

Config file: `%APPDATA%\clood\config.yaml`

Default (single-machine):
```yaml
hosts:
  - name: localhost
    url: http://localhost:11434

tiers:
  fast:
    model: qwen2.5-coder:3b
  deep:
    model: qwen2.5-coder:7b
```

## Essential Commands

```powershell
# System diagnostics
.\clood.exe doctor

# Hardware info
.\clood.exe system

# Available models
.\clood.exe models

# Ask a question
.\clood.exe ask "Explain recursion"

# Compare models
.\clood.exe catfight "Write hello world in Go"
```

## Troubleshooting

### "clood.exe is not recognized"
Add the clood directory to your PATH, or run from the directory containing clood.exe.

### "Cannot connect to Ollama"
1. Ensure Ollama is running (check system tray)
2. Try: `ollama serve` in a terminal
3. Test: `curl http://localhost:11434/api/version`

### "No GPU detected"
1. Ensure NVIDIA drivers are installed
2. Check: `nvidia-smi` in command prompt
3. Restart Ollama after driver updates

## Need Help?

- `.\clood.exe --help` - Full command list
- `.\clood.exe doctor -v` - Detailed diagnostics
- https://github.com/dirtybirdnj/clood
