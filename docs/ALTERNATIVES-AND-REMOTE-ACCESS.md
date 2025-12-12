# Alternatives to Open WebUI & Remote Access Options

> Discussion from 2025-12-11: Exploring alternatives to Open WebUI and ways to access Ubuntu workstation from Mac devices.

## Background

Open WebUI has been problematic:
- Tool calling broken in v0.6.13+ (pinned to v0.6.10)
- Poor documentation
- Complex configuration
- Web search integration issues

This document explores alternatives and how local models can use tools.

---

## Part 1: Alternatives to Open WebUI (Detailed)

### 1. oterm - Terminal UI for Ollama

**What it is:** A text-based terminal client for Ollama with a rich TUI interface.

**Key Features:**
- Multiple persistent chat sessions stored in SQLite
- **MCP (Model Context Protocol) support** - bridges MCP servers with Ollama
- Tool/function calling support (model dependent)
- Terminal image display (Sixel graphics)
- Customizable themes
- "Thinking" mode for models that support it
- Streaming with tools
- MCP Sampling support

**MCP Integration:**
oterm bridges MCP servers with Ollama. Add servers to `~/.config/oterm/config.json`:
```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/dir"]
    }
  }
}
```

**Install:**
```bash
pipx install oterm
# or
pip install oterm
```

**Usage:**
```bash
oterm
```

**Pros:**
- Native MCP support for filesystem access
- No web server needed
- Persistent sessions
- Works over SSH

**Cons:**
- Not all Ollama models support tools well
- Smaller models struggle with tool calling

