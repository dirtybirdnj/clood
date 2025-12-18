# Ollama Tuning Guide

Performance settings for the server garden.

## Mac-Mini (16GB Unified Memory)

```bash
# Kill existing and restart with tuned settings
pkill ollama

OLLAMA_HOST=0.0.0.0 \
OLLAMA_NUM_PARALLEL=2 \
OLLAMA_MAX_LOADED_MODELS=2 \
OLLAMA_FLASH_ATTENTION=1 \
OLLAMA_KEEP_ALIVE=30m \
ollama serve
```

## Ubuntu25 (Workstation)

```bash
# Kill existing and restart with tuned settings
pkill ollama

OLLAMA_HOST=0.0.0.0 \
OLLAMA_NUM_PARALLEL=4 \
OLLAMA_MAX_LOADED_MODELS=3 \
OLLAMA_FLASH_ATTENTION=1 \
OLLAMA_KEEP_ALIVE=30m \
ollama serve
```

## Environment Variables Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_HOST` | 127.0.0.1:11434 | Bind address (0.0.0.0 for network) |
| `OLLAMA_NUM_PARALLEL` | 1 | Concurrent request handling |
| `OLLAMA_MAX_LOADED_MODELS` | 1 | Models kept hot in memory |
| `OLLAMA_FLASH_ATTENTION` | 0 | Enable flash attention (faster) |
| `OLLAMA_KEEP_ALIVE` | 5m | How long models stay loaded |
| `OLLAMA_DEBUG` | 0 | Verbose logging |

## Per-Request Options

Pass in API calls or set in Modelfile:

```json
{
  "options": {
    "num_ctx": 8192,
    "num_thread": 8,
    "num_gpu": 99,
    "temperature": 0.7
  }
}
```

## Persistent Config

Create `~/.ollama/config.json`:

```json
{
  "host": "0.0.0.0:11434",
  "origins": ["*"]
}
```

Note: Environment variables override config file.

## Systemd Service (Ubuntu)

To make settings permanent, edit the service:

```bash
sudo systemctl edit ollama
```

Add:
```ini
[Service]
Environment="OLLAMA_HOST=0.0.0.0"
Environment="OLLAMA_NUM_PARALLEL=4"
Environment="OLLAMA_MAX_LOADED_MODELS=3"
Environment="OLLAMA_FLASH_ATTENTION=1"
Environment="OLLAMA_KEEP_ALIVE=30m"
```

Then:
```bash
sudo systemctl daemon-reload
sudo systemctl restart ollama
```

---

*Tuned for catfights. Let the models roar.*
