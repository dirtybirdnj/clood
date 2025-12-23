# clood Quickstart - Linux

Lightning in a Bottle - Local LLM Infrastructure

## Quick Start

```bash
# 1. Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# 2. Start Ollama
ollama serve &

# 3. Run setup wizard
./clood setup

# 4. Verify installation
./clood verify
```

## NVIDIA GPU Users

clood automatically detects your GPU via nvidia-smi. Ensure you have:
- NVIDIA drivers installed
- CUDA toolkit (optional but recommended)

Check GPU detection:
```bash
./clood system
```

## AMD GPU Users (ROCm)

For AMD GPUs, ensure ROCm is installed and Ollama is built with ROCm support.

## First Commands

```bash
# Check your system
clood doctor

# Run a catfight (model comparison)
clood catfight "Write a hello world in Go"

# Ask a question
clood ask "What is the best way to handle errors in Go?"
```

## Need Help?

- `clood --help` - Full documentation
- `clood doctor` - System diagnostics
- https://github.com/dirtybirdnj/clood
