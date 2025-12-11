# Infrastructure Setup

## Quick Start (Fresh Install)

```bash
cd infrastructure
cp .env.example .env
docker compose up -d
```

The docker-compose includes all necessary env vars for SearXNG integration.

## Configuring Existing Installation

If you already have open-webui running without the SearXNG env vars, configure via UI:

### Enable Web Search in open-webui

1. Open http://localhost:3000
2. Go to **Admin Panel** (gear icon) > **Settings** > **Web Search**
3. Toggle **Enable Web Search** ON
4. Set **Web Search Engine** to `searxng`
5. Set **SearXNG Query URL** to: `http://searxng:8080/search?q=<query>`
6. Set **Search Result Count** to 5 (or your preference)
7. Click **Save**

### Verify SearXNG is working

Test from command line:
```bash
# From host
curl "http://localhost:8888/search?q=test&format=json" | head -c 200

# From inside open-webui container
docker exec open-webui curl -s "http://searxng:8080/search?q=test&format=json" | head -c 200
```

### Using Web Search

In a chat, prefix your message with the web search icon or use a model that has web search enabled by default.

## Container Networks

Both containers must be on the same Docker network to communicate by name:

```bash
# Check networks
docker network ls

# Connect existing container to network
docker network connect clood-net open-webui
docker network connect clood-net searxng
```

## Troubleshooting

### "No search results"
- Verify SearXNG has `json` format enabled in settings.yml
- Test SearXNG directly: http://localhost:8888/?q=test

### "Connection refused"
- Containers not on same network
- Use `docker network inspect clood-net` to verify both containers are connected

### Web search not appearing
- Check Admin > Settings > Web Search is enabled
- Some models may need web search explicitly enabled in model settings
