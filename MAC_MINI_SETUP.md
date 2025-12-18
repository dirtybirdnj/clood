# Mac Mini Integration - Server Garden Setup

## The Problem
Ollama defaults to `127.0.0.1:11434` (localhost only). To join the Server Garden, it must listen on `0.0.0.0:11434` (all interfaces).

## Quick Fix (Temporary)

```bash
# Kill existing Ollama
pkill ollama

# Start with external access
OLLAMA_HOST=0.0.0.0 ollama serve
```

## Permanent Fix (macOS)

### Option 1: launchd environment
```bash
launchctl setenv OLLAMA_HOST 0.0.0.0
brew services restart ollama
```

### Option 2: Create/edit Ollama plist
```bash
# If using Homebrew, edit the plist:
nano ~/Library/LaunchAgents/homebrew.mxcl.ollama.plist

# Add to EnvironmentVariables section:
# <key>OLLAMA_HOST</key>
# <string>0.0.0.0</string>
```

### Option 3: Shell profile
```bash
# Add to ~/.zshrc or ~/.bash_profile:
export OLLAMA_HOST=0.0.0.0
```

## Verify It's Working

```bash
# Check what Ollama is listening on:
lsof -i :11434

# Should show:
# ollama  12345  user  3u  IPv4  *:11434 (LISTEN)
#                           ^^^^ this means all interfaces

# NOT:
# ollama  12345  user  3u  IPv4  127.0.0.1:11434 (LISTEN)
#                                ^^^^^^^^^ localhost only
```

## Test from Another Machine

```bash
# From ubuntu25 or laptop:
curl http://192.168.4.41:11434/api/tags
```

## Garden Config

Once working, mac-mini is configured in `~/.config/clood/config.yaml`:
```yaml
hosts:
  - name: mac-mini
    url: http://192.168.4.41:11434
    priority: 2
    enabled: true
```

## Current Garden Topology

```
        [Driver: Laptop]
              |
    ┌─────────┴─────────┐
    |                   |
[Iron Keep]        [Sentinel]
ubuntu25            mac-mini
192.168.4.64       192.168.4.41
10 models           ? models
    |                   |
    └─────────┬─────────┘
              |
        [Catfight Arena]
```

---
*The garden grows stronger when all trees can see each other.*
