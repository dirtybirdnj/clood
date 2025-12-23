# Clood Architecture

**Clood** = Claude-like experience running on local hardware.

## Vision

Replicate the Claude Code experience using local LLMs with:
1. **Project awareness** - Models know what projects exist and their structure
2. **Tiered inference** - Fast models gather context, powerful models do the thinking
3. **Tool integration** - MCP servers for filesystem, search, and GitHub
4. **Multi-machine support** - Distribute workloads across available hardware

## Core Principles

1. **Context is king** - Gather relevant context BEFORE engaging expensive models
2. **Right model for the job** - Don't use a 7B model to list files
3. **Hardware-aware** - Adapt to available GPU/CPU/RAM on each machine
4. **Learn by doing** - This is also a learning platform for understanding LLMs

---

## Tiered Model Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         User Query                               │
└─────────────────────────────────┬───────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                    TIER 1: Router / Classifier                   │
│                     (TinyLlama ~150 tok/s)                       │
│                                                                  │
│  Decides: Is this a search? File operation? Code task? Chat?    │
└─────────────────────────────────┬───────────────────────────────┘
                                  │
              ┌───────────────────┼───────────────────┐
              ▼                   ▼                   ▼
┌─────────────────────┐ ┌─────────────────┐ ┌─────────────────────┐
│  TIER 2: Retrieval  │ │  TIER 2: Tools  │ │  TIER 2: Search     │
│  (Embeddings model) │ │  (Llama 3B)     │ │  (SearXNG + 3B)     │
│                     │ │                 │ │                     │
│  - Find relevant    │ │  - List files   │ │  - Web search       │
│    files/code       │ │  - Read files   │ │  - Summarize results│
│  - Semantic search  │ │  - gh commands  │ │  - Extract facts    │
└──────────┬──────────┘ └────────┬────────┘ └──────────┬──────────┘
           │                     │                     │
           └─────────────────────┼─────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Context Assembly                              │
