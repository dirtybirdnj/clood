# Start Here - Clood Setup Checklist

Complete these steps in order to get clood running on your machine.

## Prerequisites

- [ ] Linux (Ubuntu 24/25) or macOS (M-series recommended)
- [ ] 8GB+ RAM (16GB+ recommended)
- [ ] GPU with 6GB+ VRAM for decent performance (optional but recommended)

---

## Phase 1: Core Infrastructure

### 1.1 Install Ollama

```bash
# Linux
curl -fsSL https://ollama.com/install.sh | sh

# macOS
brew install ollama
```

Verify: `ollama --version`

### 1.2 Install Docker (for SearXNG)

```bash
# Linux
sudo apt install docker.io docker-compose-v2
sudo usermod -aG docker $USER
# Log out and back in

# macOS
brew install --cask docker
# Open Docker Desktop
```

Verify: `docker ps`

### 1.3 Install Node.js (for MCP servers)

```bash
# Linux
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# macOS
brew install node
```

Verify: `node --version && npm --version`

### 1.4 Install Crush CLI

```bash
# macOS
brew install charmbracelet/tap/crush

# Linux - download from GitHub releases
# https://github.com/charmbracelet/crush/releases
# Example for amd64:
wget https://github.com/charmbracelet/crush/releases/latest/download/crush_Linux_x86_64.tar.gz
tar xzf crush_Linux_x86_64.tar.gz
sudo mv crush /usr/local/bin/
```

Verify: `crush --version`

---

## Phase 2: GPU Setup (Optional but Recommended)

### AMD GPU (RX 500/5000/6000/7000 series)

See [GPU-SETUP.md](GPU-SETUP.md) for detailed instructions.

**Quick check:**
```bash
# Check if GPU is detected
ollama run tinyllama "hello" --verbose 2>&1 | grep -i "gpu\|layer"
```

### NVIDIA GPU

Usually works out of the box. Ensure CUDA drivers installed.

### Apple Silicon

Works automatically via Metal. No setup needed.

---

## Phase 3: Pull Models

Run these manually in your terminal (not via Claude CLI):

```bash
# Essential - Tool calling (pick one)
ollama pull llama3-groq-tool-use:8b   # Best for MCP tools
ollama pull qwen3:8b                   # Alternative with tools

# Essential - Coding
ollama pull qwen2.5-coder:7b          # Best coding model for 8GB VRAM

# Fast/Small
ollama pull tinyllama                  # Router/classifier (~150 tok/s)
ollama pull qwen2.5-coder:3b          # Fast coding (~64 tok/s)

# Embeddings (for future RAG)
ollama pull nomic-embed-text
```

Verify: `ollama list`

---

## Phase 4: Start Services

### 4.1 Start SearXNG (Web Search)

```bash
cd ~/Code/clood/infrastructure
docker compose up -d searxng
```

Verify: `curl http://localhost:8888/search?q=test&format=json | head`

### 4.2 Start Ollama (if not running)

```bash
# Linux (systemd)
sudo systemctl start ollama
sudo systemctl enable ollama

# macOS
ollama serve &
```

Verify: `curl http://localhost:11434/api/tags`

---

## Phase 5: Configure Crush

### 5.1 Copy config template

```bash
mkdir -p ~/.config/crush
cp ~/Code/clood/infrastructure/configs/crush/crush.json ~/.config/crush/crush.json
```

### 5.2 Test Crush

```bash
crush
# Select a model from the list
# Try: "What is 2+2?"
```

### 5.3 Test MCP servers

```bash
# In Crush, try these prompts with llama3-groq-tool-use:8b:
"List files in /home/mgilbert/Code/clood"      # Tests filesystem MCP
"Search the web for 'ollama performance tips'" # Tests SearXNG MCP
```

---

## Phase 6: Performance Tuning (Linux)

### 6.1 CPU Governor

```bash
# Check current
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

# Set to performance (temporary)
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Make permanent
sudo apt install linux-tools-common linux-tools-$(uname -r)
sudo cpupower frequency-set -g performance
```

### 6.2 Ollama Tuning

See [ollama-tuning.md](ollama-tuning.md) for GPU-specific settings.

---

## Phase 7: Project Awareness

Create or update the projects manifest so models know about your projects:

```bash
# Check if manifest exists
cat ~/Code/projects_manifest.json

# If not, create one based on ARCHITECTURE.md template
```

---

## Verification Checklist

Run through this to confirm everything works:

- [ ] `ollama list` shows your models
- [ ] `curl http://localhost:11434/api/tags` returns JSON
- [ ] `curl http://localhost:8888` returns SearXNG HTML
- [ ] `crush` opens and connects to Ollama
- [ ] Crush can use filesystem MCP (list files)
- [ ] Crush can use searxng MCP (web search)

---

## Troubleshooting

### Ollama running on CPU instead of GPU
See [GPU-SETUP.md](GPU-SETUP.md) - likely need Vulkan backend for AMD.

### MCP servers not responding
Check that Node.js is installed and npx works: `npx --version`

### Crush can't connect to Ollama
Verify Ollama is running: `curl http://localhost:11434/api/tags`

### Out of disk space
Move Ollama models to /home partition - see [ollama-tuning.md](ollama-tuning.md#troubleshooting)

---

## Next Steps

1. Read [ARCHITECTURE.md](ARCHITECTURE.md) to understand the tiered model approach
2. Read [crush.md](crush.md) for advanced Crush configuration
3. Read [WORKFLOW.md](WORKFLOW.md) for Claude Code + Crush workflow
4. Explore [model-comparison.md](model-comparison.md) for model selection
