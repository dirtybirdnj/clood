# LAST SESSION - The Conductor's Baton

**Date:** December 23, 2025
**Session:** Orchestrator Agent & Conductor Catfight
**Status:** Ubuntu25 now has a brain. Mac-laptop provides the muscle.

---

## THE BIG PICTURE

Ubuntu25 is now a **conductor** - a lightweight orchestration agent that:
- Receives tasks via SSH
- Delegates heavy coding to mac-laptop's big models (qwen2.5-coder:32b)
- Writes files to `/data/repos/workspace/`
- Runs git operations locally

```
┌─────────────────────────────────────────────────────────────────┐
│  MAC LAPTOP                                                      │
│  ┌─────────────┐         ┌──────────────────────────────────┐   │
│  │   Crush     │ ──SSH──▶│  Ubuntu25: Conductor             │   │
│  │  (You chat) │         │  (llama3-groq-tool-use:8b)       │   │
│  └─────────────┘         │  - Orchestrates tasks            │   │
│                          │  - Writes files to /workspace    │   │
│  ┌─────────────┐         │  - Git operations                │   │
│  │  Ollama     │◀────────│                                  │   │
│  │  32B models │ delegate│  Delegates heavy coding BACK     │   │
│  │  (the beef) │─────────▶  to laptop's big GPU            │   │
│  └─────────────┘         └──────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## KEY DISCOVERIES

### 1. Conductor Catfight Results

Not all models can orchestrate. We benchmarked tool-calling ability:

| Model | Behavior | Score |
|-------|----------|-------|
| **llama3-groq-tool-use:8b** | Actually invokes tools | BEST for agentic loops |
| mistral:7b | Describes tools but doesn't call them | Good single-turn only |
| qwen2.5-coder:* | Terrible at tool format | AVOID for orchestration |
| codestral:* | No tool calls at all | AVOID |

**Key insight:** "Coder" models are bad conductors. They're optimized for generating code, not orchestrating tools.

### 2. llama.cpp vs Ollama Benchmarks

On ubuntu25 (RX 590 8GB):

| Model | Ollama | llama.cpp | Winner |
|-------|--------|-----------|--------|
| TinyLlama 1.1B | 183 tok/s | 145 tok/s | Ollama (+27%) |
| Qwen 7B | 7.4 tok/s | 30.5 tok/s | **llama.cpp (+311%)** |

**Key insight:** llama.cpp dominates for 7B+ models on RX 590.

### 3. The Orchestrator Architecture

Ubuntu25 runs a lightweight conductor (8B model) that:
- Uses minimal VRAM for orchestration
- Delegates heavy coding to mac-laptop's 32B models
- Keeps all file I/O local to ubuntu25

---

## FILES CREATED THIS SESSION

| File | Purpose |
|------|---------|
| `scripts/orchestrator.py` | The agentic conductor agent |
| `scripts/conductor-catfight.py` | Benchmark for conductor models |
| `scripts/benchmark-backends.sh` | Ollama vs llama.cpp (TinyLlama) |
| `scripts/benchmark-7b.sh` | Ollama vs llama.cpp (7B models) |
| `clood-cli/internal/commands/huggingface.go` | GGUF model management |

---

## HOW TO USE THE ORCHESTRATOR

### From mac-laptop (via Crush/Claude):

```bash
# SSH to ubuntu25 and run the orchestrator
ssh ubuntu25 "cd /data/repos/workspace && python3 orchestrator.py 'Create a todo app with add/remove functionality'"
```

### Direct on ubuntu25:

```bash
cd /data/repos/workspace
python3 orchestrator.py 'Create hello.html with nice styling'
python3 orchestrator.py --conductor mistral:7b 'List files in workspace'
```

### Orchestrator Options:

```
--conductor, -c    Conductor model (default: llama3-groq-tool-use:8b)
--max-iterations   Max agent iterations (default: 10)
```

---

## MODELS DOWNLOADING OVERNIGHT

A queue script is running on ubuntu25:

```bash
# Check progress
ssh ubuntu25 "cat /data/repos/workspace/pull-models.log"
```

Models in queue:
- qwen3:8b (best tool-caller, F1=0.933)
- hermes3:8b (NousResearch, trained for tools)
- phi4
- gemma2:9b
- granite3.1-dense:8b
- yi:9b
- qwen2.5:14b (VRAM stress test)

---

## CONDUCTOR TOOL CAPABILITIES

The orchestrator has these tools:

| Tool | Description |
|------|-------------|
| `delegate_coding(prompt, output_file)` | Send to mac-laptop's 32B model, auto-save |
| `read_file(path)` | Read from workspace |
| `write_file(path, content)` | Write to workspace |
| `list_directory(path)` | List files |
| `git_status()` | Check git state |
| `git_commit(message)` | Stage and commit |
| `git_push()` | Push to remote |
| `task_complete(summary)` | Signal done |

---

## CONFIGURATION

### Orchestrator Config (`scripts/orchestrator.py`):

```python
ORCHESTRATOR_URL = "http://localhost:11434"  # Ollama on ubuntu25
ORCHESTRATOR_MODEL = "llama3-groq-tool-use:8b"  # Conductor

CODER_HOSTS = {
    "mac-laptop": "http://192.168.4.47:11434",
    "mac-mini": "http://192.168.4.41:11434",
    "ubuntu25": "http://localhost:11434",
}

CODER_MODEL = "qwen2.5-coder:32b"  # Heavy lifting on mac-laptop
WORKSPACE = Path("/data/repos/workspace")
```

---

## NEXT STEPS

1. **Configure Crush on mac-laptop** - Add orchestrator awareness to CLAUDE.md
2. **Run conductor catfight** with new models (qwen3:8b, hermes3:8b)
3. **Test VRAM limits** with 14B+ models on ubuntu25
4. **Add more tools** to orchestrator (run tests, lint, etc.)

---

## RESUME PROMPTS

**To test the orchestrator:**
> SSH to ubuntu25 and use the orchestrator to create a simple calculator.html

**To run the catfight:**
> Run the conductor catfight benchmark with the newly downloaded models

**To check downloads:**
> Check the model download progress on ubuntu25

---

## COMMITS THIS SESSION

| Hash | Description |
|------|-------------|
| `a29d522` | feat: Add orchestrator agent and conductor benchmarking |

---

## GITHUB ISSUES

- **#187** - [EPIC] llama.cpp Integration for High-Performance Local Inference

---

```
Conductor's baton raised—
tool-use models lead the way,
coders write, not wave.
```

---

*The server garden has a new brain.*
*Ubuntu25 thinks. Mac-laptop computes.*
*The symphony plays on.*
