# clood as a Public Project

Vision for making clood useful to others beyond the original author's hardware.

---

## What clood IS

**clood = docs + config**

Not a framework. Not an application. A curated collection of:
- Configuration templates (clood, Ollama, SearXNG)
- Prompts and seed artifacts for local LLM workflows
- Hardware optimization guides
- Workflow documentation

## What clood is NOT

- Not another Claude Code clone (use [OpenCode](https://github.com/xichen1997/opencode), [Aider](https://aider.chat), [Cline](https://cline.bot))
- Not a web UI (use [Open WebUI](https://openwebui.com), [Lobe Chat](https://lobehub.com))
- Not locked to specific hardware

---

## Core Principles

### 1. Hardware Agnostic
The repo should work on any machine that can run Ollama:
- Apple Silicon Macs (M1-M4)
- AMD GPUs (ROCm or Vulkan)
- NVIDIA GPUs (CUDA)
- CPU-only systems

Hardware-specific docs belong in `hardware/` with clear machine identifiers.

### 2. Tool Agnostic
clood is the current default, but clood patterns should work with:
- Aider (terminal + Git)
- OpenCode (Claude Code clone)
- Any MCP-compatible client

### 3. Model Agnostic
Prompts should work with any capable local model:
- Qwen 2.5 Coder
- DeepSeek Coder
- Llama 3.x
- Mistral/Mixtral
- CodeGemma

---

## Directory Structure Philosophy

```
clood/
├── prompts/           # Reusable system prompts (portable)
├── seeds/             # Artifact templates (portable)
├── skills/            # Agent capabilities (portable)
│
├── infrastructure/    # Config templates (adapt to your setup)
│   └── configs/       # clood, SearXNG, etc.
│
├── hardware/          # Machine-specific (BYOH - Bring Your Own Hardware)
│   ├── OPTIMIZATION-GUIDE.md   # General principles
│   └── [machine-name].md       # Specific tuning
│
└── docs/              # Workflow guides (universal concepts)
```

### What's Portable (Copy Anywhere)
- `prompts/` - System prompts for code tasks
- `seeds/` - Skeletons, tests, checklists, transforms
- `skills/` - Slash commands, agent instructions

### What Needs Adaptation
- `infrastructure/configs/` - Paths, URLs, model names
- `hardware/` - Your specific machine profiles

---

## Contribution Guidelines

### Adding Hardware Profiles
1. Create `hardware/[machine-name].md`
2. Include: CPU, GPU, RAM specs
3. Document what works (and what doesn't)
4. Add benchmarks with specific models

### Adding Prompts
1. Place in `prompts/local-llm/`
2. Test with at least 2 different models
3. Include example input/output
4. Note any model-specific quirks

### Adding Seeds
1. Choose the right type:
   - `skeletons/` - Code with TODOs
   - `tests/` - Test-first specs
   - `checklists/` - Review criteria
   - `transforms/` - Refactoring rules
2. Keep them small and focused
3. Document the intended use case

---

## Known Limitations

### clood MCP Issues (Dec 2025)
- [Global MCP tools not loaded with project config](https://github.com/charmbracelet/clood/issues/870)
- [Broken pipe errors](https://github.com/charmbracelet/clood/issues/840)
- [Transport initialization failures](https://github.com/charmbracelet/clood/issues/475)

**Workaround:** Use Aider for file operations, clood for chat.

### Model Tool-Use Reliability
Not all "tool-capable" models reliably use tools:
- `llama3-groq-tool-use:8b` - Sometimes refuses or asks instead of acting
- `qwen2.5-coder` - Better at code, worse at tools
- `mistral` - General purpose, tool support varies

**Best current option:** Test with your specific workflow.

---

## Roadmap to 1.0

### Phase 1: Clean Separation (Current)
- [ ] Move all ubuntu25-specific content to `hardware/`
- [ ] Make `infrastructure/configs/` use placeholder paths
- [ ] Add setup script that asks for machine-specific values

### Phase 2: Multi-Tool Support
- [ ] Document Aider integration alongside clood
- [ ] Add OpenCode config template
- [ ] Create tool-agnostic prompt format

### Phase 3: Community Templates
- [ ] Hardware profile template
- [ ] Prompt submission guidelines
- [ ] Seed artifact examples (10+ per type)

### Phase 4: Automation
- [ ] Setup wizard script
- [ ] Model recommendation based on hardware
- [ ] Auto-generate hardware profile from system info

---

## Similar Projects

If clood isn't what you need:

| Project | Best For |
|---------|----------|
| [Aider](https://aider.chat) | Git-integrated coding agent |
| [OpenCode](https://github.com/xichen1997/opencode) | Claude Code experience locally |
| [Cline](https://cline.bot) | VS Code extension |
| [AnythingLLM](https://anythingllm.com) | All-in-one desktop app |
| [Open WebUI](https://openwebui.com) | Web interface with RAG |

---

## License

MIT - Use freely, contribute back if you can.
