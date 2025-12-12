# Crush Configuration

Crush is the terminal interface for **local LLM hardware only**.

**Architecture:**
- **Claude Code CLI** = Claude (cloud API) - for heavy lifting, complex tasks
- **Crush** = Local Ollama models - for offline work, privacy, cost savings

This document covers configuring Crush to use the workstation's Ollama models locally and from remote machines.

## Current Setup

**Workstation:** ubuntu25 (192.168.4.62)
- Ollama running on port 11434 (exposed to network via `OLLAMA_HOST=0.0.0.0`)
- SearXNG running on port 8888
- Open WebUI running on port 3000

**Available Models (Ollama):**
- `qwen2.5-coder:7b` - 4.7 GB, coding tasks
- `qwen2.5-coder:3b` - 1.9 GB, fast coding
- `llama3.1:8b` - 4.9 GB, general purpose
- `tinyllama:latest` - 637 MB, lightweight
- `nomic-embed-text` - 274 MB, embeddings

## Config File Location

**IMPORTANT:** The config file is `crush.json` (not `providers.json`) and goes in:
- `~/.config/crush/crush.json` (global config)
- `.crush.json` or `crush.json` in project root (per-project override)

## Workstation: Configure Crush for Local Ollama

On ubuntu25, configure Crush to use the local Ollama instance.

### Create the config file

```bash
mkdir -p ~/.config/crush
cat > ~/.config/crush/crush.json << 'EOF'
{
  "$schema": "https://charm.land/crush.json",
  "providers": {
    "ollama": {
      "name": "Local Ollama",
      "base_url": "http://localhost:11434/v1/",
      "type": "openai",
      "api_key": "ollama",
      "models": [
        {
          "name": "Qwen 2.5 Coder 7B",
          "id": "qwen2.5-coder:7b",
          "context_window": 32768,
          "default_max_tokens": 8192
        },
        {
          "name": "Qwen 2.5 Coder 3B",
          "id": "qwen2.5-coder:3b",
          "context_window": 32768,
          "default_max_tokens": 8192
        },
        {
          "name": "Llama 3.1 8B",
          "id": "llama3.1:8b",
          "context_window": 128000,
          "default_max_tokens": 8192
        }
      ]
    }
  }
}
EOF
```

### Launch Crush

```bash
crush
# Select "Local Ollama" provider and your model
```

### Web Search with Local Models