**Links:**
- [GitHub](https://github.com/ggozad/oterm)
- [MCP Documentation](https://ggozad.github.io/oterm/mcp/)

---

### 2. Crush - Charm's AI Coding Agent

**What it is:** A glamorous terminal-based AI coding agent from Charmbracelet that integrates LLMs directly into your development workflow.

**Key Features:**
- **Multi-model support** - OpenAI, Anthropic, Ollama, or any OpenAI-compatible API
- **Switch LLMs mid-session** - Change models while preserving context
- **LSP integration** - Uses Language Server Protocol for deep code understanding
- **MCP support** - Extensible via Model Context Protocol (stdio, HTTP, SSE)
- **Session management** - Multiple work sessions per project
- **Built-in tools** - File viewing, editing, bash commands, searching
- **Permission system** - Asks before running tool calls (can override)
- **Cross-platform** - macOS, Linux, Windows, BSD

**Install:**
```bash
# macOS
brew install charmbracelet/tap/crush

# Go install
go install github.com/charmbracelet/crush@latest

# Or download from releases
```

**Configuration (.crush.json):**
```json
{
  "providers": {
    "ollama": {
      "type": "openai",
      "baseURL": "http://localhost:11434/v1",
      "models": ["qwen2.5-coder:7b", "llama3.1:8b"]
    }
  },
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "."]
    }
  }
}
```

**Usage:**
```bash
crush                    # Start interactive session
crush "explain this code"  # One-shot query
```

**Pros:**
- Beautiful TUI from Charm (makers of bubbletea, lipgloss)
- First-class MCP and LSP support
- Model-agnostic (works with local and cloud)
- Active development, polished UX

**Cons:**
- Newer project (less battle-tested)
- Requires Go or Homebrew to install

**Links:**
- [GitHub](https://github.com/charmbracelet/crush)
- [Charm](https://charm.sh/)

---

### 3. gptme - CLI Agent with Tools

**What it is:** A CLI agent that can execute code, manipulate files, browse the web, and use vision.

**Key Features:**
- **Code execution** - Python (IPython) and shell commands
- **File operations** - read, write, append, patch files
- **Web browsing** - scrape and interact with websites
- **Subagents** - delegate tasks to sub-agents
- **RAG** - retrieval-augmented generation
- **Vision** - analyze images/screenshots
- **Self-correcting** - output fed back to model

**Available Tools:**
`append`, `browser`, `chats`, `computer`, `gh`, `ipython`, `patch`, `rag`, `read`, `save`, `screenshot`, `shell`, `subagent`, `tmux`, `vision`, `youtube`

**Install:**
```bash
pipx install gptme
# Python 3.10+ required
```

**Usage:**
```bash
# Interactive mode
gptme

# With Ollama
gptme --model ollama/qwen2.5-coder:7b

# Execute a task
gptme "read the README and summarize it"
```

**Pros:**
- Rich tool ecosystem out of the box
- Works with local and cloud models
- True code execution (not just generation)
- Privacy-focused (local alternative to ChatGPT Code Interpreter)

**Cons:**
- More complex than simple chat
- Local models may struggle with complex tool chains

**Links:**
- [Website](https://gptme.org/)
- [GitHub](https://github.com/gptme/gptme)
- [Tools Documentation](https://gptme.org/docs/tools.html)

---

### 4. aider - AI Pair Programming

**What it is:** AI pair programming in your terminal. Git-aware, designed for coding.

**Key Features:**
- **Git integration** - auto-commits with descriptive messages
- **Multi-file editing** - understands project context
- **Local model support** - works with Ollama
- **Offline capable** - no internet required with local models
- **Voice coding** - optional speech-to-text

**Install:**
```bash
pipx install aider-chat
```

**Usage with Ollama:**
```bash
# Set context window (important!)
export OLLAMA_CONTEXT_LENGTH=8192
ollama serve

# Run aider
aider --model ollama_chat/qwen2.5-coder:7b

# Or with specific files
aider --model ollama_chat/qwen2.5-coder:7b src/main.py
```

**Important: Context Window**
Ollama defaults to 2k context, which is too small. Aider auto-adjusts but you should set `OLLAMA_CONTEXT_LENGTH=8192` or higher.

**Recommended Models:**
- `qwen2.5-coder` - Best for code (4.7 GB)
- `deepseek-coder-v2` - Strong code understanding (8.9 GB)

**Pros:**
- Purpose-built for coding
- Excellent git integration
- Understands project structure

**Cons:**
- Focused only on coding tasks
- Local models less capable than GPT-4/Claude

**Links:**
- [Website](https://aider.chat/)
- [Ollama Setup](https://aider.chat/docs/llms/ollama.html)

---

### 5. LobeChat - Modern Web UI

**What it is:** Open-source ChatGPT alternative with modern design, plugins, and Ollama support.

**Key Features:**
- **Plugin system** - extensible function calling
- **Tool calling** - streaming tools since v1.35.0
- **MCP support** - Model Context Protocol integration
- **Multi-provider** - OpenAI, Anthropic, Ollama, etc.
- **Knowledge base** - file upload and RAG
- **Modern UI** - clean, well-designed interface

**Docker Install:**
```bash
docker run -d -p 3210:3210 \
  -e OLLAMA_PROXY_URL=http://host.docker.internal:11434/v1 \
  lobehub/lobe-chat
```

**Tool Calling:**
Works with models like `qwen2.5` that support function calling. Enable plugins in settings.

**Pros:**
- Better documentation than Open WebUI
- Active development
- Plugin marketplace
- Clean UX

**Cons:**
- Another web UI to maintain
- Tool calling still depends on model support

**Links:**
- [Website](https://lobehub.com/)
- [GitHub](https://github.com/lobehub/lobe-chat)
- [Ollama Integration](https://lobehub.com/docs/usage/providers/ollama)

---

### 6. LibreChat - Multi-Provider Platform

**What it is:** Enhanced ChatGPT clone with extensive features and MCP support.

**Key Features:**
- **MCP integration** - Full Model Context Protocol support
- **Multi-provider** - OpenAI, Anthropic, Ollama, Azure, etc.
- **Agents** - AI agents with tool use
- **Code Interpreter** - Execute code in chat
- **Artifacts** - Generate React, HTML, diagrams
- **RAG** - Document retrieval
- **Plugins** - Extensible plugin system

**Docker Install:**
```bash
git clone https://github.com/danny-avila/LibreChat.git
cd LibreChat
cp .env.example .env
# Configure Ollama in .env
docker compose up -d
```

**Ollama Configuration:**
Add to `librechat.yaml`:
```yaml
endpoints:
  custom:
    - name: "Ollama"
      apiKey: "ollama"
      baseURL: "http://host.docker.internal:11434/v1"
      models:
        default: ["qwen2.5-coder:7b", "llama3.1:8b"]
```

**Pros:**
- Most feature-complete alternative
- Strong privacy focus ("own your data")
- Air-gapped operation possible
- MCP support

**Cons:**
- Complex setup
- Heavy resource usage

**Links:**
- [Website](https://www.librechat.ai/)
- [GitHub](https://github.com/danny-avila/LibreChat)
- [Ollama Config](https://www.librechat.ai/docs/configuration/librechat_yaml/ai_endpoints/ollama)

---

### 7. AnythingLLM - Local RAG + Agents

**What it is:** All-in-one Desktop & Docker AI application with built-in RAG and AI agents.

**Key Features:**
- **AI Agents** - `@agent` commands with tool access
- **RAG** - Document upload and retrieval
- **Save Files tool** - Agent can save files to your desktop
- **Web scraping** - Scrape and summarize websites
- **Chart generation** - Create visualizations
- **MCP compatibility** - Docker deployment supports MCP Tools
- **No-code agent builder** - Build custom agents

**Default Agent Skills:**
- RAG search (enabled by default)
- Summarize documents (enabled by default)
- Scrape websites (enabled by default)
- Save files to local machine
- Web search

**Usage:**
```
@agent can you read the README and summarize it?
@agent save this information as a PDF on my desktop
exit
```

**Install:**
- Desktop app: [Download](https://anythingllm.com/)
- Docker: `docker pull mintplex/anythingllm`

**Pros:**
- Best for document Q&A
- Agent tools work well
- Desktop app is easy
- Privacy-focused defaults

**Cons:**
- Less flexible than CLI tools
- MCP only in Docker version

**Links:**
- [Website](https://anythingllm.com/)
- [GitHub](https://github.com/Mintplex-Labs/anything-llm)
- [Agent Docs](https://docs.useanything.com/agent/usage)

---

## Part 2: How Local Models Use Tools

### Understanding Tool Calling

Tool calling (aka function calling) lets an LLM request external actions. The model doesn't execute tools itself—it outputs a structured request, and the host application executes it.

**Flow:**
1. You send prompt + available tools to model
2. Model decides to call a tool, outputs JSON with tool name + args
3. Host app executes the tool
4. Tool result sent back to model
5. Model generates final response

### Ollama Tool Support

Ollama supports tool calling via its API. Not all models support it.

**Models with Tool Support:**
| Model | Tool Calling | Notes |
|-------|--------------|-------|
| `qwen2.5-coder:7b` | ✅ Yes | Good for coding tools |
| `qwen3` | ✅ Yes | Strong tool support |
| `llama3.1:8b` | ✅ Yes | General purpose |
| `llama3.2:3b` | ⚠️ Weak | May hallucinate tool calls |
| `gemma` | ❌ No | Does not support tools |
| `mistral` | ✅ Yes | Good support |

**Check supported models:**
```bash
# Models with tool support listed at:
# https://ollama.com/library (filter by "Tools")
```

### Filesystem Access Methods

#### Method 1: MCP Servers (Best for oterm, LibreChat)

MCP (Model Context Protocol) is Anthropic's standard for connecting AI to external tools.

**Filesystem MCP Server:**
```bash
# Install the filesystem server
npx -y @modelcontextprotocol/server-filesystem /path/to/allow

# Configure in oterm (~/.config/oterm/config.json):
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/home/mgilbert/Code"]
    }
  }
}
```

**Available MCP Servers:**
- `@modelcontextprotocol/server-filesystem` - File read/write
- `@modelcontextprotocol/server-github` - GitHub integration
- `@modelcontextprotocol/server-postgres` - Database access
- `@modelcontextprotocol/server-puppeteer` - Web browsing

#### Method 2: Open WebUI Tools (Current Setup)

Your `skills/open-webui/code-directory-reader.py` is a custom tool for Open WebUI:

```python
# Provides these functions to the LLM:
- list_directory(path)   # List files in /app/code
- read_file(path)        # Read file contents
- search_files(pattern)  # Find files by name
- grep_in_files(text)    # Search inside files
```

**How it works:**
1. Tool mounted in Open WebUI container
2. `/app/code` volume maps to host's code directory
3. LLM calls tool functions
4. Tool executes, returns results to LLM

#### Method 3: gptme Built-in Tools

gptme has native file operations—no configuration needed:

```bash
gptme "read the file src/main.py and explain it"
gptme "create a new file called test.py with a hello world function"
```

Tools: `read`, `save`, `append`, `patch`, `shell`

---

### Web Search: SearXNG Integration

Your stack includes **SearXNG** (not "xsearch")—a privacy-respecting metasearch engine.

**Current Setup (from docker-compose.yml):**
```yaml
searxng:
  image: searxng/searxng
  container_name: searxng
  ports:
    - "8888:8080"  # Access at http://localhost:8888
```

**How Open WebUI Uses SearXNG:**
```yaml
environment:
  - ENABLE_RAG_WEB_SEARCH=true
  - RAG_WEB_SEARCH_ENGINE=searxng
  - SEARXNG_QUERY_URL=http://searxng:8080/search?q=<query>
```

**Using SearXNG from Other Tools:**

Most tools can use SearXNG via its JSON API:
```bash
# Test from command line
curl "http://localhost:8888/search?q=test&format=json"
```

**MCP + SearXNG:**
There's no official SearXNG MCP server yet, but you could:
1. Use the `@anthropic/server-brave-search` as a template
2. Create a custom MCP server that queries your SearXNG instance

---

## Part 3: Accessing Ubuntu from Mac

### Option 1: SSH + Terminal Tools (Recommended)

```bash
# From Mac, SSH into Ubuntu and run CLI tools
ssh ubuntu-workstation
oterm  # or gptme, aider

# With tmux for persistent sessions
ssh ubuntu-workstation -t "tmux attach || tmux new"
```

**Setup (~/.ssh/config on Mac):**
```
Host ubuntu-workstation
    HostName 192.168.x.x
    User mgilbert
    ForwardAgent yes
```

### Option 2: SSH Port Forwarding

Forward web services from Ubuntu to Mac:

```bash
# Forward all services
ssh -L 3000:localhost:3000 \
    -L 11434:localhost:11434 \
    -L 8888:localhost:8888 \
    ubuntu-workstation

# Access on Mac:
# - Open WebUI/LobeChat: http://localhost:3000
# - Ollama API: http://localhost:11434
# - SearXNG: http://localhost:8888
```

**Persistent tunnels:**
```bash
brew install autossh
autossh -M 0 -f -N -L 3000:localhost:3000 ubuntu-workstation
```

### Option 3: Tailscale (Best for Always-On)

```bash
# Install on both machines
# Ubuntu:
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up

# Mac:
brew install tailscale
tailscale up

# Access via Tailscale IP
http://100.x.x.x:3000   # Open WebUI
http://100.x.x.x:11434  # Ollama API
```

### Option 4: VS Code Remote SSH

1. Install "Remote - SSH" extension in VS Code
2. Connect to Ubuntu workstation
3. Install Continue or other AI extensions
4. Code on Mac, compute on Ubuntu

### Option 5: Direct Ollama API Access

Make Ollama accessible from Mac (use with Tailscale or trusted network):

```bash
# On Ubuntu, edit Ollama service
sudo systemctl edit ollama

# Add:
[Service]
Environment="OLLAMA_HOST=0.0.0.0"

# Restart
sudo systemctl restart ollama

# From Mac
curl http://ubuntu-ip:11434/api/generate -d '{
  "model": "qwen2.5-coder:7b",
  "prompt": "Hello"
}'
```

---

## Recommendations Summary

| Use Case | Recommended Tool |
|----------|-----------------|
| Quick local chat | `ollama run` or `oterm` |
| Coding assistance | `aider` or `crush` |
| Agent with tools | `gptme`, `crush`, or `oterm` with MCP |
| Beautiful TUI + MCP | `crush` (Charm) |
| Document Q&A | AnythingLLM |
| Web UI needed | LobeChat or LibreChat |
| Mac → Ubuntu dev | VS Code Remote SSH |
| Always-on access | Tailscale |

---

## Next Steps

- [ ] Try crush (Charm's new AI coding agent)
- [ ] Install oterm and configure MCP filesystem server
- [ ] Test gptme with local Ollama models
- [ ] Try aider for coding tasks
- [ ] Set up Tailscale between Mac and Ubuntu
- [ ] Evaluate LobeChat if web UI still needed

---

## Sources

### oterm
- [GitHub - ggozad/oterm](https://github.com/ggozad/oterm)
- [oterm MCP Documentation](https://ggozad.github.io/oterm/mcp/)
- [Terminal Trove - oterm](https://terminaltrove.com/oterm/)

### Crush
- [GitHub - charmbracelet/crush](https://github.com/charmbracelet/crush)
- [Charm](https://charm.sh/)

### gptme
- [gptme.org](https://gptme.org/)
- [GitHub - gptme/gptme](https://github.com/gptme/gptme)
- [gptme Tools Documentation](https://gptme.org/docs/tools.html)

### aider
- [aider.chat](https://aider.chat/)
- [Ollama Integration](https://aider.chat/docs/llms/ollama.html)
- [LLM Connections](https://aider.chat/docs/llms.html)

### LobeChat
- [lobehub.com](https://lobehub.com/)
- [GitHub - lobehub/lobe-chat](https://github.com/lobehub/lobe-chat)
- [Ollama Provider Docs](https://lobehub.com/docs/usage/providers/ollama)

### LibreChat
- [librechat.ai](https://www.librechat.ai/)
- [GitHub - danny-avila/LibreChat](https://github.com/danny-avila/LibreChat)
- [Ollama Configuration](https://www.librechat.ai/docs/configuration/librechat_yaml/ai_endpoints/ollama)

### AnythingLLM
- [anythingllm.com](https://anythingllm.com/)
- [GitHub - Mintplex-Labs/anything-llm](https://github.com/Mintplex-Labs/anything-llm)
- [Agent Documentation](https://docs.useanything.com/agent/usage)

### Ollama Tool Calling
- [Ollama Tool Support Blog](https://ollama.com/blog/tool-support)
- [Tool Calling Documentation](https://docs.ollama.com/capabilities/tool-calling)
- [Model Library](https://ollama.com/library)

### MCP (Model Context Protocol)
- [Anthropic MCP Announcement](https://www.anthropic.com/news/model-context-protocol)
- [MCP Servers Repository](https://github.com/modelcontextprotocol/servers)

---

## Related Files in This Repo

- `README.md` - Main project documentation
- `infrastructure/docker-compose.yml` - SearXNG and Open WebUI setup
- `infrastructure/SETUP.md` - Current Open WebUI setup guide
- `skills/open-webui/code-directory-reader.py` - Custom filesystem tool
- `LAST_SESSION.md` - Previous debugging session
