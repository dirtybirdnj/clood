# Clood CLI Usage Guide

Two ways to run clood: development mode and installed mode.

---

## Development Mode

Run from the source directory during development.

### Build

```bash
cd ~/Code/clood/clood-cli
go build -o clood ./cmd/clood
```

### Run Commands

```bash
# From the clood-cli directory
./clood hosts
./clood ask "What is 2+2?"
./clood chat

# From anywhere, use full path
~/Code/clood/clood-cli/clood hosts
~/Code/clood/clood-cli/clood grep "TODO" --path ~/Code/rat-king
```

### Run in Another Project

```bash
# Navigate to target project
cd ~/Code/rat-king

# Run clood from source
~/Code/clood/clood-cli/clood ask "Explain the patterns directory"
~/Code/clood/clood-cli/clood grep "generate_.*_fill"
~/Code/clood/clood-cli/clood symbols src/patterns/
```

### Alias for Development

Add to `~/.zshrc` or `~/.bashrc`:

```bash
alias clood-dev="~/Code/clood/clood-cli/clood"
```

Then use:

```bash
cd ~/Code/rat-king
clood-dev ask "How do I add a new pattern?"
```

---

## Installed Mode

Install to system PATH for use anywhere.

### Install

```bash
cd ~/Code/clood/clood-cli
go build -o clood ./cmd/clood

# Option 1: Copy to /usr/local/bin (requires sudo)
sudo cp clood /usr/local/bin/

# Option 2: Copy to ~/bin (no sudo, add ~/bin to PATH)
mkdir -p ~/bin
cp clood ~/bin/
# Add to ~/.zshrc: export PATH="$HOME/bin:$PATH"
```

### Run Commands

```bash
# Works from anywhere
clood hosts
clood ask "What is 2+2?"

# In any project directory
cd ~/Code/rat-king
clood grep "TODO"
clood symbols src/
```

---

## Quick Reference

| Command | Purpose |
|---------|---------|
| `clood hosts` | Check Ollama host connectivity |
| `clood ask "question"` | One-shot LLM query |
| `clood chat` | Interactive chat session |
| `clood grep "pattern"` | Search codebase for pattern |
| `clood symbols <path>` | Extract code symbols |
| `clood context <files>` | Show token counts for files |
| `clood issues` | List GitHub issues (requires gh) |
| `clood handoff "summary"` | Save session context |
| `clood handoff --load` | Load last session context |

---

## Configuration

Clood looks for hosts in this order:

1. `~/.clood/config.yaml` (user config)
2. Environment variables: `CLOOD_HOST_*`
3. Default: `localhost:11434`

### Example config.yaml

```yaml
hosts:
  - name: ubuntu25
    url: http://192.168.4.63:11434
    priority: 1
  - name: mac-mini
    url: http://192.168.4.41:11434
    priority: 2
  - name: local
    url: http://localhost:11434
    priority: 3
```

---

## Workflow: Using Clood in Another Project

### Development Workflow

```bash
# Terminal 1: Keep clood source ready
cd ~/Code/clood/clood-cli
go build -o clood ./cmd/clood

# Terminal 2: Work in target project
cd ~/Code/rat-king

# Use clood for research
~/Code/clood/clood-cli/clood grep "pattern" --type rs
~/Code/clood/clood-cli/clood ask "How does the wave pattern work?"

# Check what context would be sent
~/Code/clood/clood-cli/clood context src/patterns/wave.rs src/patterns/mod.rs
```

### Production Workflow

```bash
# One-time setup
sudo cp ~/Code/clood/clood-cli/clood /usr/local/bin/

# Daily use in any project
cd ~/Code/rat-king
clood ask "Review this code" --file src/patterns/moire.rs
clood chat  # Interactive session
```

---

## Troubleshooting

### "command not found: clood"

```bash
# Check if installed
which clood

# If not found, either:
# 1. Use full path: ~/Code/clood/clood-cli/clood
# 2. Install: sudo cp ~/Code/clood/clood-cli/clood /usr/local/bin/
# 3. Add alias to shell config
```

### "connection refused" on hosts

```bash
# Check host status
clood hosts

# Test direct connection
curl http://192.168.4.63:11434/api/version

# Check if Ollama is running on target host
ssh ubuntu25 "ollama list"
```

### Rebuild after changes

```bash
cd ~/Code/clood/clood-cli
go build -o clood ./cmd/clood

# If installed, re-copy
sudo cp clood /usr/local/bin/
```

---

*"The tokens must flow."*
