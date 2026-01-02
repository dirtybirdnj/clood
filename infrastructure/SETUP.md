# Infrastructure Setup

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                   Open WebUI (localhost:3000)                    │
│  Chat UI │ RAG/Documents │ clood Tools │ Web Search             │
└──────────────────────────┬──────────────────────────────────────┘
                           │ OpenAI API
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                   clood proxy (localhost:4000)                   │
│  Routes requests to best available Ollama host                   │
└──────────────────────────┬──────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│  localhost    │  │  mac-mini     │  │  ubuntu25     │
│  :11434       │  │  :11434       │  │  :11434       │
└───────────────┘  └───────────────┘  └───────────────┘
```

## Quick Start

### 1. Start clood proxy (routes to all your Ollama hosts)

```bash
# In one terminal - keeps running
clood proxy --port 4000

# Or run in background
clood proxy --port 4000 &
```

### 2. Create Docker network (first time only)

```bash
docker network create webui-net
```

### 3. Start Open WebUI + SearXNG

```bash
cd ~/Code/clood/infrastructure
docker compose up -d
```

### 4. Open the UI

```
http://localhost:3000
```

All models from all your Ollama hosts will appear in the model dropdown.

## Adding clood Tools

Import the clood tools for codebase exploration:

1. Open http://localhost:3000
2. Go to **Workspace** > **Tools**
3. Click **+** (Add Tool)
4. Paste contents of `~/Code/clood/skills/open-webui/clood-tools.py`
5. Click **Save**

Now you can use in chat:
- `grep("pattern")` - Search codebase
- `tree("path")` - Show directory structure
- `symbols("path")` - Extract functions/types
- `ask("question")` - Query another model (inception)

## Ports

| Service | Port | Purpose |
|---------|------|---------|
| Open WebUI | 3000 | Chat interface |
| clood proxy | 4000 | OpenAI-compatible API |
| SearXNG | 8889 | Web search |
| Ollama (local) | 11434 | Local LLM |

## Troubleshooting

### "No models available"

clood proxy not running:
```bash
# Check if running
curl http://localhost:4000/v1/models

# If not, start it
clood proxy --port 4000
```

### "Connection refused to proxy"

From inside Docker, `host.docker.internal` should resolve to host machine.
Verify:
```bash
docker exec open-webui curl http://host.docker.internal:4000/v1/models
```

### Models not appearing from remote hosts

Check hosts are online:
```bash
clood hosts
```

### Direct Ollama fallback

If proxy isn't running, Open WebUI falls back to `OLLAMA_BASE_URL` (localhost:11434).
This only sees local models, not network hosts.

## Stopping

```bash
cd ~/Code/clood/infrastructure
docker compose down

# Also stop proxy if running
pkill -f "clood proxy"
```

## Web Search with SearXNG

Already configured. In chat, click the web search icon or models with web search enabled will auto-search.

Test:
```bash
curl "http://localhost:8889/search?q=test&format=json" | head -c 200
```

## Updating Open WebUI

```bash
docker compose pull
docker compose up -d
```