SearXNG is running on port 8888, but Crush doesn't have native SearXNG integration. Options:
1. **Open WebUI** (http://localhost:3000) - has SearXNG built-in
2. **Manual search** - query SearXNG at http://localhost:8888, paste relevant info into Crush

## Remote Access: Laptop & Mac Mini

To access models on the workstation from other machines.

### Workstation Prerequisites (already done)

Ollama is already exposed to the network:
- `OLLAMA_HOST=0.0.0.0:11434` set in systemd
- Port 11434 open

Verify from workstation:
```bash
ss -tlnp | grep 11434
# Should show *:11434 (listening on all interfaces)
```

### Laptop/Mac Mini Setup

#### 1. Install Crush

**macOS:**
```bash
brew install charmbracelet/tap/crush
```

**Linux:**
```bash
# Check https://github.com/charmbracelet/crush for latest install method
curl -fsSL https://charm.sh/crush.sh | bash
```

#### 2. Create config pointing to workstation

```bash
mkdir -p ~/.config/crush
cat > ~/.config/crush/crush.json << 'EOF'
{
  "$schema": "https://charm.land/crush.json",
  "providers": {
    "ollama": {
      "name": "ubuntu25 Ollama",
      "base_url": "http://192.168.4.62:11434/v1/",
      "type": "openai",
      "api_key": "ollama",
      "models": [
        {
          "name": "Qwen 2.5 Coder 7B",
          "id": "qwen2.5-coder:7b",
          "context_window": 32768,
          "default_max_tokens": 8192
        },
        {
          "name": "Qwen 2.5 Coder 3B",
          "id": "qwen2.5-coder:3b",
          "context_window": 32768,
          "default_max_tokens": 8192
        },
        {
          "name": "Llama 3.1 8B",
          "id": "llama3.1:8b",
          "context_window": 128000,
          "default_max_tokens": 8192
        }
      ]
    }
  }
}
EOF
```

#### 3. Test connection

```bash
# Test Ollama API directly first
curl http://192.168.4.62:11434/v1/models

# If that works, launch Crush
crush
```

#### 4. Troubleshooting

If connection refused:
```bash
# From laptop, check if port is reachable
nc -zv 192.168.4.62 11434

# If not, check workstation firewall
# On workstation:
sudo ufw status
sudo ufw allow from 192.168.4.0/24 to any port 11434
```

## Alternative: SSH Tunnel

If firewall is an issue or you want encryption:

```bash
# From laptop/Mac Mini, create tunnel to workstation
ssh -L 11434:localhost:11434 mgilbert@192.168.4.62 -N &

# Then use localhost in crush.json
"base_url": "http://localhost:11434/v1/"
```

## Quick Reference

| Service | Workstation URL | Purpose |
|---------|-----------------|---------|
| Ollama | http://192.168.4.62:11434 | Model API |
| SearXNG | http://192.168.4.62:8888 | Web search |
| Open WebUI | http://192.168.4.62:3000 | Web chat interface |

## Crush Config Locations

```
~/.config/crush/crush.json    # Global config (USE THIS)
.crush.json                   # Project-level override
~/.local/share/crush/         # Ephemeral data (ignore)
```

**Warning:** `crush update-providers` overwrites settings. Don't run it or you'll lose your Ollama config.

## Status

- [x] Workstation: Crush configured with Local Ollama
- [x] Workstation: Ollama exposed to network (OLLAMA_HOST=0.0.0.0)
- [x] Workstation: Port 11434 open
- [ ] Laptop: Install Crush, create crush.json
- [ ] Mac Mini: Install Crush, create crush.json

## MCP Server Configuration

Crush supports Model Context Protocol (MCP) servers for extended capabilities. The following MCP servers are configured:

### Prerequisites

```bash
# Install Node.js and npm (required for MCP servers)
sudo apt install nodejs npm

# Verify gh CLI is authenticated
gh auth status
```

### Configured MCP Servers

| Server | Purpose | Package |
|--------|---------|---------|
| `filesystem` | Read/write files in ~/Code | @modelcontextprotocol/server-filesystem |
| `searxng` | Web search via local SearXNG | @kevinwatt/mcp-server-searxng |
| `github` | Full gh CLI access (repos, PRs, issues) | any-cli-mcp-server |

### MCP Configuration in crush.json

Add to `~/.config/crush/crush.json`:

```json
{
  "mcp": {
    "filesystem": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/home/mgilbert/Code"],
      "timeout": 120,
      "disabled": false
    },
    "searxng": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@kevinwatt/mcp-server-searxng"],
      "timeout": 60,
      "disabled": false,
      "env": {
        "SEARXNG_URL": "http://localhost:8888"
      }
    },
    "github": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "any-cli-mcp-server", "gh"],
      "timeout": 120,
      "disabled": false
    }
  }
}
```

### MCP Server Details

**Filesystem Server**
- Provides: read_file, write_file, list_directory, create_directory, move_file, search_files, get_file_info
- Scoped to `/home/mgilbert/Code` for security
- [Documentation](https://www.npmjs.com/package/@modelcontextprotocol/server-filesystem)

**SearXNG Server**
- Provides: web_search tool
- Uses local SearXNG instance at http://localhost:8888
- Privacy-preserving metasearch

**GitHub CLI Server**
- Wraps the `gh` CLI tool via any-cli-mcp-server
- Provides access to all gh subcommands: repo, pr, issue, gist, etc.
- Requires `gh auth login` to be completed first
- [Documentation](https://github.com/eirikb/any-cli-mcp-server)

### Remote Workstation MCP Config

For laptop/Mac Mini connecting to ubuntu25's Ollama but using local filesystem:

```json
{
  "mcp": {
    "filesystem": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "$HOME/Code"],
      "timeout": 120
    },
    "searxng": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@kevinwatt/mcp-server-searxng"],
      "timeout": 60,
      "env": {
        "SEARXNG_URL": "http://192.168.4.62:8888"
      }
    },
    "github": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "any-cli-mcp-server", "gh"],
      "timeout": 120
    }
  }
}
```

### Disabling Individual Tools

You can disable specific tools within an MCP server:

```json
{
  "mcp": {
    "github": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "any-cli-mcp-server", "gh"],
      "disabled_tools": ["repo delete", "pr close"]
    }
  }
}
```

## Future Improvements

- [ ] mDNS/avahi for `ubuntu25.local` hostname resolution
- [ ] Pull larger models (deepseek-coder, codellama, etc.)
- [ ] Test MCP servers on laptop and Mac Mini
