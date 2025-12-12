# Crush CLI Installation Guide

[Crush](https://github.com/charmbracelet/crush) is a terminal-based chat interface by Charmbracelet that supports MCP servers for tool use.

## Installation

### macOS (Homebrew)

```bash
brew install charmbracelet/tap/crush
```

### Linux (Download Binary)

```bash
# Check latest version at https://github.com/charmbracelet/crush/releases

# AMD64
wget https://github.com/charmbracelet/crush/releases/latest/download/crush_Linux_x86_64.tar.gz
tar xzf crush_Linux_x86_64.tar.gz
sudo mv crush /usr/local/bin/
rm crush_Linux_x86_64.tar.gz

# ARM64
wget https://github.com/charmbracelet/crush/releases/latest/download/crush_Linux_arm64.tar.gz
tar xzf crush_Linux_arm64.tar.gz
sudo mv crush /usr/local/bin/
rm crush_Linux_arm64.tar.gz
```

### Linux (Go Install)

If you have Go installed:

```bash
go install github.com/charmbracelet/crush@latest
```

### Verify Installation

```bash
crush --version
```

## Configuration

### Config File Location

- Linux: `~/.config/crush/crush.json`
- macOS: `~/Library/Application Support/crush/crush.json` or `~/.config/crush/crush.json`

### Copy clood Config Template

```bash
mkdir -p ~/.config/crush
cp ~/Code/clood/infrastructure/configs/crush/crush.json ~/.config/crush/crush.json
```

### Database Location

Crush stores conversation history in:
- Linux: `~/.local/share/crush/crush.db`
- macOS: `~/Library/Application Support/crush/crush.db`

## Running Crush

```bash
# Start Crush
crush

# Use arrow keys to select a model
# Type your prompt and press Enter
# Use Ctrl+C to exit
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Enter | Send message |
| Ctrl+C | Exit |
| Up/Down | Scroll history |
| Tab | Cycle through models |

## MCP Server Integration

The clood crush.json configures three MCP servers:

### Filesystem

Provides read/write access to `~/Code`:

```json
"filesystem": {
  "type": "stdio",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/home/mgilbert/Code"]
}
```

### SearXNG (Web Search)

Requires SearXNG running on localhost:8888:

```json
"searxng": {
  "type": "stdio",
  "command": "npx",
  "args": ["-y", "@kevinwatt/mcp-server-searxng"],
  "env": {
    "SEARXNG_URL": "http://localhost:8888"
  }
}
```

### GitHub CLI

Wraps the `gh` command for GitHub operations:

```json
"github": {
  "type": "stdio",
  "command": "npx",
  "args": ["-y", "any-cli-mcp-server", "gh"]
}
```

## Testing MCP Tools

Use a tool-capable model like `llama3-groq-tool-use:8b`:

```
# Test filesystem
"List files in /home/mgilbert/Code/clood"

# Test web search
"Search the web for 'ollama performance tuning'"

# Test GitHub
"Show my GitHub repositories"
```

## Troubleshooting

### Crush can't connect to Ollama

```bash
# Check Ollama is running
curl http://localhost:11434/api/tags

# If not running
sudo systemctl start ollama  # Linux
ollama serve                  # macOS
```

### MCP servers not responding

```bash
# Check Node.js/npx
npx --version

# Test filesystem MCP manually
npx -y @modelcontextprotocol/server-filesystem /tmp
```

### Model not using tools

Not all models support tool calling. Use:
- `llama3-groq-tool-use:8b` (best)
- `qwen3:8b`
- `llama3.2:3b`
- `llama3.1:8b`

Avoid using non-tool models for MCP operations:
- `qwen2.5-coder:*` (coding-focused, no tools)
- `deepseek-coder:*` (coding-focused, no tools)
- `tinyllama` (too small for tools)

## Remote Ollama

To use Ollama on another machine, update crush.json:

```json
"providers": {
  "ollama": {
    "name": "Remote Ollama",
    "base_url": "http://192.168.4.62:11434/v1/",
    "type": "openai",
    "api_key": "ollama"
  }
}
```

## References

- [Crush GitHub](https://github.com/charmbracelet/crush)
- [MCP Protocol](https://modelcontextprotocol.io/)
- [Charmbracelet](https://charm.sh/)
