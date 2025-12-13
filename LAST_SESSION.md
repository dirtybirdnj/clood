# Last Session - 2025-12-12 (Late Night Continued)

## Big Win: Custom Scripts + Mods Working!

### Scripts Created (all in `~/Code/clood/scripts/`)

| Script | Purpose | Usage |
|--------|---------|-------|
| `code-review.py` | Review code with Ollama | `python3 code-review.py file.rs` |
| `code-review.py --edit` | Interactive Claude-style edits | `python3 code-review.py file.rs --edit` |
| `search-ask.py` | SearXNG search + Ollama answer | `python3 search-ask.py "your question"` |
| `gh-ask.py` | GitHub context + Ollama | `python3 gh-ask.py "summarize PRs" -r owner/repo` |
| `tool-proxy.py` | Inject tools into Ollama API | `python3 tool-proxy.py` (port 11435) |

### Mods Configured for Ollama + MCP

Edited `~/Library/Application Support/mods/mods.yml`:
- default-api: ollama
- default-model: llama3-groq-tool-use:8b
- MCP servers: filesystem, github

```bash
# Basic mods usage (WORKS!)
echo "hello" | mods "respond"
git diff | mods "explain"
cat file.rs | mods "review"
```

MCP with mods may need debugging - Ollama models don't handle tool calls as reliably as Claude.

---

## Scripts Usage Examples

```bash
# Code review (prose)
python3 ~/Code/clood/scripts/code-review.py ~/Code/rat-king/crates/rat-king-cli/src/main.rs

# Interactive edit mode (Claude Code style)
python3 ~/Code/clood/scripts/code-review.py ~/Code/rat-king/src/main.rs --edit

# Search + Ask
python3 ~/Code/clood/scripts/search-ask.py "ollama vulkan performance"

# GitHub context
python3 ~/Code/clood/scripts/gh-ask.py "what are recent issues" -r dirtybirdnj/clood

# Use different model
python3 ~/Code/clood/scripts/code-review.py file.py -m qwen2.5-coder:3b

# Use remote Ollama (ubuntu25)
python3 ~/Code/clood/scripts/search-ask.py "rust async" -o http://192.168.4.63:11434 -m llama3.2:3b
```

---

## Key Discovery: Crush Tool Calling Broken

- Ollama models DO support tool calling (verified via direct API)
- Crush does NOT pass tools to OpenAI-compatible APIs
- Mods has MCP support but Ollama may not handle it well
- Our custom scripts bypass this by calling Ollama directly

---

## Existing Tools to Explore

| Tool | Install | What it does |
|------|---------|--------------|
| **mods** | `brew install charmbracelet/tap/mods` | Pipe anything to LLM (configured!) |
| **llm** | `pip install llm llm-ollama` | Simon Willison's CLI, has tool support |

---

## Machine Status

| Machine | IP | Ollama | Models |
|---------|-----|--------|--------|
| MacBook Air | localhost | ✅ Running | groq-8b, qwen-3b, tinyllama |
| ubuntu25 | 192.168.4.63 | ✅ Running | groq-8b, llama3.2:3b, deepseek:6.7b, mistral:7b |
| Mac Mini | 192.168.4.41 | ❌ Needs setup | (see NEXT_SESSION.md) |

---

## Commits This Session

```
bd15c63 - Add gh-ask.py: GitHub context for Ollama
364885e - Add -o flag for remote Ollama URL
5ba261b - Add search-ask.py: SearXNG + Ollama RAG pipeline
68887bc - Add --edit mode: Claude Code style interactive apply
bc7e3d4 - Add --patch mode to code-review.py
7c602e2 - Fix code-review.py: prioritize code over docs
8d08023 - Add code-review.py - direct Ollama code reviewer
c093608 - Add tool-injection proxy for Ollama
```

---

## Next Steps

1. **Test mods MCP** - See if filesystem/github tools work with Ollama
2. **Try `llm` CLI** - `pip install llm llm-ollama` - has native tool support
3. **Mac Mini setup** - See NEXT_SESSION.md
4. **Start LiteLLM hub** - `~/Code/clood/scripts/start-litellm.sh`

---

## Files Modified

- `~/Library/Application Support/mods/mods.yml` - Ollama + MCP config
- `~/.config/crush/crush.json` - Added supports_tools flags (didn't help)
- `scripts/` - 5 new Python scripts

---

*Custom scripts work. Mods works. Crush MCP broken. Progress!*
