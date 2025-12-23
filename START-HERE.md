# Start Here - clood Setup

Get clood running on your machine.

## Prerequisites

- Go 1.21+ (to build clood)
- Ollama installed and running
- Optional: Multiple machines with Ollama for distributed inference

---

## Phase 1: Install Ollama

### Linux
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

### macOS
```bash
brew install ollama
```

Verify: `ollama --version`

---

## Phase 2: Pull Models

```bash
# Essential - pick based on your VRAM
ollama pull qwen2.5-coder:3b    # 2.5GB - fast, good for routing
ollama pull qwen2.5-coder:7b    # 6GB - balanced
ollama pull llama3.1:8b         # 6GB - general purpose

# Reasoning (if you have 8GB+ VRAM)
ollama pull deepseek-r1:14b     # 14GB - deep reasoning

# Tool calling
ollama pull llama3-groq-tool-use:8b  # 6GB - optimized for tools
```

Verify: `ollama list`

---

## Phase 3: Build clood

```bash
cd ~/Code/clood/clood-cli
go build -o clood ./cmd/clood
./clood --version
```

---

## Phase 4: Configure clood

```bash
# Run setup wizard
./clood setup

# Or manually create config
mkdir -p ~/.config/clood
cat > ~/.config/clood/config.yaml << 'EOF'
hosts:
  - name: localhost
    url: http://localhost:11434
    priority: 1
    enabled: true
tiers:
  fast:
    model: qwen2.5-coder:3b
  deep:
    model: qwen2.5-coder:7b
  analysis:
    model: deepseek-r1:14b
EOF
```

---

## Phase 5: Verify Setup

```bash
# Check what's available
./clood preflight

# Check hosts
./clood hosts

# Test a query
./clood ask "Hello, what model are you?"

# Test local tools
./clood tree .
./clood grep "func main"
```

---

## Phase 6: Add to Claude Code (Optional)

Add clood as an MCP server in Claude Code settings:

```json
{
  "mcpServers": {
    "clood": {
      "command": "/path/to/clood",
      "args": ["mcp"]
    }
  }
}
```

---

## Multi-Machine Setup

If you have multiple machines with Ollama:

```yaml
# ~/.config/clood/config.yaml
hosts:
  - name: mac-mini
    url: http://localhost:11434
    priority: 2
    enabled: true
  - name: ubuntu25
    url: http://192.168.4.64:11434
    priority: 1
    enabled: true
  - name: mac-laptop
    url: http://192.168.4.47:11434
    priority: 3
    enabled: true
```

On remote machines, expose Ollama to the network:
```bash
# Set in environment or systemd
OLLAMA_HOST=0.0.0.0 ollama serve
```

---

## Troubleshooting

### Ollama not responding
```bash
# Check if running
curl http://localhost:11434/api/tags

# Start it
ollama serve
```

### Remote host not connecting
```bash
# Test connectivity
curl http://192.168.4.64:11434/api/tags

# Check if Ollama is bound to 0.0.0.0
ssh ubuntu25 'ss -tlnp | grep 11434'
```

### Model too slow
```bash
# Check GPU usage
./clood system

# Use a smaller model
./clood ask "question" --model qwen2.5-coder:3b
```

---

## Next Steps

1. Read [clood-cli/docs/USAGE_GUIDE.md](clood-cli/docs/USAGE_GUIDE.md) for full documentation
2. Explore `clood help` for all commands
3. Try `clood preflight` at the start of each session
