# clood

We want Claude! Too bad, we're out of tokens. We have clood instead.

Local LLM agent toolkit for when you run out of tokens or want to keep things on your own hardware.

## What is this?

clood is two things:

1. **Infrastructure** - Docker setup for running local LLMs with open-webui, Ollama, and SearXNG
2. **Agent Toolkit** - Portable skills, prompts, and tools that work with any LLM (local or cloud)

The goal: point any agent at this repo and it can rebuild the entire setup or use any of the captured skills.

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Ollama installed on host (`curl -fsSL https://ollama.com/install.sh | sh`)
- Ubuntu 25 (or similar Linux - YMMV on other distros)

### 1. Start the stack

```bash
cd infrastructure
cp .env.example .env  # customize paths if needed
docker compose up -d
```

### 2. Pull models

```bash
ollama pull qwen2.5-coder:7b   # coding tasks
ollama pull llama3.1:8b        # general purpose
ollama pull nomic-embed-text   # embeddings for RAG
```

### 3. Access services

| Service | URL | Purpose |
|---------|-----|---------|
| open-webui | http://localhost:3000 | Chat interface |
| SearXNG | http://localhost:8888 | Web search |
| Ollama | http://localhost:11434 | Model API |

### 4. Configure web search in open-webui

1. Go to Admin Panel > Settings > Web Search
2. Enable Web Search
3. Set Search Engine to "searxng"
4. Set SearXNG URL to `http://searxng:8080` (container network) or `http://localhost:8888` (host)

## Directory Structure

```
clood/
├── infrastructure/          # Docker configs, SSL, networking
│   ├── docker-compose.yml   # All services
│   ├── configs/
│   │   ├── searxng/         # SearXNG settings
│   │   └── nginx/           # Reverse proxy (optional)
│   └── ssl/                 # Self-signed certs
│
├── skills/                  # Portable agent capabilities
│   ├── open-webui/          # Python tools/functions
│   ├── claude-code/         # Slash commands, CLAUDE.md templates
│   └── prompts/             # Reusable system prompts
│
├── mcp-servers/             # MCP server configs
│
├── models/                  # Modelfiles, quantization notes
│   └── ollama/
│
├── scripts/                 # Setup and utility scripts
│
└── drop-zone/               # Local file staging (gitignored)
```

## The Drop Zone

The `drop-zone/` directory is mounted into open-webui at `/app/drop-zone`. Use it to:
- Stage files for LLM analysis without web fetching
- Share documents between sessions
- Provide context that would otherwise be rate-limited

Files in drop-zone are gitignored - it's a local scratch space.

## Skills

Skills are portable agent capabilities that can be imported into various LLM interfaces.

### open-webui Tools

Python scripts in `skills/open-webui/` can be imported via:
- Admin Panel > Workspace > Tools > Import

### Claude Code Commands

Slash commands in `skills/claude-code/commands/` can be symlinked to `~/.claude/commands/`

### Prompts

System prompts in `skills/prompts/` work with any LLM.

## Current Setup

### Models (Ollama)
- `qwen2.5-coder:7b` - Fast coding model
- `llama3.1:8b` - General purpose
- `nomic-embed-text` - Embeddings for RAG

### Services
- **open-webui** - Web interface, port 3000
- **SearXNG** - Privacy-respecting metasearch, port 8888
- **Ollama** - Model serving, port 11434

## Rebuilding from Scratch

```bash
# Clone the repo
git clone https://github.com/dirtybirdnj/clood.git
cd clood

# Start infrastructure
cd infrastructure
docker compose up -d

# Pull models
ollama pull qwen2.5-coder:7b
ollama pull llama3.1:8b
ollama pull nomic-embed-text

# Import any saved skills via open-webui UI
```

## TODO

- [ ] HTTPS with self-signed cert at https://clood/
- [ ] nginx reverse proxy config
- [ ] Document CLI tools (apt, cargo, pip)
- [ ] tmux dashboard script (lazydocker, btop, terminal)
- [ ] MCP server for drop zone access
- [ ] Export/import scripts for open-webui tools