│                                                                  │
│  Combines: project manifest + retrieved files + search results  │
│           + tool outputs into structured context                 │
└─────────────────────────────────┬───────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                 TIER 3: Reasoning / Coding                       │
│              (Qwen 7B / DeepSeek 6.7B ~30-64 tok/s)             │
│                                                                  │
│  - Analyze code with full context                                │
│  - Generate solutions                                            │
│  - Create PRs                                                    │
│  - Complex reasoning                                             │
└─────────────────────────────────────────────────────────────────┘
```

---

## Model Assignments

| Tier | Role | Model | Speed | VRAM |
|------|------|-------|-------|------|
| 1 | Router/Classifier | TinyLlama | ~150 tok/s | 1GB |
| 2 | File operations | llama3.2:3b | ~60 tok/s | 2.5GB |
| 2 | Search summarization | qwen2.5-coder:3b | ~64 tok/s | 2.5GB |
| 2 | Embeddings | nomic-embed-text | N/A | 0.5GB |
| 3 | Code generation | qwen2.5-coder:7b | ~32 tok/s | 6GB |
| 3 | Tool calling | llama3-groq-tool-use:8b | ~30 tok/s | 6GB |
| 3 | Complex reasoning | qwen3:8b | ~30 tok/s | 6.5GB |

---

## Projects Manifest

Location: `~/Code/projects_manifest.json`

This file tells models about available projects without them having to discover everything.

```json
{
  "version": "1.0",
  "base_path": "/home/mgilbert/Code",
  "projects": [
    {
      "name": "clood",
      "path": "clood",
      "description": "Local LLM infrastructure and tooling - Claude-like experience on local hardware",
      "languages": ["markdown", "json", "python", "bash"],
      "key_files": [
        "README.md",
        "ARCHITECTURE.md",
        "ollama-tuning.md",
        "clood-cli/docs/USAGE_GUIDE.md"
      ],
      "tags": ["llm", "infrastructure", "tooling", "ollama", "clood"]
    },
    {
      "name": "example-project",
      "path": "example-project",
      "description": "Description of what this project does",
      "languages": ["typescript", "python"],
      "key_files": ["README.md", "src/index.ts", "package.json"],
      "tags": ["web", "api"]
    }
  ],
  "last_updated": "2025-12-12"
}
```

### Manifest Usage

Models can be instructed to:
1. Read the manifest first to understand available projects
2. Use tags to find relevant projects for a query
3. Read key_files for quick project understanding
4. Use the full path to access project files via MCP

---

## MCP Server Integration

MCP tools available in clood:

| Server | Purpose | Tier |
|--------|---------|------|
| `filesystem` | Read/write ~/Code | 2-3 |
| `searxng` | Web search | 2 |
| `github` | gh CLI commands | 3 |

### Potential Additional MCP Servers

| Server | Purpose | Package |
|--------|---------|---------|
| `memory` | Persistent context across sessions | @modelcontextprotocol/server-memory |
| `postgres` | Database queries | @modelcontextprotocol/server-postgres |
| `puppeteer` | Web scraping/automation | @modelcontextprotocol/server-puppeteer |

---

## Workflow Examples

### Example 1: "Help me fix the bug in project X"

```
1. [TinyLlama] Classify: This is a code task for project X
2. [Embeddings] Find: Relevant files in project X
3. [3B Model] Read: Key files, error logs
4. [SearXNG] Search: Similar bug reports if needed
5. [7B Model] Analyze: Full context assembled
6. [7B Model] Generate: Fix with explanation
7. [gh MCP] Create: PR with changes
```

### Example 2: "What's the best way to implement feature Y?"

```
1. [TinyLlama] Classify: Research + code generation
2. [SearXNG] Search: Best practices for feature Y
3. [3B Model] Summarize: Search results into bullet points
4. [Filesystem] Read: Existing related code
5. [7B Model] Synthesize: Recommendation with code samples
```

### Example 3: "List my projects and their status"

```
1. [TinyLlama] Classify: Simple query, no heavy model needed
2. [Filesystem] Read: projects_manifest.json
3. [3B Model] Format: Human-readable summary
```

---

## Multi-Machine Distribution

```
┌─────────────────────────────────────────────────────────────────┐
│                        Home Network                              │
│                                                                  │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐    │
│  │    ubuntu25    │  │  MacBook Air   │  │   Mac Mini     │    │
│  │   (primary)    │  │   (mobile)     │  │  (secondary)   │    │
│  │                │  │                │  │                │    │
│  │  RX 590 8GB    │  │  M4 16GB       │  │  M4 24GB       │    │
│  │  i7-8086K      │  │  unified mem   │  │  unified mem   │    │
│  │  64GB RAM      │  │                │  │                │    │
│  │                │  │                │  │                │    │
│  │  Services:     │  │  Can run:      │  │  Can run:      │    │
│  │  - Ollama      │  │  - Ollama      │  │  - Ollama      │    │
│  │  - SearXNG     │  │  - clood       │  │  - clood       │    │
│  │  - clood       │  │                │  │  - Larger      │    │
│  │                │  │  Connects to:  │  │    models      │    │
│  │  :11434        │  │  ubuntu25 for  │  │                │    │
│  │  :8888         │  │  search/models │  │  :11434        │    │
│  └────────────────┘  └────────────────┘  └────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Load Distribution Strategy

| Task | Primary Machine | Fallback |
|------|-----------------|----------|
| Web search | ubuntu25 (SearXNG) | - |
| Small models (1-3B) | Local | ubuntu25 |
| Medium models (7-8B) | Local GPU | ubuntu25 |
| Large models (14B+) | Mac Mini (24GB) | ubuntu25 CPU |

---

## Implementation Roadmap

### Phase 1: Foundation (Current)
- [x] Ollama running with GPU acceleration
- [x] clood CLI with MCP server capability
- [x] Multi-host routing configuration
- [x] Documentation framework

### Phase 2: Project Awareness
- [x] Create projects_manifest.json
- [ ] Script to auto-generate manifest from ~/Code
- [ ] Instruct models to read manifest first

### Phase 3: Tiered Inference
- [ ] Pull and configure tiered models
- [ ] Create routing prompts for TinyLlama
- [ ] Test context assembly workflow

### Phase 4: Multi-Machine
- [ ] Configure Macs to use ubuntu25 services
- [ ] Test remote Ollama access
- [ ] Document latency expectations

### Phase 5: Custom Tooling
- [ ] Build project-specific MCP servers if needed
- [ ] Create embeddings index for code search
- [ ] Automate common workflows

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `ARCHITECTURE.md` | This file - system design |
| `clood-cli/docs/USAGE_GUIDE.md` | How to use clood |
| `ollama-tuning.md` | Performance optimization |
| `hardware/i7-8086k.md` | CPU documentation |
| `~/Code/projects_manifest.json` | Project registry |
