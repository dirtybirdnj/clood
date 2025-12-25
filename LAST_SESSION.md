# Last Session: clood proxy & Issue #190

**Date:** 2025-12-24
**Previous:** ATC Dashboard row layout, model highlighting

---

## WHAT WE BUILT

### clood proxy Command (commit 08141fe)

OpenAI-compatible API server for multi-host routing:

```bash
clood proxy --port 8000 --atc http://localhost:8080
```

**Endpoints:**
- `POST /v1/chat/completions` - Routes to best available host
- `GET /v1/models` - Aggregates models from all hosts
- `GET /status` - Returns proxy statistics

**Features:**
- Host selection by lowest latency
- Model availability checking
- ATC event streaming (request_start, request_complete, request_error)
- Graceful shutdown

### GitHub Issue #190 Created

"Chat UI as command interface with ATC monitoring"

**Vision:**
- Chat UI = "radio communications" (talk to models)
- ATC = "control tower" (observe what's happening)

**Testing Plan (in issue):**
1. Start ATC dashboard
2. Start clood proxy
3. Run Open WebUI Docker container
4. Send message and verify events flow

---

## FILES CHANGED

**New:**
- `internal/commands/proxy.go` - OpenAI-compatible proxy server

**Modified:**
- `cmd/clood/main.go` - Added ProxyCmd to Infra group
- `internal/commands/atc.go` - Analysis panel WIP (not yet working)

---

## COMMITS PUSHED

```
74e182f wip: ATC analysis panel (not yet working)
08141fe feat: Add clood proxy - OpenAI-compatible API server for multi-host routing
```

---

## WHAT'S NOT WORKING

### Analysis Panel Still Not Appearing

The analysis event is sent from catfight but not showing in ATC UI.
Code is in place but needs debugging (see previous session notes).

---

## TESTING PLAN FOR TOMORROW

```bash
# Terminal 1: Start ATC dashboard
clood atc --mode active --port 8080

# Terminal 2: Start proxy
clood proxy --port 8000 --atc http://localhost:8080

# Terminal 3: Start Open WebUI
docker run -d -p 3000:8080 \
  -e OPENAI_API_BASE_URL=http://host.docker.internal:8000/v1 \
  -e OPENAI_API_KEY=not-needed \
  --name open-webui \
  ghcr.io/open-webui/open-webui:main

# Browser: Open http://localhost:3000
# Send a message, watch ATC dashboard for events
```

**Expected Flow:**
1. User sends message in Open WebUI
2. Open WebUI calls clood proxy
3. Proxy routes to best Ollama host
4. Events stream to ATC dashboard

---

## exo RESEARCH

Found exo-explore/exo - distributed AI cluster that splits large models across devices.

**Comparison:**
| Feature | clood | exo |
|---------|-------|-----|
| Splits models across GPUs | No | Yes |
| Compares models head-to-head | Yes | No |
| OpenAI-compatible API | Yes | Yes |
| Multi-host orchestration | Yes | Yes |

**Verdict:** Complementary tools. exo for running huge models, clood for comparing and routing.

---

## QUICK REFERENCE

```bash
# Test proxy directly
curl http://localhost:8000/v1/models
curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"qwen2.5-coder:3b","messages":[{"role":"user","content":"hi"}]}'

# Check proxy status
curl http://localhost:8000/status
```

---

```
Proxy routes the way,
Chat UI meets control tower—
Models await calls.
```
