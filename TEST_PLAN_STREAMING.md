# Test Plan: Streaming Features on ubuntu25

Quick reference for testing the new streaming catfight and inception features.

---

## Setup (One Time)

```bash
cd ~/Code/clood/clood-cli
git pull
go build -o ~/bin/clood ./cmd/clood
```

Verify Ollama is running:
```bash
curl http://localhost:11434/api/tags
```

---

## Test 1: Live Streaming Catfight

**What it does:** Runs multiple LLMs against the same prompt, streaming all responses simultaneously in a TUI.

**Command:**
```bash
clood catfight-live "Write a function to calculate fibonacci numbers in Go"
```

**What to look for:**
- Both cats (Siamese and Caracal) should start streaming at the same time
- You should see tokens appear from both models interleaved
- Status shows "● STREAMING" while active, "✓ DONE" when finished
- Timer in header shows elapsed time

**Controls:**
- `q` or `Esc` — quit
- `f` — toggle follow mode (auto-scroll to bottom)
- `g` — jump to top
- `G` — jump to bottom

**Variations to try:**
```bash
# Different models
clood catfight-live --models "qwen2.5-coder:3b,qwen2.5-coder:7b" "Explain recursion"

# Three cats
clood catfight-live --models "tinyllama,qwen2.5-coder:3b,llama3.1:8b" "Write hello world"
```

---

## Test 2: Inception (Interactive)

**What it does:** TUI where you chat with a model that can query "expert" models mid-response.

**Command:**
```bash
clood inception --model qwen2.5-coder:7b
```

**What to do:**
1. When prompted, type a request that might need expert knowledge
2. Example prompts:
   - "Write code to calculate escape velocity from Earth"
   - "Create a physics simulation for projectile motion"
   - "Calculate orbital parameters for a satellite"

**What to look for:**
- The model may emit `<sub-query model="science">...</sub-query>` tags
- When it does, you'll see "⏳ SUB-QUERY" in the header
- The expert response gets injected into the stream
- Main model continues with the expert knowledge

**Note:** The model needs to be "taught" to use sub-queries via the system prompt. It may or may not choose to use them depending on the prompt. This is experimental.

---

## Test 3: MCP Inception Tool (API)

**What it does:** Tests the inception tool via the MCP server directly.

**Terminal 1 — Start server:**
```bash
clood serve --sse
```

**Terminal 2 — Call inception:**
```bash
curl -X POST http://localhost:8765/call \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "clood_inception",
      "arguments": {
        "expert": "science",
        "query": "What is the gravitational constant G in SI units?"
      }
    }
  }'
```

**What to look for:**
- JSON response with the expert model's answer
- Duration field showing how long the sub-query took

---

## Quick Smoke Test

Run this one-liner to verify everything works:
```bash
clood catfight-live --models "qwen2.5-coder:3b" "Say hello"
```

Should complete in 5-10 seconds with a simple response.

---

## Troubleshooting

**"connection refused"**
→ Ollama isn't running. Start it: `ollama serve`

**"model not found"**
→ Pull the model: `ollama pull qwen2.5-coder:3b`

**UI looks broken**
→ Try a larger terminal window (at least 80x24)

**Cats not streaming**
→ Check Ollama logs: `journalctl -u ollama -f`

---

## What We're Testing

| Feature | Status | Notes |
|---------|--------|-------|
| Parallel model streaming | 🧪 Testing | Both cats stream simultaneously |
| BubbleTea TUI | 🧪 Testing | Viewport, scrolling, follow mode |
| Token counting | 🧪 Testing | Per-cat token counts and speeds |
| Inception sub-queries | 🧪 Testing | One model queries another |
| MCP tool | 🧪 Testing | clood_inception via HTTP |

---

## Known Limitations

1. **Inception is experimental** — Models may not consistently use `<sub-query>` tags
2. **No split layout yet** — Only interleaved view is implemented
3. **No judge model** — Future feature for live commentary
4. **Input handling** — Snake Road input zones not yet integrated

---

## After Testing

Please note:
- Any crashes or weird behavior
- Token speeds on different models
- Whether inception sub-queries triggered
- Ideas for improvements

Report findings in a GitHub issue or the next session.
