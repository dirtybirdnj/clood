# clood Quickstart - macOS

Lightning in a Bottle - Local LLM Infrastructure

## Quick Start

```bash
# 1. Install Ollama
brew install ollama

# 2. Start Ollama
ollama serve &

# 3. Run setup wizard
./clood setup

# 4. Verify installation
./clood verify
```

## Apple Silicon Users

Your unified memory is automatically detected. clood will configure tiers based on your available RAM:

| Total RAM | Available for Models | Recommended Tier |
|-----------|---------------------|------------------|
| 8GB | ~6GB | Budget (3b models) |
| 16GB | ~12GB | Mid-range (7b models) |
| 24GB+ | ~18GB+ | High-end (14b+ models) |

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
