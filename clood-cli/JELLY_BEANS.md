# Jelly Beans

Feature ideas crystallized for future implementation. Each bean is a facet of the vision.

---

## Bean #1: Headroom Analysis Enhancement

**Status:** Planted
**Session:** The Bar Session (Dec 17, 2025)

The `clood system` command shows RAM headroom but should also show:
- **Disk space** available vs used by models
- **Per-model disk usage** breakdown
- **Download size** vs **runtime memory** distinction
- **Concurrent model** capacity (how many 8B models can run simultaneously?)

```bash
# Current output focuses on RAM
clood system

# Should also show:
# Disk:     1.3 TB free
# Models:   42 GB used (5 models)
#   qwen2.5-coder:14b   8.4 GB
#   deepseek-r1:8b      4.3 GB
#   ...
```

**Why:** Users need to know both RAM headroom (runtime) AND disk headroom (storage) when deciding which models to pull.

---

## Bean #2: `--json` Flag for All Commands

**Status:** Sprouting (3/14 commands done)
**Session:** The Bar Session (Dec 17, 2025)

Every CLI command should support `--json` (or `-j`) flag for machine-readable output.

**Why:** When clood is used via MCP (`any-cli-mcp-server`), agents parse the output. Human-readable tables with box-drawing characters are hard to parse. JSON is universal.

```bash
# Human mode (default)
clood hosts
# ● localhost
#   http://localhost:11434
#   Latency: 1ms
#   ...

# Agent mode
clood hosts --json
# {"hosts":[{"name":"localhost","url":"http://localhost:11434","online":true,"latency_ms":1,"models":["qwen2.5-coder:14b",...]}]}
```

**Implementation:**
- Add `--json` flag to root command (inherited by all subcommands)
- Each command checks the flag and switches output format
- Use `encoding/json` with proper structs
- Suppress banners/decorative output in JSON mode

**Commands to update:**
- [x] `clood hosts` (global --json works)
- [x] `clood models` (global --json works)
- [x] `clood system` (global --json works)
- [ ] `clood bench`
- [ ] `clood health`
- [ ] `clood analyze`
- [ ] `clood grep`
- [ ] `clood symbols`
- [ ] `clood imports`
- [ ] `clood context`
- [ ] `clood tree`
- [ ] `clood summary`
- [ ] `clood agents`
- [ ] `clood issues`

**Priority:** High - this is the bridge between CLI-first and agent-first design.

---

## Bean #3: Brew Formula

**Status:** Dream
**Session:** The Bar Session (Dec 17, 2025)

> "I'll feel really fuckin' official and gold plated when it works through brew"

```bash
brew tap dirtybirdnj/clood
brew install clood
```

**Steps:**
1. Create homebrew-clood tap repository
2. Write formula pointing to release binaries
3. Set up GitHub Actions to build releases
4. Publish to tap

---

## Bean #4: SSE Streaming Server

**Status:** IMPLEMENTED
**Session:** The Bar Session (Dec 17, 2025)

Enable true streaming MCP via Server-Sent Events:

```bash
clood serve --sse --port 8765
```

crush config:
```json
"clood": {
  "type": "sse",
  "url": "http://localhost:8765/mcp/sse"
}
```

**Why:** stdio MCP is request/response. SSE enables streaming partial results during long operations like catfight. But see GOLDEN_PATHS.md - granular tools may be better than streaming monolithic ones.

---

## Bean #5: Golden Path Prompts

**Status:** Planted
**Session:** The Bar Session (Dec 17, 2025)

Create system prompts/instructions that teach crush HOW to use clood tools in sequence. Instead of one big tool call, guide the AI to:

1. Gather context first (tree, grep, symbols)
2. Explain what it found
3. Execute in small steps
4. Report progress at each step

Could be:
- A CLAUDE.md-style instruction file for crush
- MCP server metadata that describes workflows
- Example conversations that demonstrate the pattern

**Why:** The tools exist. The golden paths are documented. Now we need to teach the AI to walk them.

---

*Jelly beans planted in the server garden, waiting to bloom.*
