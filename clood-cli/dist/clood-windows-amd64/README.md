# clood Quickstart - Windows

Lightning in a Bottle - Local LLM Infrastructure

## Quick Start

### Option 1: PowerShell (Recommended)

```powershell
# 1. Install Ollama from https://ollama.ai/download

# 2. Start Ollama (runs in system tray)

# 3. Run setup wizard
.\clood.exe setup

# 4. Verify installation
.\clood.exe verify
```

### Option 2: Command Prompt

```cmd
REM 1. Install Ollama from https://ollama.ai/download

REM 2. Start Ollama (runs in system tray)

REM 3. Run setup wizard
clood.exe setup

REM 4. Verify installation
clood.exe verify
```

## NVIDIA GPU Users

clood automatically detects your GPU via nvidia-smi. Ensure you have:
- NVIDIA GeForce/RTX drivers installed
- nvidia-smi in your PATH (usually installed with drivers)

Check GPU detection:
```powershell
.\clood.exe system
```

## Configuration Location

Config file: `%APPDATA%\clood\config.yaml`

## First Commands

```powershell
# Check your system
.\clood.exe doctor

# Run a catfight (model comparison)
.\clood.exe catfight "Write a hello world in Go"

# Ask a question
.\clood.exe ask "What is the best way to handle errors in Go?"
```

## Troubleshooting

### "clood.exe is not recognized"
Add the clood directory to your PATH, or run from the directory containing clood.exe.

### "Cannot connect to Ollama"
1. Ensure Ollama is running (check system tray)
2. Try: `ollama serve` in a terminal

### "No GPU detected"
1. Ensure NVIDIA drivers are installed
2. Check: `nvidia-smi` in command prompt

## Need Help?

- `clood.exe --help` - Full documentation
- `clood.exe doctor` - System diagnostics
- https://github.com/dirtybirdnj/clood
